package ettp

import (
	"crypto/tls"
	"io"
	"net/http"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/middleware"
)

var commonHeader = make(map[string]bool)

const (
	MetricKey   claim.ContextKey = "metric"
	ResoluteKey claim.ContextKey = "resolute"
)

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
