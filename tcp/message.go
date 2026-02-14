package tcp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

const (
	BytesMessage int = 0
	TextMessage  int = 1
	PingMessage  int = 10
	PongMessage  int = 11
	ACKMessage   int = 12
	CloseMessage int = 13
	ErrorMessage int = 14
	Heartbeat    int = 15
	RequestVote  int = 16
	AuthMessage  int = 17
)

type Message struct {
	ID      string `json:"id"`
	Type    int    `json:"type"`
	Payload []byte `json:"payload"`
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
* newMessage
**/
func newMessage(tp int, message any) (*Message, error) {
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
	}

	switch tp {
	case PingMessage:
		result.Payload = []byte("PING")
	case PongMessage:
		result.Payload = []byte("PONG")
	case ACKMessage:
		result.Payload = []byte("ACK")
	case CloseMessage:
		result.Payload = []byte("CLOSE")
	}

	return result, nil
}
