package event

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var emiter *EventEmiter

func init() {
	emiter = newEventEmiter()
}

type Handler func(message Message)

type EventEmiter struct {
	events map[string]Handler `json:"-"`
	ch     chan Message       `json:"-"`
}

/**
* newEventEmiter
* @return *EventEmiter
**/
func newEventEmiter() *EventEmiter {
	result := &EventEmiter{
		events: make(map[string]Handler),
		ch:     make(chan Message),
	}

	logs.Log("event", "Event emitter initialized")
	go result.loop()
	return result
}

/**
* start
**/
func (s *EventEmiter) loop() {
	for message := range s.ch {
		fn, ok := s.events[message.Channel]
		if !ok {
			return
		}

		fn(message)
	}
}

/**
* on
* @param channel string, handler Handler
**/
func (s *EventEmiter) on(channel string, handler Handler) {
	if s.events == nil {
		s.events = make(map[string]Handler)
	}

	s.events[channel] = handler
}

/**
* emit
* @param channel string, data et.Json
**/
func (s *EventEmiter) emiter(channel string, data et.Json) {
	if s.ch == nil {
		return
	}

	message := Message{
		CreatedAt: timezone.NowTime(),
		Id:        utility.UUID(),
		Channel:   channel,
		Data:      data,
	}

	s.ch <- message
}

/**
* On
* @param channel string, handler Handler
**/
func On(channel string, handler Handler) {
	if emiter == nil {
		return
	}

	emiter.on(channel, handler)
}

/**
* emit
* @param channel string, data et.Json
**/
func Emiter(channel string, data et.Json) {
	if emiter == nil {
		return
	}

	emiter.emiter(channel, data)
}
