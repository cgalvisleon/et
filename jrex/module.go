package jrex

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

type Part string

const (
	Same    Part = "same"
	Major   Part = "major"
	Minor   Part = "minor"
	Release Part = "release"
)

/**
* ToPart
* @param value string
* @return Part, bool
**/
func ToPart(value string) (Part, bool) {
	switch value {
	case "same":
		return Same, true
	case "major":
		return Major, true
	case "minor":
		return Minor, true
	case "release":
		return Release, true
	}
	return "", false
}

type Module struct {
	ID       string  `json:"id"`
	Path     string  `json:"path"`
	Version  string  `json:"version"`
	Metadata et.Json `json:"metadata"`
	jrex     *Jrex   `json:"-"`
}

/**
* NewModule: Creates a new module
* @param jrex *Jrex, path string
* @return *Module
**/
func (s *Jrex) NewModule(path string) *Module {
	version := "1.0.0"
	id := reg.ULID()
	return &Module{
		ID:       id,
		Path:     path,
		Version:  version,
		Metadata: et.Json{},
	}
}

/**
* up
* @param jrex *Jrex
* @return *Module
**/
func (s *Module) up(jrex *Jrex) *Module {
	s.jrex = jrex
	s.jrex.Modules[s.Path] = s
	return s
}

/**
* Set
* @params name string, value interface{}
* @return error
**/
func (s *Module) Set(name string, value interface{}) *Jrex {
	return s.jrex.Set(name, value)
}

/**
* SetName
* @params name string
* @return *Module
**/
func (s *Module) SetPath(path string) *Module {
	s.Path = path
	s.ID = fmt.Sprintf("module:%s:%s", s.Path, s.Version)
	return s
}

/**
* BumpVersion
* @param part Part
* @return string
**/
func (s *Module) SetVersion(part Part) *Module {
	parts := strings.Split(s.Version, ".")
	if len(parts) != 3 {
		return s
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
		return s
	}

	s.Version = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	s.ID = fmt.Sprintf("module:%s:%s", s.Path, s.Version)
	return s
}

/**
* SetMetadata
* @param metadata et.Json
**/
func (s *Module) SetMetadata(metadata et.Json) {
	s.Metadata = metadata
}
