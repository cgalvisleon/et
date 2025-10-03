package event

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/nats-io/nats.go"
)

const (
	QUEUE_STACK      = "stack"
	EVENT            = "event"
	EVENT_LOG        = "event:log"
	EVENT_OVERFLOW   = "event:overflow"
	EVENT_WORK       = "event:worker"
	EVENT_WORK_STATE = "event:worker:state"
	EVENT_SUBSCRIBED = "event:subscribed"
)

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
* Publish
* @param channel string, data et.Json
* @return error
**/
func Publish(channel string, data et.Json) error {
	if conn == nil {
		return nil
	}

	stage := config.App.Stage
	publish(EVENT, et.Json{"channel": channel, "data": data})
	publish(fmt.Sprintf(`pipe:%s:%s`, stage, channel), data)

	return publish(channel, data)
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
		publish(EVENT_SUBSCRIBED, et.Json{"channel": channel})
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
* Unsubscribe
* @param channel string
* @return error
**/
func Unsubscribe(channel string) error {
	if conn == nil {
		return fmt.Errorf(ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return fmt.Errorf(ERR_CHANNEL_REQUIRED)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	subscribe, ok := conn.eventCreatedSub[channel]
	if !ok {
		return fmt.Errorf("channel %s not found", channel)
	}

	subscribe.Unsubscribe()
	delete(conn.eventCreatedSub, channel)

	return nil
}

/**
* Queue
* @param channel string, queue string, f func(Message)
* @return error
**/
func Queue(channel, queue string, f func(Message)) (err error) {
	if conn == nil {
		return fmt.Errorf(ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return nil
	}

	ok, err := conn.add(channel)
	if err != nil {
		return err
	}

	if ok {
		publish(EVENT_SUBSCRIBED, et.Json{"channel": channel})
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
