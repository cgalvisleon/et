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
	server *Server            `json:"-"`
	main   *Router            `json:"-"`
}

/**
* NewRouter
* @param server *Server
* @return *Router
**/
func NewRouter(server *Server, tag string) *Router {
	return &Router{
		Router: make(map[string]*Router),
		Tag:    tag,
		server: server,
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
		"solver": solver,
	}
}

/**
* addRouter
* @param tag string
* @return *Router
**/
func (s *Router) addRouter(tag string) *Router {
	router := NewRouter(s.server, tag)
	s.Router[tag] = router
	router.main = s
	return router
}

/**
* setRouter
* @param kind, method, path, solver string, header et.Json, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Router) setRouter(kind TypeRouter, method, path, parentPath, solver string, header map[string]string, excludeHeader []string, version int, packageName string) (*Solver, error) {
	pkg := s.server.Packages[packageName]
	if pkg == nil {
		pkg = NewPackage(packageName, s.server)
	}

	path = fmt.Sprintf("%s/%s", parentPath, path)
	path = strings.ReplaceAll(path, "//", "/")
	key := fmt.Sprintf("%s:%s", method, path)
	result, ok := s.server.Solvers[key]
	if ok {
		result.Solver = solver
		result.Header = header
		result.ExcludeHeader = excludeHeader
		result.Version = version
		pkg.AddSolver(result)
		return result, nil
	}

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
			target.solver = NewSolver(key, method, path)
			target.solver.Kind = kind
			target.solver.Solver = solver
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

	s.server.Solvers[target.solver.Id] = target.solver
	pkg.AddSolver(target.solver)
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
				target = router
				params[target.Param] = tag
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

/**
* addHandler
* @param method, path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) addHandler(method, path string, handlerFn http.HandlerFunc, packageName string) {
	s.server.setHandler(method, path, handlerFn, packageName)
}

/**
* Get
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Get(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(GET, path, handlerFn, packageName)
}

/**
* Delete
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Post(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(POST, path, handlerFn, packageName)
}

/**
* Put
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Put(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(PUT, path, handlerFn, packageName)
}

/**
* Patch
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Patch(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(PATCH, path, handlerFn, packageName)
}

/**
* Delete
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Delete(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(DELETE, path, handlerFn, packageName)
}

/**
* Head
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Head(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(HEAD, path, handlerFn, packageName)
}

/**
* Options
* @param path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) Options(path string, handlerFn http.HandlerFunc, packageName string) {
	s.addHandler(OPTIONS, path, handlerFn, packageName)
}
