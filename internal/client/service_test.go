package client

import (
	"testing"

	"github.com/gin-gonic/gin/internal/json"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
	"github.com/toms1441/resistance-server/internal/repo/plain"
)

var serv Service
var wconn, rconn conn.Conn

func TestNewService(t *testing.T) {
	repo := plain.NewClientRepository()

	var err error
	serv, err = NewService(repo, logger.NullLogger())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
}

func TestServiceNewClient(t *testing.T) {
	wconn, rconn = conn.NewMockConnHelper(logger.NullLogger(), logger.NullLogger())

	cl, err := serv.CreateClient(wconn, logger.NullLogger())
	if err != nil {
		t.Fatalf("serv.CreateClient: %v", err)
	}

	valid := make(chan bool)
	rconn.AddCommand("client", conn.MessageStruct{
		"get": func(log logger.Logger, body []byte) error {
			temp := &Client{}

			err = json.Unmarshal(body, temp)
			if err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}

			if *temp == *cl {
				done <- true
			}
		},
	})
}
