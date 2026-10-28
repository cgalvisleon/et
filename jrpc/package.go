package jrpc

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

/**
* Solver: Where a method is served and its input and output types.
**/
type Solver struct {
	Host   string   `json:"host"`
	Port   int      `json:"port"`
	Inputs []string `json:"inputs"`
	Output []string `json:"output"`
}

/**
* Package: The methods mounted by one service, by "Struct.Method".
**/
type Package struct {
	Name    string             `json:"name"`
	Host    string             `json:"host"`
	Port    int                `json:"port"`
	Solvers map[string]*Solver `json:"routes"`
}

/**
* newPackage
* @param name string, host string, port int
* @return *Package
**/
func newPackage(name, host string, port int) *Package {
	return &Package{
		Name:    name,
		Host:    host,
		Port:    port,
		Solvers: make(map[string]*Solver),
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
* Add
* @param method string, inputs []string, output []string
* @return *Solver
**/
func (s *Package) Add(method string, inputs []string, output []string) *Solver {
	result := &Solver{
		Host:   s.Host,
		Port:   s.Port,
		Inputs: inputs,
		Output: output,
	}
	s.Solvers[method] = result
	return result
}
