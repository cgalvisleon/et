package tcp

import (
	"bufio"
	"context"
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
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Mode int

const (
	packageName      = "tcp"
	Follower    Mode = iota
	Candidate
	Leader
	Proxy
)

type Msg struct {
	To  *Client  `json:"to"`
	Msg *Message `json:"msg"`
}

type Server struct {
	address         string                    `json:"-"`
	port            int                       `json:"-"`
	clients         map[string]*Client        `json:"-"`
	inbound         chan *Msg                 `json:"-"`
	outbound        chan *Msg                 `json:"-"`
	pending         map[string]chan *Message  `json:"-"`
	register        chan *Client              `json:"-"`
	unregister      chan *Client              `json:"-"`
	onConnection    []func(*Client)           `json:"-"`
	onDisconnection []func(*Client)           `json:"-"`
	onStart         []func(*Server)           `json:"-"`
	onError         []func(*Client, error)    `json:"-"`
	onOutbound      []func(*Client, *Message) `json:"-"`
	onInbound       []func(*Client, *Message) `json:"-"`
	mode            atomic.Value              `json:"-"`
	mu              sync.Mutex                `json:"-"`
	isDebug         bool                      `json:"-"`
	// Balancer
	proxy *Balancer `json:"-"`
	// Cluster
	peers          []*Client       `json:"-"`
	state          Mode            `json:"-"`
	term           int             `json:"-"`
	votedFor       string          `json:"-"`
	leaderID       string          `json:"-"`
	lastHeartbeat  time.Time       `json:"-"`
	muCluster      sync.Mutex      `json:"-"`
	onBecomeLeader []func(*Server) `json:"-"`
	onChangeLeader []func(*Server) `json:"-"`
}

func NewServer(port int) *Server {
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	address := fmt.Sprintf("%s:%d", host, port)
	isDebug := envar.GetBool("IS_DEBUG", false)
	result := &Server{
		address:         address,
		port:            port,
		clients:         make(map[string]*Client),
		inbound:         make(chan *Msg),
		outbound:        make(chan *Msg),
		pending:         make(map[string]chan *Message),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		onConnection:    make([]func(*Client), 0),
		onDisconnection: make([]func(*Client), 0),
		onStart:         make([]func(*Server), 0),
		onError:         make([]func(*Client, error), 0),
		onOutbound:      make([]func(*Client, *Message), 0),
		onInbound:       make([]func(*Client, *Message), 0),
		mu:              sync.Mutex{},
		isDebug:         isDebug,
		// Cluster
		peers:          make([]*Client, 0),
		state:          Follower,
		term:           0,
		votedFor:       "",
		leaderID:       "",
		lastHeartbeat:  timezone.Now(),
		muCluster:      sync.Mutex{},
		onBecomeLeader: make([]func(*Server), 0),
		onChangeLeader: make([]func(*Server), 0),
	}
	result.mode.Store(Follower)
	return result
}

/**
* error
* @param c *Client, err error
* @return error
**/
func (s *Server) error(c *Client, err error) error {
	for _, fn := range s.onError {
		fn(c, err)
	}

	return err
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
* inbox
**/
func (s *Server) inbox() {
	for msg := range s.inbound {
		s.mu.Lock()
		ch, ok := s.pending[msg.Msg.ID]
		s.mu.Unlock()

		if ok {
			ch <- msg.Msg
			return
		}

		for _, fn := range s.onInbound {
			fn(msg.To, msg.Msg)
		}

		if s.isDebug {
			logs.Debugf("recv: %s", msg.Msg.ToJson().ToString())
		}
	}
}

/**
* incoming
* @param c *Client
**/
func (s *Server) incoming(c *Client) {
	reader := bufio.NewReader(c.conn)

	for {
		// Leer tamaño (4 bytes)
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			if err != io.EOF {
				logs.Logf(packageName, msg.MSG_TCP_ERROR_READ, err)
			}
			s.unregister <- c
			return
		}

		// Leer tamaño payload
		length := binary.BigEndian.Uint32(lenBuf)
		limitReader := envar.GetInt("LIMIT_SIZE_MG", 10)
		if length > uint32(limitReader*1024*1024) {
			s.Send(c, ErrorMessage, msg.MSG_TCP_MESSAGE_TOO_LARGE)
			continue
		}

		// Leer payload completo
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			s.unregister <- c
			return
		}

		m, err := toMessage(data)
		if err != nil {
			logs.Error(err)
			continue
		}

		s.inbound <- &Msg{
			To:  c,
			Msg: m,
		}
		s.Send(c, ACKMessage, "")
	}
}

