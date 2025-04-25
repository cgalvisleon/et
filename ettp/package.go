package ettp

import (
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

type MiniRoute struct {
	Id       string
	Method   string
	Path     string
	URL      string
	Header   et.Json
	TpHeader string
	Exclude  []string
	Private  bool
	Resolve  string
}

func (r *MiniRoute) ToJson() et.Json {
	return et.Json{
		"Id":       r.Id,
		"Method":   r.Method,
		"Path":     r.Path,
		"URL":      r.URL,
		"Header":   r.Header,
		"TpHeader": r.TpHeader,
		"Exclude":  r.Exclude,
		"Private":  r.Private,
		"Resolve":  r.Resolve,
	}
}

type Package struct {
	server *Server
	Id     string
	Name   string
	Routes map[string]*MiniRoute
}

/**
* ToJson
* @return et.Json
**/
func (p *Package) ToJson() et.Json {
	var routes []et.Json
	for _, route := range p.Routes {
		routes = append(routes, route.ToJson())
	}

	result := et.Json{
		"Id":     p.Id,
		"Name":   p.Name,
		"Count":  len(routes),
		"Routes": routes,
	}
	return result
}

/**
* AddRoute
* @param method string
* @param path string
* @param route *Router
* @return *Package
**/
func (p *Package) AddRoute(method, path string, route *Router) *Package {
	url := strs.Format(`[%s]:%s`, method, path)
	if route.ExcludeHeader == nil {
		route.ExcludeHeader = []string{}
	}

	miniRoute := &MiniRoute{
		Id:       route.Id,
		Method:   method,
		Path:     path,
		URL:      url,
		Resolve:  route.Resolve,
		Header:   route.Header,
		TpHeader: route.TpHeader.String(),
		Exclude:  route.ExcludeHeader,
		Private:  route.Private,
	}
	p.Routes[url] = miniRoute

	return p
}

/**
* DeleteRoute
* @param method string
* @param path string
* @return bool
**/
func (p *Package) DeleteRoute(method, path string) bool {
	key := strs.Format(`[%s]:%s`, method, path)
	delete(p.Routes, key)

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
			delete(p.Routes, route.URL)
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
* findPakage
* @param name string
* @return *Package
**/
func findPakage(s *Server, name string) *Package {
	idx := slices.IndexFunc(s.packages, func(e *Package) bool { return strs.Lowcase(e.Name) == strs.Lowcase(name) })
	if idx == -1 {
		return nil
	}

	return s.packages[idx]
}

/**
* newPakage
* @param server *Server
* @param name string
* @return *Package
**/
func newPakage(server *Server, name string) *Package {
	result := &Package{
		Id:     utility.UUID(),
		server: server,
		Name:   name,
		Routes: make(map[string]*MiniRoute),
	}

	server.packages = append(server.packages, result)

	return result
}
