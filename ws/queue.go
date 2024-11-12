package ws

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"golang.org/x/exp/slices"
)

type Queue struct {
	Name        string        `json:"name"`
	Queue       string        `json:"queue"`
	Turn        int           `json:"turn"`
	Subscribers []*Subscriber `json:"subscribers"`
	mutex       *sync.RWMutex
}

/**
* newQueue
* @param name string
* @return *Queue
**/
func newQueue(name, queue string) *Queue {
	result := &Queue{
		Name:        name,
		Queue:       queue,
		Turn:        0,
		Subscribers: []*Subscriber{},
		mutex:       &sync.RWMutex{},
	}

	return result
}

/**
* Count return the number of subscribers
* @return int
**/
func (c *Queue) Count() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.Subscribers)
}

/**
* nextTurn return the next subscriber
* @return *Subscriber
**/
func (c *Queue) nextTurn() *Subscriber {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	n := len(c.Subscribers)
	if n == 0 {
		return nil
	}

	if c.Turn >= n {
		c.Turn = 0
	}

	result := c.Subscribers[c.Turn]
	c.Turn++

	return result
}

func (c *Queue) drain() {
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
func (c *Queue) close() {
	c.drain()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Subscribers = nil
}

/**
* describe return the channel name
* @return et.Json
**/
func (c *Queue) describe(mode int) et.Json {
	if mode == 0 {
		subscribers := []et.Json{}
		for _, subscriber := range c.Subscribers {
			subscribers = append(subscribers, subscriber.From())
		}

		return et.Json{
			"name":        c.Name,
			"queue":       c.Queue,
			"turn":        c.Turn,
			"type":        "queue",
			"subscribers": subscribers,
		}
	}

	return et.Json{
		"name":  c.Name,
		"queue": c.Queue,
		"turn":  c.Turn,
		"type":  "queue",
	}
}

/**
* queueSubscribe a client to channel
* @param client *Subscriber
**/
func (c *Queue) subscribe(client *Subscriber) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	idx := slices.IndexFunc(c.Subscribers, func(e *Subscriber) bool { return e.Id == client.Id })
	if idx != -1 {
		return
	}

	c.Subscribers = append(c.Subscribers, client)
	client.Queue[c.Name] = c
}

/**
* unsubscribe
* @param clientId string
**/
func (c *Queue) unsubscribe(client *Subscriber) {
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
func (c *Queue) broadcast(msg Message, ignored []string) int {
	result := 0
	msg.Channel = c.Name
	client := c.nextTurn()
	if client != nil && !slices.Contains(ignored, client.Id) {
		err := client.sendMessage(msg)
		if err != nil {
			logs.Alert(err)
		} else {
			result++
		}
	}

	return result
}
