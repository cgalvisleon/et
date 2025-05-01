package ws

import (
	"net/http"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

/**
* ServerHttp
* @params port int, username string, password string
* @return *Hub
**/
func ServerHttp(port int, username, password string) *Hub {
	config.Set("WS_USERNAME", username)
	config.Set("WS_PASSWORD", password)
	result := NewHub()
	result.Start()
	go startHttp(result, port)
	time.Sleep(1 * time.Second)

	return result
}

func startHttp(hub *Hub, port int) {
	http.HandleFunc("/ws", hub.HttpConnect)
	http.HandleFunc("/ws/describe", hub.HttpDescribe)
	http.HandleFunc("/ws/publications", hub.HttpGetPublications)
	http.HandleFunc("/ws/subscribers", hub.HttpGetSubscribers)
	http.HandleFunc("/master", hub.HttpLogin)
	http.HandleFunc("/realtime", hub.HttpLogin)

	logs.Logf("WebSocket", "Http server in http://localhost:%d/ws", port)
	addr := strs.Format(`:%d`, port)
	logs.Fatal(http.ListenAndServe(addr, nil))
}
