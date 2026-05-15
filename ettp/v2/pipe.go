package ettp

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"net/rpc"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

type Codec struct {
	rwc io.ReadWriteCloser
	dec *gob.Decoder
	enc *gob.Encoder
}

/**
* ReadRequestHeader
* @param r *rpc.Request
* @return error
**/
func (s *Codec) ReadRequestHeader(r *rpc.Request) error {
	err := s.dec.Decode(r)
	return err
}

/**
* ReadRequestBody
* @param body interface{}
* @return error
**/
func (s *Codec) ReadRequestBody(body interface{}) error {
	return s.dec.Decode(body)
}

/**
* WriteResponse
* @param r *rpc.Response, body interface{}
* @return error
**/
func (s *Codec) WriteResponse(r *rpc.Response, body interface{}) error {
	if err := s.enc.Encode(r); err != nil {
		return err
	}
	return s.enc.Encode(body)
}

/**
* startPipe
**/
func (s *Server) startPipe() {
	logs.Logf("Pipe", "Starting pipe on port %s", s.pipe.Addr().String())
	go func() {
		for {
			conn, err := s.pipe.Accept()
			if err != nil {
				logs.Error(err)
				continue
			}
			go s.handlerPipe(conn)
		}
	}()
}

/**
* handlerPipe
* @param conn net.Conn
**/
func (s *Server) handlerPipe(conn net.Conn) {
	defer conn.Close()

	codec := &Codec{
		rwc: conn,
		dec: gob.NewDecoder(conn),
		enc: gob.NewEncoder(conn),
	}

	var req rpc.Request
	if err := codec.ReadRequestHeader(&req); err != nil {
		logs.Error(fmt.Errorf(msg.MSG_ERROR_READING_REQUEST, err))
		return
	}

	var args et.Json
	if err := codec.ReadRequestBody(&args); err != nil {
		logs.Error(fmt.Errorf(msg.MSG_ERROR_READING_REQUEST_BODY, err))
		return
	}

	addr := conn.RemoteAddr().String()
	logs.Logf("Pipe request from %s: method:%s args:%s", addr, req.ServiceMethod, args.ToString())

	result, err := jrpc.Call(req.ServiceMethod, args)
	if err != nil {
		logs.Error(fmt.Errorf("error llamando al servicio: %v", err))
		resp := rpc.Response{
			ServiceMethod: req.ServiceMethod,
			Seq:           req.Seq,
			Error:         err.Error(),
		}

		codec.WriteResponse(&resp, nil)
		return
	}

	resp := rpc.Response{
		ServiceMethod: req.ServiceMethod,
		Seq:           req.Seq,
	}

	codec.WriteResponse(&resp, result)
}
