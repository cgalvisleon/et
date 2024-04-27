package ws

import (
	"github.com/cgalvisleon/et/strs"
	"golang.org/x/exp/slices"
)

// Channel is a hub websocket channel
type Channel struct {
	Name        string
	Subscribers []*Client
}

// NewChannel create a new channel
func newChannel(name string) *Channel {
	result := &Channel{
		Name:        strs.Lowcase(name),
		Subscribers: []*Client{},
	}

	return result
}

// Up return the channel name in uppercase
func (ch *Channel) Up() string {
	return strs.Uppcase(ch.Name)
}

// Low return the channel name in lowercase
func (ch *Channel) Low() string {
	return strs.Lowcase(ch.Name)
}

// Count return the number of subscribers in channel
func (ch *Channel) Count() int {
	return len(ch.Subscribers)
}

// Subscribe a client to channel
func (ch *Channel) Subscribe(client *Client) {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == client.Id })
	if idx == -1 {
		ch.Subscribers = append(ch.Subscribers, client)
	}
}

// Unsubcribe a client from channel
func (ch *Channel) Unsubcribe(clientId string) bool {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == clientId })
	if idx != -1 {
		ch.Subscribers = append(ch.Subscribers[:idx], ch.Subscribers[idx+1:]...)
		return true
	}

	return false
}
