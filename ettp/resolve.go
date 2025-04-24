package ettp

import (
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

type Resolve struct {
	Route   *Route
	Params  et.Json
	Resolve string
	Request *http.Request
}

/**
* NewResolve
* @return *Resolve, *http.Request
**/
func NewResolve(route *Route, params et.Json, r *http.Request) *Resolve {
	if route == nil {
		return nil
	}

	if len(route.Resolve) == 0 {
		return nil
	}

	resolve := route.Resolve
	for k, v := range params {
		v = strs.Format(`%v`, v)
		resolve = strings.ReplaceAll(resolve, k, v.(string))
		k = strings.ReplaceAll(k, "{", "")
		k = strings.ReplaceAll(k, "}", "")
		r.SetPathValue(k, v.(string))
	}

	switch route.TpParams {
	case TpQueryParams:
		sp := "?"
		ls := strings.Split(resolve, sp)
		query := params["query"].(string)
		resolve = ls[0] + sp + query
		querys := strings.Split(query, "&")
		for _, q := range querys {
			qs := strings.Split(q, "=")
			if len(qs) == 2 {
				r.SetPathValue(qs[0], qs[1])
			}
		}
	case TpMatrixParams:
		sp := ";"
		ls := strings.Split(resolve, sp)
		matrix := params["matrix"].(string)
		resolve = ls[0] + sp + matrix
		matrixs := strings.Split(matrix, ";")
		for _, m := range matrixs {
			ms := strings.Split(m, "=")
			if len(ms) == 2 {
				r.SetPathValue(ms[0], ms[1])
			}
		}
	}

	result := &Resolve{
		Route:   route,
		Params:  params,
		Resolve: resolve,
		Request: r,
	}

	return result
}

/**
* ToJson
* @return et.Json
**/
func (r *Resolve) ToJson() et.Json {
	route := et.Json{}
	if r.Route != nil {
		route = r.Route.ToJson()
	}

	return et.Json{
		"Route":   route,
		"Params":  r.Params,
		"Resolve": r.Resolve,
	}
}

/**
* findResolve
* @param method string
* @param path string
* @return *Resolve, *http.Request
**/
func findResolve(s *Server, r *http.Request) *Resolve {
	var params = et.Json{}
	var route *Route
	path := r.URL.Path
	method := r.Method
	idx := indexRoute(method, s.router)
	if idx == -1 {
		return nil
	} else {
		route = s.router[idx]
	}

	tags := strings.Split(path, "/")
	n := len(tags)
	for i := 0; i < n; i++ {
		tag := tags[i]
		if len(tag) == 0 {
			continue
		}

		find := route.find(tag)
		if find == nil {
			find, _ = route.getParamsRoute(0)
			if find == nil {
				route = nil
				break
			}

			tpParam := getTpParams(tag)
			if find.TpParams == TpPathParams {
				route = find
				params[route.Tag] = tag
				if i == n-1 {
					break
				}
			} else if find.TpParams == tpParam && tpParam == TpQueryParams {
				tags := strings.Split(tag, "?")
				if find.Tag == tags[0] {
					route = find
					var query string
					if len(tags) > 1 {
						query = tags[1]
					}
					params["query"] = query
					break
				}
			} else if find.TpParams == tpParam && tpParam == TpMatrixParams {
				tags := strings.Split(tag, ";")
				if find.Tag == tags[0] {
					route = find
					var matrix string
					for j := 1; j < len(tags); j++ {
						matrix = strs.Append(matrix, tags[j], ";")
					}
					params["matrix"] = matrix
					break
				}
			} else {
				route = nil
				break
			}
		} else {
			route = find
		}
	}

	return NewResolve(route, params, r)
}
