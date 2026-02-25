package tcp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	mg "github.com/cgalvisleon/et/msg"
)

type Mode int

const (
	packageName      = "tcp"
	Follower    Mode = iota
	Candidate
	Leader
	Proxy
)

var (
	hostName string
)

func init() {
	var err error
	hostName, err = os.Hostname()
	if err != nil {
		logs.Panic(err)
	}
}

type Node struct {
	addr       string                   `json:"-"`
	port       int                      `json:"-"`
	mode       atomic.Value             `json:"-"`
	timeout    time.Duration            `json:"-"`
	ln         net.Listener             `json:"-"`
	register   chan *Client             `json:"-"`
	unregister chan *Client             `json:"-"`
	clients    map[string]*Client       `json:"-"`
	muClients  sync.Mutex               `json:"-"`
	inbound    chan *Msg                `json:"-"`
	requests   map[string]chan *Message `json:"-"`
	muRequests sync.Mutex               `json:"-"`
	proxy      *Balancer                `json:"-"`
}

/**
* NewNode
* @param port int
* @return *Node
**/
func NewNode(port int) *Node {
	addr := fmt.Sprintf("%s:%d", hostName, port)
	timeout, err := time.ParseDuration(envar.GetStr("TIMEOUT", "10s"))
	if err != nil {
		timeout = 10 * time.Second
	}

	result := &Node{
		addr:       addr,
		port:       port,
		timeout:    timeout,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		muClients:  sync.Mutex{},
		inbound:    make(chan *Msg),
		requests:   make(map[string]chan *Message),
		muRequests: sync.Mutex{},
	}
	result.mode.Store(Follower)

	return result
}

/**
* run
**/
func (s *Node) run() {
	go func() {
		for {
			select {
			case client := <-s.register:
				s.connect(client)
			case client := <-s.unregister:
				s.disconnect(client)
			}
		}
	}()

	go func() {
		for msg := range s.inbound {
			s.inbox(msg)
		}
	}()
}

/**
* connect
* @param client *Client
**/
func (s *Node) connect(client *Client) {
	s.muClients.Lock()
	s.clients[client.Addr] = client
	s.muClients.Unlock()

	logs.Logf(packageName, mg.MSG_TCP_CONNECTED_CLIENT, client.Addr)
}

/**
* disconnect
* @param client *Client
**/
func (s *Node) disconnect(client *Client) {
	s.muClients.Lock()
	delete(s.clients, client.Addr)
	s.muClients.Unlock()

	logs.Logf(packageName, mg.MSG_TCP_DISCONNECTED_CLIENT, client.Addr)
}

/**
* inbox
* @param msg *Message
**/
func (s *Node) inbox(msg *Msg) {
	s.muRequests.Lock()
	ch, ok := s.requests[msg.ID()]
	s.muRequests.Unlock()

	logs.Logf(packageName, mg.MSG_TCP_INBOX, msg.ID())

	if ok {
		ch <- msg.Msg
		return
	}

	switch msg.Msg.Type {
	}
}

/**
* send
* @param msg *Msg
* @return error
**/
func (s *Node) send(msg *Msg) error {
	s.muClients.Lock()
	c, ok := s.clients[msg.To.Addr]
	s.muClients.Unlock()

	if !ok {
		return fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, msg.To.Addr)
	}

	err := c.send(msg.Msg)
	if err != nil {
		return err
	}

	return nil
}

/**
* handle
* @param conn net.Conn
**/
func (s *Node) handle(c *Client) {
	mode := s.mode.Load().(Mode)

	switch mode {
	case Proxy:
		s.handleBalancer(c.conn)
	default:
		s.handleClient(c)
	}
}

/**
* handleBalancer
* @param client net.Conn
**/
func (s *Node) handleBalancer(client net.Conn) {
	defer client.Close()

	if s.proxy == nil {
		s.proxy = newBalancer()
	}

	node := s.proxy.next()
	if node == nil {
		return
	}

	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	backend, err := dialer.Dial("tcp", node.Address)
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
func (s *Node) handleClient(c *Client) {
	reader := bufio.NewReader(c.conn)

	for {
		// Leer tamaño (4 bytes)
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			if err != io.EOF {
				logs.Logf(packageName, mg.MSG_TCP_ERROR_READ, err)
			}
			s.unregister <- c
			return
		}

		// Leer tamaño payload
		length := binary.BigEndian.Uint32(lenBuf)
		limitReader := envar.GetInt("LIMIT_SIZE_MG", 10)
		if length > uint32(limitReader*1024*1024) {
			s.Send(c, ErrorMessage, mg.MSG_TCP_MESSAGE_TOO_LARGE)
			continue
		}

		// Leer payload completo
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			s.unregister <- c
			return
		}

		m, err := ToMessage(data)
		if err != nil {
			logs.Error(err)
			continue
		}

		s.inbound <- &Msg{
			To:  c,
			Msg: m,
		}
	}
}

/**
* Start
* @return error
**/
func (s *Node) Start() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}

	s.run()
	return nil
}
