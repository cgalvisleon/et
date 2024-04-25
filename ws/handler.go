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

	val = ctx.Value("username")
	if val == nil {
		userName = "Anonimo"
	} else {
		userName = val.(string)
	}

	idxC := conn.hub.indexClient(clientId)
	if idxC != -1 {
		return conn.hub.clients[idxC], nil
	}

	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return conn.hub.connect(socket, clientId, userName)
}

func Broadcast(message interface{}, ignoreId string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_WS_SERVICE)
	}

	conn.hub.Broadcast(message, ignoreId)

	return nil
}

func Publish(channel string, message interface{}, ignoreId string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_WS_SERVICE)
	}

	conn.hub.Publish(channel, message, ignoreId)

	return nil
}

func SendMessage(clientId, channel string, message interface{}) (bool, error) {
	if conn == nil {
		return false, logs.Log(ERR_NOT_WS_SERVICE)
	}

	result := conn.hub.SendMessage(clientId, channel, message)

	return result, nil
}

func Subscribe(clientId string, channel string) bool {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return false
	}

	return conn.hub.Subscribe(clientId, channel)
}

func Unsubscribe(clientId string, channel string) bool {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return false
	}

	return conn.hub.Unsubscribe(clientId, channel)
}

func GetChannels() []*Channel {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return []*Channel{}
	}

	return conn.hub.channels
}

func GetClients() []*Client {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return []*Client{}
	}

	return conn.hub.clients
}

func GetSubscribers(channel string) []*Client {
	if conn == nil {
		logs.Log(ERR_NOT_WS_SERVICE)
		return []*Client{}
	}

	return conn.hub.GetSubscribers(channel)
}
