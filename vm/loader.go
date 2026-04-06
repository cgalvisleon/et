package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

type Pkg struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Version         string  `json:"version"`
	Description     string  `json:"description"`
	Main            string  `json:"main"`
	Scripts         et.Json `json:"scripts"`
	Dependencies    et.Json `json:"dependencies"`
	DevDependencies et.Json `json:"devDependencies"`
	Author          string  `json:"author"`
	License         string  `json:"license"`
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
func newLoader(baseDir, name, version string) *Loader {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		panic(err)
	}
	id := fmt.Sprintf(`pkg:%s:%s`, name, version)
	result := &Loader{
		Pkg: &Pkg{
			ID:              id,
			Name:            name,
			Version:         version,
			Scripts:         et.Json{},
			Dependencies:    et.Json{},
			DevDependencies: et.Json{},
		},
		BaseDir: absPath,
	}
	return result
}

/**
* get
* @param module string, dest any
* @return (bool, error)
**/
func (s *Loader) get(module string, dest any) (bool, error) {
	return s.store.Get(module, dest)
}

/**
* set
* @param module string, source any
* @return error
**/
func (s *Loader) set(module string, source any) error {
	return s.store.Set(module, source)
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
	if !exists(pkgFile) {
		file, err := os.Create(pkgFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ") // bonito (pretty)

		name := filepath.Base(s.BaseDir)
		s.Pkg.Description = name
		s.Pkg.Main = "index.js"
		s.Pkg.Scripts = et.Json{}
		s.Pkg.Dependencies = et.Json{}
		s.Pkg.DevDependencies = et.Json{}
		s.Pkg.Author = ""
		s.Pkg.License = ""

		if err := encoder.Encode(s.Pkg); err != nil {
			return err
		}

		if s.store != nil {
			err := s.store.Connected()
			if err != nil {
				return err
			}
		}
	} else {
		data, _ := os.ReadFile(pkgFile)
		err := json.Unmarshal(data, &s.Pkg)
		if err != nil {
			return err
		}
	}

	return nil
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
