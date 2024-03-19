package event

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func Publish(clientId, channel string, data map[string]interface{}) error {
	if conn == nil {
		return nil
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	msg := CreatedEvenMessage{
		Created_at: now,
		Id:         id,
		ClientId:   clientId,
		Channel:    channel,
		Data:       data,
	}

	dt, err := conn.encodeMessage(msg)
	if err != nil {
		return err
	}

	key := id
	cache.Set(key, msg.ToString(), 15)

	return conn.conn.Publish(msg.Type(), dt)
}

func Event(event string, data interface{}) {
	go Publish("event", "event/publish", et.Json{
		"event": event,
		"data":  data,
	})
}

func Work(work, work_id string, data interface{}) {
	go Publish("event", work, et.Json{
		"work":    work,
		"work_id": work_id,
		"data":    data,
	})
}

func Working(worker, work_id string) {
	go Publish("event", "event/working", et.Json{
		"worker":  worker,
		"work_id": work_id,
	})
}

func Done(work_id, event string) {
	go Publish("event", "event/done", et.Json{
		"work_id": work_id,
		"event":   event,
	})
}

func Rejected(work_id, event string) {
	go Publish("event", "event/rejected", et.Json{
		"work_id": work_id,
		"event":   event,
	})
}

func Action(action string, data map[string]interface{}) {
	go Publish("action", action, data)
}

func Subscribe(channel string, f func(CreatedEvenMessage)) (err error) {
	if conn == nil {
		return
	}

	msg := CreatedEvenMessage{
		Channel: channel,
	}
	conn.eventCreatedSub, err = conn.conn.Subscribe(msg.Type(), func(m *nats.Msg) {
		conn.decodeMessage(m.Data, &msg)
		f(msg)
	})

	return
}

func Stack(channel string, f func(CreatedEvenMessage)) (err error) {
	if conn == nil {
		return
	}

	msg := CreatedEvenMessage{
		Channel: channel,
	}

	conn.eventCreatedSub, err = conn.conn.Subscribe(channel, func(m *nats.Msg) {
		conn.decodeMessage(m.Data, &msg)
		key := msg.Id

		ok := conn.LockStack(key)
		if !ok {
			return
		}

		f(msg)
	})

	return
}

func WsBroadcast(message interface{}, ignoreId string) {
	go Publish("ws", "ws/broadcast", et.Json{
		"message":   message,
		"ignore_id": ignoreId,
	})
}

func WsPublish(channel string, message interface{}, ignoreId string) {
	go Publish("ws", "ws/publish", et.Json{
		"channel":   channel,
		"message":   message,
		"ignore_id": ignoreId,
	})
}

func WsSendMessage(clientId, channel string, message interface{}) {
	go Publish("ws", "ws/send_message", et.Json{
		"client_id": clientId,
		"channel":   channel,
		"message":   message,
	})
}
