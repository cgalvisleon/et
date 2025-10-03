package event

import (
	"encoding/json"
	"runtime"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/nats-io/nats.go"
)

const PackageName = "event"

var (
	conn *Conn
	os   = ""
)

func init() {
	os = runtime.GOOS
}

type Conn struct {
	*nats.Conn
	id              string
	eventCreatedSub map[string]*nats.Subscription
	mutex           *sync.RWMutex
	storage         []string
}

/**
* Save
* @return error
**/
func (s *Conn) save() error {
	bt, err := json.Marshal(s.storage)
	if err != nil {
		return err
	}

	cache.Set("event:storage", string(bt), 0)

	return nil
}

/**
* Storage
* @return []string, error
**/
func (s *Conn) load() error {
	bt, err := json.Marshal(s.storage)
	if err != nil {
		return err
	}

	scr, err := cache.Get("event:storage", string(bt))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(scr), &s.storage)
	if err != nil {
		return err
	}

	return nil
}

/**
* add
* @return error
**/
func (s *Conn) add(event string) (bool, error) {
	err := s.load()
	if err != nil {
		return false, err
	}

	idx := slices.IndexFunc(s.storage, func(e string) bool { return e == event })
	if idx == -1 {
		s.storage = append(s.storage, event)
	}

	return idx == -1, s.save()
}

/**
* Reset
* @return error
**/
func (s *Conn) Reset() error {
	s.storage = []string{}

	return s.save()
}

/**
* Remove
* @return error
**/
func (s *Conn) Remove(event string) (bool, error) {
	err := s.load()
	if err != nil {
		return false, err
	}

	idx := slices.IndexFunc(s.storage, func(e string) bool { return e == event })
	if idx == -1 {
		return false, nil
	}

	s.storage = slices.Delete(s.storage, idx, 1)

	return true, s.save()
}

/**
* Load
* @return error
**/
func Load() error {
	if !slices.Contains([]string{"linux", "darwin", "windows"}, os) {
		return nil
	}

	if conn != nil {
		return nil
	}

	err := config.Validate([]string{
		"NATS_HOST",
	})
	if err != nil {
		return err
	}

	host := config.GetStr("NATS_HOST", "")
	user := config.GetStr("NATS_USER", "")
	password := config.GetStr("NATS_PASSWORD", "")
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

/**
* Id
* @return string
**/
func Id() string {
	return conn.id
}

/**
* Events
* @return []string
**/
func Events() []string {
	if conn == nil {
		return []string{}
	}

	err := conn.load()
	if err != nil {
		return []string{}
	}

	return conn.storage
}

/**
* Reset
* @return error
**/
func Reset() error {
	if conn == nil {
		return nil
	}

	return conn.Reset()
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
