package tcp

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

type HandlerFunc func(request *Message) *Response

type Service interface {
	Execute(method string, request *Message) *Response
}

type Tcp struct {
	registry map[string]HandlerFunc
	node     *Node
	Ping     HandlerFunc
}

/**
* build: Builds the registry
* @return map[string]tcp.HandlerFunc
**/
func (s *Tcp) build() map[string]HandlerFunc {
	s.registry = map[string]HandlerFunc{
		"Ping": s.ping,
	}
	return s.registry
}

/**
* Execute: Executes a method
* @param name string, request *Message
* @return *tcp.Response
**/
func (s *Tcp) Execute(method string, request *Message) *Response {
	handler, ok := s.registry[method]
	if !ok {
		return TcpError(msg.MSG_METHOD_NOT_FOUND)
	}

	return handler(request)
}

/**
* ping
* @param request *Message
* @return *Response
**/
func (s *Tcp) ping(request *Message) *Response {
	// var id string
	// var ctx et.Json
	// err := request.GetArgs(&id, &ctx)
	// if err != nil {
	// 	return TcpError(err)
	// }

	return TcpResponse(fmt.Sprintf("Tcp.Ping hello:%s", s.node.addr))
}

/**
* newTcpService
* @param node *Node
* @return *Tcp
**/
func newTcpService(node *Node) *Tcp {
	this := &Tcp{
		node: node,
	}
	this.Ping = func(request *Message) *Response {
		var id string
		var ctx et.Json
		err := request.GetArgs(&id, &ctx)
		if err != nil {
			return TcpError(err)
		}

		return TcpResponse(fmt.Sprintf("Pong to:%s", this.node.addr))
	}

	this.build()
	return this
}
