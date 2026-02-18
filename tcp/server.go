package tcp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	mg "github.com/cgalvisleon/et/msg"
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

/**
* ID
* @return string
**/
func (s *Msg) ID() string {
	return s.Msg.ID
}

/**
* Get
* @param dest any
* @return error
**/
func (s *Msg) Get(dest any) error {
	return s.Msg.Get(dest)
}

type Server struct {
	addr            string                    `json:"-"`
	port            int                       `json:"-"`
	ln              net.Listener              `json:"-"`
	clients         map[string]*Client        `json:"-"`
	inbound         chan *Msg                 `json:"-"`
	outbound        chan *Msg                 `json:"-"`
	messages        map[string]chan *Message  `json:"-"`
	register        chan *Client              `json:"-"`
	unregister      chan *Client              `json:"-"`
	onConnection    []func(*Client)           `json:"-"`
	onDisconnection []func(*Client)           `json:"-"`
	onStart         []func(*Server)           `json:"-"`
	onError         []func(*Client, error)    `json:"-"`
	onOutbound      []func(*Client, *Message) `json:"-"`
	onInbound       []func(*Client, *Message) `json:"-"`
	onMethod        []func(*Client, *Message) `json:"-"`
	mode            atomic.Value              `json:"-"`
	timeout         time.Duration             `json:"-"`
	mu              sync.Mutex                `json:"-"`
	muMessages      sync.Mutex                `json:"-"`
	isDebug         bool                      `json:"-"`
	isTesting       bool                      `json:"-"`
	// Balancer
	proxy *Balancer `json:"-"`
	// Cluster
	Peers          []*Client       `json:"-"`
	state          Mode            `json:"-"`
	term           int             `json:"-"`
	votedFor       string          `json:"-"`
	leaderID       string          `json:"-"`
	lastHeartbeat  time.Time       `json:"-"`
	turn           int             `json:"-"`
	muRaft         sync.Mutex      `json:"-"`
	muTurn         sync.Mutex      `json:"-"`
	onBecomeLeader []func(*Server) `json:"-"`
	onChangeLeader []func(*Server) `json:"-"`
	// Call
	method map[string]interface{} `json:"-"`
}

