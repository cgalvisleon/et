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
	TpPing TpMessage = iota
	TpSetFrom
	TpSubscribe
	TpUnsubscribe
	TpQueue
	TpPublish
	TpDirect
	TpConnect
	TpDisconnect
)

func (s TpMessage) String() string {
	switch s {
	case TpPing:
		return "Ping"
	case TpSetFrom:
		return "Set id and name"
	case TpSubscribe:
		return "Subscribe"
	case TpUnsubscribe:
		return "Unsubscribe"
	case TpQueue:
		return "Queue"
	case TpPublish:
		return "Publish"
	case TpDirect:
		return "Direct"
	default:
		return "Unknown"
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
	case "Unsubscribe":
		return TpUnsubscribe
	case "Queue":
		return TpQueue
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
	Ignored    []string    `json:"ignored"`
	Tp         TpMessage   `json:"tp"`
	Channel    string      `json:"channel"`
	Queue      string      `json:"queue"`
	Data       interface{} `json:"data"`
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
		Data:       message,
		Tp:         tp,
		Ignored:    []string{},
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
* ToString return the message as string
* @return string
**/
func (e Message) ToString() string {
	b, err := e.Encode()
	if err != nil {
		return ""
	}

	return string(b)
}

/**
* ToJson return the message as et.Json
* @return et.Json
**/
func (e Message) ToJson() (et.Json, error) {
	result, err := et.Object(e)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
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

	return m, nil
}
