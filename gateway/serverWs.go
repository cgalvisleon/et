package gateway

import (
	"net/http"

	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/ws"
)

// Handler for websocket
func wsConnect(w http.ResponseWriter, r *http.Request) {
	_, err := ws.ConnectHttp(w, r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}
}
