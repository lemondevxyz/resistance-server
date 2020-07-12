package conn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/toms1441/resistance-server/internal/logger"
)

type mock struct {
	log  logger.Logger
	cmd  map[string]MessageStruct
	mtx  sync.Mutex
	done []chan bool
	pipe net.Conn
}

func NewMockConnHelper(slog, clog logger.Logger) (sconn Conn, cconn Conn) {
	spipe, cpipe := net.Pipe()

	if slog == nil {
		slog = logger.NullLogger()
	}

	if clog == nil {
		clog = logger.NullLogger()
	}

	return NewMockConn(spipe, slog), NewMockConn(cpipe, clog)
}

func NewMockConn(cl net.Conn, log logger.Logger) Conn {

	m := &mock{
		pipe: cl,
		log:  log,
	}

	if log == nil {
		m.log = logger.NullLogger()
	}

	m.cmd = map[string]MessageStruct{}

	go func(m *mock) {

		var body = make([]byte, 1024*8)
		for {
			n, err := m.pipe.Read(body)
			if n == 0 {
				continue
			}

			if err != nil {
				m.Destroy()
			}

			bts := bytes.Trim(body, "\x00")

			messagejson := MessageRecv{}
			err = json.Unmarshal(bts, &messagejson)
			if err != nil {
				fmt.Println(string(body))
				fmt.Println(err)
			}

			if len(messagejson.Group) > 0 {
				m.mtx.Lock()
				strct, ok := m.cmd[messagejson.Group]
				if ok {
					if len(messagejson.Name) > 0 {
						callback, ok := strct[messagejson.Name]
						if ok {
							err = callback(m.log, messagejson.Body)
							if err != nil {
								fullname := fmt.Sprintf("%s.%s", messagejson.Group, messagejson.Name)
								m.log.Debug("c.MessageRecv: %s %v", fullname, err)
							}
						}
					}
				} else {
					m.log.Warn("!c.cmd.(bool): %+v", strct)
				}
				m.mtx.Unlock()
			}
		}
	}(m)

	return m
}

func (m *mock) AddCommand(group string, msgstrct MessageStruct) {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	cmd, ok := m.cmd[group]
	// if group exists
	if ok {
		for k, v := range msgstrct {
			cmd[k] = v
		}

		m.cmd[group] = cmd
	} else {
		m.cmd[group] = msgstrct
	}
}

func (m *mock) ExecuteCommand(group, name string, body []byte) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	g, ok := m.cmd[group]
	if ok {
		cmd, ok := g[name]
		if ok {
			val := cmd(m.log, body)
			return val
		}

		return fmt.Errorf("c.cmd[name].(bool) != true")
	}

	return fmt.Errorf("c.cmd[group].(bool) != true")
}

func (m *mock) RemoveCommandsByGroup(group string) {
	defer m.mtx.Unlock()
	m.mtx.Lock()
	delete(m.cmd, group)
	m.log.Debug("conn.RemoveCommandsByGroup: %s", group)
}

func (m *mock) RemoveCommandsByNames(group string, names ...string) {
	_, ok := m.cmd[group]
	if ok {
		defer m.mtx.Unlock()
		m.mtx.Lock()
		for _, v := range names {
			delete(m.cmd[group], v)
		}
	}
}

func (m *mock) WriteMessage(ms MessageSend) error {
	body, err := json.Marshal(ms)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	m.WriteBytes(body)
	m.log.Debug("c.SendMessage: %s.%s", ms.Group, ms.Name)

	return nil
}

func (m *mock) WriteBytes(body []byte) {
	m.pipe.Write(body)
}

func (m *mock) GetDone() chan bool {
	var done = make(chan bool)
	if m.done == nil {
		m.done = []chan bool{}
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.done = append(m.done, done)
	return done
}

func (m *mock) Destroy() {

	m.mtx.Lock()
	for _, v := range m.done {
		v <- true
	}
	m.pipe.Close()
	m.mtx.Unlock()

}
