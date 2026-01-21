package ettp

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
)

/**
* initRouteTable
**/
func (s *Server) initRouteTable() error {
	s.Public(GET, "/version", s.getVersion, s.Name)
	s.Public(GET, "/test/{id}/{test}", s.getTest, s.Name)
	s.Private(GET, "/reset", s.reset, s.Name)
	// Develop Token
	production := envar.GetBool("PRODUCTION", true)
	if !production {
		s.Public(GET, "/tokens/develop", s.handlerDevToken, s.Name)
	}
	// Events
	s.Private(GET, "/events", s.getEvents, s.Name)
	s.Private(POST, "/events", event.HttpEventPublish, s.Name)
	s.Private(PUT, "/events/reset", s.resetEvents, s.Name)
	// Routes
	s.Private(GET, "/routes", s.getRoutes, s.Name)
	s.Private(POST, "/routes", s.upsetRouter, s.Name)
	s.Private(DELETE, "/routes/{id}", s.deleteRouteById, s.Name)
	// Packages
	s.Private(GET, "/packages", s.getPakages, s.Name)
	s.Private(DELETE, "/packages/{name}", s.deletePackage, s.Name)
	// Cache
	s.Private(GET, "/cache", s.listCache, s.Name)
	s.Private(DELETE, "/cache", s.emptyCache, s.Name)
	s.Private(GET, "/cache/{key}", s.getCache, s.Name)

	return nil
}

/**
* getVersion
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	company := envar.GetStr("COMPANY", "")
	web := envar.GetStr("WEB", "")
	help := envar.GetStr("HELP", "")
	result := et.Json{
		"created_at": s.CreatedAt.Format("02/01/2006 3:04:05 PM"),
		"version":    s.Version,
		"service":    s.Name,
		"host":       s.Host,
		"company":    company,
		"web":        web,
		"help":       help,
	}

	metric.JSON(w, r, http.StatusOK, result)
}

/**
* getTest
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getTest(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)
	id := r.PathValue("id")
	test := r.PathValue("test")

	result := et.Json{
		"created_at": s.CreatedAt.Format("02/01/2006 3:04:05 PM"),
		"id":         id,
		"test":       test,
	}

	metric.JSON(w, r, http.StatusOK, result)
}

/**
* handlerDevToken
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) handlerDevToken(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	developToken := func() string {
		production := envar.GetBool("PRODUCTION", true)
		if production {
			return ""
		}

		device := "develop"
		duration := 1 * time.Hour
		token, err := claim.NewToken(device, device, device, et.Json{}, duration)
		if err != nil {
			logs.Alert(err)
			return ""
		}

		_, err = jwt.Validate(token)
		if err != nil {
			logs.Alertf("handlerDevToken:%s", err.Error())
			return ""
		}

		return token
	}
	token := developToken()

	metric.JSON(w, r, http.StatusOK, et.Json{
		"token": token,
	})
}

/**
* getEvents
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getEvents(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

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
	metric := middleware.GetMetrics(r)

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
* getRouters
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getRoutes(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	id := r.URL.Query().Get("id")
	if len(id) != 0 {
		result, ok := s.Solvers[id]
		if !ok {
			metric.HTTPError(w, r, http.StatusNotFound, MSG_ROUTE_NOT_FOUND)
			return
		}

		metric.JSON(w, r, http.StatusOK, result.ToJson())
		return
	}

	name := r.URL.Query().Get("name")
	if len(name) != 0 {
		result, ok := s.Packages[name]
		if !ok {
			metric.HTTPError(w, r, http.StatusNotFound, MSG_ROUTE_NOT_FOUND)
			return
		}

		metric.JSON(w, r, http.StatusOK, result.ToJson())
		return
	}

	metric.JSON(w, r, http.StatusOK, s.ToJson())
}

/**
* upsetRouter
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) upsetRouter(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	result := et.Items{Result: []et.Json{}}
	body, _ := response.GetArray(r)
	n := len(body)
	for i := 0; i < n; i++ {
		item := body[i]
		method := item.Str("method")
		path := item.Str("path")
		resolve := item.Str("resolve")
		header := item.Json("header")
		tpHeader := item.Int("tp_header")
		excludeHeader := item.ArrayStr("exclude_header")
		version := item.Int("version")
		private := item.Bool("private")
		packageName := item.Str("package_name")
		router, err := s.SetRouter(method, path, resolve, tpHeader, header, excludeHeader, version, private, packageName, true)
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
	metric := middleware.GetMetrics(r)

	id := r.PathValue("id")
	err := s.RemoveRouterById(id, true)
	if err != nil {
		metric.HTTPError(w, r, http.StatusNotFound, err.Error())
		return
	}

	event.Publish(router.EVENT_REMOVE_ROUTER, et.Json{
		"id": id,
	})

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_ROUTE_DELETE,
		}})
}

/**
* getPakages
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) getPakages(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	queryParams := r.URL.Query()
	name := queryParams.Get("name")
	if len(name) == 0 {
		result := et.Items{Result: []et.Json{}}
		for _, pkg := range s.Packages {
			result.Add(pkg.ToJson())
		}

		metric.ITEMS(w, r, http.StatusOK, result)
		return
	}

	result, ok := s.Packages[name]
	if !ok {
		metric.HTTPError(w, r, http.StatusNotFound, MSG_ROUTE_NOT_FOUND)
		return
	}

	metric.JSON(w, r, http.StatusOK, result.ToJson())
}

/**
* deletePackage
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) deletePackage(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	name := r.PathValue("name")
	result, ok := s.Packages[name]
	if !ok {
		metric.HTTPError(w, r, http.StatusNotFound, MSG_ROUTE_NOT_FOUND)
		return
	}

	for _, solver := range result.Solvers {
		s.RemoveRouterById(solver.Id, false)
	}

	delete(s.Packages, name)

	s.Save()

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_PACKAGE_DELETE,
		},
	})
}

/**
* reset
* @params w http.ResponseWriter
* @params r *http.Request
*
 */
func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	s.Reset()

	for _, pk := range s.Packages {
		channel := fmt.Sprintf(`%s:%s`, router.EVENT_RESET_ROUTER, pk.Name)
		event.Publish(channel, et.Json{})
	}

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": MSG_RESET_ROUTES,
		},
	})
}

/**
* handlerCache
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) listCache(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

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
	metric := middleware.GetMetrics(r)

	queryParams := r.URL.Query()
	match := queryParams.Get("match")
	err := cache.Empty(match)
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.JSON(w, r, http.StatusOK, et.Json{"message": "Cache empty"})
}

/**
* getCache
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) getCache(w http.ResponseWriter, r *http.Request) {
	metric := middleware.GetMetrics(r)

	key := r.PathValue("key")
	result, err := cache.Get(key, "")
	if err != nil {
		metric.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	metric.JSON(w, r, http.StatusOK, result)
}
