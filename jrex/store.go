package jrex

import (
	"fmt"
	"path/filepath"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
	"github.com/fsnotify/fsnotify"
)

type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
}

type FileStore struct {
	BaseDir string
	jrex    *Jrex
}

/**
* NewStore
* @param baseDir string
* @return *FileStore, error
**/
func NewStore(baseDir string) (*FileStore, error) {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	_, err = file.MakeFolder(absPath)
	if err != nil {
		return nil, err
	}

	result := &FileStore{
		BaseDir: absPath,
	}

	return result, nil
}

/**
* up
* @param jrex *Jrex
* @return *FileStore
**/
func (s *FileStore) up(jrex *Jrex) *FileStore {
	s.jrex = jrex
	go s.hotReload()
	return s
}

/**
* Load
* @param collection, id string, dest any
* @return *Jrex, error
**/
func (s *FileStore) Get(collection, id string, dest any) (bool, error) {
	if collection == "jrex" {
		path := filepath.Join(s.BaseDir, "package.json")
		var result *Jrex
		exists, err := file.Load(path, &result)
		if err != nil {
			return false, err
		}

		if exists {
			s.up(result)
			dest = result
			return true, nil
		}

		result, err = newJrex(id, "")
		if err != nil {
			return false, err
		}

		err = file.Save(path, result)
		if err != nil {
			return false, err
		}

		s.up(result)
		dest = result
		return false, nil
	}

	path := filepath.Join(s.BaseDir, fmt.Sprintf("%s.js", id))
	var result string
	exists, err := file.Load(path, &result)
	if err != nil {
		return false, err
	}

	if exists {
		dest = result
		return true, nil
	}

	result = ""
	err = file.Save(path, result)
	if err != nil {
		return false, err
	}

	dest = result
	return false, nil
}

/**
* Set
* @param collection, id, ownerId string, obj any
* @return error
**/
func (s *FileStore) Set(collection, id, ownerId string, obj any) error {
	if collection == "jrex" {
		path := filepath.Join(s.BaseDir, "package.json")
		err := file.Save(path, obj)
		if err != nil {
			return err
		}
		return nil
	}

	path := filepath.Join(s.BaseDir, fmt.Sprintf("%s.js", id))
	err := file.Save(path, obj)
	if err != nil {
		return err
	}

	return nil
}

/**
* hotReload
* @return error
**/
func (s *FileStore) hotReload() error {
	watch, err := file.NewWatcher(s.BaseDir)
	if err != nil {
		return err
	}
	logs.Log("Watcher", fmt.Sprintf("watching %s for changes", s.BaseDir))
	err = watch.OnReload(func(info file.FileInfo, event fsnotify.Event) {
		ctx, err := s.jrex.Run()
		if err != nil {
			s.jrex.Notify("ERROR", err.Error())
		} else {
			s.jrex.Notify("CTX", ctx.ToString())
		}
	}).Load()
	if err != nil {
		return err
	}
	return nil
}
