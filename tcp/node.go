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

	"github.com/cgalvisleon/et/color"
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
	peers      []*Client                `json:"-"`
	muPeers    sync.Mutex               `json:"-"`
	method     map[string]Service       `json:"-"`
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
		peers:      make([]*Client, 0),
		muPeers:    sync.Mutex{},
		method:     make(map[string]Service),
	}
	result.mode.Store(Follower)
	result.Mount(newTcpService(result))

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
	case Method:
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
			logs.Debug(color.Red(res.Error.Error()))
			s.ResponseError(msg, res.Error)
			return
		}

		rsp, err := NewMessage(Method, res)
		if err != nil {
			logs.Error(err)
			return
		}

		err = s.response(msg.To, msg.ID(), rsp)
		if err != nil {
			logs.Error(err)
		}
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

	// Channel for response
	ch := make(chan *Message, 1)
	s.muRequests.Lock()
	s.requests[m.ID] = ch
	s.muRequests.Unlock()

	// Send
	err := s.send(&Msg{
		To:  to,
		Msg: m,
	})
	if err != nil {
		return nil, err
	}

	// Wait response or timeout
	select {
	case resp := <-ch:
		s.muRequests.Lock()
		delete(s.requests, m.ID)
		s.muRequests.Unlock()
		logs.Debugf("response: %s type:%d", resp.ID, resp.Type)
		return resp, nil

	case <-time.After(s.timeout):
		s.muRequests.Lock()
		delete(s.requests, m.ID)
		s.muRequests.Unlock()
		logs.Debugf("response timeout: %s", m.ID)
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
			s.unregister <- c
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
			s.unregister <- c
			return
		}

		m, err := ToMessage(data)
		if err != nil {
			s.SendError(c, err)
			continue
		}

		s.inbound <- &Msg{
			To:  c,
			Msg: m,
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
* Start
* @return error
**/
func (s *Node) Start() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}

	s.run()
	if s.port != 1377 {
		go test(s)
	}

	logs.Logf(packageName, mg.MSG_TCP_LISTENING, s.addr)

	return nil
}

/**
* Close
**/
func (s *Node) Close() {
	close(s.inbound)
	close(s.register)
	close(s.unregister)
	if s.ln != nil {
		s.ln.Close()
	}
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

		status := client.Status
		if ok && status == Connected {
			s.Send(client, tp, message)
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
