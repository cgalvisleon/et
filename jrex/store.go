package jrex

import (
	"fmt"
	"path/filepath"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/file"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Store interface {
	Load(tag string) (*Jrex, error)
	Save(jrex *Jrex, userId string) error
	GetModule(module string) (*Module, error)
	SetModule(module *Module) error
	DeleteModule(module string) error
	GetCode(module string) (string, error)
	SetCode(module string, code string) error
}

type FileStore struct {
	BaseDir   string
	ModuleDir string
	AuditLog  []et.Json `json:"audit_log"`
	rootDir   string
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

	modulePath := filepath.Join(absPath, ".modules")
	_, err = file.MakeFolder(modulePath)
	if err != nil {
		return nil, err
	}

	result := &FileStore{
		BaseDir:   absPath,
		ModuleDir: modulePath,
		rootDir:   ".",
	}

	return result, nil
}

/**
* Load
* @param tag string
* @return *Jrex, error
**/
func (s *FileStore) Load(tag string) (*Jrex, error) {
	module, err := s.GetModule("index")
	if err != nil {
		return nil, err
	}

	tag = utility.Normalize(tag)
	id := fmt.Sprintf("jrex:%s", tag)
	defaultValue := &Jrex{
		ID:      id,
		Tag:     tag,
		Ctx:     et.Json{},
		Modules: make(map[string]*Module),
	}
	defaultValue.Modules[module.Name] = module

	path := filepath.Join(s.BaseDir, "package.json")
	result, err := file.LoadOrCreateJSON(path, defaultValue)
	if err != nil {
		return nil, err
	}
	module.up(result)

	return result, nil
}

/**
* Save
* @param jrex *Jrex
* @return error
**/
func (s *FileStore) Save(jrex *Jrex, userId string) error {
	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     "save",
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]

	path := filepath.Join(s.BaseDir, "package.json")
	err := file.WriteJSON(path, jrex)
	if err != nil {
		return err
	}
	return nil
}

/**
* GetModule
* @param module string
* @return *Module, error
**/
func (s *FileStore) GetModule(module string) (*Module, error) {
	path := filepath.Join(s.ModuleDir, fmt.Sprintf("%s.json", module))
	result, err := file.LoadOrCreateJSON(path, &Module{
		ID:          fmt.Sprintf("module:%s:%s", module, "1.0.0"),
		Name:        module,
		Version:     "1.0.0",
		Description: "",
		Author:      "",
		License:     "MIT",
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* SetModule
* @param module *Module
* @return error
**/
func (s *FileStore) SetModule(module *Module) error {
	path := filepath.Join(s.ModuleDir, module.Name)
	err := file.WriteJSON(path, module)
	if err != nil {
		return err
	}

	return nil
}

/**
* DeleteModule
* @param module string
* @return error
**/
func (s *FileStore) DeleteModule(module string) error {
	path := filepath.Join(s.ModuleDir, module)
	_, err := file.RemoveFile(path)
	if err != nil {
		return err
	}

	path = filepath.Join(s.BaseDir, module)
	_, err = file.RemoveFile(path)
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
	fl = filepath.Join(s.rootDir, fl)
	path := filepath.Join(s.BaseDir, fl)
	code, err := file.LoadString(path, "")
	if err != nil {
		return "", err
	}
	s.rootDir = filepath.Dir(fl)

	return code, nil
}

/**
* SetCode
* @param module string, code string
* @return error
**/
func (s *FileStore) SetCode(module string, code string) error {
	path := filepath.Join(s.BaseDir, fmt.Sprintf("%s.js", module))
	err := file.WriteString(path, code)
	if err != nil {
		return err
	}
	return nil
}
