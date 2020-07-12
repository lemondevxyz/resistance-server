package client

// Repository is the repo in which we store the clients. it has abstract methods that Create, Read, Update and Delete.
type Repository interface {
	// Create pass a client and it'll insert it into the database, and will return the error.
	Create(Client) error
	// GetByID returns a client by it's id.
	GetByID(string) (Client, error)
	// GetAll returns all clients
	GetAll() ([]Client, error)
	// Update updates the client. Note: if you want to update one field you'll need to GetByID() then modify the field.
	Update(string, Client) error
	// Remove deletes a client by it's ID.
	Remove(string) error
	// IsValid returns if the repository is valid.
	IsValid() bool
}
