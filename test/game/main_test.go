package game

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
)

var sampleuser = discord.User{
	ID:            "8035111022467891-",
	Username:      "Nelly",
	Discriminator: "1337",
}

var sampleusers = [10]discord.User{}

// sn is the conn we use to communicate with game model
// cn is the conn we pass for the game model to send events to
var sn, cn = []conn.Conn{}, []conn.Conn{}

func TestMain(m *testing.M) {
	v := sampleuser

	for i := 0; i < len(sampleusers); i++ {
		v.ID = strings.ReplaceAll(sampleuser.ID, "-", strconv.Itoa(i))

		sampleusers[i] = v
	}

	// create the connections
	for _, v := range sampleusers {
		vsn, vcn := conn.NewMockConnHelper(client.Client{
			User: v,
		})

		sn = append(sn, vsn)
		cn = append(cn, vcn)
	}

	os.Exit(m.Run())
}
