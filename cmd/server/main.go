package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
)

func main() {
	server := tcp.NewServer(5050)
	err := server.Start()
	if err != nil {
		logs.Panic(err)
	}
}
