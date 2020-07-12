package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/logger"
)

// WebsocketConfig is a data structure containing config to use in NewWebsocketRoute.
type WebsocketConfig struct {
	LobbyService  lobby.Service
	ClientService client.Service
	Log           logger.Logger
	GetUser       func(c *gin.Context) (discord.User, error)
	cl            client.Client
}

type context struct {
	*gin.Context
	WebsocketConfig
}

// marshalLobbies returns []byte of all the lobbies
func (c context) marshalLobbies() ([]byte, error) {
	lserv := c.LobbyService

	lls, err := lserv.GetAllLobbies()
	if err != nil {
		return nil, err
	}

	ms := conn.MessageSend{
		Group: "lobbies",
		Name:  "get",
		Body:  lls,
	}

	// marshal once
	body, err := json.Marshal(ms)
	if err != nil {
		return nil, err
	}

	return body, nil

}

// sendLobbies sends marshalLobbies for every client
func (c context) sendLobbies() {
	log := c.Log
	cserv := c.ClientService

	cls, err := cserv.GetAllClients()
	if err != nil {
		log.Danger("client.GetAllClient: %v", err)
		return
	}

	body, err := c.marshalLobbies()
	if err != nil {
		log.Warn("json.Marshal: %v", err)
		return
	}

	for _, v := range cls {
		if len(v.LobbyID) == 0 {
			if v.Conn != nil {
				v.Conn.WriteBytes(body)
			}
		}
	}

}

// leaveLobby leaves the current lobby, if there is on
func (c context) leaveLobby() {

	log := c.Log
	lserv := c.LobbyService

	cl := c.cl

	if len(cl.LobbyID) > 0 {
		l, err := lserv.GetLobbyByID(cl.LobbyID)
		if err != nil {
			log.Warn("lserv.GetLobbyByID: %v", err)
			return
		}

		l.Leave(cl)
	}
}

// addCommands adds: "lobby_create", "lobby_join", "lobbies_get"
func (c context) addCommands() {
	lserv, cserv := c.LobbyService, c.ClientService

	msgstrct := conn.MessageStruct{
		// This command creates a new lobby
		"create": func(log logger.Logger, bytes []byte) error {
			c.leaveLobby()

			l := new(lobby.Lobby)
			err := json.Unmarshal(bytes, l)
			if err != nil {
				return fmt.Errorf("json.Unmarshal: %w", err)
			}

			err = lserv.CreateLobby(l)
			if err != nil {
				return fmt.Errorf("lserv.CreateLobby: %w", err)
			}

			// goroutine to indicate any update in the amount of players
			// in-case there is, we update every client.
			insert, remove := l.Subscribe()
			go func(cserv client.Service, lserv lobby.Service, l *lobby.Lobby) {
				for {
					select {
					case <-insert:
						c.sendLobbies()
					case <-remove:
						c.sendLobbies()
					}
				}
			}(cserv, lserv, l)

			err = l.Join(c.cl)
			if err != nil {
				return fmt.Errorf("l.Join: %w", err)
			}

			return nil
		},

		// This command joins a lobby
		"join": func(log logger.Logger, bytes []byte) error {
			c.leaveLobby()

			l := new(lobby.Lobby)
			err := json.Unmarshal(bytes, l)
			if err != nil {
				return fmt.Errorf("json.Unmarshal: %w", err)
			}

			l, err = lserv.GetLobbyByID(l.ID)
			if err != nil {
				return fmt.Errorf("json.Unmarshal: %w", err)
			}

			err = l.Join(c.cl)
			if err != nil {
				return fmt.Errorf("l.Join: %v", err)
			}

			return nil
		},
	}

	c.cl.Conn.AddCommand("lobby", msgstrct)
	c.cl.Conn.AddCommand("lobbies", conn.MessageStruct{
		"get": func(log logger.Logger, _ []byte) error {
			body, err := c.marshalLobbies()
			if err != nil {
				return fmt.Errorf("c.marshalLobbies: %w", err)
			}

			c.cl.Conn.WriteBytes(body)

			return nil
		},
	})
}

// addDestroyHandler adds a destroy handler
func (c context) addDestroyHandler() {
	<-c.cl.Conn.GetDone()
	if len(c.cl.LobbyID) > 0 {
		c.leaveLobby()
	}
}

// NewWebsocketRoute returns a new websocket handler by providing a lobby a client and a logger.
func NewWebsocketRoute(config WebsocketConfig) gin.HandlerFunc {
	config.Log.Info("Initiated route")

	debuglobby := true
	if debuglobby {
		config.Log.Debug("Debugging lobby and game")
	}

	return func(ginc *gin.Context) {
		lserv, cserv := config.LobbyService, config.ClientService
		log := config.Log

		user, err := config.GetUser(ginc)
		if err != nil {
			log.Warn("getuser.err: %v", err)

			ginc.AbortWithStatus(http.StatusForbidden)
			return
		} else {
			if !user.IsValid() {
				log.Warn("!user.IsValid()")

				ginc.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		wsconn, _, _, err := ws.UpgradeHTTP(ginc.Request, ginc.Writer)

		if err != nil {
			log.Warn("cannot upgrade: %v", err)
			return
		}

		cl, err := cserv.CreateClient(wsconn, user)
		if err != nil {
			log.Warn("cannot create client: %v", err)
			return
		}

		config.cl = cl
		c := context{
			ginc,
			config,
		}

		go c.addCommands()
		go c.addDestroyHandler()

		body, err := c.marshalLobbies()
		if err != nil {
			log.Warn("c.marshalLobbies: %v", err)
			return
		}

		cl.Conn.WriteBytes(body)

		if debuglobby {
			ls, err := lserv.GetAllLobbies()
			if err != nil {
				log.Debug("lserv.GetAllLobbies: %v", err)
				return
			}

			if len(ls) > 0 {
				// if there is an existing lobby, add new clients to it
				l := ls[0]
				time.Sleep(time.Millisecond * 50)
				err = l.Join(cl)
				if err != nil {
					log.Debug("lobby.Join(%s): %v", cl.ID, err)
				}
			} else {
				// else just create a new lobby
				bytes, err := json.Marshal(&lobby.Lobby{
					Type: lobby.TypeBasic,
				})

				if err == nil {
					time.Sleep(time.Millisecond * 50)
					err = cl.Conn.ExecuteCommand("lobby", "create", bytes)
					if err != nil {
						log.Debug("c.Conn.ExecuteCommand: %v", err)
					}
				}
			}
		}

	}
}
