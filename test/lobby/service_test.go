package lobby

import (
	"bytes"
	"testing"

	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/repo/plain"
)

var repo lobby.Repository
var lserv lobby.Service
var logoutput = bytes.NewBuffer(make([]byte, 1024))
var lobbyptr = &lobby.Lobby{
	Type: lobby.TypeBasic,
}
var otherlobby = &lobby.Lobby{
	Type: lobby.TypeAvalon,
}

func TestNewService(t *testing.T) {
	repo = plain.NewLobbyRepository()

	var err error
	_, err = lobby.NewService(nil, lobby.DefaultConfig)
	if err == nil {
		t.Fatalf("lobby.NewService should return error because repository is nil")
	}

	_, err = lobby.NewService(repo, invalidconfig)
	if err == nil {
		t.Fatalf("lobby.NewService should return error because config is invalid")
	}

	lserv, err = lobby.NewService(repo, lobby.DefaultConfig)
	if err != nil {
		t.Fatalf("lobby.NewService: %v", err)
	}

}

func TestServiceCreateLobby(t *testing.T) {

	err := lserv.CreateLobby(lobbyptr)
	if err != nil {
		t.Fatalf("lserv.CreateLobby: %v", err)
	}

	ls, err := repo.GetAll()
	if err != nil {
		t.Fatalf("repo.GetAll: %v", err)
	}

	var lslobby *lobby.Lobby
	for _, v := range ls {
		if v.ID == lobbyptr.ID {
			lslobby = v
			break
		}
	}

	if !lslobby.Equal(lobbyptr) {
		t.Fatal("lslobby != lobbyptr")
	}

}

func TestServiceGetLobbyByID(t *testing.T) {

	getlobby, err := lserv.GetLobbyByID(lobbyptr.ID)
	if err != nil {
		t.Fatalf("lserv.GetLobbyByID: %v", err)
	}

	if !getlobby.Equal(lobbyptr) {
		t.Fatal("!getlobby.Equal")
	}

}

func TestServiceGetAllLobbies(t *testing.T) {

	ls1, err := lserv.GetAllLobbies()
	if err != nil {
		t.Fatalf("lserv.GetAllLobbies: %v", err)
	}

	ls2, err := repo.GetAll()
	if err != nil {
		t.Fatalf("repo.GetAll: %v", err)
	}

	if len(ls1) != len(ls2) {
		t.Fatalf("len(ls1) != len(ls2)")
	}

	err = lserv.CreateLobby(otherlobby)
	if err != nil {
		t.Fatalf("repo.Create: %v", err)
	}

	ls1, err = lserv.GetAllLobbies()
	if err != nil {
		t.Fatalf("lserv.GetAllLobbies: %v", err)
	}

	if len(ls1) != 2 {
		t.Fatalf("len(ls1), want: %d - have: %d", 2, len(ls1))
	}

}

func TestServiceRemoveLobby(t *testing.T) {
	err := lserv.RemoveLobby(lobbyptr.ID)
	if err != nil {
		t.Fatalf("lserv.RemoveLobby: %v", err)
	}

	err = lserv.RemoveLobby(otherlobby.ID)
	if err != nil {
		t.Fatalf("lserv.RemoveLobby: %v", err)
	}

	ls, err := repo.GetAll()
	if err != nil {
		t.Fatalf("repo.GetAll: %v", err)
	}

	if len(ls) != 0 {
		t.Fatalf("len(ls), want: %d - have: %d", 0, len(ls))
	}
}
