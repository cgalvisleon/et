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
	muClients      sync.Mutex               `json:"-"`
	inbound        chan *Msg                `json:"-"`
	requests       map[string]chan *Message `json:"-"`
	muRequests     sync.Mutex               `json:"-"`
	proxy          *Balancer                `json:"-"`
	peers          []*Client                `json:"-"`
	muPeers        sync.Mutex               `json:"-"`
	method         map[string]Service       `json:"-"`
	raft           *Raft                    `json:"-"`
	closed         atomic.Bool              `json:"-"`
	onConnect      []func(*Client)          `json:"-"`
	onDisconnect   []func(*Client)          `json:"-"`
	onError        []func(*Client, error)   `json:"-"`
	onInbox        []func(*Msg)             `json:"-"`
	onSend         []func(*Msg)             `json:"-"`
	onBecomeLeader []func(*Node)            `json:"-"`
	onChangeLeader []func(*Node)            `json:"-"`
	index          atomic.Uint64            `json:"-"`
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

	ctx, cancel := context.WithCancel(context.Background())
	result := &Node{
		ctx:        ctx,
		cancel:     cancel,
		addr:       addr,
		port:       port,
		timeout:    timeout,
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		clients:    make(map[string]*Client),
		muClients:  sync.Mutex{},
		inbound:    make(chan *Msg, 256),
		requests:   make(map[string]chan *Message),
		muRequests: sync.Mutex{},
		peers:      make([]*Client, 0),
		muPeers:    sync.Mutex{},
		method:     make(map[string]Service),
		index:      atomic.Uint64{},
	}
	result.mode.Store(Follower)
	result.Mount(newTcpService(result))
	result.raft = newRaft(result)

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
	// Loop de inbound messages
	// ===============================
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return

			case msg := <-s.inbound:
				if msg != nil {
					s.inbox(msg)
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

				// Si estamos cerrando, salir limpio
				if s.ctx.Err() != nil || s.closed.Load() {
					return
				}

				// Listener cerrado
				if errors.Is(err, net.ErrClosed) {
					return
				}

				// Error transitorio → continuar
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
				// OK

			case <-s.ctx.Done():
				_ = conn.Close()
				return

			default:
				// Si el canal está lleno evitamos bloquear
				_ = conn.Close()
			}
		}
	}()
}

/**
* connectToPeersLoop
**/
func (n *Node) connectToPeersLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
		}

		n.muPeers.Lock()
		peers := make([]*Client, len(n.peers))
		copy(peers, n.peers)
		n.muPeers.Unlock()

		for _, peer := range peers {
			if peer.Status != Connected && peer.Addr != n.addr {
				_ = peer.Connect()
			}
		}
	}
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
	s.muClients.Lock()
	s.clients[client.Addr] = client
	s.muClients.Unlock()

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
	s.muClients.Lock()
	delete(s.clients, client.Addr)
	s.muClients.Unlock()

	logs.Logf(packageName, mg.MSG_TCP_DISCONNECTED_CLIENT, client.Addr)

	for _, fn := range s.onDisconnect {
		fn(client)
	}
}

