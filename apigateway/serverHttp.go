package apigateway

import (
	"io"
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/rs/cors"
)

type HttpServer struct {
	addr            string
	handler         http.Handler
	mux             *http.ServeMux
	notFoundHandler http.HandlerFunc
	// middlewares     []func(http.Handler) http.Handler
}

func NewHttpServer() *HttpServer {
	// Create a new server
	mux := http.NewServeMux()

	port := envar.EnvarInt(3300, "PORT")
	result := &HttpServer{
		addr:    et.Format(":%d", port),
		handler: cors.AllowAll().Handler(mux),
		mux:     mux,
	}

	// Handler router
	mux.HandleFunc("/version", result.Version)
	mux.HandleFunc("/", result.Handler)

	result.notFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		et.HTTPError(w, r, http.StatusNotFound, "404 Not Found.")
	}

	return result
}

func (s *HttpServer) Version(w http.ResponseWriter, r *http.Request) {
	result := Version()
	et.JSON(w, r, http.StatusOK, result)
}

func (s *HttpServer) Handler(w http.ResponseWriter, r *http.Request) {
	resolute := NewResolute(r)

	if resolute.Resolve == nil {
		s.notFoundHandler(w, r)
		return
	}

	kind := resolute.Resolve.Node.Resolve.ValStr("HTTP", "kind")
	if kind == "REST" {
		request, err := http.NewRequest(resolute.Method, resolute.URL, resolute.Body)
		if err != nil {
			et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		request.Header = resolute.Header
		client := &http.Client{}
		res, err := client.Do(request)
		if err != nil {
			et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		defer res.Body.Close()

		for key, value := range res.Header {
			w.Header().Set(key, value[0])
		}
		_, err = io.Copy(w, res.Body)
		if err != nil {
			et.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		}

		return
	}

	/*
		if kind == "WEBSOCKET" {
			// TODO

		}
	*/

	http.Redirect(w, r, resolute.URL, http.StatusSeeOther)
}

func (s *HttpServer) NotFound(handlerFn http.HandlerFunc) {
	s.notFoundHandler = handlerFn
}

func (s *HttpServer) HandlerWebSocket(handlerFn http.HandlerFunc) {

}
