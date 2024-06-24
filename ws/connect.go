package ws

import (
	"net/http"
	"net/url"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/token"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

// Connect to the ws server from the client
func ConnectWs(host, scheme, clientId, name string) (*websocket.Conn, error) {
	if scheme == "" {
		scheme = "ws"
	}

	path := strs.Format("/%s", scheme)

	u := url.URL{Scheme: scheme, Host: host, Path: path}
	header := make(http.Header)
	tk, err := token.Generate(clientId, name, "ws", "ws", "microservice", 0)
	if err != nil {
		return nil, err
	}

	tk = strs.Format(`Bearer %s`, tk)
	header.Add("Authorization", tk)
	result, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Connect to the server from the http client
func ConnectHttp(w http.ResponseWriter, r *http.Request) (*Client, error) {
	if servws == nil {
		return nil, logs.Log(ERR_NOT_WS_SERVICE)
	}

	ctx := r.Context()

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

	return servws.hub.connect(socket, clientId, name)
}
