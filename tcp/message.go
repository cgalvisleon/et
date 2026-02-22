package tcp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
)

const (
	BytesMessage int = 0
	TextMessage  int = 1
	ACKMessage   int = 13
	CloseMessage int = 15
	ErrorMessage int = 17
	Heartbeat    int = 19
	RequestVote  int = 21
	Method       int = 23
)

type Response struct {
	Response []any `json:"response"`
	Error    error `json:"error"`
}

/**
* TcpResponse
* @param res ...any
* @return *Response
**/
func TcpResponse(res ...any) *Response {
	result := &Response{
		Response: res,
		Error:    nil,
	}

	for _, r := range res {
		result.Response = append(result.Response, r)
	}

	return result
}

/**
* TcpError
* @param msg any
* @return *Response
**/
func TcpError(msg any) *Response {
	var err error

	switch v := msg.(type) {
	case error:
		err = v
	case string:
		err = errors.New(v)
	default:
		panic("TcpError only accepts error or string")
	}

	return &Response{
		Response: []any{},
		Error:    err,
	}
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Response) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* Get
* @param dest ...any
* @return error
**/
func (s *Response) Get(dest ...any) error {
	l := len(dest)
	if l > len(s.Response) {
		return errors.New(msg.MSG_INDEX_OUT_OF_RANGE)
	}

	for i, d := range dest {
		bt, err := json.Marshal(s.Response[i])
		if err != nil {
			return err
		}
		err = json.Unmarshal(bt, d)
		if err != nil {
			return err
		}
	}

	return nil
}

type Message struct {
	ID         string `json:"id"`
	Type       int    `json:"type"`
	Method     string `json:"method"`
	Payload    []byte `json:"payload"`
	Args       []any  `json:"args"`
	IsResponse bool   `json:"is_response"`
}

/**
* serialize
* @return []byte, error
**/
func (s *Message) serialize() ([]byte, error) {
	// Create a payload []byte
	payload, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	// Create a buffer
	var buf bytes.Buffer

	// Write the length of the payload
	err = binary.Write(&buf, binary.BigEndian, uint32(len(payload)))
	if err != nil {
		return nil, err
	}

	// Write the payload
	buf.Write(payload)

	// Return the buffer as []byte
	return buf.Bytes(), nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Message) ToJson() et.Json {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}
	}

	result := et.Json{}
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* Get
* @return any, error
**/
func (s *Message) Get(dest any) error {
	return json.Unmarshal(s.Payload, dest)
}

/**
* AnyGet: Gets a value from an array
* @param args []any, dest ...any
* @return error
**/
func (s *Message) GetArgs(dest ...any) error {
	l := len(dest)
	if len(s.Args) < l {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "args")
	}

	for i, d := range dest {
		bt, err := json.Marshal(s.Args[i])
		if err != nil {
			return err
		}

		err = json.Unmarshal(bt, d)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* Response
* @return *Response
**/
func (s *Message) Response() (*Response, error) {
	var result *Response
	err := s.Get(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* NewMessage
**/
func NewMessage(tp int, message any) (*Message, error) {
	bt, ok := message.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(message)
		if err != nil {
			return nil, err
		}
	}

	result := &Message{
		ID:      reg.ULID(),
		Type:    tp,
		Payload: bt,
		Args:    []any{},
	}

	switch tp {
	case ACKMessage:
		result.Payload = []byte("\n")
	case CloseMessage:
		result.Payload = []byte("CLOSE")
	}

	return result, nil
}

/**
* ToMessage
* @param bt []byte
* @return Message, error
**/
func ToMessage(bt []byte) (*Message, error) {
	var result *Message
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
