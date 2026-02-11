package tcp

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

const (
	TextMessage   int = 1
	BinaryMessage int = 2
	ACKMessage    int = 3
	CloseMessage  int = 8
	PingMessage   int = 9
	PongMessage   int = 10
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
	return json.Marshal(s)
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
func newMessage(tp int, message any) ([]byte, error) {
	bt, ok := message.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(message)
		if err != nil {
			return nil, err
		}
	}

	msg := Message{
		Type:    tp,
		Message: bt,
	}

	switch tp {
	case PingMessage:
		msg.Message = []byte("PING\n")
	case PongMessage:
		msg.Message = []byte("PONG\n")
	case ACKMessage:
		msg.Message = []byte("ACK\n")
	case CloseMessage:
		msg.Message = []byte("CLOSE\n")
	}

	result, err := msg.serialize()
	if err != nil {
		return nil, err
	}

	return result, nil
}
