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

func (l *Lobby) addCommands(c conn.Conn) {
	cl := c.GetClient()
	if !cl.IsValid() {
		return
	}

	c.RemoveCommandsByGroup("lobby")

	strct := conn.MessageStruct{
		"leave": func(log logger.Logger, bytes []byte) error {
			return l.Leave(c)
		},

		"get": func(log logger.Logger, bytes []byte) error {
			return c.WriteMessage(l.MessageSend())
		},
	}

	if len(l.conns) == 0 {
		return
	}

	tempcl, ok := l.conns[cl.ID]
	if ok && tempcl == c {
		strct["kick"] = func(log logger.Logger, bytes []byte) error {
			target := client.Client{}
			json.Unmarshal(bytes, &target)
			if len(target.ID) > 0 {
				// if the target is a valid client
				targetconn, ok := l.conns[target.ID]
				if !ok {
					return fmt.Errorf("invalid client")
				}

				l.Leave(targetconn)
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

			g, err := game.NewGame(l.conns, l.Type.Common(), gameoption)
			if err != nil {
				return fmt.Errorf("game.NewGame: %w", err)
			}
			g.SetLogger(logger.NewLogger(lc))

			go func(g *game.Game) {
				s := make(chan game.Status)
				go g.Run(s)
				lobbylog.Info("g.Run: %v", <-s)
			}(g)

			return nil
		}
	}

	c.AddCommand("lobby", strct)

}
