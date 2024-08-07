package gateway

import (
	"io"
	"net/http"
	"net/url"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/strs"
)

type Resolute struct {
	Method     string
	Proto      string
	Path       string
	RawQuery   string
	Query      url.Values
	RequestURI string
	RemoteAddr string
	Header     http.Header
	Body       io.ReadCloser
	Host       string
	Scheme     string
	Resolve    *Resolve
	URL        string
}

func (r *Resolute) Definition() js.Json {
	return js.Json{
		"method":     r.Method,
		"proto":      r.Proto,
		"path":       r.Path,
		"rawQuery":   r.RawQuery,
		"query":      r.Query,
		"requestURI": r.RequestURI,
		"remoteAddr": r.RemoteAddr,
		"header":     r.Header,
		"body":       r.Body,
		"host":       r.Host,
		"scheme":     r.Scheme,
		"resolve":    r.Resolve,
		"url":        r.URL,
	}
}

func GetResolute(r *http.Request) *Resolute {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := ""
	resolve := conn.http.GetResolve(r.Method, r.URL.Path)
	if resolve != nil {
		url = strs.Append(resolve.Resolve, r.URL.RawQuery, "?")
	}

	return &Resolute{
		Method:     r.Method,
		Proto:      r.Proto,
		Path:       r.URL.Path,
		RawQuery:   r.URL.RawQuery,
		Query:      r.URL.Query(),
		RequestURI: r.RequestURI,
		RemoteAddr: r.RemoteAddr,
		Header:     r.Header,
		Body:       r.Body,
		Host:       r.Host,
		Scheme:     scheme,
		Resolve:    resolve,
		URL:        url,
	}
}

func (r *Resolute) ToString() string {
	j := js.Json{
		"Method":     r.Method,
		"Proto":      r.Proto,
		"Path":       r.Path,
		"RawQuery":   r.RawQuery,
		"Query":      r.Query,
		"RequestURI": r.RequestURI,
		"RemoteAddr": r.RemoteAddr,
		"Header":     r.Header,
		"Body":       r.Body,
		"Host":       r.Host,
		"Scheme":     r.Scheme,
		"Resolve":    r.Resolve,
		"URL":        r.URL,
	}

	return j.ToString()
}
