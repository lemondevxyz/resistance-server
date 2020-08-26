package game

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/game"
)

var sampleuser = discord.User{
	ID:            "8035111022467891-",
	Username:      "Nelly",
	Discriminator: "133-",
}

var sampleusers = [10]discord.User{}

// sn is the conn we use to communicate with game model
// cn is the conn we pass for the game model to send events to
var sn, cn = []conn.Conn{}, []conn.Conn{}

// doesn't need to implement conn.Conn
type testConn struct {
	nc   net.Conn
	mtx  sync.Mutex
	cmds map[string]conn.MessageStruct
}

func (ts *testConn) AddCommand(group string, callback conn.MessageStruct) {
	ts.mtx.Lock()
	defer ts.mtx.Unlock()

	ts.cmds[group] = callback
}

func (ts *testConn) RemoveCommandsByGroup(group string) {
	ts.mtx.Lock()
	defer ts.mtx.Unlock()

	delete(ts.cmds, group)
}

func (ts *testConn) WriteMessage(msg conn.MessageSend) {
	body, err := json.Marshal(msg)
	if err == nil {
		err = wsutil.WriteClientMessage(ts.nc, ws.OpText, body)
		if err != nil {
			fmt.Printf("wsutil.WriteClientMessage: %s\n", err.Error())
		}
	} else {
		fmt.Printf("WriteMessage: json.Unmarshal: %s\n", err.Error())
	}
}

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

type loopParameter struct {
	g       *game.Game
	gtype   game.Type
	goption game.Option
	gi      int
	// arr of player indexes
	testgame *TestGame
	rounds   []game.Round
	round    int
	mission  int
	players  *[]int
	vsn      conn.Conn
	player   game.Player
	index    int
	mtx      sync.Mutex
}

type loopCallback func(*loopParameter) conn.MessageCallback

func testGameLoop(start int, end int, callback func(*game.Game, game.Type, game.Option, int)) {
	execfunc := func(gtype game.Type, goption game.Option) {
		for i := start; i <= end; i++ {

			mapconn := map[string]conn.Conn{}
			for _, v := range cn[:i] {
				mapconn[v.GetClient().ID] = v
			}

			newgame, err := game.NewGame(mapconn, gtype.Common(), goption)
			if err == nil {
				callback(newgame, gtype, goption, i)
			} else {
				fmt.Println("error with testGameLoop")
			}
		}
	}

	execfunc(game.TypeBasic, 0)
	// these intentionally don't work with type basic
	// execfunc(game.TypeBasic, game.OptionPercival)
	// execfunc(game.TypeBasic, game.OptionMorgana.Add(game.OptionPercival))

	execfunc(game.TypeAvalon, 0)
	execfunc(game.TypeAvalon, game.OptionPercival)
	execfunc(game.TypeAvalon, game.OptionMorgana.Add(game.OptionPercival))

}

func TestMain(m *testing.M) {
	v := sampleuser

	/* websocket version
		var wg sync.WaitGroup

		ln, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			log.Fatalf("net.Listen: %s", err.Error())
		}

		var index int
		go func(ln net.Listener) {
			for {
				netconn, err := ln.Accept()
				if err != nil {
					fmt.Printf("ln.Accept: %v", err)
					os.Exit(1)
				}

				_, err = ws.Upgrade(netconn)
				if err != nil {
					fmt.Printf("ws.Upgrade: %v", err)
					os.Exit(1)
				}

				cn = append(cn, conn.NewConn(netconn, client.Client{
					User: sampleusers[index],
				}))

				wg.Done()
			}
		}(ln)

		for i := 0; i < len(sampleusers); i++ {

			index = i

			v.ID = strings.ReplaceAll(sampleuser.ID, "-", strconv.Itoa(i))
			v.Discriminator = strings.ReplaceAll(sampleuser.Discriminator, "-", strconv.Itoa(i))

			sampleusers[i] = v

			wg.Add(1)
			nc, _, _, err := ws.Dial(context.Background(), "ws://"+ln.Addr().String())
			if err != nil {
				log.Fatalf("ws.Dial: %s", err.Error())
			}

			sn = append(sn, &testConn{
				nc:   nc,
				cmds: map[string]conn.MessageStruct{},
			})

			go func(vsn *testConn) {
				for {
					body, err := wsutil.ReadServerText(vsn.nc)
					if err != nil {
						fmt.Printf("wsutil.ReadServerText: %s\n", err.Error())
						return
					}

					mr := conn.MessageRecv{}
					err = json.Unmarshal(body, &mr)
					if err != nil {
						fmt.Printf("json.Unmarshal: %s\n", err.Error())
						return
					}

					vsn.mtx.Lock()

					cmd, ok := vsn.cmds[mr.Group
					if ok {
						for k, v := range cmd {
							if mr.Name == k {
								go v(logger.NullLogger(), mr.Body)
							}
						}
					}

					vsn.mtx.Unlock()
				}
			}(sn[i])

		}
	wg.Wait()
	*/

	// mock version
	allids := []string{}
	for i := 0; i < len(sampleusers); i++ {
		v.ID = strings.ReplaceAll(sampleuser.ID, "-", strconv.Itoa(i))
		allids = append(allids, v.ID)

		sampleusers[i] = v
	}

	for _, v := range sampleusers {
		vsn, vcn := conn.NewMockConnHelper(client.Client{
			User: v,
		})

		sn = append(sn, vsn)
		cn = append(cn, vcn)
	}

	os.Exit(m.Run())
}
