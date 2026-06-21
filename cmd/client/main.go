package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jtcp"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	addr := envar.SetStrByArg("-addr", "ADDR", "localhost:1377")

	client := jtcp.NewClient(addr)
	err := client.Connect()
	if err != nil {
		logs.Panic(err)
	}

	jtcp.StartConsole(client)
}
