package jws

import (
	"slices"
	"sync"
)

type TypeChannel string

const (
	TpQueue TypeChannel = "Queue"
	TpStack TypeChannel = "Stack"
	TpTopic TypeChannel = "Topic"
)

type Channel struct {
	Name        string      `json:"name"`
	Type        TypeChannel `json:"type"`
	Subscribers []string    `json:"subscribers"`
	Turn        int         `json:"turn"`
	mu          *sync.Mutex `json:"-"`
}

/**
* newChannel
* @param tp TypeChannel
* @return *Channel
**/
func newChannel(name string, tp TypeChannel) *Channel {
	return &Channel{
		Name:        name,
		Type:        tp,
		Subscribers: []string{},
		Turn:        0,
		mu:          &sync.Mutex{},
	}
}

/**
* subscriber
* @param subscriber string
**/
func (s *Channel) subscriber(subscriber string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.Subscribers, func(item string) bool {
		return item == subscriber
	})
	if idx != -1 {
		return
	}

	s.Subscribers = append(s.Subscribers, subscriber)
}

/**
* remove
* @param subscriber string
**/
func (s *Channel) remove(subscriber string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.Subscribers, func(item string) bool {
		return item == subscriber
	})
	if idx == -1 {
		return
	}

	s.Subscribers = append(s.Subscribers[:idx], s.Subscribers[idx+1:]...)
}

/**
* list
* @return []string - a snapshot copy of the current subscribers
**/
func (s *Channel) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]string, len(s.Subscribers))
	copy(result, s.Subscribers)
	return result
}

/**
* nextQueueTarget
* @return (string, bool) - the next subscriber to receive a queued message, round-robin
**/
func (s *Channel) nextQueueTarget() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.Subscribers)
	if n == 0 {
		return "", false
	}

	if s.Turn >= n {
		s.Turn = 0
	}
	target := s.Subscribers[s.Turn]
	s.Turn++
	return target, true
}

/**
* nextStackTarget
* @return (string, bool) - the next subscriber to receive a stacked message, reverse round-robin
**/
func (s *Channel) nextStackTarget() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.Subscribers)
	if n == 0 {
		return "", false
	}

	if s.Turn < 0 || s.Turn >= n {
		s.Turn = n - 1
	}
	target := s.Subscribers[s.Turn]
	s.Turn--
	return target, true
}
