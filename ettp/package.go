package ettp

import (
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

type Package struct {
	server *Server
	Id     string
	Name   string
	Routes map[string]*Router
}

/**
* Describe
* @return et.Json
**/
func (p *Package) Describe() et.Json {
	result := et.Json{
		"Id":     p.Id,
		"Name":   p.Name,
		"Count":  len(p.Routes),
		"Routes": p.Routes,
	}
	return result
}

/**
* addRouter
* @param route *Router
* @return *Package
**/
func (p *Package) addRouter(route *Router) *Package {
	if route.ExcludeHeader == nil {
		route.ExcludeHeader = []string{}
	}

	p.Routes[route.key()] = route

	return p
}

/**
* deleteRoute
* @param route *Router
* @return bool
**/
func (p *Package) deleteRoute(route *Router) bool {
	delete(p.Routes, route.key())

	return true
}

/**
* deleteRouteById
* @param id string
* @return bool
**/
func (p *Package) deleteRouteById(id string) bool {
	result := false
	for _, route := range p.Routes {
		if route.Id == id {
			s := p.server
			p.deleteRoute(route)
			if len(p.Routes) == 0 {
				idx := slices.IndexFunc(s.packages, func(e *Package) bool { return strs.Lowcase(e.Name) == strs.Lowcase(p.Name) })
				if idx != -1 {
					s.packages = append(s.packages[:idx], s.packages[idx+1:]...)
				}
			}
			break
		}
	}

	return result
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

/**
* newPakage
* @param server *Server, name string
* @return *Package
**/
func newPakage(server *Server, name string) *Package {
	result := &Package{
		Id:     utility.UUID(),
		server: server,
		Name:   name,
		Routes: make(map[string]*Router),
	}

	server.packages = append(server.packages, result)

	return result
}
