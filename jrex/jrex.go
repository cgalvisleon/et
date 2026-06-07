package jrex

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dop251/goja"
	"github.com/fsnotify/fsnotify"
)

type Store interface {
	SetModule(module string, source any) error
	GetModule(module string, source any) (bool, error)
	DeleteModule(module string) error
}

var (
	packageName = "jrex"
)

type Jrex struct {
	*Loader `json:"-"`
	Ctx     et.Json           `json:"ctx"`
	ID      string            `json:"id"`
	Scripts string            `json:"scripts"`
	pkgFile string            `json:"-"`
	vm      *goja.Runtime     `json:"-"`
	watch   *file.Watcher     `json:"-"`
	store   Store             `json:"-"`
	program *tea.Program      `json:"-"`
	onStart func(*Jrex) error `json:"-"`
}

/**
* New
* @param name string, store Store
* @return *Jrex
**/
func New(name string, store Store) (*Jrex, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		name = "jrex"
	}

	id := reg.GenULID(packageName)
	result := &Jrex{
		ID:    id,
		Ctx:   et.Json{},
		vm:    goja.New(),
		store: store,
	}
	result.Loader = newLoader(result, name)

	absPath, err := filepath.Abs("./")
	if err != nil {
		return nil, err
	}

	result.BaseDir = absPath
	result.pkgFile = filepath.Join(result.BaseDir, "package.json")
	return result, nil
}

/**
* Value
* @param value interface{}
* @return goja.Value
**/
func (s *Jrex) Value(value interface{}) goja.Value {
	return s.vm.ToValue(value)
}

/**
* Error
* @param err error
* @return *goja.Object
**/
func (s *Jrex) Error(err error) *goja.Object {
	return s.vm.NewGoError(err)
}

/**
* Get
* @param name string
* @return goja.Value
**/
func (s *Jrex) Get(name string) goja.Value {
	return s.vm.Get(name)
}

/**
* GetJson
* @param name string
* @return et.Json
**/
func (s *Jrex) GetJson(name string) et.Json {
	result, ok := s.vm.Get(name).Export().(et.Json)
	if !ok {
		return et.Json{}
	}
	return result
}

/**
* Set
* @params name string, value interface{}
* @return error
**/
func (s *Jrex) Set(name string, value interface{}) error {
	return s.vm.Set(name, value)
}

/**
* uppToStore
* @return error
**/
func (s *Jrex) uppToStore() error {
	id := fmt.Sprintf("pkg:%s:%s", s.Name, s.Version)
	return s.set(id, s)
}

/**
* SetModule
* @params module string, path string
* @return error
**/
func (s *Jrex) SetModule(module string, path string) error {
	_, ok := s.Modules[module]
	s.Modules[module] = path
	if !ok {
		return s.save()
	}
	return nil
}

/**
* SetDescription
* @params description string
* @return error
**/
func (s *Jrex) SetDescription(description string) error {
	s.Description = description
	return s.save()
}

/**
* SetAuthor
* @params author string
* @return error
**/
func (s *Jrex) SetAuthor(author string) error {
	s.Author = author
	return s.save()
}

/**
* SetLicense
* @params license string
* @return error
**/
func (s *Jrex) SetLicense(license string) error {
	s.License = license
	return s.save()
}

/**
* SetCtx
* @params ctx et.Json
**/
func (s *Jrex) SetCtx(ctx et.Json) {
	maps.Copy(s.Ctx, ctx)
}

/**
* OnStart: Registers a function to run when the CLI starts. Has no effect if fn is nil.
* @param fn func(*Jrex) error
* @return *Jrex
**/
func (s *Jrex) OnStart(fn func(*Jrex) error) *Jrex {
	s.onStart = fn
	return s
}

/**
* Notify: Reports a kind/message pair to the running CLI program, stripping
* ANSI escape codes so colorized log output doesn't corrupt the TUI rendering.
* @params kind string, message string
**/
func (s *Jrex) Notify(kind, message string) {
	if s.program != nil {
		s.program.Send(cliLogMsg{kind: kind, message: ansi.Strip(message)})
		return
	}
}

/**
* Run
* @params str string
* @return et.Json, error
**/
func (s *Jrex) Run(str string) (et.Json, error) {
	s.vm = goja.New()
	wrap(s)

	_, err := s.vm.RunString(requireRuntime)
	if err != nil {
		return nil, err
	}

	_, err = s.vm.RunString(str)
	if err != nil {
		return nil, err
	}

	return s.Ctx, nil
}

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
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result, err := s.Run(string(data))
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
	s.pkgFile = filepath.Join(s.BaseDir, "package.json")
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
