package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"math/rand"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/ws"
)

var conn *ws.Hub
var clients []*ws.Client

func main() {
	if conn != nil {
		return
	}

	port := config.SetIntByArg("port", 3300)
	mode := config.SetStrByArg("mode", "")
	masterURL := config.SetStrByArg("master-url", "")
	conn = ws.ServerHttp(port, mode, masterURL)
	test1(port)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logs.Log("WebSocket", "Shoutdown server...")
}

func test1(port int) {
	url := fmt.Sprintf(`ws://localhost:%d/ws`, port)

	n := 10000
	for i := 0; i < n; i++ {
		client, err := ws.NewClient(&ws.ClientConfig{
			ClientId:  fmt.Sprintf("client-%d", i),
			Name:      fmt.Sprintf("Client%d", i),
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
				"msg": fmt.Sprintf("Hola %d", idx),
			})
		}
		time.Sleep(t * time.Millisecond)
	}
}
