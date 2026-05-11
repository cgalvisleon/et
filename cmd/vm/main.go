package main

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/script"
)

func main() {
	v, err := script.New("vm")
	if err != nil {
		logs.Panic(err)
	}

	err = v.RunDev("./cmd/vm")
	if err != nil {
		logs.Panic(err)
	}

}
