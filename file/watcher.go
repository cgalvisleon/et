package file

import (
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/logs"
	"github.com/fsnotify/fsnotify"
)

const watcherPrefix = "file:watcher"

type OnWatching func(path string, d os.DirEntry, err error) error
type OnEvent func(event fsnotify.Event)

type Watcher struct {
	root         string
	watcher      *fsnotify.Watcher
	onWatching   OnWatching
	onEvent      OnEvent
	onEventError func(err error)
	isDebug      bool
}

/**
* NewWatcher
* @param root string
* @return (*Watcher, error)
**/
func NewWatcher(root string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	result := &Watcher{
		root:    root,
		watcher: watcher,
	}
	return result, nil
}

/**
* Close
* @return error
**/
func (s *Watcher) Close() error {
	return s.watcher.Close()
}

/**
* Debug
* @return *Watcher
**/
func (s *Watcher) Debug() *Watcher {
	s.isDebug = true
	return s
}

/**
* OnWatching
* @param fn OnWatching
* @return *Watcher
**/
func (s *Watcher) OnWatching(fn OnWatching) *Watcher {
	s.onWatching = fn
	return s
}

/**
* OnEvent
* @param fn OnEvent
* @return *Watcher
**/
func (s *Watcher) OnEvent(fn OnEvent) *Watcher {
	s.onEvent = fn
	return s
}

/**
* OnError
* @param fn func(err error)
* @return *Watcher
**/
func (s *Watcher) OnError(fn func(err error)) *Watcher {
	s.onEventError = fn
	return s
}

/**
* onError
* @param err error
* @return error
**/
func (s *Watcher) onError(err error) error {
	if s.isDebug {
		logs.Log(watcherPrefix, "Error:", err)
	}
	if s.onEventError != nil {
		s.onEventError(err)
	}
	return err
}

/**
* addWatch
* @param path string
* @return error
**/
func (s *Watcher) addWatch(path string) error {
	if s.isDebug {
		logs.Log(watcherPrefix, "Walking:", path)
	}
	err := s.watcher.Add(path)
	if err != nil {
		return err
	}
	return nil
}

/**
* Load
* @return error
**/
func (s *Watcher) Load() error {
	// Registrar directorio raíz y subdirectorios
	err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			err = s.addWatch(path)
			if err != nil {
				return err
			}
			if s.onWatching != nil {
				s.onWatching(path, d, err)
			}
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
			case event := <-s.watcher.Events:
				if s.isDebug {
					logs.Log(watcherPrefix, "Event:", event)
				}

				if s.onEvent != nil {
					s.onEvent(event)
				}

				// Si se crea un nuevo directorio → empezar a observarlo
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						err = s.addWatch(event.Name)
						if err != nil {
							s.onError(err)
							return
						}
					}
				}
			case err := <-s.watcher.Errors:
				s.onError(err)
			}
		}
	}()

	logs.Log(watcherPrefix, "Watching recursively:", s.root)
	<-done

	return nil
}

func WatcherPath(root string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Registrar directorio raíz y subdirectorios
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			logs.Log(watcherPrefix, "Walking:", path)
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
				logs.Log(watcherPrefix, "Event:", event)

				// Si se crea un nuevo directorio → empezar a observarlo
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						logs.Log(watcherPrefix, "New directory detected, watching:", event.Name)
						watcher.Add(event.Name)
					}
				}

			case err := <-watcher.Errors:
				logs.Log(watcherPrefix, "Error:", err)
			}
		}
	}()

	logs.Log(watcherPrefix, "Watching recursively:", root)
	<-done

	return nil
}
