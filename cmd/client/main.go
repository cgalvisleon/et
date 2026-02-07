package main

import "github.com/cgalvisleon/et/tcp"

func main() {
	client := tcp.NewClient("client", "localhost:5050")
	client.Connect()
}
