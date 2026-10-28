package event

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/nats-io/nats.go"
)

/**
* Invariants of the event package.
*
* One subscription per channel: Subscribe and Queue store the new subscription under the channel
* without unsubscribing the previous one, which keeps receiving; call Unsubscribe before
* subscribing the same channel again.
* Close iterates the subscriptions without taking the mutex: do not subscribe while closing.
* Log and Overflow are fire-and-forget through a buffer of 256 messages (asyncPublishBufSize):
* when it is full the message is dropped.
* Publish and Subscribe are no-ops without a connection (Load not called or failed); Queue and
* Unsubscribe return MSG_ERR_NOT_CONNECT.
**/

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
	EVENT_SET_ERROR               = "event:set:error"
)

/**
* Load
* @return error
**/
func Load() error {
	if conn != nil {
		return nil
	}

	var err error
	conn, err = New()
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
		conn = nil
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

	msg := NewEventMessage(channel, data)
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
			if f != nil {
				f(msg)
			}
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
					logs.Errorf(MSG_PANIC_IN_QUEUE, channel, r)
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

			if f != nil {
				f(msg)
			}
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
* Log: Publishes data on the event channel without blocking (dropped if the buffer is full).
* @param event string, data et.Json
**/
func Log(event string, data et.Json) {
	select {
	case asyncPublishCh <- asyncMsg{event, data}:
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
* @param data et.Json
* @return error
**/
func Error(data et.Json) {
	Log(EVENT_SET_ERROR, data)
}

/**
* ErrorMessage: Publishes an error with its method and function on EVENT_SET_ERROR (see Error).
* @param tenantId, message, method, functionName string
**/
func ErrorMessage(tenantId, message, method, functionName string) {
	Error(et.Json{
		"tenant_id":     tenantId,
		"message":       message,
		"method":        method,
		"function_name": functionName,
	})
}

/**
* ErrorDetails: Publishes a provider error with its status, response and attempt (see Error).
* @param tenantId string, statusCode int, provider, providerError, response string, attempt int
**/
func ErrorDetails(tenantId string, statusCode int, provider, providerError, response string, attempt int) {
	Error(et.Json{
		"tenant_id":   tenantId,
		"message":     providerError,
		"status_code": statusCode,
		"provider":    provider,
		"response":    response,
		"attempt":     attempt,
	})
}

/**
* ErrorContext: Publishes ctx as an error after setting the tenant, workflow, node and customer ids on it (see Error).
* @param tenantId, workflowId, nodeId, customerId string, ctx et.Json
**/
func ErrorContext(tenantId, workflowId, nodeId, customerId string, ctx et.Json) {
	ctx.Set("tenant_id", tenantId)
	ctx.Set("workflow_id", workflowId)
	ctx.Set("node_id", nodeId)
	ctx.Set("customer_id", customerId)
	Error(ctx)
}
