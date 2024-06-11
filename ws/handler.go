package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

// Connect to the server from the http client
func Connect(w http.ResponseWriter, r *http.Request) (*Client, error) {
	if conn == nil {
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

	return conn.hub.connect(socket, clientId, name)
}
