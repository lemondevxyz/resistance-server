package game

import (
	"encoding/json"
	"fmt"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/logger"
)

func getChooseFunc(lp *loopParameter) conn.MessageCallback {
	return func(log logger.Logger, body []byte) (err error) {
		players := *lp.players

		var id string

		err = json.Unmarshal(body, &id)
		if err != nil {
			return fmt.Errorf("json.Unmarshal: game.choose: %v", err)
		}

		captain := id
		// if this player is the captain
		// this part doesn't change per the loop index
		if captain == lp.player.GetClient().ID {

			// avoid duplicates
			exists := map[string]bool{}
			ids := []string{}

			add := func(max uint8) {
				var added uint8
				for _, v := range players {
					// avoid adding any spies cause we'll add them later.
					playa, ok := lp.g.Players[cn[v].GetClient().ID]
					if !ok {
						continue
					}

					if added == max {
						return
					}

					if lp.index == 1 {
						// for index 1, we want as many resistance as possible.
						// we'll add a spy later
						if playa.Type == game.PlayerTypeSpy || playa.Type == game.PlayerTypeMorgana {
							continue
						}
						// for index 2, we want as many resistance as possible.
						// no spies for this one
					} else if lp.index == 2 {
						if playa.Type != game.PlayerTypeResistance && playa.Type != game.PlayerTypeMerlin {
							continue
						}
					}

					exists[cn[v].GetClient().ID] = true
					ids = append(ids, cn[v].GetClient().ID)

					added++
				}

			}

			lp.mtx.Lock()
			if lp.index == 0 || lp.index == 2 {
				// index 0 add random players
				add(lp.testgame.Rounds[lp.round].Assignees)
			} else if lp.index == 1 {
				// index 1 add random players and atleast one spy
				add(lp.testgame.Rounds[lp.round].Assignees - 1)
				for _, v := range lp.g.Players {
					_, ok := exists[v.GetClient().ID]
					if !ok {
						if v.Type == game.PlayerTypeSpy {
							ids = append(ids, v.GetClient().ID)
							break
						}
					}
				}
			}
			lp.mtx.Unlock()

			lp.vsn.WriteMessage(conn.MessageSend{
				Group: "game",
				Name:  "choose",
				Body:  ids,
			})

		}

		return
	}
}

func getVoteFunc(lp *loopParameter) conn.MessageCallback {
	return func(log logger.Logger, bytes []byte) (err error) {

		players := *lp.players

		have := []string{}
		err = json.Unmarshal(bytes, &have)
		if err != nil {
			err = fmt.Errorf("json.Unmarshal: %v", err)
			return
		}

		lp.mtx.Lock()
		asgn := lp.testgame.Rounds[lp.round].Assignees
		lp.mtx.Unlock()

		want := []string{}

		for _, v := range players[:asgn] {
			want = append(want, cn[v].GetClient().ID)
		}

		var equal bool
		var i int
		if len(have) == len(want) {
			for i = 0; i < len(have); i++ {

				v := have[i]
				h := have[i]

				if h != v {
					break
				}

				if i == len(have)-1 {
					i = -1
					break
				}

			}

			if i == -1 {
				equal = true
			}
		}

		if !equal {
			fmt.Println("something is wrong, want != have")
		} else {
			if lp.index == 0 {
				// index 0 fail every vote
				lp.vsn.WriteMessage(conn.MessageSend{
					Group: "game",
					Name:  "vote",
					Body:  false,
				})
			} else {
				// any other index accept the mission
				lp.vsn.WriteMessage(conn.MessageSend{
					Group: "game",
					Name:  "vote",
					Body:  true,
				})
			}
		}

		return
	}
}
