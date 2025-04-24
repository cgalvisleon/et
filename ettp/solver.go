package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/router"
)

type TypeSolver int

const (
	SolverHandler TypeSolver = iota
	SolverRest
)

func (t TypeSolver) String() string {
	switch t {
	case SolverHandler:
		return "Handler"
	case SolverRest:
		return "Rest"
	default:
		return "Unknown"
	}
}

func ToTypeSolver(i int) TypeSolver {
	switch i {
	case 1:
		return SolverRest
	default:
		return SolverHandler
	}
}

type Solver struct {
	Method        string
	Path          string
	Resolve       string
	FuncHandler   http.HandlerFunc
	Kind          TypeSolver
	Header        et.Json
	TpHeader      router.TpHeader
	ExcludeHeader []string
	Private       bool
	PackageName   string
}
