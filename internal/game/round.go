package game

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
)

// RoundDefaults is map containing 5 Rounds represented by int.
// It's used to determine how many Assignees are required and how many players are needed to Represent a round failure.
var RoundDefaults = map[int][5]Round{
	5: {
		{
			Assignees:  2,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  2,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
	},
	// 5: 2, 3, 2, 3, 3

	6: {
		{
			Assignees:  2,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  4,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  4,
			MinFailure: 1,
		},
	},
	// 6: 2, 3, 4, 3, 4

	7: {
		{
			Assignees:  2,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  4,
			MinFailure: 2,
		},
		{
			Assignees:  4,
			MinFailure: 1,
		},
	},
	// 7: 2, 3, 3, 4*, 4

	8: {
		{
			Assignees:  3,
			MinFailure: 1,
		},
		{
			Assignees:  4,
			MinFailure: 1,
		},
		{
			Assignees:  4,
			MinFailure: 1,
		},
		{
			Assignees:  5,
			MinFailure: 2,
		},
		{
			Assignees:  5,
			MinFailure: 1,
		},
	},
	// 8: 3, 4, 4, 5*, 5
	// this is used for 9 players and 10 players aswell
}

// Won is a method that returns a boolean value if the resistance
func (r Round) GetConculsion() Status {

	var missionDeclined int
	// loop over the missions in-order to determine if the spies won
	// if all the missions(5) have been declined then the spies won
	for _, v := range r.Missions {
		// if the mission has been occupied by assignees
		if !v.IsEmpty() {
			// if the mission has been declined
			if !v.IsAccepted() {
				missionDeclined++
			}
		}
	}

	if missionDeclined == 5 {
		return StatusLost
	}

	if r.Failure >= r.MinFailure {
		return StatusLost
	} else {
		return StatusWon
	}

	return StatusDefault
}

// b = if mi == 5
func (g *Game) runRound(ri int) (b bool) {
	var mi = 0
	for ; mi < 5; mi++ {
		// if a mission has been successful then break the loop
		success := g.runMission(ri, mi)
		g.log.Debug("g.runMission(%d, %d): %v", ri, mi, success)
		if success {
			break
		}
	}
	g.log.Debug("g.runMission(%d)", ri)

	if mi == 5 {
		g.log.Debug("mi == 5", ri)
		return true
	}

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
		Name:  "decide",
		Body:  g.Rounds[ri].Missions[mi].Assignees,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	g.startDecidingPhase(cancel, ri, mi)
	<-ctx.Done()
	// after it's done
	// send the round result
	g.Rounds[ri].Failure = uint8(len(g.Rounds[ri].failure))

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "round",
		Body:  g.Rounds[ri],
	})

	g.log.Debug("g.Rounds[%d].GetConculsion: %v", ri, g.Rounds[ri].GetConculsion())

	return false
}

func (g *Game) startDecidingPhase(cancel context.CancelFunc, ri int, mi int) error {

	var assignees = g.Rounds[ri].Missions[mi].Assignees

	for _, v := range assignees {
		p := g.GetPlayer(v)

		if !p.IsValid() {
			return ErrInvalidPlayer
		}

		p.Conn.AddCommand("game", conn.MessageStruct{
			"decide": func(log logger.Logger, bytes []byte) error {

				// automatically make the round successful
				success := true

				// unless you are a spy >:)
				if p.Type == PlayerTypeSpy || p.Type == PlayerTypeMorgana {
					err := json.Unmarshal(bytes, &success)
					if err != nil {
						return fmt.Errorf("json.Unmarshal: %v", err)
					}
				}

				if !success {
					g.Rounds[ri].failure = append(g.Rounds[ri].failure, p.ID)
				} else {
					g.Rounds[ri].success = append(g.Rounds[ri].success, p.ID)
				}

				round := g.Rounds[ri]
				want := len(round.success) + len(round.failure)
				have := int(round.Assignees)
				if have != want {
					return fmt.Errorf("want: %d, have: %d", want, have)
				}

				cancel()
				return nil
			},
		})
	}

	return nil
}