package rt

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
)

var conn *ws.Client

/**
* LoadFrom
* @return erro
**/
func Load() error {
	if conn != nil {
		return nil
	}

	var err error
	params := &ws.ClientConfig{
		ClientId:  utility.UUID(),
		Name:      envar.GetStr("Real Time", "RT_NAME"),
		Url:       envar.GetStr("ws", "RT_URL"),
		Header:    http.Header{},
		Reconcect: envar.GetInt(3, "RT_RECONCECT"),
	}
	conn, err = ws.NewClient(params)
	if err != nil {
		return err
	}

	return nil
}

/**
* Close
**/
func Close() {
	if conn == nil {
		return
	}

	conn.Close()
}
