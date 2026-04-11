package tcp

import (
	"bufio"
	"context"
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
	"github.com/cgalvisleon/et/file"
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

type Config struct {
	Nodes []string `json:"nodes"`
}

type Node struct {
	ctx            context.Context          `json:"-"`
	cancel         context.CancelFunc       `json:"-"`
	addr           string                   `json:"-"`
	port           int                      `json:"-"`
	mode           atomic.Value             `json:"-"`
	timeout        time.Duration            `json:"-"`
	ln             net.Listener             `json:"-"`
	register       chan *Client             `json:"-"`
	unregister     chan *Client             `json:"-"`
	clients        map[string]*Client       `json:"-"`
	workerPool     chan struct{}            `json:"-"`
	requests       map[string]chan *Message `json:"-"`
	proxy          *Balancer                `json:"-"`
	peers          []*Client                `json:"-"`
	method         map[string]Service       `json:"-"`
	raft           *Raft                    `json:"-"`
	closed         atomic.Bool              `json:"-"`
	configFile     string                   `json:"-"`
	mu             map[string]*sync.Mutex   `json:"-"`
	onConnect      []func(*Client)          `json:"-"`
	onDisconnect   []func(*Client)          `json:"-"`
	onError        []func(*Client, error)   `json:"-"`
	onInbox        []func(*Msg)             `json:"-"`
	onSend         []func(*Msg)             `json:"-"`
	onBecomeLeader []func(*Node)            `json:"-"`
	onChangeLeader []func(*Node)            `json:"-"`
	index          atomic.Uint64            `json:"-"`
	total          atomic.Int64             `json:"-"`
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

	workerCount := envar.GetInt("WORKER_COUNT", 1000)
	ctx, cancel := context.WithCancel(context.Background())
	result := &Node{
		ctx:            ctx,
		cancel:         cancel,
		addr:           addr,
		port:           port,
		timeout:        timeout,
		register:       make(chan *Client, 64),
		unregister:     make(chan *Client, 64),
		clients:        make(map[string]*Client),
		workerPool:     make(chan struct{}, workerCount), // 🚀 máximo 100 workers concurrentes
		requests:       make(map[string]chan *Message),
		peers:          make([]*Client, 0),
		method:         make(map[string]Service),
		index:          atomic.Uint64{},
		total:          atomic.Int64{},
		configFile:     envar.GetStr("CONFIG_FILE", "./config.json"),
		mu:             make(map[string]*sync.Mutex),
		onConnect:      make([]func(*Client), 0),
		onDisconnect:   make([]func(*Client), 0),
		onError:        make([]func(*Client, error), 0),
		onInbox:        make([]func(*Msg), 0),
		onSend:         make([]func(*Msg), 0),
		onBecomeLeader: make([]func(*Node), 0),
		onChangeLeader: make([]func(*Node), 0),
	}
	result.mode.Store(Follower)
	result.Mount(newTcpService(result))
	result.raft = newRaft(result)
	result.mu["clients"] = &sync.Mutex{}
	result.mu["requests"] = &sync.Mutex{}
	result.mu["peers"] = &sync.Mutex{}

	return result
}

/**
* run
**/
func (s *Node) run() {
	// ===============================
	// Loop principal (register/unregister)
	// ===============================
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return

			case client := <-s.register:
				if client != nil {
					s.connect(client)
				}

			case client := <-s.unregister:
				if client != nil {
					s.disconnect(client)
				}
			}
		}
	}()

	// ===============================
	// Accept loop
	// ===============================
	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				if s.ctx.Err() != nil || s.closed.Load() {
					return
				}
				if errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}

			// Si el nodo ya está cerrado
			if s.closed.Load() {
				_ = conn.Close()
				return
			}

			client := s.newClient(conn)

			select {
			case s.register <- client:
			case <-s.ctx.Done():
				_ = conn.Close()
				return
			default:
				_ = conn.Close()
			}
		}
	}()
}

/**
* Error
* @param c *Client, err error
* @return error
**/
func (s *Node) Error(c *Client, err error) error {
	for _, fn := range s.onError {
		fn(c, err)
	}

	logs.Error(err)
	return err
}

/**
* connect
* @param client *Client
**/
func (s *Node) connect(client *Client) {
	s.mu["clients"].Lock()
	s.clients[client.Addr] = client
	s.mu["clients"].Unlock()

	s.handle(client)
	logs.Logf(packageName, mg.MSG_TCP_CONNECTED_CLIENT, client.Addr)

	for _, fn := range s.onConnect {
		fn(client)
	}
}

