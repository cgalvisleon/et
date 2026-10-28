package event

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var (
	emitter    *EventEmitter
	emiterOnce sync.Once
)

type Handler func(message *Message)

type EventEmitter struct {
	mu     sync.RWMutex
	events map[string]Handler `json:"-"`
	ch     chan *Message      `json:"-"`
}

/**
* newEventEmitter
* @return *EventEmitter
**/
func newEventEmitter() *EventEmitter {
	result := &EventEmitter{
		events: make(map[string]Handler),
		ch:     make(chan *Message),
	}

	logs.Log("event", "Event emitter initialized")
	go result.loop()
	return result
}

/**
* loop: Delivers each message of the channel to the handler of its channel name.
**/
func (s *EventEmitter) loop() {
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
func (s *EventEmitter) on(channel string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.events == nil {
		s.events = make(map[string]Handler)
	}

	s.events[channel] = handler
}

/**
* emitter: Queues a message for the handler of channel (blocks until loop takes it).
* @param channel string, data et.Json
**/
func (s *EventEmitter) emitter(channel string, data et.Json) {
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
		emitter = newEventEmitter()
	})

	emitter.on(channel, handler)
}

/**
* Emitter: Emits data on channel through the in-process emitter (see On).
* @param channel string, data et.Json
**/
func Emitter(channel string, data et.Json) {
	emiterOnce.Do(func() {
		emitter = newEventEmitter()
	})

	emitter.emitter(channel, data)
}
