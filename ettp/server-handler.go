package ettp

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
)

type contextKey string

var commonHeader = make(map[string]bool)

const (
	MetricKey   claim.ContextKey = "metric"
	ResoluteKey claim.ContextKey = "resolute"
)

/**
* applyMiddlewares
* @params handler http.Handler, middlewares []func(http.Handler) http.Handler
* @return http.Handler
**/
func (s *Server) applyMiddlewares(handler http.Handler, middlewares []func(http.Handler) http.Handler) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}

	return handler
}

/**
* handlerApiRest
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerApiRest(w http.ResponseWriter, r *http.Request) {
	rw := &middleware.ResponseWriterWrapper{ResponseWriter: w}

	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(rw, r, http.StatusInternalServerError, "Metric not found")
		return
	}

	resolver, ok := r.Context().Value(ResoluteKey).(*Resolver)
	if !ok {
		metric.HTTPError(rw, r, http.StatusInternalServerError, "Resolute not found")
		return
	}

	proxyReq, err := http.NewRequest(resolver.Method, resolver.URL, r.Body)
	if err != nil {
		metric.HTTPError(rw, r, http.StatusInternalServerError, err.Error())
		return
	}

	proxyReq.Header = resolver.Header
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: transport,
	}
	res, err := client.Do(proxyReq)
	if err != nil {
		metric.HTTPError(rw, r, http.StatusBadGateway, err.Error())
		return
	}
	defer res.Body.Close()

	setHeader := func(header http.Header) {
		for key, values := range header {
			joinedValues := ""
			for _, value := range values {
				if commonHeader[key] {
					continue
				} else if len(value) > 255 {
					continue
				}
				if len(joinedValues) > 0 {
					joinedValues += ", "
				}
				joinedValues += value
			}
			rw.Header().Set(key, joinedValues)
		}
	}

	setCookie := func(cookies []*http.Cookie) {
		headers := rw.Header()
		for _, cookie := range cookies {
			_, ok := headers["Set-Cookie"]
			if !ok {
				rw.Header().Add("Set-Cookie", cookie.String())
			} else {
				rw.Header().Set("Set-Cookie", cookie.String())
			}
		}
	}

	setHeader(res.Header)
	setCookie(res.Cookies())
	rw.WriteHeader(res.StatusCode)

	_, err = io.Copy(rw, res.Body)
	if err != nil {
		metric.HTTPError(rw, r, http.StatusInternalServerError, err.Error())
	}

	go metric.DoneFn(rw)
}

/**
* handlerResolve
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerResolve(w http.ResponseWriter, r *http.Request) {
	/* Begin telemetry */
	metric := middleware.NewMetric(r)
	w.Header().Set("Reqid", metric.ReqID)
	ctx := context.WithValue(r.Context(), MetricKey, metric)

	/* Get resolver */
	resolver, r := s.getResolver(r)
	console.Log("resolver", resolver.ToString())

	/* Call search time since begin */
	metric.CallSearchTime()
	metric.SetPath(resolver.GetResolve())

	/* If not found */
	if resolver.Router == nil || resolver.URL == "" {
		r.RequestURI = fmt.Sprintf(`%s://%s%s`, resolver.Scheme, resolver.Host, resolver.Path)
		s.notFoundHandler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	/* If HandlerFunc is handler */
	router := resolver.Router
	if router.Kind == TpHandler {
		h := s.handlers[router.Id]
		if h == nil {
			r.RequestURI = fmt.Sprintf(`%s://%s%s`, resolver.Scheme, resolver.Host, resolver.Path)
			s.notFoundHandler.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		handler := s.applyMiddlewares(http.HandlerFunc(h), router.middlewares)
		handler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	/* If REST is handler */
	h := s.handlerApiRest
	ctx = context.WithValue(ctx, ResoluteKey, resolver)
	handler := s.applyMiddlewares(http.HandlerFunc(h), router.middlewares)

	handler.ServeHTTP(w, r.WithContext(ctx))
}

/**
* handlerReverseProxy
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) handlerReverseProxy(w http.ResponseWriter, r *http.Request) {
	console.Debug(et.Json{
		"Method":   r.Method,
		"URL":      r.URL,
		"Host":     r.Host,
		"Path":     r.URL.Path,
		"RawQuery": r.URL.RawQuery,
		"Header":   r.Header,
		"Body":     r.Body,
	}.ToString())

	proxy, ok := s.proxy[r.URL.Path]
	if !ok {
		target, _ := url.Parse("http://localhost:8081")
		proxy = httputil.NewSingleHostReverseProxy(target)
		s.proxy[r.URL.Path] = proxy
	}

	proxy.ServeHTTP(w, r)
}

func init() {
	for _, v := range []string{
		"Content-Security-Policy",
		"Content-Length",
	} {
		commonHeader[v] = true
	}
}