/**
* disconnect
* @param client *Client
**/
func (s *Node) disconnect(client *Client) {
	s.mu["clients"].Lock()
	delete(s.clients, client.Addr)
	s.mu["clients"].Unlock()

	logs.Logf(packageName, mg.MSG_TCP_DISCONNECTED_CLIENT, client.Addr)

	for _, fn := range s.onDisconnect {
		fn(client)
	}
}

/**
* closeAllClients
**/
func (s *Node) closeAllClients() {
	s.mu["clients"].Lock()
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.clients = make(map[string]*Client)
	s.mu["clients"].Unlock()

	for _, c := range clients {
		c.Close()
	}
}

/**
* send
* @param msg *Msg
* @return error
**/
func (s *Node) send(msg *Msg) error {
	s.mu["clients"].Lock()
	c, ok := s.clients[msg.To.Addr]
	s.mu["clients"].Unlock()

	if !ok || c == nil {
		return fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, msg.To.Addr)
	}

	err := c.send(msg.Msg)
	if err != nil {
		return s.Error(c, err)
	}

	go func() {
		for _, fn := range s.onSend {
			fn(msg)
		}
	}()

	return nil
}

/**
* inbox
* @param msg *Message
**/
func (s *Node) inbox(msg *Msg) {
	s.mu["requests"].Lock()
	ch, ok := s.requests[msg.ID()]
	s.mu["requests"].Unlock()

	if ok {
		select {
		case ch <- msg.Msg:
		default:
		}
		return
	}

	switch msg.Msg.Type {
	case RequestVote:
		var args RequestVoteArgs
		err := msg.Get(&args)
		if err != nil {
			s.ResponseError(msg, err)
			return
		}

		var res RequestVoteReply
		s.raft.requestVote(&args, &res)

		rsp, err := NewMessage(RequestVote, res)
		if err != nil {
			s.ResponseError(msg, err)
			return
		}

		err = s.response(msg.To, msg.ID(), rsp)
		if err != nil {
			s.ResponseError(msg, err)
		}
	case Heartbeat:
		var args HeartbeatArgs
		err := msg.Get(&args)
		if err != nil {
			s.ResponseError(msg, err)
			return
		}

		var res HeartbeatReply
		s.raft.heartbeat(&args, &res)

		rsp, err := NewMessage(Heartbeat, res)
		if err != nil {
			s.ResponseError(msg, err)
			return
		}

		err = s.response(msg.To, msg.ID(), rsp)
		if err != nil {
			s.ResponseError(msg, err)
		}
	default:
		list := strings.Split(msg.Msg.Method, ".")
		if len(list) < 2 {
			s.ResponseError(msg, errors.New(mg.MSG_METHOD_NOT_FOUND))
			return
		}

		serviceName := list[0]
		methodName := list[1]
		service, ok := s.method[serviceName]
		if !ok {
			s.ResponseError(msg, errors.New(mg.MSG_METHOD_NOT_FOUND))
			return
		}

		res := service.Execute(methodName, msg.Msg)
		if res.Error != nil {
			s.ResponseError(msg, res.Error)
			return
		}

		rsp, err := NewMessage(Method, res)
		if err != nil {
			s.ResponseError(msg, err)
			return
		}

		err = s.response(msg.To, msg.ID(), rsp)
		if err != nil {
			s.Error(msg.To, err)
		}
	}

	go func() {
		for _, fn := range s.onInbox {
			fn(msg)
		}
	}()
}

/**
* response
* @params to *Client, id string, msg *Message
* @return error
**/
func (s *Node) response(to *Client, id string, msg *Message) error {
	if id != "" {
		msg.ID = id
		msg.IsResponse = true
	}

	return s.send(&Msg{
		To:  to,
		Msg: msg,
	})
}

/**
* request
* @param to *Client, m *Message
* @return *Message, error
**/
func (s *Node) request(to *Client, m *Message) (*Message, error) {
	s.mu["clients"].Lock()
	_, ok := s.clients[to.Addr]
	s.mu["clients"].Unlock()

	if !ok {
		return nil, fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, to.Addr)
	}

	ch := make(chan *Message, 1)

	s.mu["requests"].Lock()
	s.requests[m.ID] = ch
	s.mu["requests"].Unlock()

	err := s.send(&Msg{
		To:  to,
		Msg: m,
	})
	if err != nil {
		s.mu["requests"].Lock()
		delete(s.requests, m.ID) // 🔥 cleanup correcto
		s.mu["requests"].Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		s.mu["requests"].Lock()
		delete(s.requests, m.ID)
		s.mu["requests"].Unlock()
		return resp, nil

	case <-time.After(s.timeout):
		s.mu["requests"].Lock()
		delete(s.requests, m.ID)
		s.mu["requests"].Unlock()
		return nil, errors.New(mg.MSG_TCP_TIMEOUT)
	}
}

