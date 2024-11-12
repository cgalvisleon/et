package ws

import (
	"net/http"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/race"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

type TypeNode int

const (
	NodeWorker TypeNode = iota
	NodeMaster
)

func (t TypeNode) String() string {
	switch t {
	case NodeMaster:
		return "master"
	case NodeWorker:
		return "worker"
	}

	return "unknown"
}

func (t TypeNode) ToJson() et.Json {
	return et.Json{
		"id":   t,
		"name": t.String(),
	}
}

type Adapter struct {
	Client
	typeNode TypeNode
}

var adapter *Adapter

type AdapterConfig struct {
	Url       string
	TypeNode  TypeNode
	Reconcect int
	Header    http.Header
}

func clusterChannel(channel string) string {
	result := strs.Format(`cluster/%s`, channel)
	return utility.ToBase64(result)
}

/**
* InitMaster
* @return *Hub
**/
func (h *Hub) InitMaster() {
	if adapter != nil {
		return
	}

	adapter = &Adapter{
		Client: Client{
			Channels:  make(map[string]func(Message)),
			Attempts:  race.NewValue(0),
			Connected: race.NewValue(false),
			mutex:     &sync.Mutex{},
			clientId:  h.Id,
			name:      h.Name,
			url:       "",
			header:    http.Header{},
			reconcect: 3,
		},
		typeNode: NodeMaster,
	}
}

/**
* Join
* @param config *ClientConfig
**/
func (h *Hub) Join(config AdapterConfig) error {
	if adapter != nil {
		return nil
	}

	adapter = &Adapter{
		Client: Client{
			Channels:  make(map[string]func(Message)),
			Attempts:  race.NewValue(0),
			Connected: race.NewValue(false),
			mutex:     &sync.Mutex{},
			clientId:  h.Id,
			name:      h.Name,
			url:       config.Url,
			header:    config.Header,
			reconcect: config.Reconcect,
		},
		typeNode: config.TypeNode,
	}
	err := adapter.Connect()
	if err != nil {
		return err
	}

	adapter.SetReconnectCallback(func(c *Client) {
		logs.Debug("ReconnectCallback:", "Hola")
	})

	adapter.SetDirectMessage(func(msg Message) {
		logs.Debug("DirectMessage:", msg.ToString())
	})

	return nil
}

/**
* Live
**/
func (h *Hub) Live() {
	if adapter == nil {
		return
	}

	adapter.Close()
}

/**
* ClusterSubscribed
* @param channel string
**/
func (h *Hub) ClusterSubscribed(channel string) {
	if adapter == nil {
		return
	}

	if !adapter.Connected.Bool() {
		return
	}

	channel = clusterChannel(channel)
	adapter.Subscribe(channel, func(msg Message) {
		if msg.tp == TpDirect {
			h.SendMessage(msg.Id, msg)
		} else {
			h.Publish(msg.Channel, msg.Queue, msg, msg.Ignored, msg.From)
		}
	})
}

/**
* ClusterUnSubscribed
* @param sub channel string
**/
func (h *Hub) ClusterUnSubscribed(channel string) {
	if adapter == nil {
		return
	}

	if !adapter.Connected.Bool() {
		return
	}

	channel = clusterChannel(channel)
	adapter.Unsubscribe(channel)
}

/**
* ClusterUnSubscribed
* @param sub channel string
**/
func (h *Hub) ClusterPublish(channel string, msg Message) {
	if adapter == nil {
		return
	}

	if !adapter.Connected.Bool() {
		return
	}

	channel = clusterChannel(channel)
	adapter.Publish(channel, msg)
}
