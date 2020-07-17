package lobby

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
	"github.com/toms1441/resistance-server/internal/repo"
)

// Lobby the lobby that the clients will be in, it gathers maximum of 10 players. and is used later to transform into a Game.
// The lobby's owner is client with index 0.
type Lobby struct {
	ID      string
	Type    Type
	Private bool
	Clients []client.Client
	conns   map[string]conn.Conn

	// lobby owner
	owner string `json:"owner"`

	insert []chan conn.Conn
	remove []chan conn.Conn

	log logger.Logger
}

type Clients []client.Client

// Type needs to be between TypeBasic(0) and TypeTrumpmode(4)
type Type uint8

func (t Type) String() string {
	switch t {
	case TypeBasic:
		return "Basic"
	case TypeOriginal:
		return "Original"
	case TypeAvalon:
		return "Avalon"
	case TypeHunter:
		return "Hunter"
	case TypeTrumpmode:
		return "Trumpmode"
	}

	return ""
}

func (t Type) Common() uint8 {
	return uint8(t)
}

const (
	// TypeBasic has spies and resistance.
	TypeBasic Type = iota
	// TypeOriginal no documentation yet.
	TypeOriginal
	// TypeAvalon same as TypeBasic only with merlin.
	// Merlin is a character that sees all spies and resistance.
	TypeAvalon
	// TypeHunter no documentation
	TypeHunter
	// TypeTrumpmode no documentation
	TypeTrumpmode
)

var (
	// ErrID if len(id) != idlength.
	ErrID = errors.New("Lobby ID is not valid")
	// ErrType if type >= TypeBasic && type <= TypeTrumpmode
	ErrType = errors.New("Lobby Type is not valid")
	// ErrChannel if l.Insert == nil || l.Remove == ninl
	ErrChannel = errors.New("Lobby channels(l.Insert, l.Remove) are nil.")
)

// Equal compares two different lobbies
func (l *Lobby) Equal(l2 *Lobby) bool {
	ref1 := *l
	ref2 := *l2

	if ref1.ID != ref2.ID {
		return false
	}

	if ref1.Type != ref2.Type {
		return false
	}

	if ref1.Private != ref2.Private {
		return false
	}

	if len(ref1.Clients) != len(ref2.Clients) {
		return false
	}

	id1 := []string{}
	for _, v := range ref1.Clients {
		if v.IsValid() {
			id1 = append(id1, v.ID)
		}
	}

	id2 := []string{}
	for _, v := range ref2.Clients {
		if v.IsValid() {
			id2 = append(id2, v.ID)
		}
	}

	len1 := len(id1)
	len2 := len(id2)
	if len1 != len2 {
		return false
	}

	for k := range id1 {
		v1 := id1[k]
		v2 := id2[k]

		if v1 != v2 {
			return false
		}
	}

	return true
}

// SetLogger sets the logger for the lobby.
// Used with lobby.service to provide better logs.
func (l *Lobby) SetLogger(log logger.Logger) {
	l.log = log
}

// Join inserts a client into the lobby, and updates the rest of the clients.
func (l *Lobby) Join(c conn.Conn) error {

	if len(c.GetClient().ID) == 0 {
		return repo.ErrClientInvalid
	}

	if l.log == nil {
		l.log = logger.NullLogger()
	}

	_, ok := l.conns[c.GetClient().ID]
	if !ok {

		if l.conns == nil {
			l.conns = map[string]conn.Conn{}
		}

		l.conns[c.GetClient().ID] = c
		if l.GetClientIndex(c.GetClient().ID) == -1 {
			l.Clients = append(l.Clients, c.GetClient())
		}

		for _, v := range l.insert {
			v <- c
		}

		keys := []string{}
		for _, v := range l.conns {
			keys = append(keys, v.GetClient().ID)
		}
		sort.Strings(keys)

		sort.Slice(l.Clients, func(i, j int) bool {
			return l.Clients[i].ID < l.Clients[j].ID
		})

		if l.owner == "" {
			l.owner = keys[0]
		}

		l.log.Debug("l.Join: %v", c.GetClient().ID)

		// When the connection closes, remove the lobby.
		go func(l *Lobby, c conn.Conn) {
			for {
				select {
				case <-c.GetDone():
					l.Leave(c)
					return
				}
			}
		}(l, c)

		err := l.Send()
		if err != nil {
			l.log.Debug("l.Send: %v", err)
			return err
		}

		return nil
	}

	return repo.ErrClientExists
}

