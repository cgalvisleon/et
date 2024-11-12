package ws

import (
	"net/http"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

/**
* ServerHttp
* @params port int
* @params mode string
* @params master string
* @params schema string
* @params path string
* @return *Hub
**/
func ServerHttp(port int, mode, masterURL string) *Hub {
	result := NewHub()
	result.Start()
	switch mode {
	case "master":
		result.InitMaster()
		if masterURL != "" {
			result.Join(AdapterConfig{
				Url:       masterURL,
				TypeNode:  NodeMaster,
				Reconcect: 3,
				Header:    http.Header{},
			})
		}
	case "worker":
		if masterURL != "" {
			result.Join(AdapterConfig{
				Url:       masterURL,
				TypeNode:  NodeWorker,
				Reconcect: 3,
				Header:    http.Header{},
			})
		}
	}

	go startHttp(result, port)
	time.Sleep(1 * time.Second)

	return result
}

func startHttp(hub *Hub, port int) {
	http.HandleFunc("/ws", hub.HttpConnect)
	http.HandleFunc("/ws/describe", hub.HttpDescribe)
	http.HandleFunc("/ws/publications", hub.HttpGetPublications)
	http.HandleFunc("/ws/subscribers", hub.HttpGetSubscribers)

	logs.Logf("WebSocket", "Http server in http://localhost:%d/ws", port)
	addr := strs.Format(`:%d`, port)
	logs.Fatal(http.ListenAndServe(addr, nil))
}
