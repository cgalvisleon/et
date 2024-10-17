package ws

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"golang.org/x/exp/slices"
)

/**
* Channel
**/
type Channel struct {
	Name        string
	Group       map[string]int
	Subscribers []*Client
}

/**
* newChannel
* @param name string
* @return *Channel
**/
func newChannel(name string) *Channel {
	result := &Channel{
		Name:        strs.Lowcase(name),
		Group:       map[string]int{},
		Subscribers: []*Client{},
	}

	return result
}

/**
* Describe return the channel name
* @return et.Json
**/
func (ch *Channel) Describe() et.Json {
	result, err := et.Object(ch)
	if err != nil {
		logs.Error(err)
	}

	return result
}

/**
* Up return the channel name in uppercase
* @return string
**/
func (ch *Channel) Up() string {
	return strs.Uppcase(ch.Name)
}

/**
* Low return the channel name in lowercase
* @return string
**/
func (ch *Channel) Low() string {
	return strs.Lowcase(ch.Name)
}

/**
* Count return the number of subscribers
* @return int
**/
func (ch *Channel) Count() int {
	return len(ch.Subscribers)
}

/**
* NextTurn return the next subscriber
* @return *Client
**/
func (ch *Channel) NextTurn(queue string) *Client {
	n := ch.Count()
	if n == 0 {
		return nil
	}

	_, exist := ch.Group[queue]
	if !exist {
		ch.Group[queue] = 0
	}

	turn := ch.Group[queue]
	if turn >= n {
		turn = 0
		ch.Group[queue] = turn
	}

	result := ch.Subscribers[turn]
	ch.Group[queue]++

	return result
}

/**
* Subscribe a client to channel
* @param client *Client
**/
func (ch *Channel) Subscribe(client *Client) {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == client.Id })
	if idx == -1 {
		ch.Subscribers = append(ch.Subscribers, client)
	}
}

/**
* QueueSubscribe a client to channel
* @param client *Client
**/
func (ch *Channel) QueueSubscribe(client *Client, queue string) {
	_, exist := ch.Group[queue]
	if !exist {
		ch.Group[queue] = 0
	}

	ch.Subscribe(client)
}

/**
* Broadcast a message to all subscribers
* @param message []byte
**/
func (ch *Channel) Broadcast(message []byte) {
	for _, client := range ch.Subscribers {
		client.outbound <- message
	}
}

/**
* Unsubcribe a client from channel
* @param clientId string
* @return error
**/
func (ch *Channel) Unsubcribe(clientId string) error {
	idx := slices.IndexFunc(ch.Subscribers, func(e *Client) bool { return e.Id == clientId })
	if idx == -1 {
		return logs.Alertm(ERR_CLIENT_NOT_FOUND)
	}

	ch.Subscribers = append(ch.Subscribers[:idx], ch.Subscribers[idx+1:]...)

	return nil
}
