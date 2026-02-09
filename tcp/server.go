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
	port            int                `json:"-"`
	clients         map[string]*Client `json:"-"`
	register        chan *Client       `json:"-"`
	unregister      chan *Client       `json:"-"`
	onConnection    []func(*Client)    `json:"-"`
	onDisconnection []func(*Client)    `json:"-"`
	b               *Balancer          `json:"-"`
	mode            atomic.Value       `json:"-"`
	mu              sync.Mutex         `json:"-"`
}

func NewServer(port int) *Server {
	result := &Server{
		port:            port,
		clients:         make(map[string]*Client),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		onConnection:    make([]func(*Client), 0),
		onDisconnection: make([]func(*Client), 0),
		mu:              sync.Mutex{},
	}
	result.mode.Store(Follower)
	return result
}

/**
* run
**/
func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.onConnect(client)
		case client := <-s.unregister:
			s.onDisconnect(client)
		}
	}
}

/**
* defOnConnect
* @param *Client client
**/
func (s *Server) onConnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client.Addr] = client
	logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, client.ToJson().ToString())
	for _, fn := range s.onConnection {
		fn(client)
	}

	go s.handle(client)
}

/**
* onDisconnect
* @param *Client client
**/
func (s *Server) onDisconnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logs.Logf(packageName, msg.MSG_CLIENT_DISCONNECTED, client.Addr)

	_, ok := s.clients[client.Addr]
	if ok {
		s.clients[client.Addr].Status = Disconnected
		for _, fn := range s.onDisconnection {
			fn(client)
		}

		delete(s.clients, client.Addr)
	}
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
func (s *Server) handle(c *Client) {
	mode := s.mode.Load().(Mode)

	switch mode {
	case Leader:
		s.handleBalancer(c.conn)
	default:
		s.handleClient(c)
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

	go s.write(c)
	s.read(c)
}

/**
* disconnectClient
* @param c *Client
**/
func (s *Server) disconnectClient(c *Client) {
	s.unregister <- c
}

/**
* read
* @param c *Client
**/
func (s *Server) read(c *Client) {
	buf := make([]byte, 1024)

	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				logs.Info("cliente cerró la conexión")
			} else {
				logs.Error(err)
			}
			return
		}

		data := buf[:n]
		out, err := ToOutbound(data)
		if err != nil {
			logs.Error(err)
			return
		}

		logs.Log(packageName, msg.MSG_TCP_RECEIVED, c.Addr, out.Message)

		// ACK simple
		c.Send(TextMessage, data)
	}
}

/**
* write
* @param c *Client
**/
func (s *Server) write(c *Client) {
	for out := range c.outbound {
		if c.Status != Connected {
			return
		}

		_, err := c.conn.Write(out.Message)
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
* broadcast
* @param destination []string
* @param msg []byte
**/
func (s *Server) broadcast(destination []string, msg []byte) {
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

	go s.run()

	logs.Logf(packageName, msg.MSG_TCP_LISTENING, s.port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		client := s.newClient(conn)
		s.register <- client
	}
}
