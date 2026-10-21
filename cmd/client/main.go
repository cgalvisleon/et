package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
)

func main() {
	addr := envar.SetStrByArg("-addr", "ADDR", "localhost:1377")

	client, err := tcp.NewClient("", "client")
	if err != nil {
		logs.Panic(err)
	}

	err = client.ConnectTo(addr)
	if err != nil {
		logs.Panic(err)
	}
	defer client.Close()

	scanner := bufio.NewScanner(os.Stdin)
	for fmt.Print("> "); scanner.Scan(); fmt.Print("> ") {
		res, err := client.Request(scanner.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(string(res))
	}
}
