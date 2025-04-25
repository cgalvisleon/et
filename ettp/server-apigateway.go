package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	rt "github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
)

/**
* mountHandlerFunc
**/
func (s *Server) mountHandlerFunc() {
	s.Get("/apigateway/version", s.version, ServiceName)
	s.Get("/apigateway/test/{id}/test", s.version, ServiceName)
	s.Private().Get("/apigateway/events", s.getEvents, ServiceName)
	s.Private().Get("/apigateway/{id}", s.getRouteById, ServiceName)
	s.Private().Post("/apigateway", s.setRouter, ServiceName)
	s.Private().Delete("/apigateway/{id}", s.deleteRouteById, ServiceName)
	s.Private().Get("/apigateway/solvers", s.getSolvers, ServiceName)
	s.Private().Get("/apigateway/routers", s.getRouters, ServiceName)
	s.Private().Get("/apigateway/packages", s.getPakages, ServiceName)
	s.Private().Patch("/apigateway/reset", s.reset, ServiceName)
	// RPC
	s.Private().Get("/apigateway/rpc", s.listRpc, ServiceName)
	s.Private().Delete("/apigateway/rpc", s.deletePrcPackage, ServiceName)
	s.Private().Patch("/apigateway/rpc", s.testRpc, ServiceName)
	// Token
	s.Private().Get("/apigateway/tokens/{key}", s.getToken, ServiceName)
	s.Private().Post("/apigateway/tokens", s.setToken, ServiceName)
	s.Private().Delete("/apigateway/tokens/{key}", s.deleteToken, ServiceName)
	production := envar.GetBool(true, "PRODUCTION")
	if !production {
		s.Get("/apigateway/tokens/develop", s.handlerDevToken, ServiceName)
	}
	// Cache
	s.Private().Get("/apigateway/cache", s.listCache, ServiceName)
	s.Private().Delete("/apigateway/cache", s.emptyCache, ServiceName)

	s.Save()
}

/**
* version
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := version()
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
* setRouter
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) setRouter(w http.ResponseWriter, r *http.Request) {
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
		id := item.ValStr("-1", "_id")
		method := item.Str("method")
		path := item.Str("path")
		resolve := item.Str("resolve")
		header := item.Json("header")
		tpHeader := rt.ToTpHeader(item.Int("tp_header"))
		excludeHeader := item.ArrayStr("exclude_header")
		packageName := item.Str("package_name")
		saved := i == n-1
		router, err := s.SetResolve(private, id, method, path, resolve, header, tpHeader, excludeHeader, packageName, saved)
		if err != nil {
			metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		item.Set("from_id", s.Id)
		event.Publish(rt.APIGATEWAY_SET, item)
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

	event.Publish(rt.APIGATEWAY_DELETE, et.Json{"_id": id, "from_id": s.Id})
	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": MSG_ROUTE_DELETE}})
}

/**
* getRouters
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getRouters(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := s.GetRoutes()

	metric.ITEMS(w, r, http.StatusOK, result)
}

func (s *Server) getSolvers(w http.ResponseWriter, r *http.Request) {
	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	result := s.GetSolvers()

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

	s.Reset()

	metric.ITEM(w, r, http.StatusOK, et.Item{Ok: true, Result: et.Json{"message": strs.Format(MSG_APIGATEWAY_RESET, ServiceName)}})
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
