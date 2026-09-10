package event

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var (
	emiter     *EventEmiter
	emiterOnce sync.Once
)

type Handler func(message *Message)

type EventEmiter struct {
	mu     sync.RWMutex
	events map[string]Handler `json:"-"`
	ch     chan *Message      `json:"-"`
}

/**
* newEventEmiter
* @return *EventEmiter
**/
func newEventEmiter() *EventEmiter {
	result := &EventEmiter{
		events: make(map[string]Handler),
		ch:     make(chan *Message),
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
		if message == nil {
			continue
		}

		s.mu.RLock()
		fn, ok := s.events[message.Channel]
		s.mu.RUnlock()
		if !ok {
			continue
		}

		fn(message)
	}
}

/**
* on
* @param channel string, handler Handler
**/
func (s *EventEmiter) on(channel string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	message := &Message{
		CreatedAt: timezone.Now(),
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
	emiterOnce.Do(func() {
		emiter = newEventEmiter()
	})

	emiter.on(channel, handler)
}

/**
* emit
* @param channel string, data et.Json
**/
func Emiter(channel string, data et.Json) {
	emiterOnce.Do(func() {
		emiter = newEventEmiter()
	})

	emiter.emiter(channel, data)
}
