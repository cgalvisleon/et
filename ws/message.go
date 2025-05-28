package ws

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type TpMessage int

const (
	TpPing           TpMessage = iota // 0
	TpSetFrom                         // 1
	TpSubscribe                       // 2
	TpQueueSubscribe                  // 3
	TpStack                           // 4
	TpUnsubscribe                     // 5
	TpPublish                         // 6
	TpDirect                          // 7
	TpConnect                         // 8
	TpDisconnect                      // 9
)

func (s TpMessage) String() string {
	switch s {
	case TpPing:
		return "Ping"
	case TpSetFrom:
		return "Set id and name"
	case TpSubscribe:
		return "Subscribe"
	case TpQueueSubscribe:
		return "Queue"
	case TpStack:
		return "Stack"
	case TpUnsubscribe:
		return "Unsubscribe"
	case TpPublish:
		return "Publish"
	case TpDirect:
		return "Direct"
	case TpConnect:
		return "Connected"
	case TpDisconnect:
		return "Disconnected"
	default:
		return "Unknown"
	}
}

func (s TpMessage) ToJson() et.Json {
	return et.Json{
		"code": s,
		"name": s.String(),
	}
}

func (s TpMessage) Int() int {
	return int(s)
}

func TypeMessages() et.Json {
	return et.Json{
		"ping":        TpPing.Int(),
		"set_from":    TpSetFrom.Int(),
		"subscribe":   TpSubscribe.Int(),
		"queue":       TpQueueSubscribe.Int(),
		"stack":       TpStack.Int(),
		"unsubscribe": TpUnsubscribe.Int(),
		"publish":     TpPublish.Int(),
		"direct":      TpDirect.Int(),
		"connect":     TpConnect.Int(),
		"disconnect":  TpDisconnect.Int(),
	}
}

func ToTpMessage(s string) TpMessage {
	switch s {
	case "Ping":
		return TpPing
	case "SetFrom":
		return TpSetFrom
	case "Subscribe":
		return TpSubscribe
	case "Queue":
		return TpQueueSubscribe
	case "Stack":
		return TpStack
	case "Unsubscribe":
		return TpUnsubscribe
	case "Publish":
		return TpPublish
	case "Direct":
		return TpDirect
	default:
		return -1
	}
}

type Message struct {
	Created_at time.Time   `json:"created_at"`
	Id         string      `json:"id"`
	From       et.Json     `json:"from"`
	To         string      `json:"to"`
	Ignored    []string    `json:"-"`
	Data       interface{} `json:"data"`
	Channel    string      `json:"channel"`
	Queue      string      `json:"queue"`
	Type       et.Json     `json:"type"`
	Tp         TpMessage   `json:"tp"`
}

/**
* NewMessage
* @param et.Json
* @param interface{}
* @param TpMessage
* @return Message
**/
func NewMessage(from et.Json, message interface{}, tp TpMessage) Message {
	return Message{
		Created_at: timezone.NowTime(),
		Id:         utility.UUID(),
		From:       from,
		Ignored:    []string{},
		Data:       message,
		Type:       tp.ToJson(),
		Tp:         tp,
	}
}

/**
* Encode return the message as byte array
* @return []byte
**/
func (e Message) Encode() ([]byte, error) {
	b, err := json.Marshal(e)
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
* @param []byte
* @return Message
**/
func DecodeMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}
	m.Type = m.Tp.ToJson()

	return m, nil
}
