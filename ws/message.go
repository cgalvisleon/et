package ws

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	m "github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/utility"
)

type Message struct {
	Created_at time.Time `json:"created_at"`
	Id         string    `json:"id"`
	From       et.Json   `json:"from"`
	to         string
	Ignored    []string    `json:"ignored"`
	Tp         m.TpMessage `json:"tp"`
	Channel    string      `json:"channel"`
	Data       interface{} `json:"data"`
}

// NewMessage create a new message
func NewMessage(from et.Json, message interface{}, tp m.TpMessage) Message {
	return Message{
		Created_at: time.Now(),
		Id:         utility.UUID(),
		From:       from,
		Data:       message,
		Tp:         tp,
		Ignored:    []string{},
	}
}

// Type return the type of message
func (e Message) Type() m.TpMessage {
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

func (e Message) Json() (et.Json, error) {
	result := et.Json{}
	err := result.Scan(e.Data)
	if err != nil {
		return et.Json{}, err
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
