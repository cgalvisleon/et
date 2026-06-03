package event

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

/**
* connectTo to a host
* @param host, user, password string
* @return *Conn, error
**/
func connectTo(host, user, password string) (*Conn, error) {
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "host")
	}

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

	logs.Logf(packageName, `Connected host:%s`, host)

	return &Conn{
		id:     utility.UUID(),
		Conn:   client,
		events: map[string]*nats.Subscription{},
		mutex:  &sync.RWMutex{},
	}, nil
}
