package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
)

/**
* notFoundHandler
* @params w http.ResponseWriter
* @params r *http.Request
**/
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		http.Error(w, MSG_METRIC_NOT_FOUND, http.StatusInternalServerError)
		return
	}

	result := et.Json{
		"message": "404 Not Found.",
		"route":   r.RequestURI,
	}

	metric.JSON(w, r, http.StatusNotFound, result)
}
