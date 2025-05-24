package realtime

import (
	"net/http"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
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

	err := config.Validate([]string{
		"RT_URL",
		"RT_USERNAME",
		"RT_PASSWORD",
	})
	if err != nil {
		return nil, err
	}

	url := config.String("RT_URL", "")
	username := config.String("RT_USERNAME", "")
	password := config.String("RT_PASSWORD", "")
	reconnect := config.Int("RT_RECONNECT", 3)
	client, err := ws.Login(&ws.ClientConfig{
		ClientId:  utility.UUID(),
		Name:      name,
		Url:       url,
		Reconnect: reconnect,
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
