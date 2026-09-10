package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/middleware"
)

/**
* getPakages
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) getPakages(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		http.Error(w, MSG_METRIC_NOT_FOUND, http.StatusInternalServerError)
		return
	}

	queryParams := r.URL.Query()
	name := queryParams.Get("name")
	result := s.GetPackages(name)

	metric.ITEMS(w, r, http.StatusOK, result)
}
