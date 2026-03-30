package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/vm"
	"github.com/fsnotify/fsnotify"
)

func main() {
	v, err := vm.New("./cmd/vm/scripts")
	if err != nil {
		logs.Panic(err)
	}

	_, err = v.RunFile("/test.js")
	if err != nil {
		logs.Error(err)
	}

	watch, err := file.NewWatcher("./cmd/vm/scripts")
	if err != nil {
		logs.Error(err)
		return
	}
	defer watch.Close()

	err = watch.
		Debug().
		OnEvent(func(event fsnotify.Event) {
			logs.Debug(et.Json{
				"name": event.Name,
				"op":   event.Op.String(),
				"opts": event.Op,
			}.ToString())
		}).Load()
	if err != nil {
		logs.Error(err)
		return
	}
}
