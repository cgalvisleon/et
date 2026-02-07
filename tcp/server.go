package tcp

import (
	"fmt"
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
	port    int
	nodes   []*Node
	clients []*Client
	b       *Balancer
	mode    atomic.Value
	mu      sync.Mutex
}

func NewServer(port int) *Server {
	result := &Server{
		port:    port,
		nodes:   []*Node{},
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
	s.nodes = append(s.nodes, node)
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
* Listen
* @return error
**/
func (s *Server) Listen() error {
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
