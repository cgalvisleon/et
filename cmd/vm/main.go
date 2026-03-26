package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/vm"
)

func main() {
	v, err := vm.New("./cmd/vm/scripts")
	if err != nil {
		logs.Panic(err)
	}

	_, err = v.RunFile("/test.js")
	if err != nil {
		logs.Panic(err)
	}
}
