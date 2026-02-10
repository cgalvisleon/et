package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	client, err := tcp.NewClient("localhost:5050")
	if err != nil {
		logs.Panic(err)
	}

	client.Send(tcp.TextMessage, "Hola")
	utility.AppWait()
}
