package event

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Message interface {
	Type() string
}

type EvenMessage struct {
	Created_at time.Time `json:"created_at"`
	Id         string    `json:"id"`
	Channel    string    `json:"channel"`
	Data       et.Json   `json:"data"`
}

/**
* NewEvenMessage
* @param string channel
* @param et.Json data
* @return EvenMessage
**/
func NewEvenMessage(channel string, data et.Json) EvenMessage {
	return EvenMessage{
		Created_at: timezone.NowTime(),
		Id:         utility.UUID(),
		Channel:    channel,
		Data:       data,
	}
}

/**
* Encode
* @return []byte, error
**/
func (m EvenMessage) Encode() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

/**
* ToString
* @return string
**/
func (m EvenMessage) ToString() string {
	j, err := json.Marshal(m)
	if err != nil {
		return ""
	}

	return string(j)
}

/**
* ToJson
* @return et.Json, error
**/
func (m EvenMessage) ToJson() (et.Json, error) {
	j, err := et.Object(m)
	if err != nil {
		return et.Json{}, err
	}

	return j, nil
}

/**
* DecodeMessage
* @param []byte data
* @return EvenMessage, error
**/
func DecodeMessage(data []byte) (EvenMessage, error) {
	var m EvenMessage
	err := json.Unmarshal(data, &m)
	if err != nil {
		return EvenMessage{}, err
	}

	return m, nil
}
