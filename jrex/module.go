package jrex

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/utility"
	"github.com/dop251/goja"
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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	License     string `json:"license"`
	Code        string `json:"code"`
	jrex        *Jrex  `json:"-"`
}

func NewModule(name string) *Module {
	name = utility.Normalize(name)
	version := "1.0.0"
	id := fmt.Sprintf("module:%s:%s", name, version)
	return &Module{
		ID:          id,
		Name:        name,
		Version:     version,
		Description: "",
		Author:      "",
		License:     "MIT",
		Code:        "",
	}
}

/**
* up
* @param jrex *Jrex
* @return *Module
**/
func (s *Module) up(jrex *Jrex) *Module {
	s.jrex = jrex
	s.jrex.Modules[s.Name] = s
	return s
}

func (s *Jrex) GetModule(name string) *Module {
	name = utility.Normalize(name)
	version := "1.0.0"
	id := fmt.Sprintf("module:%s:%s", name, version)
	return &Module{
		ID:          id,
		Name:        name,
		Version:     version,
		Description: "",
		Author:      "",
		License:     "MIT",
		Code:        "",
		jrex:        s,
	}
}

func (s *Jrex) AddModule(name string) *Module {
	name = utility.Normalize(name)
	version := "1.0.0"
	id := fmt.Sprintf("module:%s:%s", name, version)
	result := &Module{
		ID:          id,
		Name:        name,
		Version:     version,
		Description: "",
		Author:      "",
		License:     "MIT",
		Code:        "",
		jrex:        s,
	}
	s.Modules[name] = result
	return result
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
* Error
* @param err error
* @return *goja.Object
**/
func (s *Module) Error(err error) *goja.Object {
	return s.jrex.Error(err)
}

/**
* SetName
* @params name string
* @return *Module
**/
func (s *Module) SetName(name string) *Module {
	s.Name = name
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

	result := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	s.ID = fmt.Sprintf("%s:%s", s.Name, result)
	s.Version = result
	return s
}

func (s *Module) SetDescription(description string) *Module {
	s.Description = description
	return s
}

func (s *Module) SetAuthor(author string) *Module {
	s.Author = author
	return s
}

func (s *Module) SetLicense(license string) *Module {
	s.License = license
	return s
}

func (s *Module) SetCode(code string) *Module {
	s.Code = code
	return s
}
