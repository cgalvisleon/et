package ws

import (
	"net/http"
	"net/url"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

// Create a Hub and run it
func connectHub() *Hub {
	hub := NewHub()
	go hub.Run()

	return hub
}

// Connect to the server from the client
func connectWs(host, scheme string) (*websocket.Conn, error) {
	if scheme == "" {
		scheme = "ws"
	}

	path := strs.Format("/%s", scheme)

	u := url.URL{Scheme: scheme, Host: host, Path: path}
	result, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Connect to the server from the http client
func Connect(w http.ResponseWriter, r *http.Request) (*Client, error) {
	if conn == nil {
		return nil, logs.Log(ERR_NOT_WS_SERVICE)
	}

	ctx := r.Context()
	logs.Debug(ctx)

	var clientId string
	val := ctx.Value("clientId")
	if val == nil {
		clientId = utility.UUID()
	} else {
		clientId = val.(string)
	}

	var name string
	val = ctx.Value("name")
	if val == nil {
		name = "Anonimo"
	} else {
		name = val.(string)
	}

	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return conn.hub.connect(socket, clientId, name)
}
