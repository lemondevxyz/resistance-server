package game

// this file is concerned with model-related tests. except Send and Run
// it's meant to be simple and easy to read.

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/logger"
)

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

func TestGameAssignPlayers(t *testing.T) {
	testValidate := func(g *game.Game) (vp validateplayer) {
		spiesmap := map[int]int{
			5:  2,
			6:  2,
			7:  3,
			8:  3,
			9:  3,
			10: 4,
		}

		if g.Type.Common() == lobby.TypeAvalon.Common() {
			vp.wantmer = 1
		}

		if g.Option.Has(game.OptionPercival) {
			vp.wantper = 1
		}

		if g.Option.Has(game.OptionMorgana) {
			vp.wantper = 1
			vp.wantmor = 1
		}

		for _, v := range g.Players {
			if v.Type == game.PlayerTypeResistance {
				vp.haveres++
			} else if v.Type == game.PlayerTypeSpy {
				vp.havespy++
			} else if v.Type == game.PlayerTypeMerlin {
				vp.havemer++
			} else if v.Type == game.PlayerTypeMorgana {
				vp.havemor++
			} else if v.Type == game.PlayerTypePercival {
				vp.haveper++
			}
		}

		vp.wantspy = -1
		vp.wantres = -1

		wantspy, ok := spiesmap[len(g.Players)]
		if !ok {
			return
		}

		vp.wantspy = wantspy
		vp.wantres = (((len(g.Players) - vp.wantspy) - vp.wantmer) - vp.wantper) - vp.wantmor

		return
	}

	// create new game made out of [5, 10]
	// then assign players and match it's validity
	validateplayertype := func(name string, want int, have int) (s string) {
		if want != have {
			s = fmt.Sprintf("%s - want: %d, have: %d\n", name, want, have)
		}

		return
	}

	testGameLoop(5, 9, func(g *game.Game, gtype game.Type, goption game.Option, i int) {
		str := ""

		strct := testValidate(g)

		str += validateplayertype("spy", strct.wantspy, strct.havespy)
		str += validateplayertype("resistance", strct.wantres, strct.haveres)
		str += validateplayertype("merlin", strct.wantmer, strct.havemer)
		str += validateplayertype("percival", strct.wantper, strct.haveper)
		str += validateplayertype("morgana", strct.wantmor, strct.havemor)

		if len(str) > 0 {
			t.Fatalf("game type: %s, player len: %d\n%s", lobby.Type(gtype.Common()).String(), i, str)
		}

	})

}

func TestGameBroadcast(t *testing.T) {

	index := len(singlegame.Players)

	msgsent := 0
	done := make(chan bool)
	var mtx sync.Mutex
	retfunc := func(log logger.Logger, bytes []byte) error {

		defer mtx.Unlock()
		mtx.Lock()

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
	testGameLoop(5, 10, func(g *game.Game, gtype game.Type, goption game.Option, index int) {
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
	testGameLoop(5, 10, func(g *game.Game, gtype game.Type, goption game.Option, index int) {
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
