package file

import (
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/logs"
	"github.com/fsnotify/fsnotify"
)

func WatcherPath(root string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Registrar directorio raíz y subdirectorios
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			logs.Log("file:watcher", "Watching:", path)
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				logs.Log("file:watcher", "Event:", event)

				// Si se crea un nuevo directorio → empezar a observarlo
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						logs.Log("file:watcher", "New directory detected, watching:", event.Name)
						watcher.Add(event.Name)
					}
				}

			case err := <-watcher.Errors:
				logs.Log("file:watcher", "Error:", err)
			}
		}
	}()

	logs.Log("file:watcher", "Watching recursively:", root)
	<-done

	return nil
}
