package ettp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/middleware"
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
* handlerResolvre
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerResolver(w http.ResponseWriter, r *http.Request) {
	/* If proxy is found */
	proxy := s.getProxyByPath(r.URL.Path)
	if proxy != nil {
		s.handlerReverseProxy(w, r)
		return
	}

	/* Begin telemetry */
	metric := middleware.NewMetric(r)
	w.Header().Set("ServiceId", metric.ServiceId)
	ctx := context.WithValue(r.Context(), MetricKey, metric)
	r = r.WithContext(ctx)

	/* Get resolver */
	resolver, r := s.getResolver(r)
	console.Log("resolver", resolver.ToString())

	/* Call search time since begin */
	metric.CallSearchTime()
	metric.SetPath(resolver.GetResolve())

	url := fmt.Sprintf(`%s://%s%s`, resolver.Scheme, resolver.Host, resolver.Path)
	/* If not found */
	if resolver.Router == nil || resolver.URL == "" {
		r.RequestURI = url
		s.notFoundHandler.ServeHTTP(w, r)
		return
	}

	/* If HandlerFunc is handler */
	router := resolver.Router
	if router.Kind == TpHandler {
		h := s.handlers[router.Id]
		if h == nil {
			r.RequestURI = url
			s.notFoundHandler.ServeHTTP(w, r)
			return
		}

		handler := s.applyMiddlewares(http.HandlerFunc(h.HandlerFn), router.middlewares)
		handler.ServeHTTP(w, r)
		return
	}

	/* If REST is handler */
	h := s.handlerApiRest
	ctx = context.WithValue(ctx, ResoluteKey, resolver)
	handler := s.applyMiddlewares(http.HandlerFunc(h), router.middlewares)

	handler.ServeHTTP(w, r.WithContext(ctx))
}

func init() {
	for _, v := range []string{
		"Content-Security-Policy",
		"Content-Length",
	} {
		commonHeader[v] = true
	}
}
