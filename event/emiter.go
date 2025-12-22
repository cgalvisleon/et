package event

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var emiter *EventEmiter

func init() {
	emiter = newEventEmiter()
	emiter.start()
}

type Handler func(message Message)

type EventEmiter struct {
	channel chan Message       `json:"-"`
	events  map[string]Handler `json:"-"`
}

/**
* newEventEmiter
* @return *EventEmiter
**/
func newEventEmiter() *EventEmiter {
	return &EventEmiter{
		channel: make(chan Message),
		events:  make(map[string]Handler),
	}
}

/**
* start
**/
func (s *EventEmiter) start() {
	go func() {
		for message := range s.channel {
			fn, ok := s.events[message.Channel]
			if !ok {
				logs.Alert(fmt.Errorf("event not found (%s)", message.Channel))
				return
			}

			fn(message)
		}
	}()
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
	if s.channel == nil {
		return
	}

	message := Message{
		CreatedAt: timezone.NowTime(),
		Id:        utility.UUID(),
		Channel:   channel,
		Data:      data,
	}

	s.channel <- message
}

/**
* On
* @param channel string, handler Handler
**/
func On(channel string, handler Handler) {
	emiter.on(channel, handler)
}

/**
* emit
* @param channel string, data et.Json
**/
func Emiter(channel string, data et.Json) {
	emiter.emiter(channel, data)
}
