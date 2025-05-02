package ettp

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
)

type Resolute struct {
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
	Resolve    *Resolve
	URL        string
}

/**
* ToJson
* @return et.Json
**/
func (rs *Resolute) ToJson() et.Json {
	resolve := et.Json{}
	if rs.Resolve != nil {
		resolve = rs.Resolve.ToJson()
	}

	return et.Json{
		"Method":     rs.Method,
		"Proto":      rs.Proto,
		"Path":       rs.Path,
		"RawQuery":   rs.RawQuery,
		"Query":      rs.Query,
		"RequestURI": rs.RequestURI,
		"RemoteAddr": rs.RemoteAddr,
		"Header":     rs.Header,
		"Body":       rs.Body,
		"Host":       rs.Host,
		"Scheme":     rs.Scheme,
		"Resolve":    resolve,
		"URL":        rs.URL,
	}
}

/**
* ToString
* @return string
**/
func (s *Resolute) ToString() string {
	resutn := s.ToJson()

	return resutn.ToString()
}

func (s *Resolute) GetResolve() string {
	if s.Resolve == nil {
		return ""
	}

	if s.Resolve.Route == nil {
		return ""
	}

	return s.Resolve.Route.Resolve
}

/**
* getResolute
* @params r *http.Request
* @return *Resolute, *http.Request
**/
func (s *Server) getResolute(r *http.Request) (*Resolute, *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	var header = http.Header{}
	var url string
	resolve := findResolve(s, r)
	if resolve != nil {
		url = strs.Append(resolve.Resolve, r.URL.RawQuery, "?")
		if resolve.Route == nil {
			console.Alertf(`%s:%s:%s`, MSG_ROUTE_NOT_FOUND, r.Method, r.URL.Path)
		} else {
			switch resolve.Route.TpHeader {
			case router.TpKeepHeader: /* Keep header */
				for key := range resolve.Route.Header {
					value := resolve.Route.Header.ArrayStr(key)
					for _, v := range value {
						header.Add(key, v)
					}
				}
			case router.TpJoinHeader: /* Join header */
				for key := range resolve.Route.Header {
					value := resolve.Route.Header.ArrayStr(key)
					for _, v := range value {
						header.Add(key, v)
					}
				}
				for key, value := range r.Header {
					if !slices.Contains(resolve.Route.ExcludeHeader, key) {
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
					if !slices.Contains(resolve.Route.ExcludeHeader, key) {
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

	result := &Resolute{
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
		Resolve:    resolve,
		URL:        url,
	}

	return result, r
}
