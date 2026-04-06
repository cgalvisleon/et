package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Pkg struct {
	Name        string  `json:"name"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
	Main        string  `json:"main"`
	Scripts     et.Json `json:"scripts"`
	Author      string  `json:"author"`
	License     string  `json:"license"`
}

type Loader struct {
	*Pkg
	BaseDir string `json:"-"`
}

/**
* newLoader
* @param baseDir string
* @return *Loader
**/
func newLoader(baseDir string) *Loader {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		panic(err)
	}
	result := &Loader{BaseDir: absPath}
	result.Init()
	return result
}

/**
* Init
* @return error
**/
func (s *Loader) Init() error {
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
		s.Pkg = &Pkg{
			Name:        name,
			Version:     "0.0.1",
			Description: name,
			Main:        "index.js",
			Scripts:     et.Json{},
			Author:      "",
			License:     "",
		}

		if err := encoder.Encode(s.Pkg); err != nil {
			return err
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
