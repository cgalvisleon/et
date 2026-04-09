package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	addr := envar.SetStrByArg("-addr", "ADDR", "localhost:5050")

	client := tcp.NewClient(addr)
	err := client.Start()
	if err != nil {
		logs.Panic(err)
	}

	res := client.Request("Tcp.Ping", "Hola")
	if res.Error != nil {
		logs.Panic(res.Error)
	}

	logs.Info(res.Response)

	utility.AppWait()
}
