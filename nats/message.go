package nats

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/utility"
)

type Message struct {
	Created_at time.Time `json:"created_at"`
	Id         string    `json:"id"`
	From       js.Json   `json:"from"`
	to         string
	Tp         message.TpMessage `json:"tp"`
	Channel    string            `json:"channel"`
	Data       interface{}       `json:"data"`
}

// NewMessage create a new message
func NewMessage(from js.Json, message interface{}) Message {
	id := utility.UUID()
	return Message{
		Created_at: time.Now(),
		Id:         id,
		From:       from,
		Data:       message,
	}
}

// Type return the type of message
func (e Message) Type() message.TpMessage {
	return e.Tp
}

// ToString return the message as string
func (e Message) ToString() string {
	j, err := json.Marshal(e)
	if err != nil {
		return ""
	}

	return string(j)
}

// Encode return the message as byte
func (e Message) Encode() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Json return the message as json
func (e Message) Json() (js.Json, error) {
	result := js.Json{}
	err := result.Scan(e.Data)
	if err != nil {
		return js.Json{}, err
	}

	return result, nil
}

// Decode return the message as struct
func DecodeMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}

	return m, nil
}
