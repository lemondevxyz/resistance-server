package conn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/logger"
	"golang.org/x/net/context"
)

var ln net.Listener
var nconn net.Conn
var iconn conn.Conn
var wg sync.WaitGroup

var connresp = make(chan int)

var cl = client.Client{
	User: discord.User{
		ID:            "80351110224678912",
		Username:      "Nelly",
		Discriminator: "1337",
	},
}

func getstrct(response chan int, ind int) conn.MessageStruct {
	return conn.MessageStruct{
		"null": func(_ logger.Logger, _ []byte) error {
			ind++
			response <- ind
			return nil
		},

		"nulltwo": func(_ logger.Logger, _ []byte) error {
			ind++
			response <- ind
			return nil
		},

		"bytes": func(_ logger.Logger, body []byte) error {
			var defBytes = []byte("message")
			mr := []byte{}

			err := json.Unmarshal(body, &mr)
			if err != nil {
				fmt.Printf("json.Unmarshal: %v\n", err)
				os.Exit(1)
			}

			result := bytes.Compare(defBytes, mr)
			if result == 0 {
				response <- 10
			}

			return nil
		},
	}
}

func TestNewListener(t *testing.T) {
	var err error
	ln, err = net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

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

			iconn = conn.NewConn(netconn, cl)
			wg.Done()
		}
	}(ln)

	time.Sleep(time.Millisecond * 20)

}

func TestNewConn(t *testing.T) {
	var err error
	wg.Add(1)

	nconn, _, _, err = ws.Dial(context.Background(), "ws://"+ln.Addr().String())

	if err != nil {
		t.Fatalf("ws.Dial: %v", err)
	}

}

func TestAddCommand(t *testing.T) {

	var defBytes = []byte("message")

	wg.Wait()
	iconn.AddCommand("test", getstrct(connresp, 0))

	ms := conn.MessageSend{
		Group: "test",
		Name:  "null",
		Body:  nil,
	}

	body, err := json.Marshal(ms)
	if err != nil {
		log.Fatalf("json.Marshal: %v", err)
	}

	err = wsutil.WriteClientMessage(nconn, ws.OpText, body)
	if err != nil {
		log.Fatalf("wsutil.WriteClientMessage: %v", err)
	}

	if val := <-connresp; val != 1 {
		log.Fatalf("want: 1, have: %d", val)
	}

	ms.Name = "bytes"
	ms.Body = defBytes

	body, err = json.Marshal(ms)
	if err != nil {
		log.Fatalf("json.Marshal: %v", err)
	}

	err = wsutil.WriteClientMessage(nconn, ws.OpText, body)
	if err != nil {
		log.Fatalf("wsutil.WriteClientMessage: %v", err)
	}

	if val := <-connresp; val != 10 {
		log.Fatalf("want: 10, have: %d", val)
	}
}

func TestExecuteCommand(t *testing.T) {

	go func(t *testing.T) {
		err := iconn.ExecuteCommand("test", "null", nil)
		if err != nil {
			t.Fatalf("iconn.ExecuteCommand: %v", err)
		}
	}(t)

	val := <-connresp
	if val != 2 {
		t.Fatalf("want: %d, have: %d", 2, val)
	}

}

func TestRemoveCommandsByNames(t *testing.T) {
	iconn.RemoveCommandsByNames("test", "null")

	body, err := json.Marshal(conn.MessageSend{
		Group: "test",
		Name:  "null",
		Body:  nil,
	})

	if err != nil {
		log.Fatalf("json.Marshal: %v", err)
	}

	err = wsutil.WriteClientMessage(nconn, ws.OpText, body)
	if err != nil {
		log.Fatalf("wsutil.WriteClientMessage: %v", err)
	}

	select {
	case <-connresp:
		t.Fatalf("we got a response after we removes the command")
	case <-time.After(time.Millisecond * 50):
		break
	}

}

func TestRemoveCommandsByGroup(t *testing.T) {

	iconn.RemoveCommandsByGroup("test")

	ms := conn.MessageSend{
		Group: "test",
		Name:  "nulltwo",
	}

	send := func() {
		body, err := json.Marshal(ms)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

		err = wsutil.WriteClientMessage(nconn, ws.OpText, body)
		if err != nil {
			t.Fatalf("WriteClientMessage: %v", err)
		}

	}
	send()
	ms.Name = "bytes"
	ms.Body = []byte("message")

	select {
	case <-connresp:
		t.Fatalf("we got a response")
	case <-time.After(time.Millisecond * 100):
		break
	}

}

func TestWriteMessage(t *testing.T) {

	go func() {
		err := iconn.WriteMessage(conn.MessageSend{
			Group: "test",
			Name:  "null",
			Body:  nil,
		})

		if err != nil {
			log.Fatalf("iconn.WriteMessage: %v", err)
		}
	}()

	body, err := wsutil.ReadServerText(nconn)
	if err != nil {
		t.Fatalf("wsutil.ReadServerText: %v", err)
	}

	mr := conn.MessageRecv{}

	err = json.Unmarshal(body, &mr)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if mr.Group != "test" || mr.Name != "null" {
		t.Fatalf(`mr != conn.MessageRecv{group: "test", name: "null"}`)
	}

}

func TestWriteBytes(t *testing.T) {

	go func() {
		iconn.WriteBytes([]byte("message"))
	}()

	body, err := wsutil.ReadServerText(nconn)
	if err != nil {
		t.Fatalf("wsutil.ReadServerText: %v", err)
	}

	if bytes.Compare(body, []byte("message")) != 0 {
		t.Fatalf("bytes are not the same")
	}

}

var wgchannel = make(chan struct{})

func TestGetDone(t *testing.T) {
	go func() {
		wg.Add(1)
		select {
		case <-iconn.GetDone():
			close(wgchannel)
			wg.Done()
		}
	}()

	time.Sleep(time.Millisecond * 10)
}

func TestDestroy(t *testing.T) {

	iconn.Destroy()

	select {
	case <-wgchannel:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out 100ms")
	}

}
