package main

import (
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	v := jrex.New("jrex", nil)
	err := v.RunDev("./cmd/vm")
	if err != nil {
		logs.Panic(err)
	}
}
