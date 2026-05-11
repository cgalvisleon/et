package cache

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

/**
* serialize
* @return []byte, error
**/
func (s Message) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s Message) ToJson() (et.Json, error) {
	definition, err := s.serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* ToString
* @return string
**/
func (s Message) ToString() string {
	j, err := s.ToJson()
	if err != nil {
		return et.Json{}.ToString()
	}

	return j.ToString()
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
* Sub: Subscribes to a channel and dispatches messages to f in a goroutine.
* A second call for the same channel is a no-op while the subscription is active.
* @param channel string
* @param f func(*redis.Message)
**/
func (s *Conn) Sub(channel string, f func(*redis.Message)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.channels[channel] != nil {
		return
	}

	sub := s.Subscribe(s.ctx, channel)
	s.channels[channel] = sub

	go func() {
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
* Unsub: Closes the active subscription for channel and removes it from the registry.
* @param channel string
* @return error
**/
func (s *Conn) Unsub(channel string) error {
	s.mutex.Lock()
	sub, ok := s.channels[channel]
	if ok {
		delete(s.channels, channel)
	}
	s.mutex.Unlock()

	if !ok || sub == nil {
		return nil
	}

	return sub.Close()
}
