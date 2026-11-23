package event

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
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
* One subscription per key: SubscribeWith, QueueWith and StackWith store it under a key of the
* caller (its own map, apart from the channels), so several listeners of one channel live side by
* side; the same key again replaces (unsubscribes) the previous one, and UnsubscribeKey cuts only
* that one. Without a connection they return MSG_ERR_NOT_CONNECT.
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
	conn, err = New(Params{
		Host:     envar.GetStr("NATS_HOST", ""),
		User:     envar.GetStr("NATS_USER", ""),
		Password: envar.GetStr("NATS_PASSWORD", ""),
	})
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
* Pub: Publishes data on channel over this connection (one of its own, New, or the package's); without a connection
* (nil) it does nothing
* @param channel string, data et.Json
* @return error
**/
func (s *Conn) Pub(channel string, data et.Json) error {
	if s == nil {
		return nil
	}

	msg := NewEventMessage(channel, data)
	msg.FromId = s.id
	dt, err := msg.Encode()
	if err != nil {
		return err
	}

	return s.Conn.Publish(msg.Channel, dt)
}

/**
* publish
* @param channel string, data et.Json
* @return error
**/
func publish(channel string, data et.Json) error {
	return conn.Pub(channel, data)
}

/**
* Publish
* @param channel string, data et.Json
* @return error
**/
func Publish(channel string, data et.Json) error {
	return conn.Pub(channel, data)
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
* handlerOf: The NATS handler of a subscription of the connection s: decodes the message, sets Myself and calls f,
* recovering a panic of f
* @param s *Conn, channel, panicMsg string, f func(Message)
* @return nats.MsgHandler
**/
func handlerOf(s *Conn, channel, panicMsg string, f func(Message)) nats.MsgHandler {
	return func(m *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				logs.Errorf(panicMsg, channel, r)
			}
		}()

		if m == nil {
			return
		}

		msg, err := DecodeMessage(m.Data)
		if err != nil {
			return
		}

		msg.Myself = msg.FromId == s.id
		if f != nil {
			f(msg)
		}
	}
}

/**
* Sub: Subscribes channel over this connection (every replica receives), stored under the channel like Subscribe;
* without a connection (nil) or channel it does nothing
* @param channel string, f func(Message)
* @return error
**/
func (s *Conn) Sub(channel string, f func(Message)) error {
	if s == nil || len(channel) == 0 {
		return nil
	}

	subscribe, err := s.Subscribe(channel, handlerOf(s, channel, MSG_PANIC_IN_SUBSCRIBE, f))
	if err != nil {
		return err
	}

	s.mutex.Lock()
	s.events[channel] = subscribe
	s.mutex.Unlock()

	return nil
}

/**
* Subscribe
* @param channel string, f func(Message)
* @return error
**/
func Subscribe(channel string, f func(Message)) (err error) {
	return conn.Sub(channel, f)
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

	subscribe, err := conn.QueueSubscribe(channel, queue, handlerOf(conn, channel, MSG_PANIC_IN_QUEUE, f))
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	conn.events[channel] = subscribe
	conn.mutex.Unlock()

	return nil
}

/**
* subscribeWith: Subscribes channel (every replica receives) or, with queue, the queue group of channel (one replica of
* the group receives), under key: the same key again unsubscribes the previous one
* @param key, channel, queue string, f func(Message)
* @return error
**/
func (s *Conn) subscribeWith(key, channel, queue string, f func(Message)) error {
	if s == nil {
		return errors.New(msg.MSG_ERR_NOT_CONNECT)
	}

	if len(key) == 0 {
		return errors.New(msg.MSG_KEY_REQUIRED)
	}

	if len(channel) == 0 {
		return errors.New(msg.MSG_ERR_CHANNEL_REQUIRED)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var subscribe *nats.Subscription
	var err error
	if queue == "" {
		subscribe, err = s.Subscribe(channel, handlerOf(s, channel, MSG_PANIC_IN_SUBSCRIBE, f))
	} else {
		subscribe, err = s.QueueSubscribe(channel, queue, handlerOf(s, channel, MSG_PANIC_IN_QUEUE, f))
	}
	if err != nil {
		return err
	}

	if previous, ok := s.keyed[key]; ok {
		previous.Unsubscribe()
	}
	s.keyed[key] = subscribe

	return nil
}

/**
* SubscribeWith: Subscribe under a key of the caller over this connection (nil: MSG_ERR_NOT_CONNECT); see the package's
* SubscribeWith
* @param key, channel string, f func(Message)
* @return error
**/
func (s *Conn) SubscribeWith(key, channel string, f func(Message)) error {
	return s.subscribeWith(key, channel, "", f)
}

/**
* QueueWith: Queue under a key of the caller over this connection; see the package's QueueWith
* @param key, channel, queue string, f func(Message)
* @return error
**/
func (s *Conn) QueueWith(key, channel, queue string, f func(Message)) error {
	if len(queue) == 0 {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "queue")
	}

	return s.subscribeWith(key, channel, queue, f)
}

/**
* UnsubscribeKey: Cuts the subscription of key over this connection; an unknown key is not an error
* @param key string
* @return error
**/
func (s *Conn) UnsubscribeKey(key string) error {
	if s == nil {
		return errors.New(msg.MSG_ERR_NOT_CONNECT)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscribe, ok := s.keyed[key]
	if !ok {
		return nil
	}

	delete(s.keyed, key)
	return subscribe.Unsubscribe()
}

/**
* HasKey: Whether this connection has a subscription under key
* @param key string
* @return bool
**/
func (s *Conn) HasKey(key string) bool {
	if s == nil {
		return false
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.keyed[key]
	return ok
}

/**
* SubscribeWith: Subscribe under a key of the caller: several listeners of one channel do not replace each other, and
* UnsubscribeKey cuts only this one; the same key again replaces the previous subscription
* @param key, channel string, f func(Message)
* @return error
**/
func SubscribeWith(key, channel string, f func(Message)) error {
	return conn.SubscribeWith(key, channel, f)
}

/**
* QueueWith: Queue under a key of the caller (one replica of the queue group receives each message); see SubscribeWith
* @param key, channel, queue string, f func(Message)
* @return error
**/
func QueueWith(key, channel, queue string, f func(Message)) error {
	return conn.QueueWith(key, channel, queue, f)
}

/**
* StackWith: QueueWith on the queue group QUEUE_STACK (shared by every StackWith and Stack of the channel)
* @param key, channel string, f func(Message)
* @return error
**/
func StackWith(key, channel string, f func(Message)) error {
	return QueueWith(key, channel, QUEUE_STACK, f)
}

/**
* UnsubscribeKey: Cuts the subscription of key (SubscribeWith, QueueWith, StackWith); an unknown key is not an error
* @param key string
* @return error
**/
func UnsubscribeKey(key string) error {
	return conn.UnsubscribeKey(key)
}

/**
* HasKey: Whether there is a subscription under key
* @param key string
* @return bool
**/
func HasKey(key string) bool {
	return conn.HasKey(key)
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
