package main

import (
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/ettp/v2"
)

func main() {
	timeout := config.GetFloat("TIMEOUT", 100)
	srv, err := ettp.New("Apigateway", &ettp.Config{
		Port:         8080,
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
		panic(err)
	}

	srv.Start()
}
