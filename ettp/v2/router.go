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
* setSolver
* @param method, path, mainPath, solver string, header et.Json, excludeHeader []string, version int, packageName string
* @return *Solver, error
**/
func (s *Router) setSolver(kind TypeApi, id, method, path, pathApi, solver string, header map[string]string, excludeHeader []string, version int, packageName string) (*Solver, error) {
	url := pathApi + path
	tags := strings.Split(url, "/")
	if len(tags) == 0 {
		return nil, fmt.Errorf("path %s is invalid", url)
	}

	regex := regexp.MustCompile(`^\{.*\}$`)
	isParam := func(tag string) bool {
		return regex.MatchString(tag)
	}

	target := s
	for i, tag := range tags {
		router, ok := target.Router[tag]
		if ok {
			target = router
			continue
		}

		target = target.addRouter(tag)
		if isParam(tag) {
			target.Param = tag
		}

		if i == len(tags)-1 {
			target.solver = NewSolver(id, method, path)
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

	return target.solver, nil
}

/**
* getSolver
* @param id string, solver *Solver, url string
* @return *Solver, error
**/
func (s *Router) getRequest(r *http.Request, solver *Solver, params map[string]string) (*Request, error) {
	if solver == nil {
		return nil, fmt.Errorf("solver not found")
	}

	result := NewRequest(r, solver, params)
	return result, nil
}

/**
* findRequest
* @param r *http.Request
* @return *Solver, error
**/
func (s *Router) findRequest(req *http.Request) (*Request, error) {
	url := req.URL.Path
	tags := strings.Split(url, "/")
	if len(tags) == 0 {
		return nil, fmt.Errorf("url %s is invalid", url)
	}

	params := make(map[string]string)
	target := s
	for _, tag := range tags {
		if len(tag) == 0 {
			return nil, fmt.Errorf("tag %s is invalid", tag)
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

		if s.Param != "" {
			router, ok := s.Router[s.Param]
			if ok {
				target = router
				params[s.Param] = tag
				continue
			}
		}

		return nil, fmt.Errorf("solver %s not found", url)
	}

	if target == nil {
		return nil, fmt.Errorf("solver %s not found", url)
	}

	return s.getRequest(req, target.solver, params)
}

/**
* addHandler
* @param method, path string, handlerFn http.HandlerFunc, packageName string
**/
func (s *Router) addHandler(method, path string, handlerFn http.HandlerFunc, packageName string) {
	key := fmt.Sprintf("%s:%s", method, path)

	s.server.handlers[key] = handlerFn
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
