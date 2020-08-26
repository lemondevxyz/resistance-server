package lobby

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
)

// Service is a service that uses the repo in-order to do database actions.
type Service interface {
	// SetLogger sets the logger for the service.
	SetLogger(log logger.Logger)
	// CreateLobby creates a lobby
	CreateLobby(lobby *Lobby) error
	// GetLobbyByID returns a lobby that has the same id
	GetLobbyByID(id string) (*Lobby, error)
	// GetAllLobbies returns all lobbies
	GetAllLobbies() ([]*Lobby, error)
	// GetLobbyByClientID returns a lobby by a client's id.
	// GetLobbyByClientID(id string) (*Lobby, error)
	// UpdateLobby updates a lobby by it's ID. Note: if you want to update a lobby you need to get it first then update that single field.
	UpdateLobby(id string, lobby *Lobby) error
	// RemoveLobby removes a lobby by it's ID.
	RemoveLobby(id string) error
}

type service struct {
	repo   Repository
	config Config
	log    logger.Logger
}

var ErrNil = errors.New("lobby is nil")
var ErrRepositoryNil = errors.New("Repository is nil")

// NewService returns a new lobby service. It can be used to create, read, update and delete.
func NewService(repo Repository, config Config) (Service, error) {

	if repo == nil {
		return nil, ErrRepositoryNil
	}

	if !repo.IsValid() {
		return nil, fmt.Errorf("repo.IsValid")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Config.Validate: %w", err)
	}

	rand.Seed(time.Now().UnixNano())

	maxstr := ""
	// should output IDLen num of 9 digits
	// say IDLen is 5, lobbyintnstr will equal to 99999
	minstr := "1"
	// should output IDLen num of 10
	// say IDLen is 5, minlenstr will equal to 10000
	for i := 1; i <= config.IDLen; i++ {
		maxstr = fmt.Sprintf("%s9", maxstr)
		if i == 1 {
			minstr = "1"
		} else {
			minstr = fmt.Sprintf("%s0", minstr)
		}
	}

	max, err := strconv.Atoi(maxstr)
	if err != nil {
		return nil, errors.New("something has went wrong with lobbyintn")
	}
	config.max = max

	min, err := strconv.Atoi(minstr)
	if err != nil {
		return nil, errors.New("something has went wrong with minlen")
	}
	config.min = min

	return &service{
		repo:   repo,
		config: config,
		log:    logger.NullLogger(),
	}, nil
}

// SetLogger sets the logger for the service
func (s *service) SetLogger(log logger.Logger) {
	if s.log != nil {
		s.log = log
		log.Info("Set Logger")
	}
}

// CreateLobby creates a new lobby.
func (s *service) CreateLobby(l *Lobby) error {

	if l == nil {
		return ErrNil
	}

	var id int

	for {
		id = s.config.min + rand.Intn(s.config.max-s.config.min)
		templ := &Lobby{
			ID: strconv.Itoa(id),
		}

		// id is valid
		if _, err := s.repo.GetByID(templ.ID); err != nil {
			break
		}
	}

	l.ID = strconv.Itoa(id)

	err := l.Validate()
	if err != nil {
		return fmt.Errorf("l.Validate: %w", err)
	}

	err = s.repo.Create(l)
	if err != nil {
		return fmt.Errorf("repo.Create: %w", err)
	}

	log := s.log.Replicate()
	space := logger.DefaultConfig.SWidth - len(l.Type.String()) - len(l.ID)

	suffix := l.Type.String() + strings.Repeat(" ", space) + l.ID
	log.SetSuffix(suffix)

	l.SetLogger(log)

	remove := l.SubscribeRemove()
	go func(l *Lobby, r chan conn.Conn) {
		for {
			select {
			case c := <-r:
				// in-case there are no players left destroy the lobby
				if len(l.conns) == 0 {
					//s.log.Debug("l.Clients == 0")
					err := s.RemoveLobby(c.GetClient().ID)
					if err != nil {
						s.log.Warn("s.RemoveLobby != nil: %v", err)
					}
				}
			}
		}
	}(l, remove)
	l.log.Info("created lobby: %v - %v", l.Type.String(), l.ID)

	return nil
}

// GetLobbyByID returns a lobby by it's id.
func (s *service) GetLobbyByID(id string) (*Lobby, error) {
	if len(id) == s.config.IDLen {
		l, err := s.repo.GetByID(id)
		if err == nil {
			return l, err
		}

		return nil, fmt.Errorf("repo.GetByID: %w", err)
	} else {
		return nil, ErrID
	}
}

// GetAllLobbies returns all lobbies.
func (s *service) GetAllLobbies() ([]*Lobby, error) {
	l, err := s.repo.GetAll()
	if err == nil {
		return l, nil
	}

	return l, fmt.Errorf("repo.GetAll: %w", err)
}

/*
func (s *service) GetLobbyByClientID(id string) (*Lobby, error) {
	_, err := s.repo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("repo.GetAll: %w", err)
	}

	return nil, nil
}
*/

// UpdateLobby updates a lobby.
func (s *service) UpdateLobby(id string, lobby *Lobby) error {
	err := s.repo.Update(id, lobby)
	if err == nil {
		return nil
	}

	return fmt.Errorf("repo.Update: %w", nil)
}

// RemoveLobby deletes a lobby from the repo.
func (s *service) RemoveLobby(id string) error {
	err := s.repo.Remove(id)
	if err == nil {
		return nil
	}

	return fmt.Errorf("repo.Update: %w", err)
}
