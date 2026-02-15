package tcp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

const (
	BytesMessage int = 0
	TextMessage  int = 1
	PingMessage  int = 10
	ACKMessage   int = 11
	CloseMessage int = 12
	ErrorMessage int = 13
	Heartbeat    int = 14
	RequestVote  int = 15
	Method       int = 16
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

type Message struct {
	ID       string `json:"id"`
	Type     int    `json:"type"`
	Method   string `json:"method"`
	Payload  []byte `json:"payload"`
	Args     []any  `json:"args"`
	Response []any  `json:"response"`
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
* Result
* @return []any, error
**/
func (s *Message) Result() ([]any, error) {
	result := make([]any, 0)
	err := error(nil)
	for _, v := range s.Response {
		_, ok := v.(error)
		if ok {
			err = v.(error)
			continue
		}
		result = append(result, v)
	}
	return result, err
}

/**
* toMessage
* @param bt []byte
* @return Message, error
**/
func toMessage(bt []byte) (*Message, error) {
	var result *Message
	err := json.Unmarshal(bt, &result)
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
		ID:       reg.ULID(),
		Type:     tp,
		Payload:  bt,
		Args:     []any{},
		Response: []any{},
	}

	switch tp {
	case PingMessage:
		result.Payload = []byte("PING")
	case ACKMessage:
		result.Payload = []byte("\n")
	case CloseMessage:
		result.Payload = []byte("CLOSE")
	}

	return result, nil
}
