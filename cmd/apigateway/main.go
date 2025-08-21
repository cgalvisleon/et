package main

import "github.com/cgalvisleon/et/ettp/v2"

func main() {
	srv := ettp.New("localhost", 8080)

	srv.Start()
}
