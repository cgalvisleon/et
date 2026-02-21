package tcp

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

const (
	PingMehtod string = "Tcp.Ping"
)

type HandlerFunc func(request *Message) *Response

type Service interface {
	Execute(name string, request *Message) *Response
}

type Tcp struct {
	registry map[string]HandlerFunc
	Ping     HandlerFunc
}

/**
* build: Builds the registry
* @return map[string]tcp.HandlerFunc
**/
func (s *Tcp) build() map[string]HandlerFunc {
	s.registry = map[string]HandlerFunc{
		"Ping": s.Ping,
	}
	return s.registry
}

/**
* Execute: Executes a method
* @param name string, request *Message
* @return *tcp.Response
**/
func (s *Tcp) Execute(name string, request *Message) *Response {
	handler, ok := s.registry[name]
	if !ok {
		return NewResponse(nil, errors.New(msg.MSG_METHOD_NOT_FOUND))
	}

	return handler(request)
}

/**
* NewTcpService
* @return *Tcp
**/
func NewTcpService() *Tcp {
	this := &Tcp{}
	this.Ping = func(request *Message) *Response {
		var id string
		var ctx et.Json
		err := request.GetArgs(&id, &ctx)
		if err != nil {
			return NewResponse(nil, err)
		}

		return NewResponse([]any{"Pong", id, ctx}, nil)
	}

	this.build()
	return this
}
