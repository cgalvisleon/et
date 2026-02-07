package main

import "github.com/cgalvisleon/et/tcp"

func main() {
	server := tcp.NewServer(8080)
	server.Start()
}
