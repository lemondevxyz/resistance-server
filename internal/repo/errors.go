package repo

import "errors"

// Errors to be used across packages
// Client
var ErrClientExists = errors.New("Client already exists")
var ErrClient404 = errors.New("Client does not exist")
var ErrClientInvalid = errors.New("Client has not been initialized")

// Lobby
var ErrLobbyExists = errors.New("Lobby already exists")
var ErrLobby404 = errors.New("Lobby does not exist")
var ErrLobbyInvalid = errors.New("Lobby has not been initialized")
