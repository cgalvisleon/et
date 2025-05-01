package ettp

import (
	"slices"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

/**
* developToken
* @return string
**/
func developToken() string {
	production := config.App.Production
	if production {
		return ""
	}

	device := "DevelopToken"
	duration := time.Hour * 24 * 7
	token, err := claim.NewToken(device, device, device, device, device, duration)
	if err != nil {
		console.Alert(err)
		return ""
	}

	_, err = claim.ValidToken(token)
	if err != nil {
		console.Alertf("GetFromToken:%s", err.Error())
		return ""
	}

	return token
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
	err := s.deleteRoute(method, id)
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
* GetPackages
* @return et.Items
**/
func (s *Server) GetPackages(name string) et.Items {
	var result = []et.Json{}
	if name != "" {
		idx := slices.IndexFunc(s.packages, func(e *Package) bool { return strs.Lowcase(e.Name) == strs.Lowcase(name) })
		if idx != -1 {
			pakage := s.packages[idx]
			result = append(result, pakage.ToJson())
		}
	} else {
		for _, pakage := range s.packages {
			result = append(result, pakage.ToJson())
		}
	}

	return et.Items{
		Ok:     len(result) > 0,
		Count:  len(result),
		Result: result,
	}
}

/**
* GetRoutes
* @return et.Items
**/
func (s *Server) GetRoutes() et.Items {
	var result = []et.Json{}
	for _, route := range s.router {
		result = append(result, route.ToJson())
	}

	return et.Items{
		Result: result,
		Count:  len(result),
		Ok:     len(result) > 0,
	}
}

/**
* GetSolvers
* @return et.Items
**/
func (s *Server) GetSolvers() et.Items {
	var result = []et.Json{}
	for _, route := range s.solvers {
		result = append(result, route.ToJson())
	}

	return et.Items{
		Result: result,
		Count:  len(result),
		Ok:     len(result) > 0,
	}
}

/**
* deleteRoute
* @param method, id string
* @return error
**/
func (s *Server) deleteRoute(method, id string) error {
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
* GetTokenByKey
* @param key string
* @return error
**/
func (s *Server) GetTokenByKey(key string) (et.Item, error) {
	if !utility.ValidStr(key, 0, []string{}) {
		return et.Item{}, console.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	result, err := cache.Get(key, "")
	if err != nil {
		return et.Item{}, err
	}

	if result == "" {
		return et.Item{}, console.Alertm(msg.RECORD_NOT_FOUND)
	}

	valid := MSG_TOKEN_VALID
	_, err = claim.ValidToken(result)
	if err != nil {
		valid = err.Error()
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"key":   key,
			"value": result,
			"valid": valid,
		},
	}, nil
}

/**
* handlerValidToken
* @param key string
* @return error
**/
func (s *Server) HandlerValidToken(key string) (et.Item, error) {
	if !utility.ValidStr(key, 0, []string{}) {
		return et.Item{}, console.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	result, err := cache.Get(key, "")
	if err != nil {
		return et.Item{}, err
	}

	if result == "" {
		return et.Item{}, console.Alertm(msg.RECORD_NOT_FOUND)
	}

	_, err = claim.ValidToken(result)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"key":   key,
			"value": result,
		},
	}, nil
}

/**
* DeleteTokenByKey
* @param id string
* @return error
**/
func (s *Server) DeleteTokenByKey(key string) error {
	if !utility.ValidStr(key, 0, []string{}) {
		return console.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	_, err := cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}
