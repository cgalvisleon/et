package main

import (
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	v, err := js.New("vm")
	if err != nil {
		logs.Panic(err)
	}

	err = v.RunDev("./cmd/vm")
	if err != nil {
		logs.Panic(err)
	}

}
