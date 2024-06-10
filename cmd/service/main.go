package main

import (
	"os"
	"os/signal"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/message"
	"github.com/cgalvisleon/et/ws"
)

func main() {
	wsc, err := ws.NewPubSub("91499023", "Cesar", inbox)
	if err != nil {
		logs.Fatal(err)
	}

	if wsc == nil {
		logs.Fatal("Error al crear el cliente websocket.")
	}

	// wsc.Publish("helo", "Hola mundo")
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

func inbox(msg message.Message) {
	dt, err := et.Marshal(msg)
	if err != nil {
		return
	}

	logs.Debug(dt.ToString())
}
