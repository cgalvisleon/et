package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
)

func main() {
	addr := envar.SetStrByArg("-addr", "ADDR", "localhost:1377")

	client := tcp.NewClient(addr)
	err := client.Connect()
	if err != nil {
		logs.Panic(err)
	}

	tcp.StartConsole(client)
}
