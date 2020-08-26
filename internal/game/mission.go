package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
)

var (
	// ErrMinAssignees is returned whenever Round.Assignees != len(plaayers)
	ErrMinAssignees = errors.New("Number of players does not match the minimum number of assignees")
	// ErrInvalidPlayer is returned whenever one or more players are invalid. it's mostly used in startChoosingPhase
	ErrInvalidPlayer = errors.New("One or more players are invalid")
)

// IsAccepted is a method that returns a boolean value that represents if the mission was accept or not.
func (m Mission) IsAccepted() bool {
	return len(m.Accept) > len(m.Decline)
}

// IsEmtpy is a method that returns a boolean value that represents if the
func (m Mission) IsEmpty() bool {
	return len(m.Accept) == 0 && len(m.Decline) == 0 && len(m.Assignees) == 0
}

func (g *Game) runMission(ri, mi int) (b bool) {

	g.log.Debug("start of runMission(%d, %d)", ri, mi)

	// if the captain is invalid then set a new captain
	if len(g.captain) == 0 {
		g.SetCaptain()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	g.startChoosingPhase(cancel, ri, mi)

	captain := g.Players[g.captain]
	g.log.Debug("captain = @%s#%s", captain.GetClient().Username, captain.GetClient().Discriminator)

	// inform the players that we're in the choosing phase
	// and give em the captain's ID
	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "choose",
		Body:  g.captain,
	})

	<-ctx.Done()
	captain.RemoveCommandsByNames("game", "choose")

	// names of the assignees
	assignees := []string{}
	for _, id := range g.Rounds[ri].Missions[mi].Assignees {
		// because they're stored in id form we have to loop
		p, ok := g.Players[id]
		if !ok {
			g.log.Debug("!p.IsValid: %s", id)
			continue
		}

		assignees = append(assignees, fmt.Sprintf("@%s#%s", p.GetClient().Username, p.GetClient().Discriminator))
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute*3)
	g.startVotingPhase(cancel, ri, mi)

	g.log.Debug("assignees = %v", assignees)

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "vote",
		Body:  g.Rounds[ri].Missions[mi].Assignees,
	})

	<-ctx.Done()
	for _, v := range g.Players {
		if v.IsValid() {
			v.RemoveCommandsByNames("game", "vote")
		}
	}

	msh := g.Rounds[ri].Missions[mi]

	g.log.Debug("missions[%d].IsAccepted: %t", mi, msh.IsAccepted())
	if msh.IsAccepted() {
		// If the mission was accepted by players
		// return true as to proceed to round decision
		b = true
	} else {
		// else just set a new captain and return false
		// this means that we get another mission in this round
		g.SetCaptain()
	}

	return
}

// startChoosingPhase is a method for adding the game_choose method.
// This function is called whenever the captain is set. which is whenever there is a mission avaliable.
func (g *Game) startChoosingPhase(cancel context.CancelFunc, ri, mi int) {
	// We don't need to validate because we already validated above
	captain, ok := g.Players[g.captain]
	if ok {
		captain.AddCommand("game", conn.MessageStruct{
			"choose": func(log logger.Logger, body []byte) error {
				var arr = []string{}

				err := json.Unmarshal(body, &arr)
				if err != nil {
					return fmt.Errorf("json.Unmarshal: %v", err)
				}

				var want = int(g.Rounds[ri].Assignees)
				var have = len(arr)

				if have != want {
					return fmt.Errorf("%w - want: %d, have: %d", ErrMinAssignees, want, have)
				}

				ids := []string{}
				exists := map[string]bool{}

				for _, v := range arr {
					_, ok := g.Players[v]
					if ok {
						// else assign the real player to the array
						if exists[v] == true {
							return fmt.Errorf("player id: %s is duplicated", v)
						}

						ids = append(ids, v)
						exists[v] = true
					} else {
						return ErrInvalidPlayer
					}
				}

				g.mtx.Lock()
				g.Rounds[ri].Missions[mi].Assignees = ids
				g.mtx.Unlock()

				cancel()

				return nil
			},
		})
	}
}

// startVotingPhase is called whenever the mission assignees have been set. It sets a command 'game.vote' to get all players' votes
func (g *Game) startVotingPhase(cancel context.CancelFunc, ri, mi int) {

	for _, v := range g.Players {
		if v.IsValid() {
			v.AddCommand("game", conn.MessageStruct{
				"vote": func(log logger.Logger, bytes []byte) error {
					// for when we have multiple commands executing at once

					var accept bool

					err := json.Unmarshal(bytes, &accept)
					if err != nil {
						return fmt.Errorf("json.Unmarshal: %w", err)
					}

					g.mtx.Lock()
					if accept {
						g.Rounds[ri].Missions[mi].Accept = append(g.Rounds[ri].Missions[mi].Accept, v.GetClient().ID)
					} else {
						g.Rounds[ri].Missions[mi].Decline = append(g.Rounds[ri].Missions[mi].Decline, v.GetClient().ID)
					}

					mission := g.Rounds[ri].Missions[mi]
					g.mtx.Unlock()

					want := len(mission.Accept) + len(mission.Decline)
					have := len(g.Players)

					// ensure there aren't any duplicates
					// once we're close to return just remove the handler
					//fmt.Println(v.GetClient().ID)
					if want != have {
						return fmt.Errorf("want: %d, have: %d", want, have)
					}

					cancel()
					return nil
				},
			})
		}
	}
}
