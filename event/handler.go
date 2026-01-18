package event

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/timezone"
	"github.com/nats-io/nats.go"
)

type EventStatus string

const (
	EventPublished    EventStatus = "published"
	EventSubscribed   EventStatus = "subscribed"
	EventUnsubscribed EventStatus = "unsubscribed"
	EventReceived     EventStatus = "received"
	QUEUE_STACK                   = "stack"
	EVENT_STATUS                  = "event:status"
	EVENT_LOG                     = "event:log"
	EVENT_OVERFLOW                = "event:overflow"
	EVENT_WORK                    = "event:work"
	EVENT_WORK_STATE              = "event:work:state"
)

/**
* publish
* @param channel string, data et.Json
* @return error
**/
func publish(channel string, data et.Json) error {
	if conn == nil {
		return nil
	}

	msg := NewEvenMessage(channel, data)
	msg.FromId = conn.id
	dt, err := msg.Encode()
	if err != nil {
		return err
	}

	_, err = conn.add(channel)
	if err != nil {
		return err
	}

	return conn.Publish(msg.Channel, dt)
}

/**
* eventState
* @params channel string, status EventStatus, data interface{}
**/
func eventState(channel string, status EventStatus, data interface{}) {
	msg := et.Json{
		"created_at": timezone.NowTime(),
		"channel":    channel,
		"event":      status,
		"host":       hostName,
		"id":         reg.GenULID("event"),
	}
	if data != nil {
		msg["msg"] = data
	}

	stage := envar.GetStr("STAGE", "DEV")
	publish(EVENT_STATUS, msg)
	publish(fmt.Sprintf(`pipe:%s:%s`, stage, channel), msg)
}

/**
* Publish
* @param channel string, data et.Json
* @return error
**/
func Publish(channel string, data et.Json) error {
	if conn == nil {
		return nil
	}

	eventState(channel, EventPublished, data)
	return publish(channel, data)
}

/**
* Unsubscribe
* @param channel string
* @return error
**/
func Unsubscribe(channel string) error {
	if conn == nil {
		return fmt.Errorf(msg.MSG_ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return fmt.Errorf(msg.MSG_ERR_CHANNEL_REQUIRED)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	subscribe, ok := conn.events[channel]
	if !ok {
		return nil
	}

	subscribe.Unsubscribe()
	conn.mutex.Lock()
	delete(conn.events, channel)
	conn.mutex.Unlock()
	eventState(channel, EventUnsubscribed, nil)

	return nil
}

/**
* Subscribe
* @param channel string, f func(Message)
* @return error
**/
func Subscribe(channel string, f func(Message)) (err error) {
	if conn == nil {
		return
	}

	if len(channel) == 0 {
		return
	}

	ok, err := conn.add(channel)
	if err != nil {
		return err
	}

	if ok {
		eventState(channel, EventSubscribed, nil)
	}

	subscribe, err := conn.Subscribe(channel,
		func(m *nats.Msg) {
			msg, err := DecodeMessage(m.Data)
			if err != nil {
				return
			}

			msg.Myself = msg.FromId == conn.id

			data, err := msg.ToJson()
			if err != nil {
				data = et.Json{}
			}

			eventState(channel, EventReceived, data)
			f(msg)
		},
	)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	conn.events[channel] = subscribe
	conn.mutex.Unlock()

	return err
}

/**
* Queue
* @param channel string, queue string, f func(Message)
* @return error
**/
func Queue(channel, queue string, f func(Message)) (err error) {
	if conn == nil {
		return fmt.Errorf(msg.MSG_ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return nil
	}

	ok, err := conn.add(channel)
	if err != nil {
		return err
	}

	if ok {
		eventState(channel, EventSubscribed, nil)
	}

	subscribe, err := conn.QueueSubscribe(
		channel,
		queue,
		func(m *nats.Msg) {
			msg, err := DecodeMessage(m.Data)
			if err != nil {
				return
			}

			msg.Myself = msg.FromId == conn.id

			data, err := msg.ToJson()
			if err != nil {
				data = et.Json{}
			}

			eventState(channel, EventReceived, data)
			f(msg)
		},
	)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	conn.events[channel] = subscribe
	conn.mutex.Unlock()

	return nil
}

/**
* Stack
* @param channel string, f func(Message)
* @return error
**/
func Stack(channel string, f func(Message)) error {
	return Queue(channel, QUEUE_STACK, f)
}

/**
* Source
* @param channel string, f func(Message)
* @return error
**/
func Source(channel string, f func(Message)) error {
	return Subscribe(channel, f)
}

/**
* Log
* @param event string, data et.Json
**/
func Log(event string, data et.Json) {
	go Publish(EVENT_LOG, data)
}

/**
* Overflow
* @param data et.Json
**/
func Overflow(data et.Json) {
	go Publish(EVENT_OVERFLOW, data)
}

/**
* Error
* @param event string,
* @return error
**/
func Error(event string, err error) error {
	go Publish(event, et.Json{"error": err.Error()})
	return err
}

/**
* HttpEventPublish
* @param w http.ResponseWriter, r *http.Request
**/
func HttpEventPublish(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	channel := body.Str("channel")
	data := body.Json("data")
	err := Publish(channel, data)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, et.Item{
		Ok: err == nil,
		Result: et.Json{
			"message": "Event published",
		},
	})
}
