package jrex

import (
	"fmt"
	"path/filepath"

	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/logs"
	"github.com/fsnotify/fsnotify"
)

type Store interface {
	Load(tag string) (*Jrex, error)
	Save(jrex *Jrex, userId string) error
	GetCode(module string) (string, error)
	SetCode(module *Module, code string) error
}

type FileStore struct {
	BaseDir string
	jrex    *Jrex
}

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
* getModule: Gets the module
* @params module string
* @return *Module, error
**/
func (s *FileStore) getModule(module string) (*Module, error) {
	mod := s.jrex.NewModule(module)
	path := filepath.Join(s.BaseDir, fmt.Sprintf("%s.json", module))
	result, err := file.LoadOrCreateJSON(path, mod)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Load
* @param tag string
* @return *Jrex, error
**/
func (s *FileStore) Load(tag string) (*Jrex, error) {
	module, err := s.getModule("index")
	if err != nil {
		return nil, err
	}

	def, err := NewJrex(tag)
	if err != nil {
		return nil, err
	}
	def.AddModule(module)

	path := filepath.Join(s.BaseDir, "package.json")
	result, err := file.LoadOrCreateJSON(path, def)
	if err != nil {
		return nil, err
	}
	s.up(result)

	return result, nil
}

/**
* Save
* @param jrex *Jrex
* @return error
**/
func (s *FileStore) Save(jrex *Jrex, userId string) error {
	path := filepath.Join(s.BaseDir, "package.json")
	err := file.WriteJSON(path, jrex)
	if err != nil {
		return err
	}
	return nil
}

/**
* GetCode
* @param module string
* @return string, error
**/
func (s *FileStore) GetCode(module string) (string, error) {
	fl := fmt.Sprintf("%s.js", module)
	path := filepath.Join(s.BaseDir, fl)
	code, err := file.LoadString(path, "")
	if err != nil {
		return "", err
	}

	return code, nil
}

/**
* SetCode
* @param module *Module, code string
* @return error
**/
func (s *FileStore) SetCode(module *Module, code string) error {
	path := filepath.Join(s.BaseDir, fmt.Sprintf("%s.js", module.Path))
	return file.WriteString(path, code)
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
