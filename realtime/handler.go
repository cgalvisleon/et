package realtime

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
)

/**
* From
* @return et.Json
**/
func From() et.Json {
	if conn == nil {
		return et.Json{}
	}

	return conn.From()
}

/**
* Ping
**/
func Ping() {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return
		}
	}

	conn.Ping()
}

/**
* SetFrom
* @param params et.Json
* @return error
**/
func SetFrom(name string) error {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return err
		}
	}

	return conn.SetFrom(name)
}

/**
* Publish a message to a channel
* @param channel string
* @param message interface{}
**/
func Publish(channel string, message interface{}) error {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return err
		}
	}

	conn.Publish(channel, message)
	return nil
}

/**
* SendMessage
* @param clientId string
* @param message interface{}
* @return error
**/
func SendMessage(clientId string, message interface{}) error {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return err
		}
	}

	return conn.SendMessage(clientId, message)
}

/**
* Subscribe to a channel
* @param channel string
* @param reciveFn func(message.ws.Message)
**/
func Subscribe(channel string, reciveFn func(ws.Message)) error {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return err
		}
	}

	conn.Subscribe(channel, reciveFn)
	return nil
}

/**
* Unsubscribe to a channel
* @param channel string
**/
func Unsubscribe(channel string) {
	if conn == nil {
		return
	}

	conn.Unsubscribe(channel)
}

/**
* Queue to a channel
* @param channel string
* @param reciveFn func(message.ws.Message)
**/
func Queue(channel, queue string, reciveFn func(ws.Message)) {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return
		}
	}

	conn.Queue(channel, queue, reciveFn)
}

/**
* Stack to a channel
* @param channel string
* @param reciveFn func(message.ws.Message)
**/
func Stack(channel string, reciveFn func(ws.Message)) {
	if conn == nil {
		_, err := Load()
		if err != nil {
			return
		}
	}

	conn.Queue(channel, utility.QUEUE_STACK, reciveFn)
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
func WorkState(work_id string, status event.WorkStatus, data et.Json) {
	work := et.Json{
		"update_at": timezone.Now(),
		"_id":       work_id,
		"status":    status.String(),
		"data":      data,
	}
	switch status {
	case event.WorkStatusPending:
		work["pending_at"] = utility.Now()
	case event.WorkStatusAccepted:
		work["accepted_at"] = utility.Now()
	case event.WorkStatusProcessing:
		work["processing_at"] = utility.Now()
	case event.WorkStatusCompleted:
		work["completed_at"] = utility.Now()
	case event.WorkStatusFailed:
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
func Source(channel string, reciveFn func(ws.Message)) error {
	return Subscribe(channel, reciveFn)
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
* SetDirectMessageCallback
* @param reciveFn func(ws.Message)
**/
func SetDirectMessageCallback(reciveFn func(ws.Message)) {
	if conn == nil {
		return
	}

	conn.SetDirectMessageCallback(reciveFn)
}

/**
* SetReconnectCallback
* @param reciveFn func(ws.Client)
**/
func SetReconnectCallback(reciveFn func(c *ws.Client)) {
	if conn == nil {
		return
	}

	conn.SetReconnectCallback(reciveFn)
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
