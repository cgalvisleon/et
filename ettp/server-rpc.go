package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
)

/**
* getRpcAll
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) listRpc(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result, err := jrpc.GetRouters()
	if err != nil {
		metric.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* deletePrcPackage
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) deletePrcPackage(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	body, _ := response.GetBody(r)
	host := body.Str("host")
	packageName := body.Str("packageName")
	result, err := jrpc.DeleteRouters(host, packageName)
	if err != nil {
		metric.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	metric.ITEM(w, r, http.StatusOK, result)
}

/**
* handlerTest
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) testRpc(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	body, _ := response.GetBody(r)
	method := body.Str("method")
	data := body.Json("data")
	result, err := jrpc.CallItem(method, data)
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.ITEM(w, r, http.StatusOK, result)
}
