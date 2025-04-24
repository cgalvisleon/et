package ettp

import (
	"net/http"
	"strconv"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
)

/**
* handlerCache
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) listCache(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	queryParams := r.URL.Query()
	search := queryParams.Get("search")
	page, err := strconv.Atoi(queryParams.Get("page"))
	if err != nil {
		page = 1
	}
	rows, err := strconv.Atoi(queryParams.Get("rows"))
	if err != nil {
		rows = 30
	}

	result, err := cache.AllCache(search, page, rows)
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.JSON(w, r, http.StatusOK, result)
}

/**
* emptyCache
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) emptyCache(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	queryParams := r.URL.Query()
	match := queryParams.Get("match")
	err := cache.Empty(match)
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.JSON(w, r, http.StatusOK, et.Json{"message": "Cache empty"})
}
