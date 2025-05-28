package event

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Message struct {
	CreatedAt time.Time `json:"created_at"`
	FromId    string    `json:"from_id"`
	Id        string    `json:"id"`
	Channel   string    `json:"channel"`
	Data      et.Json   `json:"data"`
	Myself    bool      `json:"myself"`
}

/**
* NewEvenMessage
* @param string channel
* @param et.Json data
* @return Message
**/
func NewEvenMessage(channel string, data et.Json) Message {
	return Message{
		CreatedAt: timezone.NowTime(),
		Id:        utility.UUID(),
		Channel:   channel,
		Data:      data,
	}
}

/**
* Encode
* @return []byte, error
**/
func (m Message) Encode() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

/**
* serialize
* @return []byte, error
**/
func (s Message) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s Message) ToJson() (et.Json, error) {
	definition, err := s.serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* ToString
* @return string
**/
func (s Message) ToString() string {
	j, err := s.ToJson()
	if err != nil {
		return et.Json{}.ToString()
	}

	return j.ToString()
}

/**
* DecodeMessage
* @param []byte data
* @return Message, error
**/
func DecodeMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}

	return m, nil
}
