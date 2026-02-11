package tcp

import (
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
	Type    int         `json:"type"`
	Message interface{} `json:"message"`
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
func newMessage(tp int, message any) (Message, error) {
	result := Message{
		Type:    tp,
		Message: message,
	}

	switch tp {
	case PingMessage:
		result.Message = "PING\n"
	case PongMessage:
		result.Message = "PONG\n"
	case ACKMessage:
		result.Message = "ACK\n"
	case CloseMessage:
		result.Message = "CLOSE\n"
	}

	return result, nil
}
