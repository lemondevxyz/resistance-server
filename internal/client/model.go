package client

import (
	"github.com/toms1441/resistance-server/internal/discord"
)

// Client is a client made out of a discord user(discord.User)
type Client struct {
	discord.User `json:"user"`
}
