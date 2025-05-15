package ettp

import (
	"slices"
	"strings"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/router"
)

/**
* setRoute
* @param id, method, path, resolve string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, packageName string, save bool
* @return *Router
**/
func (s *Server) setRoute(id, method, path, resolve string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, packageName string, save bool) *Router {
	if len(method) == 0 {
		return nil
	}

	if len(path) == 0 {
		return nil
	}

	confirm := func(action string) {
		console.Logf(s.Name, `[%s] %s:%s -> %s | TpHeader:%s | Private:%v | %s`, action, method, path, resolve, tpHeader.String(), private, packageName)
	}

	var router *Router
	idx := slices.IndexFunc(s.solvers, func(e *Router) bool { return e.Id == id })
	if idx != -1 {
		router = s.solvers[idx]
		router.Id = id
		router.Resolve = resolve
		router.Kind = kind
		router.Header = header
		router.TpHeader = tpHeader
		router.ExcludeHeader = excludeHeader
		router.Private = private

		if router.Private {
			router.addMiddleware(router.server.authenticator)
		} else {
			router.removeMiddleware(router.server.authenticator)
		}

		router.SetPakage(packageName)

		confirm("RESET")
	} else {
		idx = getRouteIndex(method, s.router)
		if idx == -1 {
			router, s.router = newRoute(s, method, s.router, packageName)
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

			find := router.find(tag)
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
				router = router.addRoute(id, method, tag, kind, header, tpHeader, excludeHeader, private, tpParams)
			} else {
				router = find
			}

			if i == n-1 && router != nil {
				router.Path = path
				router.Resolve = resolve
				router.SetPakage(packageName)
				s.solvers = append(s.solvers, router)
			}
		}

		confirm("SET")
	}

	if save {
		go s.Save()
	}

	return router
}

/**
* GetRouteById
* @param id string
* @return *Router
**/
func (s *Server) GetRouteById(id string) *Router {
	idx := slices.IndexFunc(s.solvers, func(e *Router) bool { return e.Id == id })
	if idx == -1 {
		return nil
	}

	return s.solvers[idx]
}

/**
* DeleteRouteById
* @param id string
* @return error
**/
func (s *Server) DeleteRouteById(id string, save bool) error {
	idx := slices.IndexFunc(s.solvers, func(e *Router) bool { return e.Id == id })
	if idx == -1 {
		return mistake.New(MSG_ROUTE_NOT_FOUND)
	}

	router := s.solvers[idx]
	pkg := router.pkg
	if pkg != nil {
		pkg.deleteRouteById(id)
	}

	method := router.Method
	err := s.deleteRouteByMethod(method, id)
	if err != nil {
		return err
	}

	console.Logf("Api gateway", `[DELETE] %s:%s -> %s`, router.Method, router.Path, router.Resolve)
	s.solvers = append(s.solvers[:idx], s.solvers[idx+1:]...)

	if save {
		go s.Save()
	}

	return nil
}

/**
* deleteRouteByMethod
* @param method, id string
* @return error
**/
func (s *Server) deleteRouteByMethod(method, id string) error {
	idx := slices.IndexFunc(s.router, func(e *Router) bool { return e.Tag == method })
	if idx == -1 {
		return console.Alertm("Method route not found")
	}

	router := s.router[idx]
	ok := router.deleteById(id, true)
	if !ok {
		return console.Alertm("Route not found")
	}

	return nil
}

/**
* SetResolve
* @param private bool, id, method, path, resolve string, header et.Json, tpHeader router.TpHeader, excludeHeader []string, packageName string, saved bool
* @return *Router, error
**/
func (s *Server) SetRouter(private bool, id, method, path, resolve string, header et.Json, tpHeader router.TpHeader, excludeHeader []string, packageName string, saved bool) (*Router, error) {
	method = strings.ToUpper(method)
	ok := methodMap[method]
	if !ok {
		return nil, console.Alertf(MSG_METHOD_NOT_FOUND, method)
	}

	route := s.setRoute(id, method, path, resolve, TpRest, header, tpHeader, excludeHeader, private, packageName, saved)
	if route == nil {
		return nil, mistake.New(MSG_ROUTE_NOT_REGISTER)
	}

	return route, nil
}
