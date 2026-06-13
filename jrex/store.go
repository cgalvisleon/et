package jrex

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
