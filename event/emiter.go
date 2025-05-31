package event

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Handler func(message Message)

type EventEmiter struct {
	channel chan Message       `json:"-"`
	events  map[string]Handler `json:"-"`
}

var emiter *EventEmiter

func init() {
	emiter = NewEventEmiter()
	emiter.Start()
}

/**
* NewEventEmiter
* @return *EventEmiter
**/
func NewEventEmiter() *EventEmiter {
	return &EventEmiter{
		channel: make(chan Message),
		events:  make(map[string]Handler),
	}
}

/**
* EventEmiter
* @param message Message
**/
func (s *EventEmiter) eventEmiter(message Message) {
	if s.events == nil {
		s.events = make(map[string]Handler)
	}

	eventEmiter, ok := s.events[message.Channel]
	if !ok {
		logs.Alert(fmt.Errorf("event not found (%s)", message.Channel))
		return
	}

	eventEmiter(message)
}

/**
* Start
**/
func (s *EventEmiter) Start() {
	go func() {
		for message := range s.channel {
			s.eventEmiter(message)
		}
	}()
}

/**
* On
* @param channel string, handler Handler
**/
func (s *EventEmiter) On(channel string, handler Handler) {
	if s.events == nil {
		s.events = make(map[string]Handler)
	}

	s.events[channel] = handler
}

/**
* Emit
* @param channel string, data et.Json
**/
func (s *EventEmiter) Emit(channel string, data et.Json) {
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
	if emiter == nil {
		emiter = NewEventEmiter()
	}

	emiter.On(channel, handler)
}

/**
* Emit
* @param channel string, data et.Json
**/
func Emit(channel string, data et.Json) {
	emiter.Emit(channel, data)
}
