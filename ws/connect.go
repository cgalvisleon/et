package ws

import (
	"net/http"
	"net/url"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/token"
	"github.com/cgalvisleon/et/utility"
	"github.com/gorilla/websocket"
)

/**
* ConnectWs connect to the server using the websocket
* @param host string
* @param scheme string
* @param clientId string
* @param name string
* @return *websocket.Conn
* @return error
**/
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

/**
* ConnectHttp connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
* @return *Client
* @return error
**/
func ConnectHttp(w http.ResponseWriter, r *http.Request) (*Client, error) {
	if servws == nil {
		return nil, logs.Log(ERR_NOT_WS_SERVICE)
	}

	ctx := r.Context()

	var clientId string
	val := ctx.Value(middleware.ClientIdKey)

	logs.Debug("WS ConnectHttp: clientId:", val)

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
