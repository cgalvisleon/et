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
	Tag    string             `json:"tag"`
	Param  string             `json:"param"`
	Solver *Solver            `json:"solver"`
	Router map[string]*Router `json:"router"`
	owner  *Router            `json:"-"`
}

/**
* newRouter
* @param tag string
* @return *Router
**/
func newRouter(tag string) *Router {
	return &Router{
		Tag:    tag,
		Router: make(map[string]*Router),
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
	router.owner = s
	s.Router[router.Tag] = router
	return router
}

/**
* set
* @param kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int
* @return *Solver, error
**/
func (s *Router) set(kind TypeRouter, method, path, solver string, typeHeader TpHeader, header map[string]string, excludeHeader []string, version int) (*Solver, error) {
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
		}
	}

	if target.Solver == nil {
		return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_BUILD, path)
	}

	return target.Solver, nil
}

/**
* find
* @param path string
* @return *Router, error
**/
func (s *Router) find(path string) (*Router, error) {
	tags := strings.Split(path, "/")
	if len(tags) == 0 {
		return nil, fmt.Errorf(msg.MSG_PATH_INVALID, path)
	}

	params := make(map[string]string)
	result := s
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}

		router, ok := result.Router[tag]
		if ok {
			result = router
			continue
		}

		querys := strings.Split(tag, "?")
		if len(querys) > 0 {
			query := querys[0]
			router, ok := result.Router[query]
			if ok {
				result = router
			}

			if result.Solver == nil {
				return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
			}

			if result.Solver.Method != GET {
				return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
			}

			break
		}

		matrix := strings.Split(tag, ";")
		if len(matrix) > 0 {
			matrix := matrix[0]
			router, ok := result.Router[matrix]
			if ok {
				result = router
			}

			if result.Solver == nil {
				return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
			}

			if result.Solver.Method != GET {
				return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
			}

			break
		}

		if result.Param != "" {
			router, ok := result.Router[result.Param]
			if ok {
				result = router
				params[result.Param] = tag
				continue
			}
		}

		return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
	}

	if result.Solver == nil {
		return nil, fmt.Errorf(msg.MSG_SOLVER_NOT_FOUND, path)
	}

	return result, nil
}

/**
* findResolver
* @param r *http.Request
* @return *Resolver, error
**/
func (s *Router) findResolver(req *http.Request) (*Resolver, error) {
	path := req.URL.Path
	target, err := s.find(path)
	if err != nil {
		return nil, err
	}

	return newResolver(req, target.Solver, nil)
}

/**
* delete
* @param path string
* @return bool, error
**/
func (s *Router) delete(path string) (bool, error) {
	target, err := s.find(path)
	if err != nil {
		return false, err
	}

	ok := target.owner != nil
	for ok {
		owner := target.owner
		delete(owner.Router, target.Tag)
		if len(owner.Router) > 0 {
			break
		}
		target = owner
		ok = target.owner != nil
	}

	return true, nil
}