/**
* closeAllClients
**/
func (s *Node) closeAllClients() {
	s.muClients.Lock()
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.clients = make(map[string]*Client)
	s.muClients.Unlock()

	for _, c := range clients {
		c.Close()
	}
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

	for _, fn := range s.onInbox {
		fn(msg)
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

	if !ok || c == nil {
		return fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, msg.To.Addr)
	}

	if err := c.send(msg.Msg); err != nil {
		return s.Error(c, err)
	}

	for _, fn := range s.onSend {
		fn(msg)
	}

	return nil
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
	s.muClients.Lock()
	_, ok := s.clients[to.Addr]
	s.muClients.Unlock()

	if !ok {
		return nil, fmt.Errorf(mg.MSG_TCP_CLIENT_NOT_FOUND, to.Addr)
	}

	ch := make(chan *Message, 1)

	s.muRequests.Lock()
	s.requests[m.ID] = ch
	s.muRequests.Unlock()

	err := s.send(&Msg{
		To:  to,
		Msg: m,
	})
	if err != nil {
		s.muRequests.Lock()
		delete(s.requests, m.ID) // 🔥 cleanup correcto
		s.muRequests.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		s.muRequests.Lock()
		delete(s.requests, m.ID)
		s.muRequests.Unlock()
		return resp, nil

	case <-time.After(s.timeout):
		s.muRequests.Lock()
		delete(s.requests, m.ID)
		s.muRequests.Unlock()
		return nil, fmt.Errorf(mg.MSG_TCP_TIMEOUT)
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

	for {
		// Leer tamaño (4 bytes)
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			if err != io.EOF {
				logs.Logf(packageName, mg.MSG_TCP_ERROR_READ, err)
			}
			select {
			case s.unregister <- c:
			case <-s.ctx.Done():
			}
			return
		}

		// Leer tamaño payload
		length := binary.BigEndian.Uint32(lenBuf)
		limitReader := envar.GetInt("LIMIT_SIZE_MG", 10)
		if length > uint32(limitReader*1024*1024) {
			s.SendError(c, errors.New(mg.MSG_TCP_MESSAGE_TOO_LARGE))
			continue
		}

		// Leer payload completo
		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			select {
			case s.unregister <- c:
			case <-s.ctx.Done():
			}
			return
		}

		m, err := ToMessage(data)
		if err != nil {
			s.SendError(c, err)
			continue
		}

		select {
		case s.inbound <- &Msg{
			To:  c,
			Msg: m,
		}:
		case <-s.ctx.Done():
			return
		}
	}
}

/**
* newClient
* @param conn net.Conn
* @return *Client
**/
func (s *Node) newClient(conn net.Conn) *Client {
	timeout, err := time.ParseDuration(envar.GetStr("TIMEOUT", "10s"))
	if err != nil {
		timeout = 10 * time.Second
	}

	result := &Client{
		CreatedAt: timezone.Now(),
		ID:        reg.ULID(),
		Addr:      conn.RemoteAddr().String(),
		Status:    Connected,
		Ctx:       et.Json{},
		conn:      conn,
		timeout:   timeout,
	}
	result.alive.Store(true)
	return result
}

/**
* Next
* @return *Client
**/
func (s *Node) Next() *Client {
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
func (s *Node) electionLoop() {
	s.raft.electionLoop()
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
	go s.connectToPeersLoop()
	go s.electionLoop()
	// go s.raft.start()

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

	logs.Logf(packageName, mg.MSG_TCP_SHUTTING_DOWN)

	s.cancel()

	if s.ln != nil {
		_ = s.ln.Close()
	}

	s.closeAllClients()
}

/**
* Send
* @param to *Client, tp int, message any
* @return error
**/
func (s *Node) Send(to *Client, tp int, message any) error {
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
func (s *Node) Broadcast(destination []string, tp int, message any) {
	for _, addr := range destination {
		s.muClients.Lock()
		client, ok := s.clients[addr]
		s.muClients.Unlock()

		if !ok || client == nil {
			continue
		}

		if client.Status == Connected {
			_ = s.Send(client, tp, message)
		}
	}
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
* GetMethod
* @return map[string]map[string]*Mtd
**/
func (s *Node) GetMethod() et.Json {
	result := et.Json{}
	for pkg, mt := range s.method {
		result[pkg] = mt
	}
	return result
}

/**
* AddNode
* @param addr string
**/
func (s *Node) AddNode(addr string) {
	node := NewClient(addr)
	node.isNode = true
	s.muPeers.Lock()
	s.peers = append(s.peers, node)
	s.muPeers.Unlock()
}

/**
* RemoveNode
* @param addr string
**/
func (s *Node) RemoveNode(addr string) {
	s.muPeers.Lock()
	defer s.muPeers.Unlock()

	idx := slices.IndexFunc(s.peers, func(e *Client) bool { return e.Addr == addr })
	if idx != -1 {
		s.peers = append(s.peers[:idx], s.peers[idx+1:]...)
	}
}

/**
* GetPeer
* @return *Client
**/
func (s *Node) GetPeer(addr string) *Client {
	s.muPeers.Lock()
	defer s.muPeers.Unlock()

	idx := slices.IndexFunc(s.peers, func(e *Client) bool { return e.Addr == addr })
	if idx != -1 {
		return s.peers[idx]
	}
	return nil
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
