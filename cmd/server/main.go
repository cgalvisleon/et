package main

import "github.com/cgalvisleon/et/tcp"

func main() {
	server := tcp.NewServer(5050)
	server.Start()
}
