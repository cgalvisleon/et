package tcp

import (
	"net"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
)

type Client struct {
	addr      string
	conn      net.Conn
	mu        sync.Mutex
	connected bool
	isDebug   bool
}

func newClient(addr string) *Client {
	return &Client{
		addr:      addr,
		mu:        sync.Mutex{},
		connected: false,
	}
}

/**
* connect
* @return error
**/
func (s *Client) connect() error {
	conn, err := net.DialTimeout("tcp", s.addr, 3*time.Second)
	if err != nil {
		return err
	}

	tcp := conn.(*net.TCPConn)
	tcp.SetKeepAlive(true)
	tcp.SetKeepAlivePeriod(10 * time.Second)

	s.conn = conn
	s.connected = true

	if s.isDebug {
		logs.Debug("conectado a", s.addr)
	}
	return nil
}

/**
* close
**/
func (s *Client) close() {
	if s.conn != nil {
		s.conn.Close()
	}
	s.connected = false
}

func (s *Client) connectionCheck() {
	for {
		if err := s.connect(); err != nil {
			logs.Debug("Reintentando conexión...")
			time.Sleep(2 * time.Second)
			continue
		}
		return
	}
}
