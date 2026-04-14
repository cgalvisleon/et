package ettp

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type Package struct {
	Name    string             `json:"name"`
	Solvers map[string]*Solver `json:"solvers"`
	server  *Server            `json:"-"`
}

/**
* newPackage
* @param name string, server *Server
* @return *Package
**/
func newPackage(name string, server *Server) *Package {
	result := &Package{
		Name:    name,
		Solvers: make(map[string]*Solver),
		server:  server,
	}

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
**/
func (s *Package) addSolver(solver *Solver) {
	_, exists := s.Solvers[solver.ID]
	if exists {
		s.Solvers[solver.ID] = solver
		return
	}

	s.Solvers[solver.ID] = solver
	if solver.PackageName != s.Name {
		oldPackage := s.server.packages[solver.PackageName]
		if oldPackage != nil {
			oldPackage.removeSolver(solver)
		}
	}

	solver.PackageName = s.Name
}

/**
* removeSolver
* @param solver *Solver
**/
func (s *Package) removeSolver(solver *Solver) {
	_, exists := s.Solvers[solver.ID]
	if !exists {
		return
	}

	delete(s.Solvers, solver.ID)
}
