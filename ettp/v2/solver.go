package ettp

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
)

type TypeRouter int

const (
	TpHandler TypeRouter = iota + 1
	TpApiRest
)

func (t TypeRouter) String() string {
	switch t {
	case TpHandler:
		return "handler"
	case TpApiRest:
		return "app"
	default:
		return "Unknown"
	}
}

func StringToTypeRouter(s string) TypeRouter {
	switch s {
	case "api":
		return TpApiRest
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

/**
* String
* @return string
**/
func (t TpHeader) String() string {
	switch t {
	case TpKeepHeader:
		return "Keep Header"
	case TpJoinHeader:
		return "Join Header"
	case TpReplaceHeader:
		return "Replace Header"
	default:
		return "Keep Header"
	}
}

type Solver struct {
	Id            string                            `json:"id"`
	Kind          TypeRouter                        `json:"kind"`
	Method        string                            `json:"method"`
	Path          string                            `json:"path"`
	Solver        string                            `json:"solver"`
	TypeHeader    TpHeader                          `json:"type_header"`
	Header        map[string]string                 `json:"header"`
	ExcludeHeader []string                          `json:"exclude_header"`
	Version       int                               `json:"version"`
	Private       bool                              `json:"private"`
	PackageName   string                            `json:"package_name"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
	handlerFn     http.HandlerFunc                  `json:"-"`
}

/**
* ToJson
* @return et.Json
**/
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
		"private":      s.Private,
		"package_name": s.PackageName,
	}
}

/**
* NewSolver
* @param id, method, path string
* @return *Solver
**/
func NewSolver(method, path string) *Solver {
	key := fmt.Sprintf("%s:%s", method, path)
	return &Solver{
		Id:          key,
		Method:      method,
		Path:        path,
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}
