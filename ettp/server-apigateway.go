package ettp

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	rt "github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
)

/**
* mountApiGatewayFunc
**/
func (s *Server) mountApiGatewayFunc() {
	s.Get("/apigateway/version", s.getVersion, s.Name)
	s.Get("/apigateway/test/{id}/test", s.getVersion, s.Name)
	s.Private().Get("/apigateway/events", s.getEvents, s.Name)
	s.Private().Get("/apigateway/{id}", s.getRouteById, s.Name)
	s.Private().Post("/apigateway", s.upsetRouter, s.Name)
	s.Private().Delete("/apigateway/{id}", s.deleteRouteById, s.Name)
	s.Private().Get("/apigateway/router", s.getRouter, s.Name)
	s.Private().Get("/apigateway/packages", s.getPakages, s.Name)
	s.Private().Post("/apigateway/reset", s.reset, s.Name)
	/* RPC */
	s.Private().Get("/apigateway/rpc", jrpc.ListRouters, s.Name)
	s.Private().Post("/apigateway/rpc", jrpc.HttpCalcItem, s.Name)
	/* Token */
	s.Private().Get("/apigateway/tokens/{key}", s.getToken, s.Name)
	s.Private().Post("/apigateway/tokens", s.setToken, s.Name)
	s.Private().Delete("/apigateway/tokens/{key}", s.deleteToken, s.Name)
	production := config.App.Production
	if !production {
		s.Get("/apigateway/tokens/develop", s.handlerDevToken, s.Name)
	}
	/* Cache */
	s.Private().Get("/apigateway/cache", s.listCache, s.Name)
	s.Private().Delete("/apigateway/cache", s.emptyCache, s.Name)

	s.Save()
}

/**
* getVersion
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := s.version()
	metric.JSON(w, r, http.StatusOK, result)
}

/**
* getEvents
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getEvents(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, "Metric not found")
		return
	}

	result := et.Items{
		Ok:     true,
		Count:  len(event.Events),
		Result: []et.Json{},
	}
	for _, event := range event.Events {
		result.Result = append(result.Result, et.Json{"event": event})
	}

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* getRouteById
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getRouteById(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	id := r.PathValue("id")
	result := s.GetRouteById(id)
	if result == nil {
		metric.HTTPError(w, r, http.StatusNotFound, MSG_ROUTE_NOT_FOUND)
		return
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: result.ToJson()})
}

/**
* upsetRouter
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) upsetRouter(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := et.Items{Result: []et.Json{}}
	body, _ := response.GetArray(r)
	n := len(body)
	for i := 0; i < n; i++ {
		item := body[i]
		private := item.Bool("private")
		id := item.ValStr("", "id")
		method := item.Str("method")
		path := item.Str("path")
		resolve := item.Str("resolve")
		header := item.Json("header")
		tpHeader := rt.ToTpHeader(item.Int("tp_header"))
		excludeHeader := item.ArrayStr("exclude_header")
		packageName := item.Str("package_name")
		saved := i == n-1
		router, err := s.SetRouter(private, id, method, path, resolve, header, tpHeader, excludeHeader, packageName, saved)
		if err != nil {
			metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		result.Add(router.ToJson())
	}

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* deleteRouteById
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) deleteRouteById(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	id := r.PathValue("id")
	err := s.DeleteRouteById(id, true)
	if err != nil {
		metric.HTTPError(w, r, http.StatusNotFound, err.Error())
		return
	}

	event.Publish(rt.APIGATEWAY_DELETE, et.Json{
		"id": id,
	})
	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": MSG_ROUTE_DELETE}})
}

/**
* getRouter
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getRouter(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := s.GetRouter()

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* reset
* @params w http.ResponseWriter
* @params r *http.Request
*
 */
func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	if err := s.Reset(); err != nil {
		metric.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	for _, pk := range s.packages {
		channel := fmt.Sprintf(`%s/%s`, rt.APIGATEWAY_RESET, pk.Name)
		event.Publish(channel, et.Json{})
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": strs.Format(MSG_APIGATEWAY_RESET, s.Name)}})
}

/**
* handlerDevToken
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) handlerDevToken(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	token := developToken()

	metric.JSON(w, r, http.StatusOK, et.Json{
		"token": token,
	})
}