/**
* NewServer
* @param port int
* @return *Server
**/
func NewServer(port int) *Server {
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	isDebug := envar.GetBool("IS_DEBUG", false)
	isTesting := envar.GetBool("IS_TESTING", false)
	timeout, err := time.ParseDuration(envar.GetStr("TIMEOUT", "10s"))
	if err != nil {
		timeout = 10 * time.Second
	}
	result := &Server{
		addr:            addr,
		port:            port,
		clients:         make(map[string]*Client),
		inbound:         make(chan *Msg),
		outbound:        make(chan *Msg),
		messages:        make(map[string]chan *Message),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		onConnection:    make([]func(*Client), 0),
		onDisconnection: make([]func(*Client), 0),
		onStart:         make([]func(*Server), 0),
		onError:         make([]func(*Client, error), 0),
		onOutbound:      make([]func(*Client, *Message), 0),
		onInbound:       make([]func(*Client, *Message), 0),
		onMethod:        make([]func(*Client, *Message), 0),
		mu:              sync.Mutex{},
		muMessages:      sync.Mutex{},
		timeout:         timeout,
		isDebug:         isDebug,
		isTesting:       isTesting,
		// Cluster
		Peers:          make([]*Client, 0),
		state:          Follower,
		term:           0,
		votedFor:       "",
		leaderID:       "",
		lastHeartbeat:  timezone.Now(),
		muRaft:         sync.Mutex{},
		onBecomeLeader: make([]func(*Server), 0),
		onChangeLeader: make([]func(*Server), 0),
		// Call
		method: make(map[string]interface{}),
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
* newClient
* @param conn net.Conn
* @return *Client
**/
func (s *Server) newClient(conn net.Conn) *Client {
	timeout, err := time.ParseDuration(envar.GetStr("TIMEOUT", "10s"))
	if err != nil {
		timeout = 10 * time.Second
	}
	return &Client{
		CreatedAt: timezone.Now(),
		ID:        reg.ULID(),
		Addr:      conn.RemoteAddr().String(),
		Status:    Connected,
		Ctx:       et.Json{},
		conn:      conn,
		timeout:   timeout,
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
	logs.Logf(packageName, mg.MSG_CLIENT_CONNECTED_FROM, client.toJson().ToString())
	go s.handle(client)
	for _, fn := range s.onConnection {
		fn(client)
	}
}

/**
* onDisconnect
* @param *Client client
**/
func (s *Server) onDisconnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logs.Logf(packageName, mg.MSG_CLIENT_DISCONNECTED, client.Addr)

	_, ok := s.clients[client.Addr]
	if ok {
		client.Status = Disconnected
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
func (s *Server) handleClient(c *Client) {
	go s.incoming(c)
}

/**
* inbox
**/
func (s *Server) inbox() {
	for msg := range s.inbound {
		s.mu.Lock()
		ch, ok := s.messages[msg.Msg.ID]
		s.mu.Unlock()

		if ok {
			ch <- msg.Msg
			return
		}

		switch msg.Msg.Type {
		case PingMessage:
			var args string
			err := msg.Get(&args)
			if err != nil {
				s.ResponseError(msg.To, msg.ID(), err)
				return
			}

			rsp, err := NewMessage(PingMessage, "PONG")
			if err != nil {
				logs.Error(err)
				return
			}

			err = s.response(msg.To, msg.ID(), rsp)
			if err != nil {
				logs.Error(err)
			}
		case RequestVote:
			var args RequestVoteArgs
			err := msg.Get(&args)
			if err != nil {
				s.ResponseError(msg.To, msg.ID(), err)
				return
			}

			var res RequestVoteReply
			err = s.requestVote(&args, &res)
			if err != nil {
				s.ResponseError(msg.To, msg.ID(), err)
				return
			}

			rsp, err := NewMessage(RequestVote, res)
			if err != nil {
				logs.Error(err)
				return
			}

			err = s.response(msg.To, msg.ID(), rsp)
			if err != nil {
				logs.Error(err)
			}
		case Heartbeat:
			var args HeartbeatArgs
			err := msg.Get(&args)
			if err != nil {
				s.ResponseError(msg.To, msg.ID(), err)
				return
			}

			var res HeartbeatReply
			err = s.heartbeat(&args, &res)
			if err != nil {
				s.ResponseError(msg.To, msg.ID(), err)
				return
			}

			rsp, err := NewMessage(Heartbeat, res)
			if err != nil {
				logs.Error(err)
				return
			}

			err = s.response(msg.To, msg.ID(), rsp)
			if err != nil {
				logs.Error(err)
			}
		case Method:
			for _, fn := range s.onMethod {
				fn(msg.To, msg.Msg)
			}
		default:
			for _, fn := range s.onInbound {
				fn(msg.To, msg.Msg)
			}

			if s.isDebug {
				logs.Debugf("inbox: %s", msg.Msg.ToJson().ToString())
			}
		}
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
			logs.Debugf("send: %s to %s", msg.Msg.ToJson().ToString(), msg.To.Addr)
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

		m, err := toMessage(data)
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
* response
* @params to *Client, id string, tp int, message any
* @return error
**/
func (s *Server) response(to *Client, id string, msg *Message) error {
	s.mu.Lock()
	_, ok := s.clients[to.Addr]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, to.Addr)
	}

	if id != "" {
		msg.ID = id
	}

	s.outbound <- &Msg{
		To:  to,
		Msg: msg,
	}

	return nil
}

/**
* request
* @param to *Client, tp int, payload any
* @return *Message, error
**/
func (s *Server) request(to *Client, m *Message) (*Message, error) {
	s.mu.Lock()
	_, ok := s.clients[to.Addr]
	s.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, to.Addr)
	}

	// Channel for response
	ch := make(chan *Message, 1)
	s.muMessages.Lock()
	s.messages[m.ID] = ch
	s.muMessages.Unlock()

	// Send
	s.outbound <- &Msg{
		To:  to,
		Msg: m,
	}

	// Wait response or timeout
	select {
	case resp := <-ch:
		s.muMessages.Lock()
		delete(s.messages, m.ID)
		s.muMessages.Unlock()
		return resp, nil

	case <-time.After(s.timeout):
		s.muMessages.Lock()
		delete(s.messages, m.ID)
		s.muMessages.Unlock()
		return nil, fmt.Errorf(mg.MSG_TCP_TIMEOUT)
	}
}

/**
* AddNode
* @param addr string
**/
func (s *Server) AddNode(addr string) {
	if s.mode.Load() == Proxy {
		if s.proxy == nil {
			s.proxy = newBalancer()
		}

		node := newNode(addr)
		s.proxy.nodes = append(s.proxy.nodes, node)
	} else {
		if addr == s.addr {
			return
		}

		node := NewNode(addr)
		s.Peers = append(s.Peers, node)
	}
}

/**
* RemoveNode
* @param addr string
**/
func (s *Server) RemoveNode(addr string) {
	idx := slices.IndexFunc(s.Peers, func(e *Client) bool { return e.Addr == addr })
	if idx != -1 {
		s.Peers = append(s.Peers[:idx], s.Peers[idx+1:]...)
	}
}

/**
* Start
* @return error
**/
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	var err error
	s.ln, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go s.run()
	go s.inbox()
	go s.send()

	if len(s.Peers) > 0 {
		go s.ElectionLoop()
	}

	logs.Logf(packageName, mg.MSG_TCP_LISTENING, s.addr)

	for _, fn := range s.onStart {
		fn(s)
	}

	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				continue
			}

			client := s.newClient(conn)
			s.register <- client
		}
	}()

	return nil
}

