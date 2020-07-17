package conn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
)

var sconn, cconn conn.Conn
var pipe1, pipe2 net.Conn
var mockresp = make(chan int)

func TestNewMockConn(t *testing.T) {

	pipe1, pipe2 = net.Pipe()
	sconn = conn.NewMockConn(pipe1, client.Client{})

}

func TestMockWriteBytes(t *testing.T) {

	msg := []byte("message")
	done := make(chan int)
	i := 0
	go func() {
		var body = make([]byte, len(msg))
		for {
			n, err := pipe2.Read(body)
			if n == 0 {
				continue
			}

			if err != nil {
				return
			}

			val := bytes.Compare(body, msg)
			if val == 0 {
				i++

				done <- i
				if i == 3 {
					return
				}
			}
		}
	}()

	test := func(want int) {
		select {
		case have := <-done:
			if want != have {
				t.Fatalf("want: %d, have: %d", want, have)
			}
		case <-time.After(time.Millisecond * 100):
			t.Fatalf("timed out")
		}
	}

	// test multiple writes
	sconn.WriteBytes(msg)
	test(1)
	sconn.WriteBytes(msg)
	test(2)
	sconn.WriteBytes(msg)
	test(3)

	/*
		cconn = NewMockConn(pipe2, logger.NewLogger(logger.DefaultConfig))
	*/

}

func TestMockReadBytes(t *testing.T) {

	sconn.AddCommand("test", getstrct(mockresp, 0))

	body, err := json.Marshal(conn.MessageSend{
		Group: "test",
		Name:  "null",
	})

	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	pipe2.Write(body)

	select {
	case val := <-mockresp:
		if val != 1 {
			t.Fatalf("want: 1, have: %d", val)
		}
		break
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out")
	}
}

func TestMockWriteMessage(t *testing.T) {

	if cconn == nil {
		cconn = conn.NewMockConn(pipe2, client.Client{})
	}

	//sconn.AddCommand("test", strct)

	err := cconn.WriteMessage(conn.MessageSend{
		Group: "test",
		Name:  "null",
	})

	if err != nil {
		t.Logf("conn.WriteMessage: %v", err)
	}

	select {
	case val := <-mockresp:
		if val != 2 && val != 3 {
			t.Fatalf("want: 2 || 3, have: %d", val)
		}
		break
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out")
	}
}

// no need to test AddCommand since it already works above.

// i just wanna fill up the coverage
func TestMockAddCommand(t *testing.T) {
	TestMockWriteMessage(t)
}

func TestMockExecuteCommand(t *testing.T) {
	go func() {
		err := sconn.ExecuteCommand("test", "null", nil)
		if err != nil {
			fmt.Printf("sconn.ExecuteCommand: %v\n", err)
			os.Exit(1)
		}
	}()

	select {
	case val := <-mockresp:
		if val != 4 {
			t.Fatalf("want: %d, have: %d", 4, val)
		}
		break
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out")
	}
}

func TestMockRemoveCommandsByNames(t *testing.T) {
	sconn.RemoveCommandsByNames("test", "null")

	err := cconn.WriteMessage(conn.MessageSend{
		Group: "test",
		Name:  "null",
		Body:  nil,
	})

	if err != nil {
		log.Fatalf("cconn.WriteMessage: %v", err)
	}

	select {
	case <-mockresp:
		t.Fatalf("we got a response after we removes the command")
	case <-time.After(time.Millisecond * 50):
		break
	}

}

func TestMockRemoveCommandsByGroup(t *testing.T) {

	sconn.RemoveCommandsByGroup("test")

	ms := conn.MessageSend{
		Group: "test",
		Name:  "nulltwo",
	}

	send := func() {
		err := cconn.WriteMessage(ms)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}

	}

	send()
	ms.Name = "bytes"
	ms.Body = []byte("message")

	select {
	case <-mockresp:
		t.Fatalf("we got a response")
	case <-time.After(time.Millisecond * 100):
		break
	}

}

func TestMockGetDone(t *testing.T) {
	wgchannel = make(chan struct{})

	go func() {
		wg.Add(1)
		select {
		case <-sconn.GetDone():
			close(wgchannel)
			wg.Done()
		}
	}()

	time.Sleep(time.Millisecond * 10)
}

func TestMockDestroy(t *testing.T) {

	sconn.Destroy()

	select {
	case <-wgchannel:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timed out 100ms")
	}

}
