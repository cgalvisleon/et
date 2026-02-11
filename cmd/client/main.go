package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
)

func main() {
	client := tcp.NewClient("localhost:5050")
	err := client.Start()
	if err != nil {
		logs.Panic(err)
	}
}
