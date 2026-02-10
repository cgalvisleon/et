package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	client := tcp.NewClient("client", "localhost:5050")
	err := client.Connect()
	if err != nil {
		logs.Panic(err)
	}

	utility.AppWait()
}
