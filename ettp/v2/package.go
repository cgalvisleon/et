package ettp

import (
	"slices"

	"github.com/cgalvisleon/et/et"
)

type Package struct {
	Name    string    `json:"name"`
	Solvers []*Solver `json:"solvers"`
	server  *Server   `json:"-"`
}

/**
* NewPackage
* @param name string, server *Server
* @return *Package
**/
func NewPackage(name string, server *Server) *Package {
	result := &Package{
		Name:    name,
		Solvers: make([]*Solver, 0),
		server:  server,
	}

	server.Packages[name] = result
	return result
}

/**
* ToJson
* @return et.Json
**/
func (p *Package) ToJson() et.Json {
	solvers := make([]et.Json, 0)
	for _, solver := range p.Solvers {
		solvers = append(solvers, solver.ToJson())
	}

	return et.Json{
		"name":    p.Name,
		"solvers": solvers,
	}
}

/**
* AddSolver
* @param solver *Solver
**/
func (p *Package) AddSolver(solver *Solver) {
	idx := slices.IndexFunc(p.Solvers, func(s *Solver) bool {
		return s.Id == solver.Id
	})

	if idx == -1 {
		if solver.PackageName != p.Name {
			oldPackage := p.server.Packages[solver.PackageName]
			if oldPackage != nil {
				oldPackage.RemoveSolver(solver)
			}
		}

		solver.PackageName = p.Name
		p.Solvers = append(p.Solvers, solver)
	}
}

/**
* RemoveSolver
* @param solver *Solver
**/
func (p *Package) RemoveSolver(solver *Solver) {
	idx := slices.IndexFunc(p.Solvers, func(s *Solver) bool {
		return s.Id == solver.Id
	})

	if idx != -1 {
		p.Solvers = slices.Delete(p.Solvers, idx, 1)
	}
}
