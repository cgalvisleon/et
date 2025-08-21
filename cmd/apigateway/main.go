package main

import (
	"time"

	"github.com/cgalvisleon/et/ettp/v2"
)

func main() {
	srv := ettp.New(8080, &ettp.Config{
		PathApi:      "/api",
		PathApp:      "/app",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
		TLS:          false,
		CertFile:     "",
		KeyFile:      "",
	})

	srv.Start()
}
