package event

import (
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

const PackageName = "event"

/**
* asyncMsg holds a channel name and payload for non-blocking publish operations.
**/
type asyncMsg struct {
	channel string
	data    et.Json
}

// asyncPublishBufSize is the capacity of the fire-and-forget publish channel.
const asyncPublishBufSize = 256

var (
	conn           *Conn
	oS             = ""
	hostName       string
	loadMu         sync.Mutex
	// asyncPublishCh is consumed by a single background worker started in init().
	asyncPublishCh = make(chan asyncMsg, asyncPublishBufSize)
)

func init() {
	oS = runtime.GOOS
	hostName, _ = os.Hostname()
	go func() {
		for m := range asyncPublishCh {
			publish(m.channel, m.data)
		}
	}()
}

type Conn struct {
	*nats.Conn
	id     string
	events map[string]*nats.Subscription
	mutex  *sync.RWMutex
}

/**
* Load
* @return error
**/
func Load() error {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, oS) {
		return nil
	}

	loadMu.Lock()
	defer loadMu.Unlock()

	if conn != nil {
		return nil
	}

	err := envar.Validate([]string{
		"NATS_HOST",
	})
	if err != nil {
		return err
	}

	host := envar.GetStr("NATS_HOST", "")
	user := envar.GetStr("NATS_USER", "")
	password := envar.GetStr("NATS_PASSWORD", "")
	conn, err = ConnectTo(host, user, password)
	if err != nil {
		return err
	}

	return nil
}

/**
* Close unsubscribes all active subscriptions and closes the NATS connection.
**/
func Close() {
	if conn == nil {
		return
	}

	for _, sub := range conn.events {
		sub.Unsubscribe()
	}

	conn.Close()

	logs.Log(PackageName, `Disconnect...`)
}

/**
* Id
* @return string
**/
func Id() string {
	return conn.id
}

/**
* HealthCheck
* @return bool
**/
func HealthCheck() bool {
	if conn == nil {
		return false
	}

	return conn.IsConnected()
}
