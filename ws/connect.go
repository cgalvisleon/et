package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/token"
	"github.com/cgalvisleon/et/utility"
)

/**
* HttpConnect connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
	}

	// Identify of the client
	clientId := r.URL.Query().Get("clientId")
	if clientId == "" {
		clientId = utility.UUID()
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Anonimo"
	}

	ctx := r.Context()
	clientId = token.ClientIdKey.String(ctx, clientId)
	name = token.NameKey.String(ctx, name)

	_, err = h.connect(conn, clientId, name)
	if err != nil {
		logs.Alert(err)
	}
}

/**
* HttpStream connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpStream(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
	}

	// Identify of the client
	clientId := r.URL.Query().Get("clientId")
	if clientId == "" {
		clientId = utility.UUID()
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Anonimo"
	}

	ctx := r.Context()
	clientId = token.ClientIdKey.String(ctx, clientId)
	name = token.NameKey.String(ctx, name)

	_, err = h.streaming(conn, clientId, name)
	if err != nil {
		logs.Alert(err)
	}
}
