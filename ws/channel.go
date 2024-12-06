package ws

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"golang.org/x/exp/slices"
)

/**
* Channel
**/
type Channel struct {
	Name        string        `json:"name"`
	Subscribers []*Subscriber `json:"subscribers"`
	mutex       *sync.RWMutex
}

/**
* newChannel
* @param name string
* @return *Channel
**/
func newChannel(name string) *Channel {
	result := &Channel{
		Name:        name,
		Subscribers: []*Subscriber{},
		mutex:       &sync.RWMutex{},
	}

	return result
}

/**
* drain
**/
func (c *Channel) drain() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, client := range c.Subscribers {
		if client == nil {
			continue
		}

		delete(client.Channels, c.Name)
	}

	c.Subscribers = []*Subscriber{}
}

/**
* close
**/
func (c *Channel) close() {
	c.drain()
}

/**
* describe return the channel name
* @return et.Json
**/
func (c *Channel) describe(mode int) et.Json {
	if mode == 0 {
		subscribers := []et.Json{}
		for _, subscriber := range c.Subscribers {
			subscribers = append(subscribers, subscriber.From())
		}

		return et.Json{
			"name":        c.Name,
			"type":        "channel",
			"count":       len(c.Subscribers),
			"subscribers": subscribers,
		}
	}

	return et.Json{
		"name": c.Name,
		"type": "channel",
	}
}

/**
* Count return the number of subscribers
* @return int
**/
func (c *Channel) Count() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.Subscribers)
}

/**
* subscribe a client to channel
* @param client *Subscriber
**/
func (c *Channel) subscribe(client *Subscriber) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	idx := slices.IndexFunc(c.Subscribers, func(e *Subscriber) bool { return e.Id == client.Id })
	if idx != -1 {
		return
	}

	c.Subscribers = append(c.Subscribers, client)
	client.Channels[c.Name] = c
}

/**
* unsubscribe
* @param clientId string
**/
func (c *Channel) unsubscribe(client *Subscriber) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	idx := slices.IndexFunc(c.Subscribers, func(e *Subscriber) bool { return e.Id == client.Id })
	if idx == -1 {
		return
	}

	c.Subscribers = append(c.Subscribers[:idx], c.Subscribers[idx+1:]...)
	delete(client.Channels, c.Name)
}

/**
* broadcast
* @param msg Message
* @param ignored []string
* @return int
**/
func (c *Channel) broadcast(msg Message, ignored []string) int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := 0
	msg.Channel = c.Name
	for _, client := range c.Subscribers {
		if !slices.Contains(ignored, client.Id) {
			err := client.send(msg)
			if err != nil {
				logs.Alert(err)
			} else {
				result++
			}
		}
	}

	if result != 0 {
		logs.Logf(ServiceName, "Broadcast channel:%s sent to:%d", c.Name, result)
	}

	return result
}
