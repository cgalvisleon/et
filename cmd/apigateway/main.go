package main

import (
	"time"

	"github.com/cgalvisleon/et/ettp/v2"
)

func main() {
	srv := ettp.New("Apigateway", &ettp.Config{
		Port:         8080,
		Parent:       "/api",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
		TLS:          false,
		CertFile:     "",
		KeyFile:      "",
		Debug:        true,
	})

	srv.Start()
}
