package ettp

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/ws"
)

/**
* loadHandlerFuncWS
**/
func (s *Server) loadHandlerFuncWS() {
	s.Get("/realtime", s.wsRealtime, "Websocket")
	s.Private().Get("/ws", s.wsUpgrade, "Websocket")
	s.Private().Get("/realtime/publications", s.wsChannels, "Websocket")
	s.Private().Get("/realtime/subscribers", s.wsClients, "Websocket")
	s.Private().Post("/realtime", s.wsPublish, "Websocket")

	s.Save()
}

/**
* wsUpgrade
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) wsUpgrade(w http.ResponseWriter, r *http.Request) {
	if s.ws == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "Websocket not found")
		return
	}

	s.ws.HttpConnect(w, r)
}

/**
* wsRealtime
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) wsRealtime(w http.ResponseWriter, r *http.Request) {
	if s.ws == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "Websocket not found")
		return
	}

	s.ws.HttpLogin(w, r)
}

/**
* wsChannels
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) wsChannels(w http.ResponseWriter, r *http.Request) {
	if s.ws == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "Websocket not found")
		return
	}

	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	name := r.PathValue("name")
	queue := r.PathValue("queue")
	result := s.ws.GetChannels(name, queue)

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* wsClients
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) wsClients(w http.ResponseWriter, r *http.Request) {
	if s.ws == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "Websocket not found")
		return
	}

	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	key := r.PathValue("key")
	result := s.ws.GetClients(key)

	metric.ITEMS(w, r, http.StatusOK, result)
}

/**
* wsPublish
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (s *Server) wsPublish(w http.ResponseWriter, r *http.Request) {
	if s.ws == nil {
		response.HTTPError(w, r, http.StatusBadRequest, "Websocket not found")
		return
	}

	metric, ok := r.Context().Value(MetricKey).(*middleware.Metrics)
	if !ok {
		metric.HTTPError(w, r, http.StatusInternalServerError, MSG_METRIC_NOT_FOUND)
		return
	}

	body, _ := response.GetBody(r)
	channel := body.Str("channel")
	queue := body.Str("queue")
	ignored := body.ArrayStr("ignored")
	from := body.Json("from")
	message := body["message"]
	tpStr := body.Str("tp")
	tp := ws.ToTpMessage(tpStr)
	msg := ws.NewMessage(from, message, tp)
	s.ws.Publish(channel, queue, msg, ignored, from)

	metric.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"message": "Message published",
			"channel": channel,
		},
	})
}
