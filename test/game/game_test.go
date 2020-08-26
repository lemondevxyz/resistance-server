package game

// this file is concerned with game-logic
import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/logger"
)

type TestGame struct {
	Type    game.Type      `json:"type"`
	Rounds  [5]game.Round  `json:"rounds"`
	Players map[string]int `json:"players"`
	Option  game.Option    `json:"option"`
}

var singlegame *game.Game

func TestGameSend(t *testing.T) {
	validateplayerhave := func(ps map[string]int) (vp validateplayer) {
		for _, v := range ps {
			ptype := game.PlayerType(v)
			if ptype == game.PlayerTypeResistance {
				vp.haveres++
			} else if ptype == game.PlayerTypeSpy {
				vp.havespy++
			} else if ptype == game.PlayerTypeMerlin {
				vp.havemer++
			} else if ptype == game.PlayerTypeMorgana {
				vp.havemor++
			} else if ptype == game.PlayerTypePercival {
				vp.haveper++
			}
		}

		return
	}

	spiesmap := map[int]int{
		5:  2,
		6:  2,
		7:  3,
		8:  3,
		9:  3,
		10: 4,
	}

	validateplayertype := func(name string, want int, have int) (s string) {
		if want != have {
			s = fmt.Sprintf("%s - want: %d, have: %d\n", name, want, have)
		}

		return
	}

	validateall := func(g *game.Game, goption game.Option, gtype game.Type, p game.Player, thisgame *TestGame) validateplayer {

		vp := validateplayerhave(thisgame.Players)
		if p.Type == game.PlayerTypeResistance {
			vp.wantres = len(g.Players)
		} else if p.Type == game.PlayerTypeSpy || p.Type == game.PlayerTypeMerlin || p.Type == game.PlayerTypeMorgana {
			vp.wantspy = spiesmap[len(g.Players)]
			vp.wantres = len(g.Players) - vp.wantspy
			// if we're a merlin then decrease the desired resistance
			if p.Type == game.PlayerTypeMerlin {
				vp.wantres = vp.wantres - 1
				vp.wantmer = 1
			} else if p.Type == game.PlayerTypeMorgana {
				//vp.wantspy = vp.wantspy - 1
				vp.wantres = vp.wantres - 1
				vp.wantmor = 1
			} else if p.Type == game.PlayerTypeSpy {
				if g.Option.Has(game.OptionMorgana) {
					//vp.wantspy = vp.wantspy - 1
					vp.wantres--
					vp.wantmor = 1
				}
			}
		} else if p.Type == game.PlayerTypePercival {
			vp.wantres = len(g.Players) - 1
			vp.wantper = 1
			if goption.Has(game.OptionPercival) {
				vp.wantmer = 1
				if goption.Has(game.OptionMorgana) {
					vp.wantmer++
				}

				vp.wantres = vp.wantres - vp.wantmer
			}

			//vp.wantres = vp.wantres - vp.wantper
		}

		return vp
	}

	testGameLoop(5, 9, func(g *game.Game, gtype game.Type, goption game.Option, index int) {
		var mtx sync.Mutex
		num := 0
		done := make(chan int)

		for i := 0; i < len(cn); i++ {
			vsn := sn[i]
			vsn.RemoveCommandsByGroup("game")

			p, ok := g.Players[cn[i].GetClient().ID]
			if !ok {
				continue
			}

			vsn.AddCommand("game", conn.MessageStruct{
				"get": func(log logger.Logger, bytes []byte) error {
					thisgame := &TestGame{}
					err := json.Unmarshal(bytes, thisgame)
					if err != nil {
						t.Fatalf("json.Unmarshal: %v", err)
					}

					vp := validateall(g, goption, gtype, p, thisgame)

					str := []string{}
					spy := validateplayertype("spy", vp.wantspy, vp.havespy)
					res := validateplayertype("resistance", vp.wantres, vp.haveres)
					mer := validateplayertype("merlin", vp.wantmer, vp.havemer)
					mor := validateplayertype("morgana", vp.wantmor, vp.havemor)
					per := validateplayertype("percival", vp.wantper, vp.haveper)

					if len(spy) > 0 {
						str = append(str, spy)
					}

					if len(res) > 0 {
						str = append(str, res)
					}

					if len(mer) > 0 {
						t.Logf("%d", vp.havemer)
						str = append(str, mer)
					}

					if len(mor) > 0 {
						str = append(str, mor)
					}

					if len(per) > 0 {
						str = append(str, per)
					}

					if len(str) > 0 {
						for _, v := range str {
							t.Logf("%s", v)
						}
						//t.Logf("%v %v", vp.wantspy, vp.wantspy)
						//t.Logf("%+v", vp)
						//t.Logf("%d", p.Type)
					}

					mtx.Lock()
					defer mtx.Unlock()
					num++
					// if all players finished
					if num == len(g.Players) {
						done <- index
					}

					return nil
				},
			})

		}

		g.Send()

		<-done
		//t.Logf("finished game: type: %s, option: %d", lobby.Type(gtype.Common()).String(), goption)
	})

}