/**
* handle
* @param conn net.Conn
**/
func (s *Node) handle(c *Client) {
	mode := s.mode.Load().(Mode)

	switch mode {
	case Proxy:
		go handleBalancer(c.conn)
	default:
		go s.handleClient(c)
	}
}

/**
* handleClient
* @param c *Client
**/
func (s *Node) handleClient(c *Client) {
	reader := bufio.NewReader(c.conn)
	lenBuf := make([]byte, 4)
	limit := envar.GetInt("LIMIT_SIZE_MG", 10)
	maxSize := uint32(limit * 1024 * 1024)

	for {
		// Leer tamaño (4 bytes)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			s.unregisterClient(c, err)
			return
		}

		// Leer tamaño payload
		length := binary.BigEndian.Uint32(lenBuf)

		if length > maxSize {
			io.CopyN(io.Discard, reader, int64(length))
			s.SendError(c, errors.New(mg.MSG_TCP_MESSAGE_TOO_LARGE))
			continue
		}

		// Leer payload completo
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			s.unregisterClient(c, err)
			return
		}

		m, err := ToMessage(data)
		if err != nil {
			s.SendError(c, err)
			continue
		}

		msg := &Msg{
			To:  c,
			Msg: m,
		}

		// 🔥 NUNCA BLOQUEAR EL READER
		s.dispatch(msg)
	}
}

/**
* unregisterClient
* @param c *Client, err error
**/
func (s *Node) unregisterClient(c *Client, err error) {
	if err != io.EOF {
		logs.Logf(packageName, mg.MSG_TCP_ERROR_READ, err)
	}

	select {
	case s.unregister <- c:
	case <-s.ctx.Done():
	}
}

/**
* dispatch
* @param msg *Msg
**/
func (s *Node) dispatch(msg *Msg) {
	select {
	case s.workerPool <- struct{}{}:
		go func() {
			defer func() { <-s.workerPool }()
			s.inbox(msg)
		}()
	default:
		// 🔥 backpressure: sistema saturado
		logs.Warn("worker pool full, dropping message")
	}
}

/**
* newClient
* @param conn net.Conn
* @return *Client
**/
func (s *Node) newClient(conn net.Conn) *Client {
	result := newClient()
	result.Addr = conn.RemoteAddr().String()
	result.conn = conn
	result.alive.Store(true)

	return result
}

/**
* Next
* @return *Client
**/
func (s *Node) Next() *Client {
	s.mu["peers"].Lock()
	defer s.mu["peers"].Unlock()

	n := uint64(len(s.peers))
	for i := uint64(0); i < n; i++ {
		idx := (s.index.Add(1)) % n
		node := s.peers[idx]
		if node.alive.Load() {
			return node
		}
	}
	return nil
}

/**
* electionLoop
**/
func (s *Node) electionLoop() error {
	time.Sleep(300 * time.Millisecond)
	var config *Config
	err := file.Read(s.configFile, &config)
	if err != nil {
		return err
	}

	if config == nil {
		config = &Config{
			Nodes: []string{},
		}

		file.Write(s.configFile, config)
	}

	for _, node := range config.Nodes {
		s.AddNode(node)
	}

	go s.raft.electionLoop()

	return nil
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
	err = s.electionLoop()
	if err != nil {
		return
	}

	logs.Logf(packageName, mg.MSG_TCP_LISTENING, s.addr)

	return nil
}

/**
* Close
**/
func (s *Node) Close() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}

	logs.Log(packageName, mg.MSG_TCP_SHUTTING_DOWN)
	s.cancel()

	if s.ln != nil {
		_ = s.ln.Close()
	}

	s.closeAllClients()
}

/**
* LeaderID
* @return string, bool
**/
func (s *Node) LeaderID() (string, bool) {
	return s.raft.LeaderID()
}

/**
* AddNode
* @param addr string
**/
func (s *Node) AddNode(addr string) {
	if addr == s.addr {
		return
	}

	node := NewClient(addr)
	node.isNode = true
	node.onConnect = append(node.onConnect, func(c *Client) {
		s.total.Add(1)
	})
	node.onDisconnect = append(node.onDisconnect, func(c *Client) {
		s.total.Add(-1)
	})

	node.Connect()

	s.mu["peers"].Lock()
	s.peers = append(s.peers, node)
	s.mu["peers"].Unlock()
}

