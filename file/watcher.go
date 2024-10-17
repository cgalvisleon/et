package file

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func WatcherPath(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("Evento:", event)
			// Aquí puedes manejar diferentes tipos de eventos, como creación, modificación o eliminación de archivos.
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error:", err)
		}
	}
}
