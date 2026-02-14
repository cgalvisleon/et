package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	port := envar.SetIntByArg("-port", "PORT", 1377)

	server := tcp.NewServer(port)
	err := server.Start()
	if err != nil {
		logs.Panic(err)
	}

	utility.AppWait()
}
