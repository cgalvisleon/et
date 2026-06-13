package jrex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/file"
	"github.com/fsnotify/fsnotify"
)

type Store interface {
	SetModule(module string, source any) error
	GetModule(module string, source any) (bool, error)
	DeleteModule(module string) error
}

var moduleFileSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// FileStore is a jrex.Store implementation backed by JSON files on the
// local filesystem, one file per module key.
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
	watch   *file.Watcher
	watchMu sync.Mutex
}

/**
* NewFileStore: Creates a file-based Store rooted at baseDir, creating the
* directory if it does not exist.
* @param baseDir string
* @return *FileStore, error
**/
func NewFileStore(baseDir string) (*FileStore, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, err
	}

	return &FileStore{baseDir: abs}, nil
}

/**
* resolve: Returns p as an absolute path: p itself if already absolute,
* otherwise p joined with baseDir.
* @param p string
* @return string
**/
func (s *FileStore) resolve(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.baseDir, p)
}

/**
* Exists: Reports whether p (absolute, or relative to baseDir) exists.
* @param p string
* @return bool
**/
func (s *FileStore) Exists(p string) bool {
	_, err := os.Stat(s.resolve(p))
	return err == nil
}

/**
* ReadTextFile: Reads p (absolute, or relative to baseDir) as text.
* @param p string
* @return string, error
**/
func (s *FileStore) ReadTextFile(p string) (string, error) {
	data, err := os.ReadFile(s.resolve(p))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

/**
* ReadJSON: Reads p (absolute, or relative to baseDir) and decodes it into
* dest. Returns (false, nil) without modifying dest if the file does not
* exist.
* @param p string, dest any
* @return bool, error
**/
func (s *FileStore) ReadJSON(p string, dest any) (bool, error) {
	full := s.resolve(p)

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, err
	}

	return true, nil
}

/**
* WriteJSON: Pretty-encodes src as JSON and writes it to p (absolute, or
* relative to baseDir).
* @param p string, src any
* @return error
**/
func (s *FileStore) WriteJSON(p string, src any) error {
	full := s.resolve(p)

	data, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return os.WriteFile(full, data, 0644)
}

/**
* moduleFilename: Converts a module key into a safe, flat filename. Note that
* two different keys could in theory sanitize to the same filename; this is
* accepted in exchange for human-readable filenames.
* @param module string
* @return string
**/
func moduleFilename(module string) string {
	clean := path.Clean("/" + module)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		clean = "_"
	}

	return moduleFileSanitizer.ReplaceAllString(clean, "_") + ".json"
}

/**
* pathFor: Resolves the absolute file path for module, ensuring it stays
* within baseDir.
* @param module string
* @return string, error
**/
func (s *FileStore) pathFor(module string) (string, error) {
	name := moduleFilename(module)
	full := filepath.Join(s.baseDir, name)

	rel, err := filepath.Rel(s.baseDir, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid module key: %q", module)
	}

	return full, nil
}

/**
* SetModule: Serializes source as JSON and writes it to the file for module.
* @param module string, source any
* @return error
**/
func (s *FileStore) SetModule(module string, source any) error {
	full, err := s.pathFor(module)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return os.WriteFile(full, data, 0644)
}

/**
* GetModule: Reads the file for module and decodes it into source. Returns
* (false, nil) without modifying source if the file does not exist.
* @param module string, source any
* @return bool, error
**/
func (s *FileStore) GetModule(module string, source any) (bool, error) {
	full, err := s.pathFor(module)
	if err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(data, source); err != nil {
		return false, err
	}

	return true, nil
}

/**
* DeleteModule: Removes the file for module. It is not an error if the file
* does not exist.
* @param module string
* @return error
**/
func (s *FileStore) DeleteModule(module string) error {
	full, err := s.pathFor(module)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err = os.Remove(full)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

/**
* Watch: Starts watching baseDir for external changes to module files and
* invokes fn for each create/write/remove/rename event on a .json file. The
* module argument passed to fn is the sanitized filename without the .json
* extension (see moduleFilename) — it may not match the original module key
* exactly in case of sanitization collisions. Watching runs in a background
* goroutine; call Close to stop it.
* @param fn func(module string, event fsnotify.Event)
* @return error
**/
func (s *FileStore) Watch(fn func(module string, event fsnotify.Event)) error {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()

	if s.watch != nil {
		return errors.New("file store is already watching")
	}

	w, err := file.NewWatcher(s.baseDir)
	if err != nil {
		return err
	}

	handler := func(info file.FileInfo, event fsnotify.Event) {
		if info.IsDir || filepath.Ext(event.Name) != ".json" {
			return
		}
		if fn != nil {
			name := strings.TrimSuffix(filepath.Base(event.Name), ".json")
			fn(name, event)
		}
	}

	w.OnCreate(handler).OnWrite(handler).OnRemove(handler).OnRename(handler)
	s.watch = w

	go w.Load()

	return nil
}

/**
* Close: Stops the watcher started by Watch. It is not an error to call
* Close if Watch was never called.
* @return error
**/
func (s *FileStore) Close() error {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()

	if s.watch == nil {
		return nil
	}

	err := s.watch.Close()
	s.watch = nil
	return err
}
