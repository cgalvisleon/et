package cache

import (
	"github.com/cgalvisleon/et/et"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func (s Message) ToJson() (et.Json, error) {
	result, err := et.Object(s)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* Pub
* @param channel string
* @param message interface{}
* @return error
**/
func (s *Conn) Pub(channel string, message []byte) error {
	err := s.Publish(s.ctx, channel, message).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* Sub
* @param channel string
* @param f func(interface{})
**/
func (s *Conn) Sub(channel string, f func(*redis.Message)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.channels[channel] {
		return
	}

	s.channels[channel] = true

	go func() {
		sub := s.Subscribe(s.ctx, channel)
		defer sub.Close()

		ch := sub.Channel()

		for msg := range ch {
			f(msg)
		}

		s.mutex.Lock()
		delete(s.channels, channel)
		s.mutex.Unlock()
	}()
}

/**
* Unsub
* @param channel string
* @return error
**/
func (s *Conn) Unsub(channel string) error {
	sub := s.Subscribe(s.ctx, channel)
	defer sub.Close()

	err := sub.Unsubscribe(s.ctx, channel)
	if err != nil {
		return err
	}

	return nil
}
