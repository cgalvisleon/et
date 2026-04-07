package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

type Pkg struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Main            string            `json:"main"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Author          string            `json:"author"`
	License         string            `json:"license"`
}

type Mode string

const (
	Develop    Mode = "develop"
	Production Mode = "production"
	Building   Mode = "building"
)

type Loader struct {
	*Pkg
	mode    Mode   `json:"-"`
	BaseDir string `json:"-"`
	store   Store  `json:"-"`
}

/**
* newLoader
* @param baseDir, name, version string
* @return *Loader
**/
func newLoader(name, version string) *Loader {
	id := fmt.Sprintf(`pkg:%s:%s`, name, version)
	result := &Loader{
		Pkg: &Pkg{
			ID:              id,
			Name:            name,
			Version:         version,
			Scripts:         make(map[string]string),
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
		},
	}
	return result
}

/**
* init
* @return error
**/
func (s *Loader) init() error {
	if s.mode == Production {
		if s.store == nil {
			return fmt.Errorf(msg.MSG_STORE_REQUIRED)
		}

		err := s.store.Connected()
		if err != nil {
			return err
		}

		var pkg Pkg
		ok, err := s.get(s.ID, &pkg)
		if err != nil {
			return err
		}
		if ok {
			s.Pkg = &pkg
		}

		return nil
	}

	pkgFile := filepath.Join(s.BaseDir, "package.json")
	if exists(pkgFile) {
		data, _ := os.ReadFile(pkgFile)
		err := json.Unmarshal(data, &s.Pkg)
		if err != nil {
			return err
		}
	} else {
		s.Pkg.Description = ""
		s.Pkg.Main = "index.js"
		s.Pkg.Scripts = make(map[string]string)
		s.Pkg.Dependencies = make(map[string]string)
		s.Pkg.DevDependencies = make(map[string]string)
		s.Pkg.Author = ""
		s.Pkg.License = ""
		if err := s.save(); err != nil {
			return err
		}
	}

	return nil
}

/**
* save
* @return error
**/
func (s *Loader) save() error {
	pkgFile := filepath.Join(s.BaseDir, "package.json")
	file, err := os.Create(pkgFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // bonito (pretty)

	if err := encoder.Encode(s.Pkg); err != nil {
		return err
	}

	return nil
}

/**
* get
* @param module string, dest any
* @return (bool, error)
**/
func (s *Loader) get(module string, dest any) (bool, error) {
	if s.store == nil {
		return false, nil
	}
	return s.store.Get(module, dest)
}

/**
* set
* @param module string, source any
* @return error
**/
func (s *Loader) set(module string, source any) error {
	if s.store == nil {
		return nil
	}
	return s.store.Set(module, source)
}

/**
* Resolve
* @param modulePath string, currentDir string
* @return (string, error)
**/
func (s *Loader) Resolve(modulePath string, currentDir string) (string, error) {
	// 1. Relativo ./ o ../
	if strings.HasPrefix(modulePath, "./") || strings.HasPrefix(modulePath, "../") {
		full := filepath.Join(currentDir, modulePath)
		result, err := s.resolveAsFileOrDir(full)
		if err != nil {
			return "", err
		}
		logs.Log("Resolve", result)
		return result, nil
	}

	// 2. node_modules
	nm := filepath.Join(s.BaseDir, "node_modules", modulePath)
	result, err := s.resolveAsFileOrDir(nm)
	if err != nil {
		return "", err
	}
	logs.Log("Resolve", result)
	return result, nil
}

/**
* resolveAsFileOrDir
* @param base string
* @return (string, error)
**/
func (s *Loader) resolveAsFileOrDir(base string) (string, error) {
	// archivo directo
	if exists(base) {
		return base, nil
	}

	// archivo .js
	if exists(base + ".js") {
		return base + ".js", nil
	}

	// carpeta con package.json
	pkgFile := filepath.Join(base, "package.json")
	if exists(pkgFile) {
		data, _ := os.ReadFile(pkgFile)

		json.Unmarshal(data, &s.Pkg)

		if s.Pkg.Main != "" {
			return filepath.Join(base, s.Pkg.Main), nil
		}
	}

	// fallback index.js
	index := filepath.Join(base, "index.js")
	if exists(index) {
		return index, nil
	}

	return "", fmt.Errorf("module not found: %s", base)
}

/**
* exists
* @param path string
* @return bool
**/
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
