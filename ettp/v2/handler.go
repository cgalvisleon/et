package ettp

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/middleware"
)

var commonHeader = make(map[string]bool)

const (
	ResoluteKey claim.ContextKey = "resolute"
)

func init() {
	for _, v := range []string{
		"Content-Security-Policy",
		"Content-Length",
	} {
		commonHeader[v] = true
	}
}

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
* handlerRouteTable
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Server) handlerRouteTable(w http.ResponseWriter, r *http.Request) {
	/* Begin telemetry */
	metric := middleware.GetMetrics(r)
	ctx := context.WithValue(r.Context(), middleware.MetricKey, metric)
	r = r.WithContext(ctx)

	/* Get resolver */
	resolver, err := s.FindResolver(r)
	if err != nil {
		metric.HTTPError(w, r, http.StatusNotFound, err.Error())
		return
	}

	if s.debug {
		console.Log("Route Table", resolver.ToJson().ToString())
	}

	/* Call search time since begin */
	w.Header().Set("serviceId", resolver.Id)
	metric.CallSearchTime()
	metric.SetPath(resolver.solver.Path)

	/* If HandlerFunc is handler */
	if resolver.solver.Kind == TpHandler {
		h := s.handlers[resolver.solver.Id]
		if h == nil {
			go s.HTTPError(resolver, metric, w, r, http.StatusNotFound, "Handler not found")
			return
		}

		handler := s.applyMiddlewares(http.HandlerFunc(h), resolver.solver.middlewares)
		handler.ServeHTTP(w, r)
		return
	}

	/* If WebApp is handler */
	if resolver.solver.Kind == TpWepApp {
		h := s.handlerWebApp
		ctx = context.WithValue(ctx, ResoluteKey, resolver)
		handler := s.applyMiddlewares(http.HandlerFunc(h), resolver.solver.middlewares)
		handler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	/* If API REST is handler */
	h := s.handlerApiRest
	ctx = context.WithValue(ctx, ResoluteKey, resolver)
	handler := s.applyMiddlewares(http.HandlerFunc(h), resolver.solver.middlewares)
	handler.ServeHTTP(w, r.WithContext(ctx))
}

/**
* handlerWebApp
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerWebApp(w http.ResponseWriter, r *http.Request) {
	rw := &middleware.ResponseWriterWrapper{ResponseWriter: w}
	metric := middleware.GetMetrics(r)

	resolver, ok := r.Context().Value(ResoluteKey).(*Resolver)
	if !ok {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, "Resolver not found")
		return
	}

	frontendURL, _ := url.Parse(resolver.URL)
	proxy := httputil.NewSingleHostReverseProxy(frontendURL)
	proxy.ServeHTTP(w, r)

	// http.Redirect(w, r, resolver.URL, http.StatusTemporaryRedirect)
}

/**
* handlerApiRest
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerApiRest(w http.ResponseWriter, r *http.Request) {
	rw := &middleware.ResponseWriterWrapper{ResponseWriter: w}
	metric := middleware.GetMetrics(r)

	resolver, ok := r.Context().Value(ResoluteKey).(*Resolver)
	if !ok {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, "Resolver not found")
		return
	}

	proxyReq, err := http.NewRequest(resolver.Method, resolver.URL, r.Body)
	if err != nil {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, err.Error())
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
		s.HTTPError(resolver, metric, rw, r, http.StatusBadGateway, err.Error())
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
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, err.Error())
	}

	go s.HTTPSuccess(resolver, metric, rw)
}
