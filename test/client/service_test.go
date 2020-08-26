package main

import (
	"testing"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/logger"
	"github.com/toms1441/resistance-server/internal/repo/plain"
)

var repo = plain.NewClientRepository()
var cserv client.Service
var wconn, rconn conn.Conn
var cl client.Client

var user = discord.User{
	ID:            "80351110224678912",
	Username:      "Nelly",
	Discriminator: "1337",
}

func TestNewService(t *testing.T) {
	var err error

	cserv, err = client.NewService(repo, logger.NullLogger())
	if err != nil {
		t.Fatalf("client.NewService: %v", err)
	}

	wconn, rconn = conn.NewMockConnHelper(client.Client{user})
}

func TestServiceNewClient(t *testing.T) {
	var err error

	cl, err = cserv.CreateClient(user)
	if err != nil {
		t.Fatalf("cserv.CreateClient: %v", err)
	}

	temp, err := repo.GetByID(cl.ID)
	if err != nil {
		t.Fatalf("client gets created but never inserted into repo. error: %v", err)
	}

	if temp != cl {
		t.Fatal("temp != cl")
	}

}

func TestServiceGetClientByID(t *testing.T) {

	id := cl.ID

	_, err := cserv.GetClientByID(id)
	if err != nil {
		t.Fatalf("cserv.GetClientByID: %v", err)
		temp, err := repo.GetByID(id)
		if err != nil {
			t.Fatalf("repo.GetByID: %v", err)
		}

		if cl != temp {
			t.Fatal("cl != temp")
		}
	}

}

func TestServiceGetAllClients(t *testing.T) {

	tempuser := user
	tempuser.ID += "1"
	temp := client.Client{
		User: tempuser,
	}

	err := repo.Create(temp)
	if err != nil {
		t.Fatalf("cserv.CreateClient: %v", err)
	}

	cls, err := cserv.GetAllClients()
	if err != nil {
		t.Fatalf("cserv.GetAllClients: %v", err)
	}

	var exists int
	for _, v := range cls {
		if v == cl || v == temp {
			exists++
		}
	}

	if exists != 2 {
		t.Fatalf("did not get two matching clients")
	}

}

func TestServiceUpdateClient(t *testing.T) {
	newuser := user
	newuser.Username = "Yelln"

	temp := client.Client{
		User: newuser,
	}

	err := cserv.UpdateClient(user.ID, temp)
	if err != nil {
		t.Fatalf("cserv.UpdateClient: %v", err)
	}

	cl, err := repo.GetByID(user.ID)
	if err != nil {
		t.Fatalf("repo.GetByID: %v", err)
	}

	if cl != temp {
		t.Logf("%v %v", cl, temp)
		t.Fatal("cl != temp")
	}

}

func TestServiceRemoveClient(t *testing.T) {
	err := cserv.RemoveClient(cl.ID)
	if err != nil {
		t.Fatalf("cserv.RemoveClient: %v", err)
	}

	_, err = repo.GetByID(user.ID)
	if err == nil {
		t.Fatalf("repo.GetByID: nil")
	}

	id := user.ID + "1"

	err = cserv.RemoveClient(id)
	if err != nil {
		t.Fatalf("cserv.RemoveClient: %v", err)
	}

	_, err = repo.GetByID(id)
	if err == nil {
		t.Fatalf("repo.GetByID: nil")
	}

}
