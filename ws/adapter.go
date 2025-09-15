package ws

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

type Adapter interface {
	ConnectTo(hub *Hub, params et.Json) error
	Close()
	Subscribed(channel string)
	UnSubscribed(channel string)
	Publish(channel string, msg Message) error
}

func clusterChannel(channel string) string {
	result := fmt.Sprintf(`cluster/%s`, channel)
	result = utility.ToBase64(result)

	return result
}
