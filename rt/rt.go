package rt

import (
	"net/url"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	"github.com/gorilla/websocket"
)

var conn *ClientWS

type ClientWS struct {
	socket    *websocket.Conn
	Host      string
	ClientId  string
	Name      string
	channels  map[string]func(ws.Message)
	connected bool
}

/**
* ConnectWs connect to the server using the websocket
* @param host string
* @param scheme string
* @param clientId string
* @param name string
* @return *websocket.Conn
* @return error
**/
func Load() error {
	if conn != nil {
		return nil
	}

	name := envar.GetStr("Client", "RT_NAME")
	host := envar.GetStr("localhost", "RT_HOST")
	scheme := envar.GetStr("ws", "RT_SCHEME")

	id := utility.UUID()
	path := strs.Format("/%s", scheme)
	params := url.Values{}
	params.Add("clientId", id)
	params.Add("name", name)
	serverURL := url.URL{Scheme: scheme, Host: host, Path: path, RawQuery: params.Encode()}
	wsocket, _, err := websocket.DefaultDialer.Dial(serverURL.String(), nil)
	if err != nil {
		return err
	}

	conn = &ClientWS{
		socket:   wsocket,
		Host:     host,
		ClientId: id,
		Name:     name,
		channels: make(map[string]func(ws.Message)),
	}

	go conn.read()

	logs.Logf("Real time", "Connected clientId:%s name:%s host:%s%s", id, name, host, path)

	return nil
}

/**
* Close
**/
func Close() {
	if conn != nil {
		conn.socket.Close()
	}
}
