package event

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

type WorkStatus int

const (
	WorkStatusPending WorkStatus = iota
	WorkStatusAccepted
	WorkStatusProcessing
	WorkStatusCompleted
	WorkStatusFailed
)

/**
* String
* @return string
**/
func (s WorkStatus) String() string {
	switch s {
	case WorkStatusPending:
		return "Pending"
	case WorkStatusAccepted:
		return "Accepted"
	case WorkStatusProcessing:
		return "Processing"
	case WorkStatusCompleted:
		return "Completed"
	case WorkStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

/**
* ToWorkStatus
* @param int n
* @return WorkStatus
**/
func ToWorkStatus(n int) WorkStatus {
	switch n {
	case 0:
		return WorkStatusPending
	case 1:
		return WorkStatusAccepted
	case 2:
		return WorkStatusProcessing
	case 3:
		return WorkStatusCompleted
	case 4:
		return WorkStatusFailed
	default:
		return WorkStatusPending
	}
}

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

	return conn.conn.Publish(msg.Channel, dt)
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

	conn.eventCreatedSub, err = conn.conn.Subscribe(channel,
		func(m *nats.Msg) {
			msg, err := DecodeMessage(m.Data)
			if err != nil {
				return
			}

			f(msg)
		},
	)

	return
}

/**
* Queue
* @param string channel
* @param func(EvenMessage) f
* @return error
**/
func Queue(channel, queue string, f func(EvenMessage)) (err error) {
	if conn == nil {
		return logs.NewError(ERR_NOT_CONNECT)
	}

	if len(channel) == 0 {
		return nil
	}

	conn.eventCreatedSub, err = conn.conn.QueueSubscribe(
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

	return nil
}

/**
* Stack
* @param channel string
* @param f func(EvenMessage)
* @return error
**/
func Stack(channel string, f func(EvenMessage)) error {
	return Queue(channel, "stack", f)
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
		"event":      event,
		"data":       data,
	}

	go Publish("event/worker", work)
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

	go Publish("event/worker/state", work)
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
	go Publish("log", data)
}

/**
* Telemetry
* @param data et.Json
**/
func Telemetry(data et.Json) {
	go Publish("telemetry", data)
}

/**
* Overflow
* @param data et.Json
**/
func Overflow(data et.Json) {
	go Publish("requests/overflow", data)
}

/**
* TokenLastUse
* @param data et.Json
**/
func TokenLastUse(data et.Json) {
	go Publish("telemetry.token.last_use", data)
}

/**
* RealTime
* @param data et.Json
**/
func RealTime(channel string, from et.Json, message interface{}) {
	data := et.Json{
		"created_at": timezone.Now(),
		"_id":        utility.UUID(),
		"channel":    channel,
		"from":       from,
		"message":    message,
	}

	go Publish("realtime", data)
}

/**
* Test event, testing message broker
* @param w http.ResponseWriter
* @param r *http.Request
**/
func Test(w http.ResponseWriter, r *http.Request) {
	body, _ := response.GetBody(r)
	event := body.Str("event")
	data := body.Json("data")

	work := Work(event, data)

	response.JSON(w, r, http.StatusOK, work)
}
