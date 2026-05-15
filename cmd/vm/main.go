package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/vm"
)

func main() {
	v, err := vm.New("vm")
	if err != nil {
		logs.Panic(err)
	}

	err = v.RunDev("./cmd/vm")
	if err != nil {
		logs.Panic(err)
	}

}
