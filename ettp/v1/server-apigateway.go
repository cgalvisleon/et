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
)

/**
* mountApiGatewayFunc
**/
func (s *Server) mountApiGatewayFunc() {
	s.Get("/version", s.getVersion, s.Name)
	s.Get("/test/{id}/test", s.getVersion, s.Name)
	s.Private().Get("/events", s.getEvents, s.Name)
	s.Private().Post("/events", event.HttpEventPublish, s.Name)
	s.Private().Put("/events/reset", s.resetEvents, s.Name)
	s.Private().Get("/routes", s.getRoutes, s.Name)
	s.Private().Post("/routes", s.upsetRouter, s.Name)
	s.Private().Delete("/routes/{id}", s.deleteRouteById, s.Name)
	s.Private().Get("/packages", s.getPakages, s.Name)
	s.Private().Put("/reset", s.reset, s.Name)
	/* RPC */
	s.Private().Get("/rpc", jrpc.HttpListRouters, s.Name)
	s.Private().Post("/rpc", jrpc.HttpCalcItem, s.Name)
	/* Token */
	production := config.App.Production
	if !production {
		s.Get("/develop/token", s.handlerDevToken, s.Name)
	}
	/* Cache */
	s.Private().Get("/cache", s.listCache, s.Name)
	s.Private().Delete("/cache", s.emptyCache, s.Name)
	s.Private().Get("/cache/{key}", s.getCache, s.Name)

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

	events := event.Events()
	result := et.Items{
		Ok:     true,
		Count:  len(events),
		Result: []et.Json{},
	}
	for _, event := range events {
		result.Result = append(result.Result, et.Json{"event": event})
	}

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* resetEvents
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) resetEvents(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	if err := event.Reset(); err != nil {
		metric.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": "Events reset",
		},
	})
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
		method := item.Str("method")
		path := item.Str("path")
		resolve := item.Str("resolve")
		header := item.Json("header")
		tpHeader := rt.ToTpHeader(item.Int("tp_header"))
		excludeHeader := item.ArrayStr("exclude_header")
		packageName := item.Str("package_name")
		saved := i == n-1
		router, err := s.SetRouter(private, method, path, resolve, header, tpHeader, excludeHeader, packageName, saved)
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

	event.Publish(rt.EVENT_REMOVE_ROUTER, et.Json{
		"id": id,
	})
	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": MSG_ROUTE_DELETE}})
}

/**
* getRouters
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getRoutes(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		name := r.URL.Query().Get("name")
		result := s.GetPackages(name)
		metric.ITEMS(w, r, http.StatusOK, result)
		return
	}

	result := s.GetRouteById(id)
	if result == nil {
		metric.HTTPError(w, r, http.StatusNotFound, MSG_ROUTE_NOT_FOUND)
		return
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result.ToJson(),
	})
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
		channel := fmt.Sprintf(`%s:%s`, rt.EVENT_RESET_ROUTER, pk.Name)
		event.Publish(channel, et.Json{})
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": fmt.Sprintf(MSG_APIGATEWAY_RESET, s.Name)}})
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
