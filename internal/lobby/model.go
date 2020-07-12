package lobby

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
	"github.com/toms1441/resistance-server/internal/repo"
	"gopkg.in/go-playground/validator.v9"
)

// Lobby the lobby that the clients will be in, it gathers maximum of 10 players. and is used later to transform into a Game.
// The lobby's owner is client with index 0.
type Lobby struct {
	ID      string          `json:"id"`
	Type    Type            `json:"type"`
	Private bool            `json:"private"`
	Clients []client.Client `json:"clients"`

	insert []chan client.Client
	remove []chan client.Client

	log logger.Logger
}

// Type needs to be between TypeBasic(0) and TypeTrumpmode(4)
type Type uint8

func (t Type) String() string {
	switch t {
	case TypeBasic:
		return "Basic"
	case TypeOriginal:
		return "Original"
	case TypeAvalon:
		return "Avalon"
	case TypeHunter:
		return "Hunter"
	case TypeTrumpmode:
		return "Trumpmode"
	}

	return ""
}

func (t Type) Common() uint8 {
	return uint8(t)
}

const (
	// TypeBasic has spies and resistance.
	TypeBasic Type = iota
	// TypeOriginal no documentation yet.
	TypeOriginal
	// TypeAvalon same as TypeBasic only with merlin.
	// Merlin is a character that sees all spies and resistance.
	TypeAvalon
	// TypeHunter no documentation
	TypeHunter
	// TypeTrumpmode no documentation
	TypeTrumpmode
)

var validate *validator.Validate

var (
	// ErrID if len(id) != idlength.
	ErrID = errors.New("Lobby ID is not valid")
	// ErrType if type >= TypeBasic && type <= TypeTrumpmode
	ErrType = errors.New("Lobby Type is not valid")
	// ErrChannel if l.Insert == nil || l.Remove == ninl
	ErrChannel = errors.New("Lobby channels(l.Insert, l.Remove) are nil.")
)

// SetLogger sets the logger for the lobby.
// Used with lobby.service to provide better logs.
func (l *Lobby) SetLogger(log logger.Logger) {
	l.log = log
}

// GetClientIndex returns the client index by it's id.
func (l *Lobby) GetClientIndex(id string) int {
	for k, v := range l.Clients {
		if v.ID == id {
			return k
		}
	}

	return -1
}

// GetClient returns a client by it's id.
func (l *Lobby) GetClient(id string) (c client.Client) {
	c = client.Client{}
	if i := l.GetClientIndex(id); i >= 0 {
		c = l.Clients[i]
	}

	return
}

// Join inserts a client into the lobby, and updates the rest of the clients.
func (l *Lobby) Join(c client.Client) error {

	i := l.GetClientIndex(c.ID)
	if i == -1 {
		l.Clients = append(l.Clients, c)

		c.LobbyID = l.ID
		// update all lobby handlers
		for _, v := range l.insert {
			v <- c
		}

		l.addCommands(l.Clients[0])
		l.log.Debug("l.Join: %v", c.ID)

		// When the connection closes, remove the lobby.
		go func(l *Lobby, c client.Client, done chan bool) {
			for {
				select {
				case <-done:
					l.Leave(c)
					return
				default:
					if c.LobbyID != l.ID {
						return
					}
				}
			}
		}(l, c, c.Conn.GetDone())

		err := l.Send()
		if err != nil {
			l.log.Debug("l.Send: %v", err)
			return err
		}

		return nil
	}

	return repo.ErrClientExists
}

// Leave removes a client from the lobby, and updates the rest of the clients.
func (l *Lobby) Leave(c client.Client) error {

	i := l.GetClientIndex(c.ID)
	if i >= 0 {
		if len(l.Clients) == 0 {
			return nil
		}

		// it's just easier to copy the value and replace
		// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
		l.Clients[len(l.Clients)-1], l.Clients[i] = l.Clients[i], l.Clients[len(l.Clients)-1]
		l.Clients = l.Clients[:len(l.Clients)-1]

		// double-check cause we might've removed the last player
		if len(l.Clients) > 0 {
			firstc := l.Clients[0]
			if firstc.IsValid() {
				l.addCommands(firstc)
			}
		}

		c.LobbyID = ""
		for _, v := range l.remove {
			v <- c
		}

		l.log.Debug("l.Remove: %v", c.ID)
		err := l.Send()
		l.log.Debug("l.Send: %v", err)
		if err != nil {
			return err
		}

		return nil
	}

	return repo.ErrClient404
}

// Subscribe returns two channels, one for insert and remove.
// It's used to indicate a change in the client list
func (l *Lobby) Subscribe() (insert, remove chan client.Client) {
	return l.SubscribeInsert(), l.SubscribeRemove()
}

func (l *Lobby) SubscribeInsert() (insert chan client.Client) {
	insert = make(chan client.Client)
	if l.insert == nil {
		l.insert = []chan client.Client{}
	}

	l.insert = append(l.insert, insert)
	return insert
}

func (l *Lobby) SubscribeRemove() (remove chan client.Client) {
	remove = make(chan client.Client)

	if l.remove == nil {
		l.remove = []chan client.Client{}
	}

	l.remove = append(l.remove, remove)
	return remove
}

// Send send the clients information about the lobby, it's called whenever a client joins or leaves the lobby.
func (l *Lobby) Send() error {
	bytes, err := json.Marshal(l.MessageSend())

	if err != nil {
		l.log.Debug("l.Send: %v", err)
		return err
	}

	for _, v := range l.Clients {
		v.Conn.WriteBytes(bytes)
	}

	return nil
}

// MessageSend is a method that returns conn.MessageSend
func (l *Lobby) MessageSend() conn.MessageSend {
	return conn.MessageSend{
		Group: "lobby",
		Name:  "get",
		Body:  l,
	}
}

// Validate validates the lobby, it includes ID and Type validators as well as validate.Struct(lobby)
func (l *Lobby) Validate() (err error) {
	if validate == nil {
		validate = validator.New()
	}

	if err = validate.Struct(l); err != nil {
		return
	}

	_, err = strconv.Atoi(l.ID)
	// i.e 	the length of digits is same of LOBBY_ID_LENGTH
	// 		and the charaters are all numbers
	if len(l.ID) != gConfig.IDLen || err != nil {
		return ErrID
	}

	// i.e	it's between BASIC(which is 0) and TRUMPMODE(which is 4)
	if l.Type < TypeBasic || l.Type > TypeTrumpmode {
		return ErrType
	}

	return
}
