package realtime

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/ws"
)

const ServiceName = "Real Time"

var conn *ws.Client

/**
* Load
* @return erro
**/
func Load(name string) (*ws.Client, error) {
	if conn != nil {
		return conn, nil
	}

	url := envar.GetStr("", "RT_URL")
	if url == "" {
		return nil, logs.Alertm(MSG_RT_URL_REQUIRED)
	}

	username := envar.GetStr("", "WS_USERNAME")
	if username == "" {
		return nil, logs.Alertm(ERR_WS_USERNAME_REQUIRED)
	}

	password := envar.GetStr("", "WS_PASSWORD")
	if password == "" {
		return nil, logs.Alertm(ERR_WS_PASSWORD_REQUIRED)
	}

	client, err := ws.Login(&ws.ClientConfig{
		ClientId:  reg.Id("RealTime"),
		Name:      name,
		Url:       url,
		Reconnect: envar.GetInt(3, "RT_RECONCECT"),
		Header: http.Header{
			"username": []string{username},
			"password": []string{password},
		},
	})
	if err != nil {
		return nil, err
	}

	conn = client

	logs.Logf(ServiceName, `Connected host:%s`, url)

	return conn, nil
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
