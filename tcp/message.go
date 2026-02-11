package tcp

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

const (
	BinaryMessage int = 0
	TextMessage   int = 1
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
		result.Message = []byte("PING\n")
	case PongMessage:
		result.Message = []byte("PONG\n")
	case ACKMessage:
		result.Message = []byte("ACK\n")
	case CloseMessage:
		result.Message = []byte("CLOSE\n")
	}

	return result, nil
}
