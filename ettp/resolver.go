package ettp

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
)

type solver struct {
	router  *Router
	params  et.Json
	resolve string
}

/**
* newSolver
* @param route *Router, params et.Json, r *http.Request
* @return *solver
**/
func newSolver(route *Router, params et.Json, r *http.Request) *solver {
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

	result := &solver{
		router:  route,
		params:  params,
		resolve: resolve,
	}

	return result
}

/**
* findSolver
* @param s *Server, r *http.Request
* @return *solver
**/
func findSolver(s *Server, r *http.Request) *solver {
	var params = et.Json{}
	var router *Router
	path := r.URL.Path
	method := r.Method
	idx := getRouteIndex(method, s.router)
	if idx == -1 {
		return nil
	} else {
		router = s.router[idx]
	}

	tags := strings.Split(path, "/")
	n := len(tags)
	for i := 0; i < n; i++ {
		tag := tags[i]
		if len(tag) == 0 {
			continue
		}

		find := router.find(tag)
		if find == nil {
			find, _ = router.getParams(0)
			if find == nil {
				router = nil
				break
			}

			tpParam := getTpParams(tag)
			if find.TpParams == TpPathParams {
				router = find
				params[router.Tag] = tag
				if i == n-1 {
					break
				}
			} else if find.TpParams == tpParam && tpParam == TpQueryParams {
				tags := strings.Split(tag, "?")
				if find.Tag == tags[0] {
					router = find
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
					router = find
					var matrix string
					for j := 1; j < len(tags); j++ {
						matrix = strs.Append(matrix, tags[j], ";")
					}
					params["matrix"] = matrix
					break
				}
			} else {
				router = nil
				break
			}
		} else {
			router = find
		}
	}

	return newSolver(router, params, r)
}

type Resolver struct {
	Server     *Server
	Method     string
	Proto      string
	Path       string
	RawQuery   string
	Query      url.Values
	RequestURI string
	RemoteAddr string
	Header     http.Header
	Body       interface{}
	Host       string
	Scheme     string
	URL        string
	Router     *Router
}

/**
* ToJson
* @return et.Json
**/
func (s *Resolver) ToJson() et.Json {
	router := et.Json{}
	if s.Router != nil {
		router = s.Router.ToJson()
	}

	return et.Json{
		"Method":     s.Method,
		"Proto":      s.Proto,
		"Path":       s.Path,
		"RawQuery":   s.RawQuery,
		"Query":      s.Query,
		"RequestURI": s.RequestURI,
		"RemoteAddr": s.RemoteAddr,
		"Header":     s.Header,
		"Body":       s.Body,
		"Host":       s.Host,
		"Scheme":     s.Scheme,
		"URL":        s.URL,
		"Router":     router,
	}
}

/**
* ToString
* @return string
**/
func (s *Resolver) ToString() string {
	resutn := s.ToJson()

	return resutn.ToString()
}

/**
* GetResolve
* @return string
**/
func (s *Resolver) GetResolve() string {
	if s.Router == nil {
		return ""
	}

	return s.Router.Resolve
}

/**
* getResolver
* @params r *http.Request
* @return *Resolver, *http.Request
**/
func (s *Server) getResolver(r *http.Request) (*Resolver, *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	var header = http.Header{}
	var url string
	solver := findSolver(s, r)
	if solver != nil {
		url = strs.Append(solver.resolve, r.URL.RawQuery, "?")
		if solver.router == nil {
			console.Alertf(`%s:%s:%s`, MSG_ROUTE_NOT_FOUND, r.Method, r.URL.Path)
		} else {
			switch solver.router.TpHeader {
			case router.TpKeepHeader: /* Keep header */
				for key := range solver.router.Header {
					value := solver.router.Header.ArrayStr(key)
					for _, v := range value {
						header.Add(key, v)
					}
				}
			case router.TpJoinHeader: /* Join header */
				for key := range solver.router.Header {
					value := solver.router.Header.ArrayStr(key)
					for _, v := range value {
						header.Add(key, v)
					}
				}
				for key, value := range r.Header {
					if !slices.Contains(solver.router.ExcludeHeader, key) {
						for i, v := range value {
							if i == 0 {
								header.Set(key, v)
							} else {
								header.Add(key, v)
							}
						}
					}
				}
			case router.TpReplaceHeader: /* Replace header */
				for key, value := range r.Header {
					if !slices.Contains(solver.router.ExcludeHeader, key) {
						for i, v := range value {
							if i == 0 {
								header.Set(key, v)
							} else {
								header.Add(key, v)
							}
						}
					}
				}
			}
		}
	}

	var body interface{}
	requestBody, err := request.ReadBody(r.Body)
	if err != nil {
		body = et.Json{
			"result": []byte(err.Error()),
		}
	} else {
		jbody, err := requestBody.ToJson()
		if err != nil {
			body = requestBody.ToString()
		}
		body = jbody
	}

	r.Body = io.NopCloser(bytes.NewBuffer(requestBody.Data))

	result := &Resolver{
		Server:     s,
		Method:     r.Method,
		Proto:      r.Proto,
		Path:       r.URL.Path,
		RawQuery:   r.URL.RawQuery,
		Query:      r.URL.Query(),
		RequestURI: r.RequestURI,
		RemoteAddr: r.RemoteAddr,
		Header:     header,
		Body:       body,
		Host:       r.Host,
		Scheme:     scheme,
		URL:        url,
		Router:     solver.router,
	}

	return result, r
}
