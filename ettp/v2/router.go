package ettp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
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

var methods = map[string]bool{
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
	Solver *Solver            `json:"solver"`
}

/**
* newRouter
* @param tag string
* @return *Router
**/
func newRouter(tag string) *Router {
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
* add
* @param tag string
* @return *Router
**/
func (s *Router) add(tag string) *Router {
	router := newRouter(tag)
	s.Router[tag] = router
	return router
}

/**
* setRouter
* @param kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int, packageName string
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
		if !ok {
			if isParam(tag) {
				target.Param = tag
			}
			router = target.add(tag)
		}

		target = router
		if i == len(tags)-1 {
			target.Solver = newSolver(method, path)
			target.Solver.Kind = kind
			target.Solver.Solver = solver
			target.Solver.TypeHeader = typeHeader
			target.Solver.Header = header
			target.Solver.ExcludeHeader = excludeHeader
			target.Solver.Version = version
			target.Solver.PackageName = packageName
		}
	}

	if target.Solver == nil {
		return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_BUILD, path)
	}

	return target.Solver, nil
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
		return nil, fmt.Errorf(msg.MSG_PATH_INVALID, path)
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

		return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND_TAG, path, tag)
	}

	if target.Solver == nil {
		return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
	}

	return s.getResolver(req, target.Solver, params)
}

/**
* getResolver
* @param id string, solver *Solver, url string
* @return *Solver, error
**/
func (s *Router) getResolver(r *http.Request, solver *Solver, params map[string]string) (*Resolver, error) {
	result := newResolver(r, solver, params)
	return result, nil
}
