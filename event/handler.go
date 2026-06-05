package event

import (
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
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
* Load
* @return error
**/
func Load(cfg Config) error {
	if conn != nil {
		return nil
	}

	var err error
	conn, err = New(cfg)
	if err != nil {
		return err
	}

	return nil
}

/**
* Close
**/
func Close() {
	if conn != nil {
		conn.Close()
	}
}

/**
* IsLoad
* @return bool
**/
func IsLoad() bool {
	return conn != nil
}

/**
* HealthCheck
* @return bool
**/
func HealthCheck() bool {
	if conn == nil {
		return false
	}

	return conn.HealthCheck()
}

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

	return conn.Publish(msg.Channel, dt)
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

	return publish(channel, data)
}

/**
* Unsubscribe
* @param channel string
* @return error
**/
func Unsubscribe(channel string) error {
	if conn == nil {
		return errors.New(msg.MSG_ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return errors.New(msg.MSG_ERR_CHANNEL_REQUIRED)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	subscribe, ok := conn.events[channel]
	if !ok {
		return nil
	}

	subscribe.Unsubscribe()
	delete(conn.events, channel)

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

	subscribe, err := conn.Subscribe(channel,
		func(m *nats.Msg) {
			defer func() {
				if r := recover(); r != nil {
					logs.Errorf(MSG_PANIC_IN_SUBSCRIBE, channel, r)
				}
			}()

			if m == nil {
				return
			}

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
		return errors.New(msg.MSG_ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return nil
	}

	subscribe, err := conn.QueueSubscribe(
		channel,
		queue,
		func(m *nats.Msg) {
			defer func() {
				if r := recover(); r != nil {
					logs.Errorf("panic in Queue channel:%s err:%v", channel, r)
				}
			}()

			if m == nil {
				return
			}

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
	select {
	case asyncPublishCh <- asyncMsg{EVENT_LOG, data}:
	default:
	}
}

/**
* Overflow
* @param data et.Json
**/
func Overflow(data et.Json) {
	select {
	case asyncPublishCh <- asyncMsg{EVENT_OVERFLOW, data}:
	default:
	}
}

/**
* Error
* @param event string,
* @return error
**/
func Error(event string, err error) error {
	select {
	case asyncPublishCh <- asyncMsg{event, et.Json{"error": err.Error()}}:
	default:
	}
	return err
}

/**
* HttpEventPublish
* @param w http.ResponseWriter, r *http.Request
**/
func HttpEventPublish(w http.ResponseWriter, r *http.Request) {
	body, _ := request.GetBody(r)
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
