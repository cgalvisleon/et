package nats

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/cgalvisleon/et/utility"
)

type Message struct {
	Created_at time.Time `json:"created_at"`
	Id         string    `json:"id"`
	From       et.Json   `json:"from"`
	To         et.Json   `json:"to"`
	Kind       string    `json:"type"`
	Data       et.Json   `json:"data"`
}

func NewMessage(kind string, from, to, data et.Json) *Message {
	id := utility.UUID()
	return &Message{
		Created_at: time.Now(),
		Id:         id,
		From:       from,
		To:         to,
		Data:       data,
	}
}

func (e *Message) Type() string {
	return e.Kind
}

func (e *Message) ToString() string {
	j, err := json.Marshal(e)
	if err != nil {
		return ""
	}

	return string(j)
}

func (n *Conn) encodeMessage(m pubsub.Message) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (n *Conn) decodeMessage(data []byte, m interface{}) error {
	return json.Unmarshal(data, &m)
}
