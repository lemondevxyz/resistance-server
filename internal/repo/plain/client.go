package plain

import (
	"sync"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/repo"
)

type clientRepository struct {
	db map[string]client.Client
	mx sync.Mutex
}

func NewClientRepository() client.Repository {
	repo := &clientRepository{
		db: map[string]client.Client{},
	}

	return repo
}

func (r *clientRepository) Create(c client.Client) error {
	if r.db == nil {
		return repo.ErrClientInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	_, ok := r.db[c.ID]
	if ok {
		return repo.ErrClientExists
	}

	r.db[c.ID] = c
	return nil
}

func (r *clientRepository) GetByID(id string) (client.Client, error) {
	var c client.Client
	if r.db == nil {
		return c, repo.ErrClientInvalid
	}

	c, ok := r.db[id]
	if !ok {
		return c, repo.ErrClient404
	}

	return c, nil
}

func (r *clientRepository) GetAll() ([]client.Client, error) {
	arr := []client.Client{}
	if r.db == nil {
		return arr, repo.ErrClientInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()
	for _, v := range r.db {
		arr = append(arr, v)
	}

	return arr, nil
}

func (r *clientRepository) Update(id string, c client.Client) error {
	if r.db == nil {
		return repo.ErrClientInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	c, ok := r.db[id]
	if !ok {
		return repo.ErrClient404
	}

	r.db[id] = c
	return nil
}

func (r *clientRepository) Remove(id string) error {
	if r.db == nil {
		return repo.ErrClientInvalid
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	_, ok := r.db[id]
	if !ok {
		return repo.ErrClient404
	}

	delete(r.db, id)
	return nil
}

func (r *clientRepository) IsValid() bool {
	if r.db == nil {
		return false
	}

	return true
}
