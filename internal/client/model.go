package client

import (
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/discord"
)

// Client is a client made out of a network connection(net.Conn), and a discord user(discord.User)
type Client struct {
	ID   string       `json:"id"`
	User discord.User `json:"user"`
	Conn conn.Conn    `json:"-"`
	// Lazy way to connect lobby and client
	LobbyID string `json:"-"`
}

// Send is a method that sends the user information to the connection
func (c Client) Send() {
	c.Conn.WriteMessage(conn.MessageSend{
		Group: "client",
		Name:  "get",
		Body:  c,
	})
}

// IsValid is a method that checks the validity of the client.
func (c Client) IsValid() bool {
	if c != (Client{}) {
		if c.Conn != nil {
			return true
		}
	}

	return false
}
