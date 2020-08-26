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

// GetConculsion is a method that returns a boolean value if the resistance
func (r Round) GetConculsion() Status {

	var missionDeclined int
	// loop over the missions in-order to determine if the spies won
	// if all the missions(5) have been declined then the spies won

	if r.Missions[0].IsEmpty() {
		return StatusDefault
	}

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

}

// b = if mi == 5
func (g *Game) runRound(ri int) (b bool) {
	var mi = 0
	for ; mi < 5; mi++ {
		// if a mission has been successful then break the loop
		success := g.runMission(ri, mi)
		g.log.Debug("end of g.runMission(%d, %d): %t", ri, mi, success)
		if success {
			break
		}
	}
	g.log.Debug("g.runMission(%d)", ri)

	if mi == 5 {
		g.log.Debug("mi == 5")
		return true
	}

	assignees := []string{}
	for _, id := range g.Rounds[ri].Missions[mi].Assignees {
		// because they're stored in id form we have to loop
		p, ok := g.Players[id]
		if ok {
			if !p.IsValid() {
				g.log.Debug("!p.IsValid: %s", id)
				continue
			}
		}

		assignees = append(assignees, fmt.Sprintf("@%s#%s", p.GetClient().Username, p.GetClient().Discriminator))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

	g.startDecidingPhase(cancel, ri, mi)

	g.log.Debug("assignees = %v", assignees)

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "decide",
		Body:  g.Rounds[ri].Missions[mi].Assignees,
	})

	<-ctx.Done()
	// after it's done
	// send the round result
	g.mtx.Lock()
	g.Rounds[ri].Failure = uint8(len(g.Rounds[ri].failure))
	g.mtx.Unlock()

	g.Broadcast(conn.MessageSend{
		Group: "game",
		Name:  "round",
		Body:  g.Rounds[ri],
	})

	g.log.Debug("g.Rounds[%d].GetConculsion: %v", ri, g.Rounds[ri].GetConculsion())

	spies := 0
	resistance := 0

	for _, v := range g.Rounds {
		status := v.GetConculsion()

		if status == StatusWon {
			resistance++
		} else if status == StatusLost {
			spies++
		}
	}

	// gg
	if spies == 3 || resistance == 3 {
		return true
	}

	return false
}

func (g *Game) startDecidingPhase(cancel context.CancelFunc, ri int, mi int) error {

	var assignees = g.Rounds[ri].Missions[mi].Assignees

	for _, v := range assignees {
		p, ok := g.Players[v]

		if ok {
			if !p.IsValid() {
				return ErrInvalidPlayer
			}
		}

		p.AddCommand("game", conn.MessageStruct{
			"decide": func(log logger.Logger, bytes []byte) error {
				defer p.RemoveCommandsByNames("game", "decide")
				// in-case we got multiple executions at once

				// automatically make the round successful
				success := true

				// unless you are a spy >:)
				if p.Type == PlayerTypeSpy || p.Type == PlayerTypeMorgana {
					err := json.Unmarshal(bytes, &success)
					if err != nil {
						return fmt.Errorf("json.Unmarshal: %v", err)
					}
				}

				g.mtx.Lock()
				if !success {
					g.Rounds[ri].failure = append(g.Rounds[ri].failure, p.GetClient().ID)
				} else {
					g.Rounds[ri].success = append(g.Rounds[ri].success, p.GetClient().ID)
				}

				round := g.Rounds[ri]
				g.mtx.Unlock()
				want := int(round.Assignees)
				have := len(round.success) + len(round.failure)

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
