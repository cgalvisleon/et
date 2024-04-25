package gateway

import (
	"net/http"

	"github.com/cgalvisleon/elvis/response"
	"github.com/cgalvisleon/elvis/ws"
)

// Handler for websocket
func wsConnect(w http.ResponseWriter, r *http.Request) {
	_, err := ws.Connect(w, r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}
}
