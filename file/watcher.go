package file

import (
	"os"
	"path/filepath"
	"time"

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
	onCreate     func(FileInfo, fsnotify.Event)
	onWrite      func(FileInfo, fsnotify.Event)
	onRemove     func(FileInfo, fsnotify.Event)
	onRename     func(FileInfo, fsnotify.Event)
	onChmod      func(FileInfo, fsnotify.Event)
	onReload     func(FileInfo, fsnotify.Event)
	onEventError func(err error)
	reloadFile   map[string]fsnotify.Op
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
		root:       root,
		watcher:    watcher,
		reloadFile: make(map[string]fsnotify.Op),
	}
	result.onEvent = func(event fsnotify.Event) {
		inf := ExistPath(event.Name)
		if !inf.IsDir {
			op, ok := result.reloadFile[event.Name]
			if ok {
				if op == fsnotify.Write && event.Op == fsnotify.Chmod {
					if result.onReload != nil {
						result.onReload(inf, event)
					}
				}
			}
			result.reloadFile[event.Name] = event.Op
			time.AfterFunc(3*time.Second, func() {
				delete(result.reloadFile, event.Name)
			})
		}
		switch event.Op {
		case fsnotify.Create:
			if result.onCreate != nil {
				result.onCreate(inf, event)
			}
		case fsnotify.Write:
			if result.onWrite != nil {
				result.onWrite(inf, event)
			}
		case fsnotify.Remove:
			if result.onRemove != nil {
				result.onRemove(inf, event)
			}
		case fsnotify.Rename:
			if result.onRename != nil {
				result.onRename(inf, event)
			}
		case fsnotify.Chmod:
			if result.onChmod != nil {
				result.onChmod(inf, event)
			}
		}
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
* OnCreate
* @param fn func(FileInfo, fsnotify.Event)
* @return *Watcher
**/
func (s *Watcher) OnCreate(fn func(FileInfo, fsnotify.Event)) *Watcher {
	s.onCreate = fn
	return s
}

/**
* OnWrite
* @param fn func(FileInfo, fsnotify.Event)
* @return *Watcher
**/
func (s *Watcher) OnWrite(fn func(FileInfo, fsnotify.Event)) *Watcher {
	s.onWrite = fn
	return s
}

/**
* OnRemove
* @param fn func(FileInfo, fsnotify.Event)
* @return *Watcher
**/
func (s *Watcher) OnRemove(fn func(FileInfo, fsnotify.Event)) *Watcher {
	s.onRemove = fn
	return s
}

/**
* OnRename
* @param fn func(FileInfo, fsnotify.Event)
* @return *Watcher
**/
func (s *Watcher) OnRename(fn func(FileInfo, fsnotify.Event)) *Watcher {
	s.onRename = fn
	return s
}

/**
* OnChmod
* @param fn func(FileInfo, fsnotify.Event)
* @return *Watcher
**/
func (s *Watcher) OnChmod(fn func(FileInfo, fsnotify.Event)) *Watcher {
	s.onChmod = fn
	return s
}

/**
* OnReload
* @param fn func(FileInfo, fsnotify.Event)
* @return *Watcher
**/
func (s *Watcher) OnReload(fn func(FileInfo, fsnotify.Event)) *Watcher {
	s.onReload = fn
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
