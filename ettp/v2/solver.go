package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
)

type TypeApi int

const (
	TpHandler TypeApi = iota
	TpApiRest
	TpPortForward
	TpProxy
)

func (t TypeApi) String() string {
	switch t {
	case TpHandler:
		return "Handler"
	case TpApiRest:
		return "Api REST"
	case TpPortForward:
		return "Port Forward"
	case TpProxy:
		return "Proxy"
	default:
		return "Unknown"
	}
}

func IntToTypeApi(i int) TypeApi {
	switch i {
	case 1:
		return TpApiRest
	case 2:
		return TpProxy
	default:
		return TpHandler
	}
}

type TpHeader int

const (
	TpKeepHeader TpHeader = iota
	TpJoinHeader
	TpReplaceHeader
)

func (t TpHeader) String() string {
	switch t {
	case TpKeepHeader:
		return "Keep Header"
	case TpJoinHeader:
		return "Join Header"
	case TpReplaceHeader:
		return "Replace Header"
	}

	return "Unknown"
}

type Solver struct {
	Id            string                            `json:"id"`
	Kind          TypeApi                           `json:"kind"`
	Method        string                            `json:"method"`
	Path          string                            `json:"path"`
	Solver        string                            `json:"solver"`
	TypeHeader    TpHeader                          `json:"type_header"`
	Header        map[string]string                 `json:"header"`
	ExcludeHeader []string                          `json:"exclude"`
	Version       int                               `json:"version"`
	PackageName   string                            `json:"package_name"`
	router        *Router                           `json:"-"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
}

func (s *Solver) ToJson() et.Json {
	return et.Json{
		"id":           s.Id,
		"kind":         s.Kind.String(),
		"method":       s.Method,
		"path":         s.Path,
		"solver":       s.Solver,
		"type_header":  s.TypeHeader.String(),
		"header":       s.Header,
		"exclude":      s.ExcludeHeader,
		"version":      s.Version,
		"package_name": s.PackageName,
	}
}

/**
* NewSolver
* @param id string, solver *Solver, url string
* @return *Solver
**/
func NewSolver(id string, method, path string) *Solver {
	return &Solver{
		Id:          id,
		Method:      method,
		Path:        path,
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}
