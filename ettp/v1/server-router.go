package ettp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	rt "github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/utility"
)

/**
* setRouter
* @param method, path, resolve string, kind TypeApi, header et.Json, tpHeader rt.TpHeader, excludeHeader []string, private bool, packageName string, save bool
* @return *Router
**/
func (s *Server) setRouter(method, path, resolve string, kind TypeApi, header et.Json, tpHeader rt.TpHeader, excludeHeader []string, private bool, packageName string, save bool) (*Router, error) {
	if !utility.ValidStr(method, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "method")
	}

	if !utility.ValidStr(path, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "path")
	}

	if !utility.ValidStr(resolve, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ATRIB_REQUIRED, "resolve")
	}

	key := fmt.Sprintf("%s:%s", method, path)
	confirm := func(action string) {
		logs.Logf(s.Name, `[%s] %s:%s -> %s | TpHeader:%s | Private:%v | %s`, action, method, path, resolve, tpHeader.String(), private, packageName)
	}

	var router *Router
	idx := slices.IndexFunc(s.solvers, func(e *Router) bool { return e.Id == key })
	if idx != -1 {
		router = s.solvers[idx]
		router.Id = key
		router.Resolve = resolve
		router.Kind = kind
		router.Header = header
		router.TpHeader = tpHeader
		router.ExcludeHeader = excludeHeader
		router.Private = private

		confirm("RESET")
	} else {
		idx = getRouterIndexByTag(method, s.router)
		if idx == -1 {
			router = s.newRouter(method, packageName)
		} else {
			router = s.router[idx]
		}

		path = strings.TrimSuffix(path, "/")
		tags := strings.Split(path, "/")
		n := len(tags)
		for i := 0; i < n; i++ {
			tag := tags[i]
			if len(tag) == 0 {
				continue
			}

			find := router.getRouterByTag(tag)
			if find == nil {
				tpParams := getTpParams(tag)
				switch tpParams {
				case TpQueryParams:
					querys := strings.Split(tag, "?")
					tag = querys[0]
				case TpMatrixParams:
					matrix := strings.Split(tag, ";")
					tag = matrix[0]
				}
				router = router.addRoute(method, tag, kind, header, tpHeader, excludeHeader, private, tpParams)
			} else {
				router = find
			}

			if i == n-1 && router != nil {
				router.Path = path
				router.Resolve = resolve
				s.solvers = append(s.solvers, router)
			}
		}

		confirm("SET")
	}

	if router != nil {
		router.setPakage(packageName)
	}

	if save {
		go s.Save()
	}

	return router, nil
}

/**
* GetRouteById
* @param id string
* @return *Router
**/
func (s *Server) GetRouteById(id string) *Router {
	for _, router := range s.router {
		find := router.getRouterById(id)
		if find != nil {
			return find
		}
	}

	return nil
}

/**
* DeleteRouteById
* @param id string
* @return error
**/
func (s *Server) DeleteRouteById(id string, save bool) error {
	router := s.GetRouteById(id)
	if router == nil {
		return fmt.Errorf(MSG_ROUTE_NOT_FOUND)
	}

	idx := slices.IndexFunc(s.solvers, func(e *Router) bool { return e.Id == id })
	if idx != -1 {
		s.solvers = append(s.solvers[:idx], s.solvers[idx+1:]...)
	}

	if router.main != nil {
		idx := slices.IndexFunc(router.main.Routes, func(e *Router) bool { return e.Id == id })
		if idx != -1 {
			logs.Logf(packageName, `[DELETE] %s:%s -> %s`, router.Method, router.Path, router.Resolve)
			router.main.Routes = append(router.main.Routes[:idx], router.main.Routes[idx+1:]...)
		}
	}

	if save {
		go s.Save()
	}

	return nil
}

/**
* SetRouter
* @param private bool, method, path, resolve string, header et.Json, tpHeader rt.TpHeader, excludeHeader []string, packageName string, saved bool
* @return *Router, error
**/
func (s *Server) SetRouter(private bool, method, path, resolve string, header et.Json, tpHeader rt.TpHeader, excludeHeader []string, packageName string, saved bool) (*Router, error) {
	method = strings.ToUpper(method)
	ok := methodMap[method]
	if !ok {
		return nil, logs.Alertf(MSG_METHOD_NOT_FOUND, method)
	}

	route, err := s.setRouter(method, path, resolve, TpApiRest, header, tpHeader, excludeHeader, private, packageName, saved)
	if err != nil {
		return nil, fmt.Errorf(MSG_ROUTE_NOT_REGISTER)
	}

	event.Publish(rt.EVENT_SET_ROUTER, et.Json{
		"private":        private,
		"id":             route.Id,
		"method":         method,
		"path":           path,
		"resolve":        resolve,
		"header":         header,
		"tp_header":      tpHeader,
		"exclude_header": excludeHeader,
		"package_name":   packageName,
	})

	return route, nil
}
