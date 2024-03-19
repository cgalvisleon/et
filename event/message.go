package event

import (
	"encoding/json"
	"time"
)

type Message interface {
	Type() string
}

type CreatedEvenMessage struct {
	Created_at time.Time              `json:"created_at"`
	Id         string                 `json:"id"`
	ClientId   string                 `json:"client_id"`
	Channel    string                 `json:"channel"`
	Data       map[string]interface{} `json:"data"`
}

func (m CreatedEvenMessage) Type() string {
	return m.Channel
}

func (m CreatedEvenMessage) ToString() string {
	j, err := json.Marshal(m)
	if err != nil {
		return ""
	}

	return string(j)
}

func (n *Conn) encodeMessage(m Message) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (n *Conn) decodeMessage(data []byte, m interface{}) error {
	return json.Unmarshal(data, &m)
}
