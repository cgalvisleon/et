package event

import (
	"net/http"
	"slices"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/nats-io/nats.go"
)

const QUEUE_STACK = "stack"
const EVENT_LOG = "log"
const EVENT_TELEMETRY = "telemetry"
const EVENT_OVERFLOW = "requests:overflow"
const EVENT_TELEMETRY_TOKEN_LAST_USE = "telemetry:token:last_use"
const EVENT_WORK = "event:worker"
const EVENT_WORK_STATE = "event:worker:state"

var Events = []string{}

func publish(channel string, data et.Json) error {
	if conn == nil {
		return nil
	}

	msg := NewEvenMessage(channel, data)
	dt, err := msg.Encode()
	if err != nil {
		return err
	}
	msg.FromId = conn.id

	return conn.Publish(msg.Channel, dt)
}

/**
* Publish
* @param channel string
* @param data et.Json
* @return error
**/
func Publish(channel string, data et.Json) error {
	stage := config.App.Stage
	publish(strs.Format(`event:chanels:%s`, stage), et.Json{"channel": channel})
	publish(strs.Format(`pipe:%s:%s`, stage, channel), data)

	return publish(channel, data)
}

/**
* Subscribe
* @param channel string
* @param f func(Message)
* @return error
**/
func Subscribe(channel string, f func(Message)) (err error) {
	if conn == nil {
		return
	}

	if len(channel) == 0 {
		return
	}

	idx := slices.IndexFunc(Events, func(e string) bool { return e == channel })
	if idx == -1 {
		Events = append(Events, channel)
	}

	subscribe, err := conn.Subscribe(channel,
		func(m *nats.Msg) {
			msg, err := DecodeMessage(m.Data)
			if err != nil {
				return
			}

			msg.Myself = msg.FromId == conn.id
			f(msg)
		},
	)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	conn.eventCreatedSub[channel] = subscribe
	conn.mutex.Unlock()

	return err
}

/**
* Queue
* @param string channel
* @param func(Message) f
* @return error
**/
func Queue(channel, queue string, f func(Message)) (err error) {
	if conn == nil {
		return mistake.New(ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return nil
	}

	subscribe, err := conn.QueueSubscribe(
		channel,
		queue,
		func(m *nats.Msg) {
			msg, err := DecodeMessage(m.Data)
			if err != nil {
				return
			}

			f(msg)
		},
	)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	conn.eventCreatedSub[channel] = subscribe
	conn.mutex.Unlock()

	return nil
}

/**
* Stack
* @param channel string
* @param f func(Message)
* @return error
**/
func Stack(channel string, f func(Message)) error {
	return Queue(channel, QUEUE_STACK, f)
}

/**
* Source
* @param string channel
* @param func(Message) reciveFn
* @return error
**/
func Source(channel string, f func(Message)) error {
	return Subscribe(channel, f)
}

/**
* Log
* @param event string
* @param data et.Json
**/
func Log(event string, data et.Json) {
	go Publish(EVENT_LOG, data)
}

/**
* Telemetry
* @param data et.Json
**/
func Telemetry(data et.Json) {
	go Publish(EVENT_TELEMETRY, data)
}

/**
* Overflow
* @param data et.Json
**/
func Overflow(data et.Json) {
	go Publish(EVENT_OVERFLOW, data)
}

/**
* TokenLastUse
* @param data et.Json
**/
func TokenLastUse(data et.Json) {
	go Publish(EVENT_TELEMETRY_TOKEN_LAST_USE, data)
}

/**
* Error
* @param event string
* @param err error
* @return error
**/
func Error(event string, err error) error {
	go Publish(event, et.Json{"error": err.Error()})
	return err
}

/**
* HttpEventWork
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpEventWork(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	event := body.Str("event")
	data := body.Json("data")
	work := Work(event, data)

	response.JSON(w, r, http.StatusOK, work)
}
