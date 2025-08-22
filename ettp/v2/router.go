package ettp

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/et"
)

const (
	GET     = "GET"
	POST    = "POST"
	PUT     = "PUT"
	PATCH   = "PATCH"
	DELETE  = "DELETE"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
	RPC     = "RPC"
)

var methodMap = map[string]bool{
	GET:     true,
	POST:    true,
	PUT:     true,
	PATCH:   true,
	DELETE:  true,
	HEAD:    true,
	OPTIONS: true,
	RPC:     true,
}

type Router struct {
	Router map[string]*Router `json:"router"`
	Tag    string             `json:"tag"`
	Param  string             `json:"param"`
	solver *Solver            `json:"-"`
	main   *Router            `json:"-"`
}

/**
* NewRouter
* @param tag string
* @return *Router
**/
func NewRouter(tag string) *Router {
	return &Router{
		Router: make(map[string]*Router),
		Tag:    tag,
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Router) ToJson() et.Json {
	solver := et.Json{}
	if s.solver != nil {
		solver = s.solver.ToJson()
	}

	return et.Json{
		"router": s.Router,
		"tag":    s.Tag,
		"param":  s.Param,
		"solver": solver,
	}
}

/**
* addRouter
* @param tag string
* @return *Router
**/
func (s *Router) addRouter(tag string) *Router {
	router := NewRouter(tag)
	s.Router[tag] = router
	router.main = s
	return router
}

/**
* setRouter
* @param kind, method, path, solver string, typeHeader TpHeader, header et.Json, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Router) setRouter(kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, packageName string) (*Solver, error) {
	tags := strings.Split(path, "/")
	if len(tags) == 0 {
		return nil, fmt.Errorf("path %s is invalid", path)
	}

	regex := regexp.MustCompile(`^\{.*\}$`)
	isParam := func(tag string) bool {
		return regex.MatchString(tag)
	}

	target := s
	for i, tag := range tags {
		if tag == "" {
			continue
		}

		router, ok := target.Router[tag]
		if ok {
			target = router
			continue
		}

		target = target.addRouter(tag)
		if isParam(tag) {
			target.main.Param = tag
		}

		if i == len(tags)-1 {
			target.solver = NewSolver(method, path)
			target.solver.Kind = kind
			target.solver.Solver = solver
			target.solver.TypeHeader = typeHeader
			target.solver.Header = header
			target.solver.ExcludeHeader = excludeHeader
			target.solver.Version = version
			target.solver.PackageName = packageName
			target.solver.router = target
		}
	}

	if target.solver == nil {
		return nil, fmt.Errorf("solver %s not build", path)
	}

	return target.solver, nil
}

/**
* getResolver
* @param id string, solver *Solver, url string
* @return *Solver, error
**/
func (s *Router) getResolver(r *http.Request, solver *Solver, params map[string]string) (*Resolver, error) {
	if solver == nil {
		return nil, fmt.Errorf("solver not found")
	}

	result := NewResolver(r, solver, params)
	return result, nil
}

/**
* findResolver
* @param r *http.Request
* @return *Resolver, error
**/
func (s *Router) findResolver(req *http.Request) (*Resolver, error) {
	path := req.URL.Path
	tags := strings.Split(path, "/")
	if len(tags) == 0 {
		return nil, fmt.Errorf("path:%s is invalid", path)
	}

	params := make(map[string]string)
	target := s
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}

		router, ok := target.Router[tag]
		if ok {
			target = router
			continue
		}

		if tag[0] == '?' {
			break
		}

		querys := strings.Split(tag, "?")
		if len(querys) > 1 {
			query := querys[1]
			router, ok := s.Router[query]
			if ok {
				target = router
				break
			}
		}

		matrix := strings.Split(tag, ";")
		if len(matrix) > 1 {
			matrix := matrix[1]
			router, ok := s.Router[matrix]
			if ok {
				target = router
				break
			}
		}

		if target.Param != "" {
			router, ok := target.Router[target.Param]
			if ok {
				params[target.Param] = tag
				target = router
				continue
			}
		}

		return nil, fmt.Errorf("solver:%s not found tag:%s", path, tag)
	}

	if target == nil {
		return nil, fmt.Errorf("solver:%s not found", path)
	}

	return s.getResolver(req, target.solver, params)
}
