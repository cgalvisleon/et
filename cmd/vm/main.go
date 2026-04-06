package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/vm"
)

func main() {
	_, err := vm.Dev("./cmd/vm", "vm", "0.0.1")
	if err != nil {
		logs.Panic(err)
	}

}
