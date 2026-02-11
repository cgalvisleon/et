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
* ToMessage
* @param bt []byte
* @return Message, error
**/
func ToMessage(bt []byte) (Message, error) {
	var result Message
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return Message{}, err
	}

	return result, nil
}
