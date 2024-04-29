package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

func Connect(w http.ResponseWriter, r *http.Request) (*Client, error) {
	if conn == nil {
		return nil, logs.Log(ERR_NOT_WS_SERVICE)
	}

	var clientId string
	var userName string
	ctx := r.Context()
	val := ctx.Value("clientId")
	if val == nil {
		clientId = utility.UUID()
	} else {
		clientId = val.(string)
	}

	val = ctx.Value("name")
	if val == nil {
		userName = "Anonimo"
	} else {
		userName = val.(string)
	}

	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return conn.hub.connect(socket, clientId, userName)
}

/*
func Broadcast(message interface{}, ignoreId string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_WS_SERVICE)
	}

	conn.Broadcast(message, ignoreId)

	return nil
}

func Publish(channel string, message interface{}, ignoreId string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_WS_SERVICE)
	}

	conn.Publish(channel, message, ignoreId)

	return nil
}

func SendMessage(clientId, channel string, message interface{}) (bool, error) {
	if conn == nil {
		return false, logs.Log(ERR_NOT_WS_SERVICE)
	}

	result := conn.SendMessage(clientId, channel, message)

	return result, nil
}

func Subscribe(clientId string, channel string) bool {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return false
	}

	return conn.Subscribe(clientId, channel)
}

func Unsubscribe(clientId string, channel string) bool {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return false
	}

	return conn.Unsubscribe(clientId, channel)
}

func GetChannels() []*Channel {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return []*Channel{}
	}

	return conn.channels
}

func GetClients() []*Client {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return []*Client{}
	}

	return conn.clients
}

func GetSubscribers(channel string) []*Client {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return []*Client{}
	}

	return conn.GetSubscribers(channel)
}
*/
