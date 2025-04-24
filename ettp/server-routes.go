package ettp

import (
	"net/http"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
)

/**
* UpsetRoute
* @param id, method, path, resolve string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, packageName string, save bool
* @return *Route
**/
func (s *Server) UpsetRoute(id, method, path, resolve string, kind TypeApi, header et.Json, tpHeader router.TpHeader, excludeHeader []string, private bool, packageName string, save bool) *Route {
	if len(method) == 0 {
		return nil
	}

	if len(path) == 0 {
		return nil
	}

	confirm := func() {
		console.Logf(ServiceName, `[SET] %s:%s -> %s | TpHeader:%s | Private:%v | %s`, method, path, resolve, tpHeader.String(), private, packageName)
	}

	var router *Route
	idx := slices.IndexFunc(s.solvers, func(e *Route) bool { return e.Method == method && e.Path == path })
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
	} else {
		idx = indexRoute(method, s.router)
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
				router = router.newRoute(id, method, tag, kind, header, tpHeader, excludeHeader, private, tpParams)
			} else {
				router = find
			}

			if i == n-1 && router != nil {
				router.Path = path
				router.Resolve = resolve
				router.SetPakage(packageName)
				s.solvers = append(s.solvers, router)

				confirm()
			}
		}
	}

	if save {
		go s.Save()
	}

	return router
}

/**
* setHandlerFunc
* @param method string
* @param path string
* @param handlerFn http.HandlerFunc
* @param packageName string
**/
func (s *Server) setHandlerFunc(method, path string, handlerFn http.HandlerFunc, packageName string) *Route {
	method = strs.Uppcase(method)
	ok := methodMap[method]
	if !ok {
		console.Alertf(`'%s' http method is not supported.`, method)
		return nil
	}

	url := strs.Format("%s%s", s.host, path)
	url = strings.ReplaceAll(url, "//", "/")
	id := strs.Format("%s:%s", method, url)

	route := s.UpsetRoute(id, method, url, url, TpHandler, et.Json{}, router.TpReplaceHeader, []string{}, false, packageName, false)
	if route != nil {
		s.handlers[route.Id] = handlerFn
	}

	return route
}

/**
* loadRoute
* @param route *Route
* @return *Route
**/
func (s *Server) loadRoute(route *Route) {
	s.UpsetRoute(
		route.Id,
		route.Method,
		route.Path,
		route.Resolve,
		route.Kind,
		route.Header,
		route.TpHeader,
		route.ExcludeHeader,
		route.Private,
		route.PackageName,
		false,
	)
}

/**
* SetResolve
* @param private bool
* @param id string
* @param method string
* @param path string
* @param resolve string
* @param header et.Json
* @param tpHeader router.TpHeader
* @param packageName string
* @param saved bool
* @return error
**/
func (s *Server) SetResolve(private bool, id, method, path, resolve string, header et.Json, tpHeader router.TpHeader, excludeHeader []string, packageName string, saved bool) (*Route, error) {
	method = strings.ToUpper(method)
	ok := methodMap[method]
	if !ok {
		return nil, console.Alertf(MSG_METHOD_NOT_FOUND, method)
	}

	route := s.UpsetRoute(id, method, path, resolve, TpRest, header, tpHeader, excludeHeader, private, packageName, saved)
	if route == nil {
		return nil, mistake.New(MSG_ROUTE_NOT_REGISTER)
	}

	return route, nil
}
