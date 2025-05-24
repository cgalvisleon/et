package event

import (
	"sync"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

const PackageName = "event"

var conn *Conn

type Conn struct {
	*nats.Conn
	id              string
	eventCreatedSub map[string]*nats.Subscription
	mutex           *sync.RWMutex
}

/**
* Load
* @return error
**/
func Load() error {
	if conn != nil {
		return nil
	}

	err := config.Validate([]string{
		"NATS_HOST",
	})
	if err != nil {
		return err
	}

	host := config.String("NATS_HOST", "")
	user := config.String("NATS_USER", "")
	password := config.String("NATS_PASSWORD", "")
	conn, err = ConnectTo(host, user, password)
	if err != nil {
		return err
	}

	return nil
}

/**
* Close the connection to the service pubsub
**/
func Close() {
	if conn == nil {
		return
	}

	for _, sub := range conn.eventCreatedSub {
		sub.Unsubscribe()
	}

	conn.Close()

	logs.Log(PackageName, `Disconnect...`)
}

func Id() string {
	return conn.id
}
