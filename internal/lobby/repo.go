package lobby

// Repository is the repo in which we store the lobbies. it has abstract methods that Create, Read, Update and Delete.
type Repository interface {
	// Create pass a lobby and it'll insert it into the database, and will return the error.
	Create(l *Lobby) error
	// GetByID returns a lobby by it's id.
	GetByID(id string) (*Lobby, error)
	// GetAll returns all lobbies
	GetAll() ([]*Lobby, error)
	// Update updates the lobby. Note: if you want to update one field you'll need to GetByID() then modify the field.
	Update(id string, l *Lobby) error
	// Remove deletes a lobby by it's ID.
	Remove(id string) error
	// IsValid returns a boolean value indicating the validity of the repository.
	IsValid() bool
}
