package tcp

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/timezone"
)

type Mode int

const (
	packageName      = "tcp"
	Follower    Mode = iota
	Leader
)

type Server struct {
	port    int                `json:"-"`
	clients map[string]*Client `json:"-"`
	b       *Balancer          `json:"-"`
	mode    atomic.Value       `json:"-"`
	mu      sync.Mutex         `json:"-"`
}

func NewServer(port int) *Server {
	result := &Server{
		port:    port,
		clients: make(map[string]*Client),
		mu:      sync.Mutex{},
	}
	result.mode.Store(Follower)
	return result
}

/**
* SetMode
* @param m Mode
**/
func (s *Server) SetMode(m Mode) {
	s.mode.Store(m)
}

/**
* AddNode
* @param address string
**/
func (s *Server) AddNode(address string) {
	node := newNode(address)
	s.b.nodes = append(s.b.nodes, node)
}

/**
* handle
* @param conn net.Conn
**/
func (s *Server) handle(conn net.Conn) {
	mode := s.mode.Load().(Mode)

	switch mode {
	case Leader:
		s.handleBalancer(conn)
	default:
		s.handleServer(conn)
	}
}

/**
* handleServer
* @param conn net.Conn
**/
func (s *Server) handleServer(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		data := buf[:n]
		logs.Logf(packageName, msg.MSG_TCP_RECEIVED, string(data))

		conn.Write([]byte("ACK: "))
		conn.Write(data)
	}
}

/**
* handleBalancer
* @param client net.Conn
**/
func (s *Server) handleBalancer(client net.Conn) {
	defer client.Close()

	node := s.b.next()
	if node == nil {
		return
	}

	backend, err := net.Dial("tcp", node.Address)
	if err != nil {
		return
	}
	defer backend.Close()

	node.Conns.Add(1)
	defer node.Conns.Add(-1)

	go io.Copy(backend, client)
	io.Copy(client, backend)
}

/**
* handleClient
* @param c *Client
**/
func (s *Server) handleClient(c *Client) {
	defer s.disconnectClient(c)

	go s.writeLoop(c)
	s.readLoop(c)
}

/**
* disconnectClient
* @param c *Client
**/
func (s *Server) disconnectClient(c *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Status == Disconnected {
		return
	}

	c.Status = Disconnected
	c.conn.Close()
	close(c.outbound)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, c.Addr)

	logs.Log(packageName, msg.MSG_CLIENT_DISCONNECTED, c.Addr)
}

/**
* readLoop
* @param c *Client
**/
func (s *Server) readLoop(c *Client) {
	buf := make([]byte, 1024)

	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			logs.Log(packageName, msg.MSG_CLIENT_DISCONNECTED, c.Addr)
			return
		}

		data := buf[:n]
		logs.Log(packageName, msg.MSG_TCP_RECEIVED, c.Addr, string(data))

		// ACK simple
		c.Send(TextMessage, data)
	}
}

/**
* writeLoop
* @param c *Client
**/
func (s *Server) writeLoop(c *Client) {
	for out := range c.outbound {
		if c.Status != Connected {
			return
		}

		_, err := c.conn.Write(out.message)
		if err != nil {
			return
		}
	}
}

/**
* newClient
* @param conn net.Conn
* @return *Client
**/
func (s *Server) newClient(conn net.Conn) *Client {
	return &Client{
		Created_at: timezone.Now(),
		Addr:       conn.RemoteAddr().String(),
		Status:     Connected,
		conn:       conn,
		outbound:   make(chan Outbound, 128),
		ctx:        context.Background(),
	}
}

/**
* Broadcast
* @param destination []string
* @param msg []byte
**/
func (s *Server) Broadcast(destination []string, msg []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, addr := range destination {
		client, ok := s.clients[addr]
		if ok && client.Status == Connected {
			client.Send(TextMessage, msg)
		}
	}
}

/**
* Start
* @return error
**/
func (s *Server) Start() error {
	address := fmt.Sprintf(":%d", s.port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	logs.Logf(packageName, msg.MSG_TCP_LISTENING, s.port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		client := s.newClient(conn)

		s.mu.Lock()
		s.clients[client.Addr] = client
		s.mu.Unlock()

		logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, client.Addr)

		go s.handle(conn)
	}
}
