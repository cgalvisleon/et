package main

import (
	"log"
	"net/http"

	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/ws"
)

var conn *ws.Hub

func main() {
	if conn != nil {
		return
	}

	conn = ws.NewWs()
	conn.Start()

	http.HandleFunc("/ws", wsHandler)
	log.Println("Servidor WebSocket iniciado en :3300")
	log.Fatal(http.ListenAndServe(":3300", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := conn.ConnectHttp(w, r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
	}
}
