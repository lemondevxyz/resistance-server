package game

import (
	"encoding/json"
	"fmt"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/logger"
)

func getDecideFunc(lp *loopParameter) conn.MessageCallback {
	return func(log logger.Logger, body []byte) error {
		lp.mtx.Lock()
		lp.mission++
		lp.mtx.Unlock()

		ids := []string{}
		if err := json.Unmarshal(body, &ids); err != nil {
			return fmt.Errorf("json.Unmarshal: %w", err)
		}

		isassignee := false
		for _, v := range ids {
			if v == lp.player.GetClient().ID {
				isassignee = true
				break
			}
		}

		//playertype := g.Players[playerid].Type
		if isassignee {
			if lp.index == 1 || lp.index == 2 {
				// one of this assignees will be a spy
				lp.vsn.WriteMessage(conn.MessageSend{
					Group: "game",
					Name:  "decide",
					Body:  false,
				})
			}
		}

		return nil
	}
}

func getRoundFunc(lp *loopParameter) conn.MessageCallback {
	return func(log logger.Logger, body []byte) error {

		bodyround := game.Round{}
		if err := json.Unmarshal(body, &bodyround); err != nil {
			return fmt.Errorf("json.Unmarshal: %w", err)
		}

		lp.mtx.Lock()
		lp.rounds = append(lp.rounds, bodyround)

		lp.round++
		// reset the mission
		lp.mission = 0
		lp.mtx.Unlock()

		return nil
	}
}
