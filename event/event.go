package event

import (
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

const PackageName = "event"

var (
	conn     *Conn
	oS       = ""
	hostName string
)

func init() {
	oS = runtime.GOOS
	hostName, _ = os.Hostname()
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
* Close the connection to the service pubsub
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
