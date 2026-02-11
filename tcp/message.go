package tcp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

const (
	TextMessage  int = 1
	ACKMessage   int = 3
	CloseMessage int = 8
	PingMessage  int = 9
	PongMessage  int = 10
)

type Message struct {
	Type    int    `json:"type"`
	Message []byte `json:"message"`
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
	return et.Json{
		"type":    s.Type,
		"message": s.Message,
	}
}

/**
* toMessage
* @param bt []byte
* @return Message, error
**/
func toMessage(bt []byte) (Message, error) {
	var result Message
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return Message{}, err
	}

	return result, nil
}

/**
* newMessage
**/
func newMessage(tp int, message any) (Message, error) {
	bt, ok := message.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(message)
		if err != nil {
			return Message{}, err
		}
	}

	result := Message{
		Type:    tp,
		Message: bt,
	}

	switch tp {
	case PingMessage:
		result.Message = []byte("PING")
	case PongMessage:
		result.Message = []byte("PONG")
	case ACKMessage:
		result.Message = []byte("ACK")
	case CloseMessage:
		result.Message = []byte("CLOSE")
	}

	return result, nil
}
