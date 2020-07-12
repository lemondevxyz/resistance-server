package lobby

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/logger"
)

func (l *Lobby) addCommands(c client.Client) {
	if !c.IsValid() {
		return
	}

	c.Conn.RemoveCommandsByGroup("lobby")

	strct := conn.MessageStruct{
		"leave": func(log logger.Logger, bytes []byte) error {
			return l.Leave(c)
		},

		"get": func(log logger.Logger, bytes []byte) error {
			return c.Conn.WriteMessage(l.MessageSend())
		},
	}

	if len(l.Clients) == 0 {
		return
	}

	if l.Clients[0] == c {
		strct["kick"] = func(log logger.Logger, bytes []byte) error {
			target := client.Client{}
			json.Unmarshal(bytes, &target)
			if len(target.ID) > 0 {
				// if the target is a valid client
				target = l.GetClient(target.ID)
				if !target.IsValid() {
					return fmt.Errorf("!target.IsValid")
				}

				l.Leave(target)
			}

			return nil
		}

		lobbylog := l.log
		strct["start"] = func(log logger.Logger, bytes []byte) error {
			gameoption := game.OptionNone

			err := json.Unmarshal(bytes, &gameoption)
			if err != nil {
				return fmt.Errorf("json.Unmarshal: %v", err)
			}

			lc := logger.DefaultConfig
			lc.PAttr = color.New(color.FgYellow, color.Italic)
			lc.Prefix = "game"
			lc.Suffix = l.log.GetSuffix()
			lc.Debug = true

			g, err := game.NewGame(logger.NewLogger(lc), l.Clients, l.Type.Common(), gameoption)
			if err != nil {
				return fmt.Errorf("game.NewGame: %w", err)
			}

			go func(g *game.Game) {
				s := make(chan game.Status)
				go g.Run(s)
				lobbylog.Info("g.Run: %v", <-s)
			}(g)

			return nil
		}
	}

	c.Conn.AddCommand("lobby", strct)

}
