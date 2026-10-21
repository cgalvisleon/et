package main

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
)

func main() {
	port := envar.SetIntByArg("-port", "PORT", 1377)

	srv, err := tcp.NewServer("", "server")
	if err != nil {
		logs.Panic(err)
	}

	srv.OnRequest(func(conn *tcp.Client, message []byte) ([]byte, error) {
		return message, nil
	})

	err = srv.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		logs.Panic(err)
	}

	utility.AppWait()
}
