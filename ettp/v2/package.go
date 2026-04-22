package ettp

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type Package struct {
	Name   string  `json:"name"`
	Routes et.Json `json:"routes"`
	server *Server `json:"-"`
}

/**
* newPackage
* @param name string, server *Server
* @return *Package
**/
func newPackage(name string, server *Server) *Package {
	result := &Package{
		Name:   name,
		Routes: et.Json{},
		server: server,
	}

	server.Packages[name] = result

	return result
}

/**
* ToJson
* @return (et.Json, error)
**/
func (s *Package) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* addSolver
* @param solver *Solver
* @return *Solver
**/
func (s *Package) addSolver(solver *Solver) *Solver {
	solver.PackageName = s.Name
	s.Routes[solver.ID] = solver.Solver
	return solver
}

/**
* removeSolver
* @param solver *Solver
**/
func (s *Package) removeSolver(solver *Solver) {
	delete(s.Routes, solver.ID)
}