/**
* Close
* @return error
**/
func (s *Server) Close() error {
	return s.ln.Close()
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
* LeaderID
* @return string, bool
**/
func (s *Server) LeaderID() (string, bool) {
	s.muRaft.Lock()
	defer s.muRaft.Unlock()
	return s.leaderID, s.state == Leader
}

/**
* GetLeader
* @return *Client, bool
**/
func (s *Server) GetLeader() (*Client, bool) {
	leader, imLeader := s.LeaderID()
	if imLeader {
		return nil, true
	}

	idx := slices.IndexFunc(s.Peers, func(e *Client) bool { return e.Addr == leader })
	if idx == -1 {
		return nil, false
	}
	return s.Peers[idx], false
}

/**
* NextTurn
* @return *Client
**/
func (s *Server) NextTurn() *Client {
	s.muTurn.Lock()
	defer s.muTurn.Unlock()
	result := s.Peers[s.turn]
	s.turn++
	return result
}

/**
* Address
* @return string, error
**/
func (s *Server) Address() string {
	return s.addr
}

/**
* Port
* @return int
**/
func (s *Server) Port() int {
	return s.port
}

/**
* Send
* @param to *Client, tp int, message any
* @return error
**/
func (s *Server) Send(to *Client, tp int, message any) error {
	msg, err := NewMessage(tp, message)
	if err != nil {
		return err
	}

	return s.response(to, "", msg)
}

/**
* SendError
* @param to *Client, err error
* @return error
**/
func (s *Server) SendError(to *Client, err error) error {
	msg, err := NewMessage(ErrorMessage, err.Error())
	if err != nil {
		return err
	}

	return s.response(to, "", msg)
}

/**
* ResponseError
* @param to *Client, id string, err error
* @return error
**/
func (s *Server) ResponseError(to *Client, id string, err error) error {
	msg, err := NewMessage(ErrorMessage, err.Error())
	if err != nil {
		return err
	}

	return s.response(to, id, msg)
}

/**
* Broadcast
* @param destination []string
* @param msg []byte
**/
func (s *Server) Broadcast(destination []string, tp int, message any) {
	for _, addr := range destination {
		client, ok := s.clients[addr]
		if ok && client.Status == Connected {
			s.Send(client, tp, message)
		}
	}
}

/**
* Request
* @param to *Client, method string, request ...any
* @return *Response
**/
func (s *Server) Request(to *Client, method string, request ...any) *Response {
	m, err := NewMessage(Method, "")
	if err != nil {
		return newResponse(nil, err)
	}
	m.Method = method
	m.Args = request

	res, err := s.request(to, m)
	if err != nil {
		return newResponse(nil, err)
	}

	response, err := res.Result()
	if err != nil {
		return newResponse(nil, err)
	}

	return newResponse(response, nil)
}

/**
* Mount
* @param services any
**/
func (s *Server) Mount(services any) error {
	if services == nil {
		return errors.New(mg.MSG_SERVICE_REQUIRED)
	}
	tipoStruct := reflect.TypeOf(services)
	structName := tipoStruct.String()
	list := strings.Split(structName, ".")
	structName = list[len(list)-1]
	name := structName
	s.method[name] = services
	return nil
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
