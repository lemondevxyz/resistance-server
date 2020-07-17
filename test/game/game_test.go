package game

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/logger"
)

type TestGame struct {
	Type    game.Type      `json:"type"`
	Players map[string]int `json:"players"`
	Option  game.Option    `json:"option"`
}

var singlegame *game.Game

func TestNewGame(t *testing.T) {

	testGameLoop(0, 9, func(g *game.Game, err error, gtype game.Type, goption game.Option, i int) {
		if i >= 5 {
			if err != nil {
				t.Fatalf("game.NewGame: %v", err)
			}
		} else if i < 5 {
			if err != game.ErrInvalidClients {
				t.Fatalf("game.NewGame.(error) == nil, when it should give an error")
			}
		}
	})

}

func TestGameSetLogger(t *testing.T) {
	done := make(chan bool)

	reader, writer := io.Pipe()

	lc := logger.DefaultConfig
	lc.Writer = writer

	mapconn := map[string]conn.Conn{}
	for _, v := range cn[:5] {
		mapconn[v.GetClient().ID] = v
	}

	go func() {
		bytes := make([]byte, 1024)
		for {
			n, err := reader.Read(bytes)
			if n == 0 {
				continue
			}

			if err != nil {
				fmt.Printf("%v", err)
				continue
			}

			done <- true
		}
	}()

	var err error

	singlegame, err = game.NewGame(mapconn, game.TypeBasic.Common(), 0)
	if err != nil {
		t.Fatalf("game.NewGame: %v", err)
	}

	singlegame.SetLogger(logger.NewLogger(lc))

	select {
	case <-done:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out")
	}

	singlegame.SetLogger(logger.NullLogger())

}

func TestGameBroadcast(t *testing.T) {

	index := len(singlegame.Players)

	msgsent := 0
	done := make(chan bool)
	retfunc := func(log logger.Logger, bytes []byte) error {

		msgsent++
		if msgsent == 5 {
			done <- true
		}

		return nil
	}

	for i := 0; i < index; i++ {
		sn[i].AddCommand("test", conn.MessageStruct{
			"null": retfunc,
		})
	}

	singlegame.Broadcast(conn.MessageSend{
		Group: "test",
		Name:  "null",
	})

	select {
	case <-done:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out")
	}

	for i := 0; i < index; i++ {
		sn[i].RemoveCommandsByGroup("test")
	}

}

func TestGameJSON(t *testing.T) {
	testGameLoop(5, 10, func(g *game.Game, err error, gtype game.Type, goption game.Option, index int) {
		marshal, err := json.Marshal(g)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		unmarshal := &TestGame{}
		err = json.Unmarshal(marshal, unmarshal)
		if err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
	})
}

func TestGameMessageSend(t *testing.T) {
	testGameLoop(5, 10, func(g *game.Game, err error, gtype game.Type, goption game.Option, index int) {
		ms := conn.MessageSend{
			Group: "game",
			Name:  "get",
			Body:  g,
		}

		marshal, err := json.Marshal(ms)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		thisgame := &TestGame{}
		err = json.Unmarshal(marshal, thisgame)
		if err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
	})
}

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

	testGameLoop(5, 9, func(g *game.Game, err error, gtype game.Type, goption game.Option, index int) {
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
						t.Logf("%d", p.Type)
					}

					num++
					// we ran this function for all players with no error
					if num == len(g.Players) {
						done <- index
					}

					return nil
				},
			})

		}

		g.Send()

		<-done
		t.Logf("finished game: type: %s, option: %d", lobby.Type(gtype.Common()).String(), goption)
	})

}
