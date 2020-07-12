package client

import (
	"fmt"
	"net"
	"time"

	"github.com/kjk/betterguid"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/logger"
)

// Service is a service that uses the repo in-order to do database actions.
type Service interface {
	// CreateClient creates a client from a network connection and a discord user
	CreateClient(nconn net.Conn, du discord.User) (Client, error)
	// GetClientByID returns a client by id
	GetClientByID(id string) (Client, error)
	// GetAllClients returns all clients, or an empty array with an error
	GetAllClients() ([]Client, error)
	// UpdateClient updates a client from id and client. if you want to update a single field you need to get the client first then modify said field.
	UpdateClient(id string, client Client) error
	// RemoveClient deletes a client from the repo
	RemoveClient(id string) error
}

type service struct {
	repo Repository
	log  logger.Logger
}

// NewService returns a new user service. It can be used to create, read, update and delete.
func NewService(repo Repository, log logger.Logger) (Service, error) {
	if !repo.IsValid() {
		return nil, fmt.Errorf("!repo.IsValid()")
	}

	log.Info("Initiated service")

	return &service{
		repo: repo,
		log:  log,
	}, nil
}

func (s *service) CreateClient(nconn net.Conn, du discord.User) (Client, error) {
	c := Client{
		User: du,
		ID:   betterguid.New(),
	}

	if !du.IsValid() {
		return c, fmt.Errorf("!discord.User.IsValid()")
	}

	cls, err := s.GetAllClients()
	// kick out the other connection made by the same user
	if err == nil {
		for _, v := range cls {
			if v.User.ID == c.User.ID {
				// we're assuming that there is only ONE connection besides the current one.
				v.Conn.Destroy()
				s.RemoveClient(v.ID)
				time.Sleep(time.Millisecond * 50)
				break
			}
		}
	}

	log := s.log.Replicate()
	log.SetPrefix("conn")
	log.SetSuffix(c.ID)

	wsconn := conn.NewConn(nconn, log)
	c.Conn = wsconn

	// when the connection gets destroyed this function gets executed
	go func(s *service, wsconn conn.Conn, log logger.Logger, id string) {
		<-wsconn.GetDone()
		s.RemoveClient(id)
	}(s, wsconn, log, c.ID)

	err = s.repo.Create(c)
	if err != nil {
		return c, fmt.Errorf("repo.Create: %w", err)
	}

	c.Send()

	s.log.Debug("s.CreateClient: %s", c.ID)

	return c, nil
}

func (s *service) GetClientByID(id string) (Client, error) {
	client, err := s.repo.GetByID(id)
	if err != nil {
		return client, fmt.Errorf("Repository error: %w", err)
	}

	s.log.Debug("s.GetClientByID: %s", id)
	return client, nil
}

func (s *service) GetAllClients() ([]Client, error) {
	return s.repo.GetAll()
}

func (s *service) UpdateClient(id string, client Client) error {
	err := s.repo.Update(id, client)
	if err != nil {
		return fmt.Errorf("Repository error: %w", err)
	}

	s.log.Debug("s.UpdateClient: %s %v", id, client)
	return nil
}

func (s *service) RemoveClient(id string) error {
	err := s.repo.Remove(id)
	if err != nil {
		return fmt.Errorf("Repository error: %w", err)
	}

	s.log.Debug("s.RemoveClient: %s", id)
	return nil
}
