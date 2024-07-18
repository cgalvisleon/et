package response

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/utility"
	"github.com/go-chi/chi/v5"
)

type Result struct {
	Ok     bool        `json:"ok"`
	Result interface{} `json:"result"`
}

// ScanBody function to scan the body of a request
func ScanBody(r io.Reader) (js.Json, error) {
	var result js.Json
	err := json.NewDecoder(r).Decode(&result)
	if err != nil {
		return js.Json{}, err
	}

	return result, nil
}

// ScanStr function to scan a string
func ScanStr(value string) (js.Json, error) {
	return ScanBody(strings.NewReader(value))
}

// ScanJson function to scan a json
func ScanJson(value map[string]interface{}) (js.Json, error) {
	var result js.Json = value
	return result, nil
}

// Client function to get the client information
func Client(r *http.Request) js.Json {
	now := utility.Now()
	ctx := r.Context()

	return js.Json{
		"date_of":   now,
		"client_id": generic.New(ctx.Value("clientId")).Str(),
		"name":      generic.New(ctx.Value("name")).Str(),
	}
}

// GetBody function to get the body of a request
func GetBody(r *http.Request) (js.Json, error) {
	var result js.Json
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		return js.Json{}, err
	}
	defer r.Body.Close()

	return result, nil
}

// GetQuery function to get the query of a request
func GetQuery(r *http.Request) js.Json {
	var result js.Json = js.Json{}
	values := r.URL.Query()
	for key, value := range values {
		if len(value) > 0 {
			result.Set(key, value[0])
		}
	}

	return result
}

// GetParam function to get the param of a request
func GetParam(r *http.Request, key string) *generic.Any {
	val := chi.URLParam(r, key)
	result := generic.New(val)

	return result
}

// WriteResponse function to write a response
func WriteResponse(w http.ResponseWriter, statusCode int, e []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(e)

	return nil
}

// JSON function to return a json response
func JSON(w http.ResponseWriter, r *http.Request, statusCode int, dt interface{}) error {
	if dt == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	result := Result{
		Ok:     http.StatusOK == statusCode,
		Result: dt,
	}

	e, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

// ITEM function to return a json response
func ITEM(w http.ResponseWriter, r *http.Request, statusCode int, dt js.Item) error {
	if &dt == (&js.Item{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

// ITEMS function to return a json response
func ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, dt js.Items) error {
	if &dt == (&js.Items{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

// HTTPError function to return a json response
func HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	msg := js.Json{
		"message": message,
	}

	return JSON(w, r, statusCode, msg)
}

// HTTPAlert function to return a json response
func HTTPAlert(w http.ResponseWriter, r *http.Request, message string) error {
	return HTTPError(w, r, http.StatusBadRequest, message)
}

// Stream function to return a json response
func Stream(w http.ResponseWriter, r *http.Request, statusCode int, dt interface{}) error {
	if dt == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}

	WriteResponse(w, statusCode, e)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

// HTTPApp function to return a json response
func HTTPApp(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		h := http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP
		r.Get(path, h)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
