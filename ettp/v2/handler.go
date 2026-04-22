package ettp

import (
	"context"
	"io"
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/request"
)

var commonHeader = make(map[string]bool)

const (
	ResoluteKey request.ContextKey = "resolute"
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
* handler
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
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
		logs.Log("Route Table", resolver.ToJson().ToString())
	}

	/* Call search time since begin */
	w.Header().Set("ServiceId", resolver.ID)
	metric.CallSearchTime()
	metric.SetPath(resolver.Path)

	/* If HandlerFunc is handler */
	if resolver.Kind == TpHandler {
		h := resolver.handlerFn
		if h == nil {
			go s.HTTPError(resolver, metric, w, r, http.StatusNotFound, "Handler not found")
			return
		}

		handler := s.applyMiddlewares(http.HandlerFunc(h), resolver.middlewares)
		handler.ServeHTTP(w, r)
		return
	}

	/* If API REST is handler */
	h := s.handlerApi
	ctx = context.WithValue(ctx, ResoluteKey, resolver)
	handler := s.applyMiddlewares(http.HandlerFunc(h), resolver.middlewares)
	handler.ServeHTTP(w, r.WithContext(ctx))
}

/**
* handlerApi
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerApi(w http.ResponseWriter, r *http.Request) {
	rw := &middleware.ResponseWriterWrapper{ResponseWriter: w}
	metric := middleware.GetMetrics(r)

	resolver, ok := r.Context().Value(ResoluteKey).(*Resolver)
	if !ok {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, "Resolver not found")
		return
	}

	if resolver.URL == "" {
		s.HTTPError(resolver, metric, rw, r, http.StatusNotFound, "Resolver not found")
		return
	}

	proxyReq, err := http.NewRequest(resolver.Method, resolver.URL, r.Body)
	if err != nil {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, err.Error())
		return
	}

	proxyReq.Header = resolver.Header
	res, err := s.client.Do(proxyReq)
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
