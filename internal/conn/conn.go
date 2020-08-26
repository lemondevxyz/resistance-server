package conn

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/logger"
)

type sendChannel struct {
	c      chan []byte
	closed bool
	mutex  sync.Mutex
}

// conn is a websocket-friendly wrapper to net.Conn, it helpers to ease development.
type connStruct struct {
	log  logger.Logger            // a logger used for debugging
	conn net.Conn                 // an actual connection
	send *sendChannel             // a channel to send bytes to the connection
	cmd  map[string]MessageStruct // a map of commands
	cl   client.Client

	mtx sync.Mutex

	// Done is an indicator if the connection has been closed. it's meant for external use.
	done []chan bool
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

func newSendChannel(C chan []byte) *sendChannel {
	return &sendChannel{
		c: C,
	}
}

func (s *sendChannel) safesend(bytes []byte) {
	if !s.isclosed() {
		s.c <- bytes
	}
}

func (s *sendChannel) close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.closed {
		close(s.c)
		s.closed = true
	}
}

func (s *sendChannel) isclosed() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.closed
}

var connsmtx sync.Mutex

// repository cannot type interface and have access to functions.
var conns = map[string]Conn{}

func AllConn() map[string]Conn {
	return conns
}

// NewConn creates the helper from an upgraded websocket connection and logger.
func NewConn(conn net.Conn, cl client.Client) Conn {

	c := &connStruct{
		conn: conn,
		send: newSendChannel(make(chan []byte, 256)),
		log:  logger.NullLogger(),
		cmd:  map[string]MessageStruct{},
		cl:   cl,
	}

	connsmtx.Lock()
	oldconn, ok := conns[cl.ID]
	if ok {
		done := oldconn.GetDone()
		oldconn.Destroy()
		<-done
		delete(conns, cl.ID)
	}
	connsmtx.Unlock()

	conns[cl.ID] = c

	go c.write()
	go c.read()

	return c
}

// AddCommand adds a command to the command list.
func (c *connStruct) AddCommand(group string, msgstrct MessageStruct) {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	cmd, ok := c.cmd[group]
	// if group exists
	if ok {
		for k, v := range msgstrct {
			cmd[k] = v
		}

		c.cmd[group] = cmd
	} else {
		c.cmd[group] = msgstrct
	}
}

func (c *connStruct) ExecuteCommand(group, name string, bytes []byte) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	g, ok := c.cmd[group]
	if ok {
		cmd, ok := g[name]
		if ok {
			val := cmd(c.log, bytes)
			return val
		}

		return fmt.Errorf("c.cmd[name].(bool) != true")
	}

	return fmt.Errorf("c.cmd[group].(bool) != true")
}

// RemoveCommandByGroup removes a whole group from the command list
func (c *connStruct) RemoveCommandsByGroup(group string) {
	defer c.mtx.Unlock()
	c.mtx.Lock()
	delete(c.cmd, group)
	c.log.Debug("conn.RemoveCommandsByGroup: %s", group)
}

func (c *connStruct) RemoveCommandsByNames(group string, names ...string) {
	_, ok := c.cmd[group]
	if ok {
		defer c.mtx.Unlock()
		c.mtx.Lock()
		for _, v := range names {
			delete(c.cmd[group], v)
		}
	}
}

// Destroy gets called whenever the connection has been closed, or when the send channel closes. or when it gets called externally.
func (c *connStruct) Destroy() {
	c.conn.Close()
	if !c.send.isclosed() {
		c.mtx.Lock()
		for _, v := range c.done {
			v <- true
		}
		c.mtx.Unlock()
	}

	c.send.close()
}

// SendMessage sends a message wrapped in MessageSend struct.
func (c *connStruct) WriteMessage(ms MessageSend) error {
	bytes, err := json.Marshal(ms)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	c.WriteBytes(bytes)
	c.log.Debug("c.SendMessage: %s.%s", ms.Group, ms.Name)

	return nil
}

func (c *connStruct) WriteBytes(bytes []byte) {
	c.send.safesend(bytes)
}

// GetDone returns a channel bool that gets set when the Destroy function gets called.
func (c *connStruct) GetDone() chan bool {
	var done = make(chan bool)
	if c.done == nil {
		c.done = []chan bool{}
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.done = append(c.done, done)
	return done
}

func (c *connStruct) GetClient() client.Client {
	return c.cl
}

func (c *connStruct) SetLogger(log logger.Logger) {
	c.log = log
}

func (c *connStruct) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func(c *connStruct) {
		ticker.Stop()
		c.Destroy()
	}(c)

	for {
		select {
		case message, ok := <-c.send.c:
			if !ok {
				// The hub closed the channel.
				wsutil.WriteServerMessage(c.conn, ws.OpClose, nil)
				return
			}

			writer := wsutil.NewWriter(c.conn, ws.StateServerSide, ws.OpText)
			_, err := writer.Write(message)
			if err != nil {
				return
			}
			writer.Flush()

			// Add queued chat messages to the current websocket message.
			for i := 0; i < len(c.send.c); i++ {
				_, err := writer.Write(<-c.send.c)
				if err != nil {
					return
				}
				writer.Flush()
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsutil.WriteServerMessage(c.conn, ws.OpPing, nil); err != nil {
				return
			}
		}
	}
}

func (c *connStruct) read() {

	defer func(c *connStruct) {
		c.Destroy()
	}(c)

	for {
		bytes, opcode, err := wsutil.ReadClientData(c.conn)
		if err != nil {
			return
		}

		if opcode == ws.OpText {
			messagejson := &MessageRecv{}
			json.Unmarshal(bytes, messagejson)
			if len(messagejson.Group) > 0 {
				c.mtx.Lock()
				strct, ok := c.cmd[messagejson.Group]
				if ok {
					if len(messagejson.Name) > 0 {
						callback, ok := strct[messagejson.Name]
						if ok {
							err = callback(c.log, messagejson.Body)
							if err != nil {
								fullname := fmt.Sprintf("%s.%s", messagejson.Group, messagejson.Name)
								c.log.Debug("c.MessageRecv: %s %v", fullname, err)
							}
						}
					}
				} else {
					c.log.Warn("!c.cmd.(bool): %+v", strct)
				}
				c.mtx.Unlock()
			}
		} else if opcode == ws.OpClose {
			return
		}

	}
}
