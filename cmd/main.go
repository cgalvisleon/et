package main

import (
	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
)

func main() {
	_, err := cache.Load()
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		id := reg.Id("test")
		logs.Debug(id)
	}
}
