package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/utility"
)

/**
* ConnectHttp connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
* @return *Client
* @return error
**/
func (h *Hub) ConnectHttp(w http.ResponseWriter, r *http.Request) (*Client, error) {
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	clientId := r.URL.Query().Get("clientId")
	name := r.URL.Query().Get("name")
	if clientId == "" {
		clientId = utility.UUID()
	}
	if name == "" {
		name = "Anonimo"
	}

	ctx := r.Context()
	clientId = middleware.ClientIdKey.String(ctx, clientId)
	name = middleware.NameKey.String(ctx, name)

	return h.connect(socket, clientId, name)
}
