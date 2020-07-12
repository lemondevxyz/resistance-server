package conn

import (
	"encoding/json"

	"github.com/toms1441/resistance-server/internal/logger"
)

type Conn interface {
	// AddCommands adds a command by a group and message struct. if the group already exists, we added the strct to the group.
	AddCommand(group string, strct MessageStruct)
	// ExecuteCommand executes a command manually by providing group, name, and body.
	ExecuteCommand(group, name string, body []byte) error
	// RemoveCommandsByGroup removes all commands by matching group.
	RemoveCommandsByGroup(group string)
	// RemoveCommandsByNames removes all commands by matching group and name.
	RemoveCommandsByNames(group string, name ...string)

	// WriteMessage writes a message to the connection.
	WriteMessage(ms MessageSend) error
	// WriteBytes writes bytes to the connection.
	WriteBytes(body []byte)
	// GetDone returns a channel that gets set, if the connection has been destroyed.
	GetDone() chan bool
	// Destroy destroys the connection.
	Destroy()
}

// MessageStruct is a struct to store commands.
type MessageStruct map[string]MessageCallback

// MessageCallback is used whenever a command gets called it has a name field, if the name field exists in Conn.CMD MessageCallback gets executed.
type MessageCallback func(log logger.Logger, bytes []byte) error

// MessageRecv is the struct we use whenever the client has sent a message.
type MessageRecv struct {
	Group string          `json:"group"`
	Name  string          `json:"name"`
	Body  json.RawMessage `json:"body"`
}

// MessageSend is the struct we use whenever we want to send the client a message. We could use it through conn.SendMessage
type MessageSend struct {
	Group string      `json:"group"`
	Name  string      `json:"name"`
	Body  interface{} `json:"body"`
}
