package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/cgalvisleon/et/ws"
)

var wcs *ws.Client

func main() {
	wsc := ws.NewPubSub("", "91499023", "Cesar", inbox)
	if wsc == nil {
		logs.Fatal("Error al crear el cliente websocket.")
	}

	wsc.Publish("helo", "Hola mundo")
	// serv, err := New()
	// if err != nil {
	// 	logs.Fatal(err)
	// }

	// go serv.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// serv.Close()
}

func inbox(msg pubsub.Message) {
	logs.Debug(msg)
}