// boy is this test going to be long
func TestGameRun(t *testing.T) {

	// index 0 will decline any mission, in-order to test round failure
	// index 1 will recruit a spy on each mission and fail every mission
	// index 2 will recruit an all resistance team on each mission and try to fail every mission
	// index 3 will recruit randomly, even if it's a spy. if one of the player is a spy, that spy will fail the round.
	dolog := false // set this to true if you're debugging

	lc := logger.DefaultConfig
	lc.SWidth = 0
	lc.PWidth = 10
	// cause yeah
	lc.PAttr = color.New(color.Bold)
	lc.Debug = true

	for k, v := range cn {
		newlc := lc
		newlc.Prefix = fmt.Sprintf("client %02d", k+1)
		if dolog {
			v.SetLogger(logger.NewLogger(newlc))
		}
	}

	results := map[int]game.Status{
		0: game.StatusLost,
		1: game.StatusLost,
		2: game.StatusWon,
	}

	for index := 0; index <= 2; index++ {
		index := index
		/*
			if index == 1 || index == 2 {
				dolog = true
			} else {
				dolog = false
			}
		*/

		var done = make(chan game.Status)

		log := logger.NullLogger()
		if dolog {
			lc.Prefix = fmt.Sprintf("game %02d", index+1)
			log = logger.NewLogger(lc)
		}

		go testGameLoop(5, 5, func(g *game.Game, gtype game.Type, goption game.Option, gi int) {
			if goption != 0 || gtype != game.TypeBasic {
				return
			}

			g.SetLogger(log)

			players := []int{}

			for i := 0; i < len(cn); i++ {
				cl := cn[i].GetClient()
				playerid := cl.ID

				vsn := sn[i]

				player, ok := g.Players[playerid]
				if !ok {
					continue
				}

				testgame := &TestGame{}

				players = append(players, i)

				lp := &loopParameter{
					g:       g,
					gtype:   gtype,
					goption: goption,
					gi:      gi,

					testgame: testgame,
					rounds:   []game.Round{},
					players:  &players,
					vsn:      vsn,
					player:   player,
					index:    index,
				}

				vsn.AddCommand("game", conn.MessageStruct{
					// this command is sent whenever a mission has started
					// now by started I mean a new start of a round or a failure of a previous mission.
					// in this case body is basically an id.
					// we have to do the ID matching manually.
					"choose": getChooseFunc(lp),
					// vote is sent to all players, it basically sends an array of user IDs as body.
					// we expect each player to respond to vote with a true or false
					"vote": getVoteFunc(lp),
					// decide is sent whenever a mission has been accepted.
					// basically whenever a mission have been accepted, the mission assignees(players that are in the mission) get to decide if the mission fails or succeeds
					// if the player is a spy(morgana is also a spy), their vote is accounted for.
					// else the vote succeeds
					"decide": getDecideFunc(lp),

					"round": getRoundFunc(lp),

					"get": func(log logger.Logger, body []byte) error {
						json.Unmarshal(body, testgame)
						return nil
					},
				})

				// when the game finishes remove all the commands defined above
				defer vsn.RemoveCommandsByGroup("game")
			}

			g.Run(done)
		})

		select {
		case have := <-done:
			t.Logf("game[%d].Status %s\n", index, have.String())
			want := results[index]
			if have != want {
				t.Fatalf("want: '%s', have: '%s'", want.String(), have.String())
			}
		case <-time.After(time.Second * 2):
			t.Fatal("timed out")
			break
		}
	}
}
