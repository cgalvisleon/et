package tcp

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

const (
	AuthMethod string = "Auth.Auth"
)

type HandlerFunc func(request *Message) *Response

type Service interface {
	Execute(name string, request *Message) *Response
}

type Auth struct {
	registry map[string]HandlerFunc
	Auth     HandlerFunc
}

/**
* NewAuthService
* @return *Auth
**/
func NewAuthService() *Auth {
	this := &Auth{}
	this.Auth = func(request *Message) *Response {
		var id string
		var ctx et.Json
		err := request.GetArgs(&id, &ctx)
		if err != nil {
			return NewResponse(nil, err)
		}

		return NewResponse([]any{"Hola", id, ctx}, nil)
	}

	this.build()
	return this
}

/**
* build: Builds the registry
* @return map[string]tcp.HandlerFunc
**/
func (s *Auth) build() map[string]HandlerFunc {
	s.registry = map[string]HandlerFunc{
		"Auth": s.Auth,
	}
	return s.registry
}

/**
* Execute: Executes a method
* @param name string, request *Message
* @return *tcp.Response
**/
func (s *Auth) Execute(name string, request *Message) *Response {
	handler, ok := s.registry[name]
	if !ok {
		return NewResponse(nil, errors.New(msg.MSG_METHOD_NOT_FOUND))
	}

	return handler(request)
}
