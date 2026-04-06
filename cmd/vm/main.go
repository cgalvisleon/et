package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/vm"
)

func main() {
	v, err := vm.New("./cmd/vm")
	if err != nil {
		logs.Panic(err)
	}

	_, err = v.RunFile(v.Main)
	if err != nil {
		logs.Error(err)
	}

	err = v.HotReload()
	if err != nil {
		logs.Error(err)
		return
	}
}
