package tcp

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

type Mode int

const (
	Follower Mode = iota
	Leader
)

type Server struct {
	port    int          `json:"-"`
	clients []*Client    `json:"-"`
	b       *Balancer    `json:"-"`
	mode    atomic.Value `json:"-"`
	mu      sync.Mutex   `json:"-"`
}

func NewServer(port int) *Server {
	result := &Server{
		port:    port,
		clients: []*Client{},
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
		logs.Log("TCP", "Recibido:", string(data))

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
* Start
* @return error
**/
func (s *Server) Start() error {
	address := fmt.Sprintf(":%d", s.port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	logs.Logf("TCP", msg.MSG_TCP_LISTENING, s.port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go s.handle(conn)
	}
}
