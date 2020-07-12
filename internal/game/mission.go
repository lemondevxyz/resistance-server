package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

	g.log.Debug("g.runMission(%d, %d):", ri, mi)

	// if the captain is invalid then set a new captain
	captain := g.GetCaptain()
	if !captain.IsValid() {
		g.captain = rand.Intn(len(g.Players))
	}

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "choose",
		Body:  captain.ID,
	})

	g.log.Debug("captain = @%s#%s", captain.User.Username, captain.User.Discriminator)

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "votemax",
		Body:  g.Rounds[ri].Assignees,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	g.startChoosingPhase(cancel, ri, mi)
	<-ctx.Done()
	captain.Conn.RemoveCommandsByGroup("game")

	// names of the assignees
	assignees := []string{}
	for _, id := range g.Rounds[ri].Missions[mi].Assignees {
		// because they're stored in id form we have to loop
		p := g.GetPlayer(id)
		if !p.IsValid() {
			g.log.Debug("!p.IsValid: %s", id)
			continue
		}

		assignees = append(assignees, fmt.Sprintf("@%s#%s", p.User.Username, p.User.Discriminator))
	}

	g.log.Debug("assignees = %v", assignees)

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "vote",
		Body:  g.Rounds[ri].Missions[mi].Assignees,
	})

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute*3)

	g.startVotingPhase(cancel, ri, mi)
	<-ctx.Done()
	for _, v := range g.Players {
		if v.IsValid() {
			v.Conn.RemoveCommandsByGroup("game")
		}
	}

	msh := g.Rounds[ri].Missions[mi]

	g.log.Debug("missions[%d].IsAccepted: %t", mi, msh.IsAccepted())
	if msh.IsAccepted() {
		// If the mission was accepted by players
		// return true as to proceed to roundVote
		b = true
	} else {
		// else set the new captain as the next player in line
		i := g.GetPlayerIndex(captain.ID) + 1
		if len(g.Players) <= i {
			i = 0
		}

		g.SetCaptain(i)
	}

	return
}

// startChoosingPhase is a method for adding the game_choose method.
// This function is called whenever the captain is set. which is whenever there is a mission avaliable.
func (g *Game) startChoosingPhase(cancel context.CancelFunc, ri, mi int) {
	// We don't need to validate because we already validated above
	captain := g.GetCaptain()
	captain.Conn.AddCommand("game", conn.MessageStruct{
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

			var ids []string
			for _, v := range arr {
				p := g.GetPlayer(v)
				// if one of the players is not valid then cancel the whole function
				if !p.IsValid() {
					return ErrInvalidPlayer
				}
				// else assign the real player to the array
				ids = append(ids, v)
			}

			g.Rounds[ri].Missions[mi].Assignees = ids

			cancel()

			return nil
		},
	})
}

// startVotingPhase is called whenever the mission assignees have been set. It sets a command 'game.vote' to get all players' votes
func (g *Game) startVotingPhase(cancel context.CancelFunc, ri, mi int) {
	for _, v := range g.Players {
		if v.IsValid() {
			v.Conn.AddCommand("game", conn.MessageStruct{
				"vote": func(log logger.Logger, bytes []byte) error {
					var accept bool

					err := json.Unmarshal(bytes, &accept)
					if err != nil {
						return fmt.Errorf("json.Unmarshal: %w", err)
					}

					if accept {
						g.Rounds[ri].Missions[mi].Accept = append(g.Rounds[ri].Missions[mi].Accept, v.ID)
					} else {
						g.Rounds[ri].Missions[mi].Decline = append(g.Rounds[ri].Missions[mi].Decline, v.ID)
					}

					mission := g.Rounds[ri].Missions[mi]
					want := len(mission.Accept) + len(mission.Decline)
					have := len(g.Players)

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
