package jrex

import (
	"fmt"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/fsnotify/fsnotify"
)

/**
* RunCode
* @params code []byte
* @return et.Json, error
**/
func (s *Jrex) RunByBt(code []byte) (et.Json, error) {
	return s.Run(string(code))
}

/**
* RunByFile
* @params path string
* @return et.Json, error
**/
func (s *Jrex) RunByFile(path string) (et.Json, error) {
	path = filepath.Join(s.Loader.BaseDir, path)
	data, err := s.files.ReadTextFile(path)
	if err != nil {
		return nil, err
	}

	result, err := s.Run(data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* RunBySource
* @param path string
* @return (et.Json, error)
**/
func (s *Jrex) RunBySource(path string) (et.Json, error) {
	var scr Module
	exists, err := s.get(path, &scr)
	if err != nil {
		panic(s.Error(err))
	}

	if !exists {
		panic(s.Error(fmt.Errorf("script not found: %s", path)))
	}
	return s.Run(scr.Scripts)
}

/**
* RunDev
* @param baseDir string
* @return error
**/
func (s *Jrex) RunDev(baseDir string) error {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	s.BaseDir = absPath
	s.files, err = NewFileStore(s.BaseDir)
	if err != nil {
		return err
	}
	s.mode = Develop
	err = s.init()
	if err != nil {
		return err
	}

	s.OnStart(func(s *Jrex) error {
		_, err := s.RunByFile(s.Main)
		if err != nil {
			return err
		}
		return nil
	})

	return s.RunCli()
}

/**
* hotReload
* @return error
**/
func (s *Jrex) hotReload() error {
	watch, err := file.NewWatcher(s.BaseDir)
	if err != nil {
		return err
	}
	s.watch = watch
	s.Notify("Watcher", fmt.Sprintf("watching %s for changes", s.BaseDir))
	err = s.watch.OnReload(func(info file.FileInfo, event fsnotify.Event) {
		_, err := s.RunByFile(s.Main)
		if err != nil {
			s.Notify("ERROR", err.Error())
		} else {
			s.Notify("CTX", s.Ctx.ToString())
		}
	}).Load()
	if err != nil {
		return err
	}
	return nil
}
