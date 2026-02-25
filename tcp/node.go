package tcp

import (
	"fmt"
	"net"
	"os"
	"sync"

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
	mode       Mode                     `json:"-"`
	ln         net.Listener             `json:"-"`
	register   chan *Client             `json:"-"`
	unregister chan *Client             `json:"-"`
	clients    map[string]*Client       `json:"-"`
	muClients  sync.Mutex               `json:"-"`
	inbound    chan *Msg                `json:"-"`
	requests   map[string]chan *Message `json:"-"`
	muRequests sync.Mutex               `json:"-"`
}

/**
* NewNode
* @param port int
* @return *Node
**/
func NewNode(port int) *Node {
	addr := fmt.Sprintf("%s:%d", hostName, port)
	result := &Node{
		addr:       addr,
		port:       port,
		mode:       Follower,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		muClients:  sync.Mutex{},
		inbound:    make(chan *Msg),
		requests:   make(map[string]chan *Message),
		muRequests: sync.Mutex{},
	}
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
