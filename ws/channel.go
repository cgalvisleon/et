package ws

import (
	"golang.org/x/exp/slices"
)

// Channel is a hub websocket channel
type Channel struct {
	hub         *Hub
	Name        string
	Subscribers []*Client
}

// NewChanel create a new channel
func NewChanel(hub *Hub, name string) *Channel {
	result := &Channel{
		hub:         hub,
		Name:        name,
		Subscribers: []*Client{},
	}
	hub.channels = append(hub.channels, result)

	return result
}

// Unsubcribe a client from channel
func (ch *Channel) Unsubcribe(clientId string) {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == clientId })
	if idx != -1 {
		ch.Subscribers = append(ch.Subscribers[:idx], ch.Subscribers[idx+1:]...)
	}

	count := len(ch.Subscribers)
	if count == 0 {
		hub := ch.hub
		idxC := slices.IndexFunc(hub.channels, func(e *Channel) bool { return e.Name == ch.Name })
		if idxC != -1 {
			hub.channels = append(hub.channels[:idxC], hub.channels[idxC+1:]...)
		}
	}
}
