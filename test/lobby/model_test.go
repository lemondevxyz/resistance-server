package lobby

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/logger"
)

var cl = client.Client{
	discord.User{
		ID:            "80351110224678912",
		Username:      "Nelly",
		Discriminator: "1337",
	},
}

var lb = &lobby.Lobby{}
var sc, cc conn.Conn

func TestLobbyEqual(t *testing.T) {

	cl2 := cl
	cl2.ID = "Yelln"

	cls := []client.Client{
		cl,
	}

	l1 := &lobby.Lobby{
		ID:      "1234",
		Type:    lobby.TypeBasic,
		Private: false,
		Clients: cls,
	}

	l2 := &lobby.Lobby{
		ID:      "5678",
		Type:    lobby.TypeOriginal,
		Private: true,
	}

	check := func(t *testing.T, str string) {
		if l1.Equal(l2) {
			t.Fatalf("lobby.Equal: %s", str)
		}
	}
	check(t, "")

	l2.ID = l1.ID
	check(t, "ID")

	l2.Type = l1.Type
	check(t, "Type")

	l2.Private = l1.Private
	check(t, "Private")

	l2.Clients = []client.Client{client.Client{}}
	check(t, "Clients")

	l2.Clients = []client.Client{cl2}
	check(t, "Clients 2")

	l2.Clients = []client.Client{cl}
	if !l1.Equal(l2) {
		t.Fatalf("!l1.Equal(l2)")
	}

}

func TestLobbyJoin(t *testing.T) {

	sc, cc = conn.NewMockConnHelper(cl)
	err := lb.Join(cc)
	if err != nil {
		t.Fatalf("lb.Join: %v", err)
	}

	if lb.Clients[0] != cc.GetClient() {
		t.Fatal("lb.Clients[0] != cl")
	}

}

func TestLobbyRemove(t *testing.T) {
	err := lb.Leave(cc)
	if err != nil {
		t.Fatalf("lb.Leave: %v", err)
	}

	if len(lb.Clients) > 0 {
		t.Fatal("lb.Leave did not remove client")
	}

}

func TestLobbySubscribeInsert(t *testing.T) {
	insert := lb.SubscribeInsert()
	go func() {
		lb.Join(cc)
	}()

	select {
	case <-insert:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("lb.SubscribeInsert does not work")
	}

	lb.RemoveSubscribeInsert(insert)
}

func TestLobbySubscribeRemove(t *testing.T) {
	remove := lb.SubscribeRemove()
	go func() {
		lb.Leave(cc)
	}()

	select {
	case <-remove:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("lb.SubscribeInsert does not work")
	}

	lb.RemoveSubscribeRemove(remove)
}

func TestLobbyMessageSend(t *testing.T) {

	want := conn.MessageSend{
		Group: "lobby",
		Name:  "get",
	}

	have := lb.MessageSend()
	if want.Group != have.Group || want.Name != have.Name {
		t.Fatal("want != have")
	}

}

func TestLobbySend(t *testing.T) {

	time.Sleep(time.Millisecond * 50)
	done := make(chan *lobby.Lobby)

	sc.AddCommand("lobby", conn.MessageStruct{
		"get": func(log logger.Logger, bytes []byte) error {
			unmarshal := &lobby.Lobby{}

			err := json.Unmarshal(bytes, unmarshal)
			if err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}

			done <- unmarshal

			return nil
		},
	})

	lb.Join(cc)

	select {
	case have := <-done:
		if !have.Equal(lb) {
			t.Fatalf("want: %+v, have: %+v", lb, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timed out")
	}

}

func TestLobbyGetClientIndex(t *testing.T) {

	i := lb.GetClientIndex(cc.GetClient().ID)
	if i == -1 {
		t.Fatalf("l.GetClientIndex == -1")
	}

	lb.Leave(cc)

}

func TestLobbyValidate(t *testing.T) {
	lb.Type = lobby.Type(6)
	if lb.Validate() == nil {
		t.Fatalf("lb.Validate == nil")
	}

	lb.Type = lobby.TypeBasic
	if err := lb.Validate(); err != nil {
		t.Fatalf("lb.Validate: %v", err)
	}
}