/**
* Send
* @param to *Client, tp TpMessage, message any
* @return error
**/
func (s *Node) Send(to *Client, tp TpMessage, message any) error {
	msg, err := NewMessage(tp, message)
	if err != nil {
		return err
	}

	return s.send(&Msg{
		To:  to,
		Msg: msg,
	})
}

/**
* SendError
* @param to *Client, err error
* @return error
**/
func (s *Node) SendError(to *Client, err error) error {
	msg, errM := NewMessage(ErrorMessage, "")
	if errM != nil {
		return errM
	}
	msg.Error = err.Error()

	return s.send(&Msg{
		To:  to,
		Msg: msg,
	})
}

/**
* Mount
* @param services any
**/
func (s *Node) Mount(service Service) error {
	if service == nil {
		return errors.New(mg.MSG_SERVICE_REQUIRED)
	}

	tipoStruct := reflect.TypeOf(service)
	pkgName := tipoStruct.String()
	list := strings.Split(pkgName, ".")
	pkgName = list[len(list)-1]
	s.method[pkgName] = service
	return nil
}

/**
* GetMethods
* @return et.Json
**/
func (s *Node) GetMethods() et.Json {
	result := et.Json{}
	for pkg, mt := range s.method {
		result[pkg] = mt
	}
	return result
}

/**
* Request
* @param to *Client, method string, args ...any
* @return *Response
**/
func (s *Node) Request(to *Client, method string, args ...any) *Response {
	msg, err := NewMessage(Method, "")
	if err != nil {
		return TcpError(err)
	}
	msg.Method = method
	for _, arg := range args {
		msg.Args = append(msg.Args, arg)
	}

	res, err := s.request(to, msg)
	if err != nil {
		return TcpError(err)
	}

	result, err := res.Response()
	if err != nil {
		return TcpError(err)
	}

	return result
}

/**
* ResponseError
* @param to *Client, id string, err error
* @return error
**/
func (s *Node) ResponseError(msg *Msg, err error) error {
	res, _ := NewMessage(ErrorMessage, "")
	res.Error = err.Error()

	return s.response(msg.To, msg.ID(), res)
}

/**
* Broadcast
* @param destination []string
* @param msg []byte
**/
func (s *Node) Broadcast(destination []string, tp TpMessage, message any) {
	for _, addr := range destination {
		s.mu["clients"].Lock()
		client, ok := s.clients[addr]
		s.mu["clients"].Unlock()

		if !ok || client == nil {
			continue
		}

		if client.Status == Connected {
			_ = s.Send(client, tp, message)
		}
	}
}

/**
* RemoveNode
* @param addr string
**/
func (s *Node) RemoveNode(addr string) {
	s.mu["peers"].Lock()
	defer s.mu["peers"].Unlock()

	idx := slices.IndexFunc(s.peers, func(e *Client) bool { return e.Addr == addr })
	if idx != -1 {
		s.peers[idx].Close()
		s.peers = append(s.peers[:idx], s.peers[idx+1:]...)
	}
}

/**
* GetPeers
* @return []*Client
**/
func (s *Node) GetPeers() []*Client {
	s.mu["peers"].Lock()
	defer s.mu["peers"].Unlock()

	result := make([]*Client, len(s.peers))
	copy(result, s.peers)
	return result
}

/**
* OnConnect
* @param fn func(*Client)
**/
func (s *Node) OnConnect(fn func(*Client)) {
	s.onConnect = append(s.onConnect, fn)
}

/**
* OnDisconnect
* @param fn func(*Client)
**/
func (s *Node) OnDisconnect(fn func(*Client)) {
	s.onDisconnect = append(s.onDisconnect, fn)
}

/**
* OnError
* @param fn func(*Client, error)
**/
func (s *Node) OnError(fn func(*Client, error)) {
	s.onError = append(s.onError, fn)
}

/**
* OnInbox
* @param fn func(*Msg)
**/
func (s *Node) OnInbox(fn func(*Msg)) {
	s.onInbox = append(s.onInbox, fn)
}

/**
* OnSend
* @param fn func(*Msg)
**/
func (s *Node) OnSend(fn func(*Msg)) {
	s.onSend = append(s.onSend, fn)
}

/**
* OnBecomeLeader
* @param fn func(*Node)
**/
func (s *Node) OnBecomeLeader(fn func(*Node)) {
	s.onBecomeLeader = append(s.onBecomeLeader, fn)
}

/**
* OnChangeLeader
* @param fn func(*Node)
**/
func (s *Node) OnChangeLeader(fn func(*Node)) {
	s.onChangeLeader = append(s.onChangeLeader, fn)
}
