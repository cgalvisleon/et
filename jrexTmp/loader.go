package jrex

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type Pkg struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Main            string            `json:"main"`
	Modules         map[string]string `json:"modules"`
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
)

type Loader struct {
	*Pkg
	mode    Mode   `json:"-"`
	BaseDir string `json:"-"`
	jrex    *Jrex  `json:"-"`
}

/**
* newLoader
* @param name string
* @return *Loader
**/
func newLoader(jrex *Jrex, name string) *Loader {
	result := &Loader{
		Pkg: &Pkg{
			Name:            name,
			Version:         "0.0.1",
			Description:     "",
			Main:            "index.js",
			Modules:         make(map[string]string),
			Scripts:         make(map[string]string),
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
		},
		mode: Production,
		jrex: jrex,
	}
	return result
}

/**
* save
* @return error
**/
func (s *Loader) save() error {
	return s.jrex.files.WriteJSON("package.json", s.Pkg)
}

/**
* init
* @return error
**/
func (s *Loader) init() error {
	if s.mode == Production {
		id := fmt.Sprintf("pkg:%s:%s", s.Name, s.Version)
		var pkg Pkg
		ok, err := s.get(id, &pkg)
		if err != nil {
			return err
		}
		if ok {
			s.Pkg = &pkg
		}

		return nil
	} else {
		ok, err := s.jrex.files.ReadJSON("package.json", &s.Pkg)
		if err != nil {
			return err
		}
		if !ok {
			if err := s.save(); err != nil {
				return err
			}
		}
	}

	return nil
}

/**
* SetVersion
* @param version string
* @return error
**/
func (s *Loader) SetVersion(version string) error {
	s.Version = version
	return s.save()
}

/**
* BumpVersion
* @param part Part
* @return string
**/
func (s *Loader) BumpVersion(part Part) (string, error) {
	parts := strings.Split(s.Version, ".")
	if len(parts) != 3 {
		return s.Version, nil
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	switch part {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "release":
		patch++
	default:
		return s.Version, nil
	}

	result := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	return result, s.SetVersion(result)
}

/**
* get
* @param module string, dest any
* @return (bool, error)
**/
func (s *Loader) get(module string, dest any) (bool, error) {
	return s.jrex.store.GetModule(module, dest)
}

/**
* set
* @param module string, source any
* @return error
**/
func (s *Loader) set(module string, source any) error {
	return s.jrex.store.SetModule(module, source)
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
		s.jrex.Notify("LOG", fmt.Sprintf("Resolve: %s", result))
		return result, nil
	}

	// 2. node_modules
	nm := filepath.Join(s.BaseDir, "node_modules", modulePath)
	result, err := s.resolveAsFileOrDir(nm)
	if err != nil {
		return "", err
	}
	s.jrex.Notify("LOG", fmt.Sprintf("Resolve: %s", result))
	return result, nil
}

/**
* resolveAsFileOrDir
* @param base string
* @return (string, error)
**/
func (s *Loader) resolveAsFileOrDir(base string) (string, error) {
	// archivo directo
	if s.jrex.files.Exists(base) {
		return base, nil
	}

	// archivo .js
	if s.jrex.files.Exists(base + ".js") {
		return base + ".js", nil
	}

	// carpeta con package.json
	if ok, _ := s.jrex.files.ReadJSON("package.json", &s.Pkg); ok {
		if s.Pkg.Main != "" {
			return filepath.Join(base, s.Pkg.Main), nil
		}
	}

	// fallback index.js
	index := filepath.Join(base, "index.js")
	if s.jrex.files.Exists(index) {
		return index, nil
	}

	return "", fmt.Errorf("module not found: %s", base)
}
