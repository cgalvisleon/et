package ws

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"golang.org/x/exp/slices"
)

type TpBroadcast int

const (
	TpAll TpBroadcast = iota
	TpRoundRobin
	TpDirect
	TpCommand
)

// Channel is a hub websocket channel
type Channel struct {
	Name        string
	TpBroadcast TpBroadcast
	Subscribers []*Client
	turn        int
}

// String return the string representation of the broadcast type
func (tp TpBroadcast) String() string {
	switch tp {
	case TpRoundRobin:
		return "roundrobin"
	default:
		return "all"
	}
}

// NewChannel create a new channel
func newChannel(name string) *Channel {
	result := &Channel{
		Name:        strs.Lowcase(name),
		TpBroadcast: TpAll,
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

func (ch *Channel) NextTurn() *Client {
	n := ch.Count()
	if n == 0 {
		return nil
	}

	if ch.turn >= n {
		ch.turn = 0
	}

	result := ch.Subscribers[ch.turn]
	ch.turn++

	return result
}

// Subscribe a client to channel
func (ch *Channel) Subscribe(client *Client) {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == client.Id })
	if idx == -1 {
		ch.Subscribers = append(ch.Subscribers, client)
	}
}

// Unsubcribe a client from channel
func (ch *Channel) Unsubcribe(clientId string) error {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == clientId })
	if idx == -1 {
		return logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	ch.Subscribers = append(ch.Subscribers[:idx], ch.Subscribers[idx+1:]...)

	return nil
}
