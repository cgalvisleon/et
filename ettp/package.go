package ettp

import (
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

type Package struct {
	server  *Server   `json:"-"`
	Id      string    `json:"id"`
	Name    string    `json:"name"`
	routes  []*Router `json:"-"`
	proxies []*Proxy  `json:"-"`
}

/**
* newPakage
* @param server *Server, name string
* @return *Package
**/
func newPakage(server *Server, name string) *Package {
	result := &Package{
		Id:      utility.UUID(),
		server:  server,
		Name:    name,
		routes:  []*Router{},
		proxies: []*Proxy{},
	}

	server.packages = append(server.packages, result)

	return result
}

/**
* Describe
* @return et.Json
**/
func (s *Package) Describe() et.Json {
	result := et.Json{
		"id":   s.Id,
		"name": s.Name,
		"routes": et.Json{
			"count": len(s.routes),
			"items": s.routes,
		},
		"proxies": et.Json{
			"count": len(s.proxies),
			"items": s.proxies,
		},
	}
	return result
}

/**
* addRouter
* @param route *Router
* @return *Package
**/
func (s *Package) addRouter(route *Router) *Package {
	if route.ExcludeHeader == nil {
		route.ExcludeHeader = []string{}
	}

	idx := slices.IndexFunc(s.routes, func(e *Router) bool { return e.Id == route.Id })
	if idx != -1 {
		s.routes[idx] = route
	} else {
		s.routes = append(s.routes, route)
	}

	return s
}

/**
* addProxy
* @param proxy *Proxy
* @return *Package
**/
func (s *Package) addProxy(proxy *Proxy) *Package {
	idx := slices.IndexFunc(s.proxies, func(e *Proxy) bool { return e.Path == proxy.Path })
	if idx != -1 {
		s.proxies[idx] = proxy
	} else {
		s.proxies = append(s.proxies, proxy)
	}

	return s
}

/**
* deleteRoute
* @param route *Router
* @return bool
**/
func (s *Package) deleteRoute(route *Router) bool {
	idx := slices.IndexFunc(s.routes, func(e *Router) bool { return e.Id == route.Id })
	if idx != -1 {
		s.routes = append(s.routes[:idx], s.routes[idx+1:]...)
	}

	return true
}

/**
* deleteProxy
* @param proxy *Proxy
* @return bool
**/
func (s *Package) deleteProxy(proxy *Proxy) bool {
	idx := slices.IndexFunc(s.proxies, func(e *Proxy) bool { return e.Path == proxy.Path })
	if idx != -1 {
		s.proxies = append(s.proxies[:idx], s.proxies[idx+1:]...)
	}

	return true
}

/**
* deleteRouteById
* @param id string
* @return bool
**/
func (s *Package) deleteRouteById(id string) bool {
	idx := slices.IndexFunc(s.routes, func(e *Router) bool { return e.Id == id })
	if idx == -1 {
		return false
	}

	s.routes = append(s.routes[:idx], s.routes[idx+1:]...)
	if len(s.routes) == 0 {
		idx := slices.IndexFunc(s.server.packages, func(e *Package) bool { return strs.Lowcase(e.Id) == strs.Lowcase(s.Id) })
		if idx != -1 {
			s.server.packages = append(s.server.packages[:idx], s.server.packages[idx+1:]...)
		}
	}

	return true
}

/**
* deleteProxyById
* @param id string
* @return bool
**/
func (s *Package) deleteProxyById(id string) bool {
	idx := slices.IndexFunc(s.proxies, func(e *Proxy) bool { return e.Id == id })
	if idx == -1 {
		return false
	}

	s.proxies = append(s.proxies[:idx], s.proxies[idx+1:]...)

	return true
}

/**
* getPackageByName
* @param name string
* @return *Package
**/
func getPackageByName(s *Server, name string) *Package {
	idx := slices.IndexFunc(s.packages, func(e *Package) bool { return strs.Lowcase(e.Name) == strs.Lowcase(name) })
	if idx == -1 {
		return nil
	}

	return s.packages[idx]
}
