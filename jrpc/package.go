package jrpc

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type Solver struct {
	Method string   `json:"method"`
	Inputs []string `json:"inputs"`
	Output []string `json:"output"`
}

type Package struct {
	Name    string    `json:"name"`
	Host    string    `json:"host"`
	Port    int       `json:"port"`
	Solvers []*Solver `json:"routes"`
}

/**
* NewPackage
* @param name string, host string, port int
* @return *Package
**/
func NewPackage(name string, host string, port int) *Package {
	return &Package{
		Name:    name,
		Host:    host,
		Port:    port,
		Solvers: make([]*Solver, 0),
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Package) ToJson() et.Json {
	dt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(dt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* Mount
* @param services any
* @return error
**/
func (s *Package) Mount(services any) error {
	solvers, err := Mount(s.Host, services)
	if err != nil {
		return err
	}

	for method, solver := range solvers {
		s.Solvers = append(s.Solvers, &Solver{
			Method: method,
			Inputs: solver.ArrayStr("inputs"),
			Output: solver.ArrayStr("output"),
		})
	}

	return nil
}
