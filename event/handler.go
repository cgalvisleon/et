package event

import (
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

const QUEUE_STACK = "stack"
const EVENT_LOG = "log"
const EVENT_TELEMETRY = "telemetry"
const EVENT_OVERFLOW = "requests/overflow"
const EVENT_LAT_TOKEN_USE = "telemetry.token.last_use"
const EVENT_WORK = "event/worker"
const EVENT_WORK_STATE = "event/worker/state"

/**
* Publish
* @param channel string
* @param data et.Json
* @return error
**/
func Publish(channel string, data et.Json) error {
	if conn == nil {
		return nil
	}

	msg := NewEvenMessage(channel, data)
	dt, err := msg.Encode()
	if err != nil {
		return err
	}

	return conn.Publish(msg.Channel, dt)
}

/**
* Subscribe
* @param channel string
* @param f func(EvenMessage)
* @return error
**/
func Subscribe(channel string, f func(EvenMessage)) (err error) {
	if conn == nil {
		return
	}

	if len(channel) == 0 {
		return
	}

	subscribe, err := conn.Subscribe(channel,
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

	return err
}

/**
* Queue
* @param string channel
* @param func(EvenMessage) f
* @return error
**/
func Queue(channel, queue string, f func(EvenMessage)) (err error) {
	if conn == nil {
		return errors.New(ERR_NOT_CONNECT)
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
* @param f func(EvenMessage)
* @return error
**/
func Stack(channel string, f func(EvenMessage)) error {
	return Queue(channel, QUEUE_STACK, f)
}

/**
* Work
* @param event string
* @param data et.Json
**/
func Work(event string, data et.Json) et.Json {
	work := et.Json{
		"created_at": timezone.Now(),
		"_id":        utility.UUID(),
		"from_id":    conn._id,
		"event":      event,
		"data":       data,
	}

	go Publish(EVENT_WORK, work)
	go Publish(event, work)

	return work
}

/**
* WorkState
* @param work_id string
* @param status WorkStatus
* @param data et.Json
**/
func WorkState(work_id string, status WorkStatus, data et.Json) {
	work := et.Json{
		"update_at": timezone.Now(),
		"_id":       work_id,
		"from_id":   conn._id,
		"status":    status.String(),
		"data":      data,
	}
	switch status {
	case WorkStatusPending:
		work["pending_at"] = utility.Now()
	case WorkStatusAccepted:
		work["accepted_at"] = utility.Now()
	case WorkStatusProcessing:
		work["processing_at"] = utility.Now()
	case WorkStatusCompleted:
		work["completed_at"] = utility.Now()
	case WorkStatusFailed:
		work["failed_at"] = utility.Now()
	}

	go Publish(EVENT_WORK_STATE, work)
}

/**
* Data
* @param string channel
* @param func(Message) reciveFn
* @return error
**/
func Data(channel string, data et.Json) error {
	return Publish(channel, data)
}

/**
* Source
* @param string channel
* @param func(Message) reciveFn
* @return error
**/
func Source(channel string, f func(EvenMessage)) error {
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
	go Publish(EVENT_LAT_TOKEN_USE, data)
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