/**
* send
**/
func (s *Server) send() {
	for msg := range s.outbound {
		s.mu.Lock()
		c, ok := s.clients[msg.To.Addr]
		if !ok {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		if c.Status != Connected {
			return
		}

		bt, err := msg.Msg.serialize()
		if err != nil {
			s.error(msg.To, err)
			return
		}

		_, err = msg.To.conn.Write(bt)
		if err != nil {
			s.error(msg.To, err)
			return
		}

		for _, fn := range s.onOutbound {
			fn(msg.To, msg.Msg)
		}

		if s.isDebug {
			logs.Debugf("send: %s", msg.To.Addr+":"+msg.Msg.ToJson().ToString())
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
		ID:         reg.ULID(),
		Addr:       conn.RemoteAddr().String(),
		Status:     Connected,
		conn:       conn,
		ctx:        context.Background(),
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
	logs.Logf(packageName, msg.MSG_CLIENT_CONNECTED, client.toJson().ToString())
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
* handle
* @param conn net.Conn
**/
func (s *Server) handle(c *Client) {
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
func (s *Server) handleBalancer(client net.Conn) {
	defer client.Close()

	if s.proxy == nil {
		s.proxy = newBalancer()
	}

	node := s.proxy.next()
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
	if s.isDebug {
		go func() {
			for {
				time.Sleep(3 * time.Second)
				s.Send(c, PongMessage, "")
			}
		}()
	}

	go s.incoming(c)
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
	go s.inbox()
	go s.send()

	nodes, err := GetNodes()
	if err != nil {
		return err
	}

	for _, addr := range nodes {
		s.AddNode(addr)
	}

	go s.ElectionLoop()

	logs.Logf(packageName, msg.MSG_TCP_LISTENING, s.port)

	for _, fn := range s.onStart {
		fn(s)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		client := s.newClient(conn)
		s.register <- client
	}
}

/**
* StartProxy
* @return error
**/
func (s *Server) StartProxy() error {
	s.mode.Store(Proxy)
	return s.Start()
}

/**
* AddNode
* @param address string
**/
func (s *Server) AddNode(address string) {
	if s.mode.Load() == Proxy {
		if s.proxy == nil {
			s.proxy = newBalancer()
		}

		node := newNode(address)
		s.proxy.nodes = append(s.proxy.nodes, node)
	} else {
		if address == s.address {
			return
		}

		node := NewClient(address)
		err := node.Start()
		if err != nil {
			return
		}

		s.peers = append(s.peers, node)
	}
}

/**
* Send
* @param c *Client, tp int, message any
* @return error
**/
func (s *Server) Send(c *Client, tp int, message any) error {
	msg, err := newMessage(tp, message)
	if err != nil {
		return err
	}

	s.outbound <- &Msg{
		To:  c,
		Msg: msg,
	}

	return nil
}

/**
* Broadcast
* @param destination []string
* @param msg []byte
**/
func (s *Server) Broadcast(destination []string, tp int, message any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, addr := range destination {
		client, ok := s.clients[addr]
		if ok && client.Status == Connected {
			s.Send(client, tp, message)
		}
	}
}

/**
* OnConnect
* @param fn func(*Client)
**/
func (s *Server) OnConnect(fn func(*Client)) {
	s.onConnection = append(s.onConnection, fn)
}

/**
* OnDisconnect
* @param fn func(*Client)
**/
func (s *Server) OnDisconnect(fn func(*Client)) {
	s.onDisconnection = append(s.onDisconnection, fn)
}

/**
* OnStart
* @param fn func(*Server)
**/
func (s *Server) OnStart(fn func(*Server)) {
	s.onStart = append(s.onStart, fn)
}

/**
* OnError
* @param fn func(*Client, error)
**/
func (s *Server) OnError(fn func(*Client, error)) {
	s.onError = append(s.onError, fn)
}

/**
* OnOutbound
* @param fn func(*Client, *Message)
**/
func (s *Server) OnOutbound(fn func(*Client, *Message)) {
	s.onOutbound = append(s.onOutbound, fn)
}

/**
* OnInbound
* @param fn func(*Client, *Message)
**/
func (s *Server) OnInbound(fn func(*Client, *Message)) {
	s.onInbound = append(s.onInbound, fn)
}
