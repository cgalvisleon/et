package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"math/rand"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/ws"
)

var conn *ws.Hub
var clients []*ws.Client

func main() {
	if conn != nil {
		return
	}

	envar.SetInt("port", 3300, "Port server", "PORT")
	envar.SetStr("mode", "", "Modo cluster: master or worker", "WS_MODE")
	envar.SetStr("master-url", "", "Master host", "WS_MASTER_URL")

	port := envar.GetInt(3300, "PORT")
	mode := envar.GetStr("", "WS_MODE")
	masterURL := envar.GetStr("", "WS_MASTER_URL")

	conn = ws.ServerHttp(port, mode, masterURL)

	test1(port)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logs.Log("WebSocket", "Shoutdown server...")
}

func test1(port int) {
	url := strs.Format(`ws://localhost:%d/ws`, port)

	n := 10000
	for i := 0; i < n; i++ {
		client, err := ws.NewClient(&ws.ClientConfig{
			ClientId:  strs.Format("client-%d", i),
			Name:      strs.Format("Client%d", i),
			Url:       url,
			Reconnect: 3,
		})
		if err != nil {
			logs.Alert(err)
		}

		client.Subscribe("Hola", func(msg ws.Message) {
			logs.Debug("client1", msg.ToString())
		})

		clients = append(clients, client)
	}

	rand.NewSource(time.Now().UnixNano())

	t := time.Duration(100)
	for {
		idx := rand.Intn(n)
		client := clients[idx]
		if client != nil {
			client.Publish("Hola", et.Json{
				"msg": strs.Format("Hola %d", idx),
			})
		}
		time.Sleep(t * time.Millisecond)
	}
}
