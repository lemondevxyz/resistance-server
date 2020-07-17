package game

import (
	"fmt"
	"testing"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/game"
	"github.com/toms1441/resistance-server/internal/lobby"
)

type validateplayer struct {
	wantres int
	wantspy int

	haveres int
	havespy int

	havemor int
	wantmor int

	haveper int
	wantper int

	havemer int
	wantmer int
}

func testGameLoop(start int, end int, callback func(*game.Game, error, game.Type, game.Option, int)) {
	execfunc := func(gtype game.Type, goption game.Option) {
		for i := start; i <= end; i++ {

			mapconn := map[string]conn.Conn{}
			for _, v := range cn[:i] {
				mapconn[v.GetClient().ID] = v
			}

			newgame, err := game.NewGame(mapconn, gtype.Common(), goption)
			callback(newgame, err, gtype, goption, i)
		}
	}

	execfunc(game.TypeBasic, 0)
	// these don't work with type basic
	// execfunc(game.TypeBasic, game.OptionPercival)
	// execfunc(game.TypeBasic, game.OptionMorgana.Add(game.OptionPercival))

	execfunc(game.TypeAvalon, 0)
	execfunc(game.TypeAvalon, game.OptionPercival)
	execfunc(game.TypeAvalon, game.OptionMorgana.Add(game.OptionPercival))

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

	testGameLoop(5, 9, func(g *game.Game, err error, gtype game.Type, goption game.Option, i int) {
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
