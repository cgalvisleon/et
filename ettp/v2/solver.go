package ettp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
)

type TypeRouter int

const (
	TpHandler TypeRouter = iota + 1
	TpApiRest
	TpStatic // files of an origin (a frontend or a plugin): proxied or redirected, with the cache policy of handlerStatic
)

// The modes of a static route: the gateway copies the file (proxy) or answers 302 to the origin (redirect); the entry
// files (index.html, mf-manifest.json, remoteEntry.js) are always proxied, so the page and the manifest keep the origin
// of the gateway
const (
	ModeProxy    = "proxy"
	ModeRedirect = "redirect"
)

/**
* String: Name of the route kind ("handler" or "app").
* @return string
**/
func (t TypeRouter) String() string {
	switch t {
	case TpHandler:
		return "handler"
	case TpApiRest:
		return "app"
	case TpStatic:
		return "static"
	default:
		return "Unknown"
	}
}

func StringToTypeRouter(s string) TypeRouter {
	switch s {
	case "api":
		return TpApiRest
	case "static":
		return TpStatic
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

/**
* Solver: One route of the gateway (ID = METHOD:path): a local handler (TpHandler), a proxied URL (TpApiRest)
* with its header policy, version and package, or the files of an origin (TpStatic, path and resolve ending in /*)
* with its Mode and, for a SPA, the Fallback file of the paths without an extension.
**/
type Solver struct {
	ID            string                            `json:"id"`
	Kind          TypeRouter                        `json:"kind"`
	Method        string                            `json:"method"`
	Path          string                            `json:"path"`
	Solver        string                            `json:"solver"`
	TypeHeader    TpHeader                          `json:"type_header"`
	Header        map[string]string                 `json:"header"`
	ExcludeHeader []string                          `json:"exclude_header"`
	Version       int                               `json:"version"`
	PackageName   string                            `json:"package_name"`
	Mode          string                            `json:"mode,omitempty"`
	Fallback      string                            `json:"fallback,omitempty"`
	middlewares   []func(http.Handler) http.Handler `json:"-"`
	handlerFn     http.HandlerFunc                  `json:"-"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Solver) ToJson() et.Json {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* newSolver
* @param method, path string
* @return *Solver
**/
func newSolver(method, path string) *Solver {
	key := fmt.Sprintf("%s:%s", method, path)
	return &Solver{
		ID:          key,
		Method:      method,
		Path:        path,
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}
