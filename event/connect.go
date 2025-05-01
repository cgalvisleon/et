package event

import (
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
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

	client, err := nats.Connect(host, nats.UserInfo(user, password))
	if err != nil {
		return nil, err
	}

	logs.Logf("NATS", `Connected host:%s`, host)

	return &Conn{
		Id:              reg.GenId("nats"),
		Conn:            client,
		eventCreatedSub: map[string]*nats.Subscription{},
		mutex:           &sync.RWMutex{},
	}, nil
}
