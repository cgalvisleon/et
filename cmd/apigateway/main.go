package main

import (
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/ettp/v2"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	timeout := envar.GetFloat("TIMEOUT", 100)
	srv, err := ettp.New("Apigateway", &ettp.Config{
		Port:         4040,
		RpcPort:      4400,
		Parent:       "/api",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
		Timeout:      time.Duration(timeout) * time.Second,
		IsTLS:        false,
		CertFile:     "",
		KeyFile:      "",
		UseCache:     false,
		UseEvent:     false,
		Debug:        true,
	})

	if err != nil {
		logs.Panic(err)
	}

	srv.Start()
}
