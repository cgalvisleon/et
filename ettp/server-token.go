package ettp

import (
	"net/http"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
)

/**
* handlerSetToken
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) setToken(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	body, _ := response.GetBody(r)
	app := body.Str("app")
	device := body.Str("device")
	id := body.Str("id")
	token := body.Str("token")
	second := body.Num("duration")
	duration := time.Duration(second) * time.Second
	key := claim.SetToken(app, device, id, token, duration)
	result := et.Json{
		"key":      key,
		"duration": duration,
		"message":  "Token setted",
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result})
}

/**
* getToken
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) getToken(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	key := r.PathValue("key")
	result, err := s.GetTokenByKey(key)
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.ITEM(w, r, http.StatusOK, result)
}

/**
* deleteToken
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) deleteToken(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	key := r.PathValue("key")
	err := s.DeleteTokenByKey(key)
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": "Token deleted"}})
}
