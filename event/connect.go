package event

import (
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
	"github.com/nats-io/nats.go"
)

func ConnectTo(host, user, password string) (*Conn, error) {
	if !utility.ValidStr(host, 0, []string{}) {
		return nil, logs.Alertf(msg.MSG_ATRIB_REQUIRED, "nats_host")
	}

	client, err := nats.Connect(host, nats.UserInfo(user, password))
	if err != nil {
		return nil, err
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return &Conn{
		Conn:            client,
		_id:             utility.UUID(),
		eventCreatedSub: map[string]*nats.Subscription{},
		mutex:           &sync.RWMutex{},
	}, nil
}

/**
* Connect to a host
* @return *Conn, error
**/
func connect() (*Conn, error) {
	host := envar.GetStr("", "NATS_HOST")
	if host == "" {
		return nil, logs.Alertf(msg.ERR_ENV_REQUIRED, "NATS_HOST")
	}

	user := envar.GetStr("", "NATS_USER")
	password := envar.GetStr("", "NATS_PASSWORD")

	return ConnectTo(host, user, password)
}
