package plain

import (
	"fmt"
	"sync"

	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/repo"
)

type lobbyRepository struct {
	db map[string]*lobby.Lobby
	mx sync.Mutex
}

func NewLobbyRepository() lobby.Repository {
	return &lobbyRepository{
		db: map[string]*lobby.Lobby{},
	}
}

func (r *lobbyRepository) Create(l *lobby.Lobby) error {
	if r.db == nil {
		return repo.ErrLobbyInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	_, ok := r.db[l.ID]
	if ok {
		return repo.ErrLobbyExists
	}

	if err := l.Validate(); err != nil {
		return fmt.Errorf("l.Validate: %w", err)
	}

	r.db[l.ID] = l

	return nil
}

func (r *lobbyRepository) GetByID(id string) (*lobby.Lobby, error) {
	if r.db == nil {
		return nil, repo.ErrLobbyInvalid
	}

	l, ok := r.db[id]
	if !ok {
		return nil, repo.ErrLobby404
	}

	return l, nil
}

func (r *lobbyRepository) GetAll() ([]*lobby.Lobby, error) {
	ls := []*lobby.Lobby{}
	if r.db == nil {
		return ls, repo.ErrLobbyInvalid
	}

	for _, v := range r.db {
		ls = append(ls, v)
	}

	return ls, nil
}

func (r *lobbyRepository) Update(id string, l *lobby.Lobby) error {
	if r.db == nil {
		return repo.ErrLobbyInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	_, ok := r.db[id]
	if !ok {
		return repo.ErrLobby404
	}

	if err := l.Validate(); err != nil {
		return fmt.Errorf("l.Validate: %w", err)
	}

	return nil
}

func (r *lobbyRepository) Remove(id string) error {
	if r.db == nil {
		return repo.ErrLobbyInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	_, ok := r.db[id]
	if !ok {
		return repo.ErrLobby404
	}

	delete(r.db, id)
	return nil
}

func (r *lobbyRepository) IsValid() bool {
	if r.db == nil {
		return false
	}

	return true
}
