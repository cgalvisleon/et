package jrpc

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

type Solver struct {
	Host       string   `json:"host"`
	Port       int      `json:"port"`
	StructName string   `json:"structName"`
	Method     string   `json:"method"`
	Inputs     et.Json  `json:"inputs"`
	Output     []string `json:"output"`
}

/**
* serialize
* @return et.Json
**/
func (s *Solver) serialize() et.Json {
	return et.Json{
		"host":       s.Host,
		"port":       s.Port,
		"structName": s.StructName,
		"method":     s.Method,
		"inputs":     s.Inputs,
		"outputs":    s.Output,
	}
}

/**
* getSolver
* @param method string
* @return *Solver
* @return error
**/
func getSolver(method string) (*Solver, error) {
	lst := strings.Split(method, ".")
	if len(lst) != 2 {
		return nil, fmt.Errorf(msg.MSG_METHOD_NOT_FOUND, method)
	}

	packageName := lst[0]
	methodName := lst[1]
	packages, err := getPackages()
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(packages, func(p *Package) bool { return p.Name == packageName })
	if idx == -1 {
		return nil, fmt.Errorf(msg.MSG_ERR_PACKAGE_NOT_FOUND, packageName)
	}

	pkg := packages[idx]
	idx = slices.IndexFunc(pkg.Solvers, func(s *Solver) bool { return s.Method == methodName })
	if idx == -1 {
		return nil, fmt.Errorf(msg.MSG_ERR_METHOD_NOT_FOUND, method)
	}

	solver := pkg.Solvers[idx]
	if solver == nil {
		return nil, fmt.Errorf(msg.MSG_ERR_METHOD_NOT_FOUND, method)
	}

	return solver, nil
}
