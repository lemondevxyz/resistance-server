package client

import (
	"fmt"

	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/logger"
	"github.com/toms1441/resistance-server/internal/repo"
)

// Service is a service that uses the repo in-order to do database actions.
type Service interface {
	// CreateClient creates a client from a websocket connection and a discord user
	CreateClient(du discord.User) (Client, error)
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

func (s *service) CreateClient(du discord.User) (Client, error) {
	c := Client{
		User: du,
	}

	if !du.IsValid() {
		return c, fmt.Errorf("!discord.User.IsValid()")
	}

	_, err := s.GetClientByID(c.ID)
	if err == nil {
		return c, repo.ErrClientExists
	}

	err = s.repo.Create(c)
	if err != nil {
		return c, fmt.Errorf("repo.Create: %w", err)
	}

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
