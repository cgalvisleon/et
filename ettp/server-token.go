package ettp

import (
	"net/http"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/console"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
)

/**
* developToken
* @return string
**/
func developToken() string {
	production := config.App.Production
	if production {
		return ""
	}

	device := "DevelopToken"
	duration := time.Hour * 24 * 7
	token, err := claim.NewToken(device, device, device, device, device, duration)
	if err != nil {
		console.Alert(err)
		return ""
	}

	_, err = claim.ValidToken(token)
	if err != nil {
		console.Alertf("GetFromToken:%s", err.Error())
		return ""
	}

	return token
}

/**
* GetTokenByKey
* @param key string
* @return error
**/
func (s *Server) GetTokenByKey(key string) (et.Item, error) {
	if !utility.ValidStr(key, 0, []string{}) {
		return et.Item{}, console.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	result, err := cache.Get(key, "")
	if err != nil {
		return et.Item{}, err
	}

	if result == "" {
		return et.Item{}, console.Alertm(msg.RECORD_NOT_FOUND)
	}

	valid := MSG_TOKEN_VALID
	_, err = claim.ValidToken(result)
	if err != nil {
		valid = err.Error()
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"key":   key,
			"value": result,
			"valid": valid,
		},
	}, nil
}

/**
* handlerValidToken
* @param key string
* @return error
**/
func (s *Server) HandlerValidToken(key string) (et.Item, error) {
	if !utility.ValidStr(key, 0, []string{}) {
		return et.Item{}, console.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	result, err := cache.Get(key, "")
	if err != nil {
		return et.Item{}, err
	}

	if result == "" {
		return et.Item{}, console.Alertm(msg.RECORD_NOT_FOUND)
	}

	_, err = claim.ValidToken(result)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"key":   key,
			"value": result,
		},
	}, nil
}

/**
* DeleteTokenByKey
* @param id string
* @return error
**/
func (s *Server) DeleteTokenByKey(key string) error {
	if !utility.ValidStr(key, 0, []string{}) {
		return console.Alertf(msg.MSG_ATRIB_REQUIRED, "key")
	}

	_, err := cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}

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
	duration := body.Int("duration")
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