// Leave removes a client from the lobby, and updates the rest of the clients.
func (l *Lobby) Leave(c conn.Conn) error {

	if l.log == nil {
		l.log = logger.NullLogger()
	}

	if len(c.GetClient().ID) == 0 {
		return repo.ErrClientInvalid
	}

	if i := l.GetClientIndex(c.GetClient().ID); i >= 0 {
		// order matters
		l.Clients = append(l.Clients[:i], l.Clients[i+1:]...)
	}

	_, ok := l.conns[c.GetClient().ID]
	// i.e the connection is in the map
	if ok {
		if len(l.conns) == 0 {
			return nil
		}

		// it's just easier to copy the value and replace
		// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
		delete(l.conns, c.GetClient().ID)

		// double-check cause we might've removed the last player
		if len(l.conns) > 0 {
			id := []string{}
			for k := range l.conns {
				id = append(id, k)
			}
			sort.Strings(id)

			firstc := l.conns[id[0]]
			if firstc.GetClient().IsValid() {
				l.addCommands(firstc)
			}
		}

		for _, v := range l.remove {
			v <- c
		}

		l.log.Debug("l.Remove: %v", c.GetClient().ID)
		err := l.Send()
		l.log.Debug("l.Send: %v", err)
		if err != nil {
			return err
		}

		return nil
	}

	return repo.ErrClient404
}

// SubscribeInsert returns a channel that gets set whenever a client joins
func (l *Lobby) SubscribeInsert() (insert chan conn.Conn) {
	insert = make(chan conn.Conn)
	if l.insert == nil {
		l.insert = []chan conn.Conn{}
	}

	l.insert = append(l.insert, insert)
	return insert
}

// SubscribeRemove returns a channel that gets set whenever a client leaves
func (l *Lobby) SubscribeRemove() (remove chan conn.Conn) {
	remove = make(chan conn.Conn)
	if l.remove == nil {
		l.remove = []chan conn.Conn{}
	}

	l.remove = append(l.remove, remove)
	return remove
}

func (l *Lobby) removesubscribe(slice []chan conn.Conn, chcon chan conn.Conn) []chan conn.Conn {
	a := slice

	for i, v := range slice {
		if v == chcon {

			a[i] = a[len(a)-1] // Copy last element to index i.
			a[len(a)-1] = nil  // Erase last element (write zero value).
			a = a[:len(a)-1]

			break
		}
	}

	return a
}

func (l *Lobby) RemoveSubscribeInsert(insert chan conn.Conn) {
	l.insert = l.removesubscribe(l.insert, insert)
}

func (l *Lobby) RemoveSubscribeRemove(remove chan conn.Conn) {
	l.remove = l.removesubscribe(l.remove, remove)
}

// Send send the clients information about the lobby, it's called whenever a client joins or leaves the lobby.
func (l *Lobby) Send() error {
	bytes, err := json.Marshal(l.MessageSend())

	if err != nil {
		l.log.Debug("l.Send: %v", err)
		return err
	}

	for _, v := range l.conns {
		v.WriteBytes(bytes)
	}

	return nil
}

// GetClientIndex returns the client index by id.
func (l *Lobby) GetClientIndex(id string) (i int) {
	i = -1

	for k, v := range l.Clients {
		if v.ID == id {
			i = k
			break
		}
	}

	return
}

// MessageSend is a method that returns conn.MessageSend
func (l *Lobby) MessageSend() conn.MessageSend {
	return conn.MessageSend{
		Group: "lobby",
		Name:  "get",
		Body:  l,
	}
}

// Validate validates the lobby, it includes ID and Type validators as well as validate.Struct(lobby)
func (l *Lobby) Validate() (err error) {

	// i.e	it's between BASIC(which is 0) and TRUMPMODE(which is 4)
	if l.Type < TypeBasic || l.Type > TypeTrumpmode {
		return ErrType
	}

	return
}
