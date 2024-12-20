package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/sesion"
	"github.com/cgalvisleon/et/utility"
)

/**
* HttpCluster connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpLogin(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("username")
	password := r.Header.Get("password")

	ws_username := envar.GetStr("", "WS_USERNAME")
	if !utility.ValidStr(ws_username, 0, []string{}) {
		response.HTTPError(w, r, http.StatusInternalServerError, mistake.New(ERR_NOT_SIGNATURE).Error())
	}

	ws_password := envar.GetStr("", "WS_PASSWORD")
	if !utility.ValidStr(ws_password, 0, []string{}) {
		response.HTTPError(w, r, http.StatusInternalServerError, mistake.New(ERR_NOT_SIGNATURE).Error())
	}

	if username != ws_username || password != ws_password {
		response.HTTPError(w, r, http.StatusInternalServerError, mistake.New(ERR_NOT_SIGNATURE).Error())
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
	}

	clientId := utility.UUID()
	name := utility.GetOTP(6)
	_, err = h.connect(conn, clientId, name)
	if err != nil {
		logs.Alert(err)
	}
}

/**
* HttpConnect connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (h *Hub) HttpConnect(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	clientId := query.ValStr("", "clientid")
	if clientId == "" {
		clientId = utility.UUID()
	}

	name := query.ValStr("", "name")
	if name == "" {
		name = "Anonimo"
	}

	ctx := r.Context()
	clientId = sesion.ClientIdKey.String(ctx, clientId)
	name = sesion.NameKey.String(ctx, name)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
	}

	_, err = h.connect(conn, clientId, name)
	if err != nil {
		logs.Alert(err)
	}
}
