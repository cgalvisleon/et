package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/vm"
)

func main() {
	v := vm.New("vm")
	err := v.RunDev("./cmd/vm")
	if err != nil {
		logs.Panic(err)
	}

}
