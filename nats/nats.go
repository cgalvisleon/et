package nats

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
)

var (
	conn *PubSub
)

// Load a new client websocket connection
func Load(cache cache.Cache, clientId, name string, reciveFn func(pubsub.Message)) (*PubSub, error) {
	if conn != nil {
		return conn, nil
	}

	host := envar.GetStr("", "NATS_HOST")
	if host == "" {
		return nil, logs.Alertm("NATS_HOST not found")
	}

	conn := NewPubSub(host, clientId, name, cache, reciveFn)

	return conn, nil
}

// Close the connection
func Close() {
	if conn.conn != nil {
		conn.conn.Close()
	}
}
