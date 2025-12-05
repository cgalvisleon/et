package event

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

/**
* Connect to a host
* @param host, user, password string
* @return *Conn, error
**/
func ConnectTo(host, user, password string) (*Conn, error) {
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "nats_host")
	}

	opts := []nats.Option{
		nats.UserInfo(user, password),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logs.Logf("NATS", `Reconnected host:%s`, host)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logs.Logf("NATS", `Closed host:%s`, host)
		}),
	}
	client, err := nats.Connect(host, opts...)
	if err != nil {
		return nil, err
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return &Conn{
		id:              utility.UUID(),
		Conn:            client,
		eventCreatedSub: map[string]*nats.Subscription{},
		mutex:           &sync.RWMutex{},
		storage:         []string{},
	}, nil
}
