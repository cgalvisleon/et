package event

import (
	"os"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

/**
* Conn: The NATS connection of the package plus its id, one subscription per channel (Subscribe, Queue, Stack) and
* one per key (SubscribeWith, QueueWith, StackWith), kept apart so a key never takes the place of a channel.
**/
type Conn struct {
	*nats.Conn
	id     string
	events map[string]*nats.Subscription
	keyed  map[string]*nats.Subscription
	mutex  *sync.RWMutex
}

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
	packageName    = "event"
	conn           *Conn
	oS             = ""
	hostName       string
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

/**
* New: Opens the NATS connection from NATS_HOST, NATS_USER and NATS_PASSWORD.
* @return *Conn, error
**/
func New() (*Conn, error) {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, oS) {
		return nil, logs.Alertf(MSG_UNSUPPORTED_OS, oS)
	}

	host := envar.GetStr("NATS_HOST", "")
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "host")
	}

	user := envar.GetStr("NATS_USER", "")
	password := envar.GetStr("NATS_PASSWORD", "")
	opts := []nats.Option{
		nats.UserInfo(user, password),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logs.Logf(packageName, `Reconnected host:%s`, host)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logs.Logf(packageName, `Closed host:%s`, host)
		}),
	}
	client, err := nats.Connect(host, opts...)
	if err != nil {
		return nil, err
	}

	logs.Logf("nats", `Connected host:%s`, host)

	return &Conn{
		id:     utility.UUID(),
		Conn:   client,
		events: map[string]*nats.Subscription{},
		keyed:  map[string]*nats.Subscription{},
		mutex:  &sync.RWMutex{},
	}, nil
}

/**
* Close unsubscribes all active subscriptions (by channel and by key) and closes the NATS connection.
**/
func (s *Conn) Close() {
	s.mutex.Lock()
	for _, sub := range s.events {
		sub.Unsubscribe()
	}
	for _, sub := range s.keyed {
		sub.Unsubscribe()
	}
	s.events = map[string]*nats.Subscription{}
	s.keyed = map[string]*nats.Subscription{}
	s.mutex.Unlock()

	s.Conn.Close()

	logs.Log(packageName, `Disconnect...`)
}

/**
* HealthCheck
* @return bool
**/
func (s *Conn) HealthCheck() bool {
	return s.IsConnected()
}
