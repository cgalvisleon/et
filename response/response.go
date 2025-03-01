package response

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
	"github.com/go-chi/chi/v5"
)

type Result struct {
	Ok     bool        `json:"ok"`
	Result interface{} `json:"result"`
}

/**
* ScanBody
* @param r io.Reader
* @return et.Json, error
**/
func ScanBody(r io.Reader) (et.Json, error) {
	var result et.Json
	err := json.NewDecoder(r).Decode(&result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* ScanStr
* @param value string
* @return et.Json, error
**/
func ScanStr(value string) (et.Json, error) {
	return ScanBody(strings.NewReader(value))
}

/**
* ScanJson
* @param value map[string]interface{}
* @return et.Json, error
**/
func ScanJson(value map[string]interface{}) (et.Json, error) {
	var result et.Json = value
	return result, nil
}

/**
* GetBody
* @param r *http.Request
* @return et.Json, error
**/
func GetBody(r *http.Request) (et.Json, error) {
	body, err := request.ReadBody(r.Body)
	if err != nil {
		return et.Json{}, err
	}

	result, err := body.ToJson()
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* GetArray
* @param r *http.Request
* @return []et.Json, error
**/
func GetArray(r *http.Request) ([]et.Json, error) {
	body, err := request.ReadBody(r.Body)
	if err != nil {
		return []et.Json{}, err
	}

	result, err := body.ToArrayJson()
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* GetQuery
* @param r *http.Request
* @return et.Json
**/
func GetQuery(r *http.Request) et.Json {
	var result et.Json = et.Json{}
	values := r.URL.Query()
	for key, value := range values {
		if len(value) > 0 {
			result.Set(key, value[0])
		}
	}

	return result
}

/**
* GetParam
* @param r *http.Request
* @param key string
* @return string
**/
func GetParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

/**
* WriteResponse
* @param w http.ResponseWriter
* @param statusCode int
* @param e []byte
* @return error
**/
func WriteResponse(w http.ResponseWriter, statusCode int, e []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(e)

	return nil
}

/**
* RJson
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data interface{}
* @return error
**/
func RESULT(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error {
	if data == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

/**
* JSON
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data interface{}
* @return error
**/
func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error {
	if data == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	result := Result{
		Ok:     http.StatusOK == statusCode,
		Result: data,
	}

	e, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

/**
* ITEM
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data et.Item
* @return error
**/
func ITEM(w http.ResponseWriter, r *http.Request, statusCode int, data et.Item) error {
	if &data == (&et.Item{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

/**
* ITEMS
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data et.Items
* @return error
**/
func ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, data et.Items) error {
	if &data == (&et.Items{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

/**
* DATA
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data et.Items
* @return error
**/
func DATA(w http.ResponseWriter, r *http.Request, statusCode int, data et.Json) error {
	if &data == (&et.Json{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return WriteResponse(w, statusCode, e)
}

/**
* ANY
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data et.Items
* @return error
**/
func ANY(w http.ResponseWriter, r *http.Request, statusCode int, result interface{}) error {
	switch v := result.(type) {
	case et.Item:
		return ITEM(w, r, statusCode, v)
	case et.Items:
		return ITEMS(w, r, statusCode, v)
	default:
		return JSON(w, r, statusCode, result)
	}
}

/**
* HTTPError
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param message string
* @return error
**/
func HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	msg := et.Json{
		"message": message,
	}

	return JSON(w, r, statusCode, msg)
}

func HTTPAlert(w http.ResponseWriter, r *http.Request, message string) error {
	return HTTPError(w, r, http.StatusBadRequest, message)
}

func Unauthorized(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, r, http.StatusUnauthorized, "401 Unauthorized")
}

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	HTTPError(w, r, http.StatusInternalServerError, "500 Autentication Server Error - "+err.Error())
}

func Forbidden(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, r, http.StatusForbidden, "403 Forbidden")
}

/**
* Stream
* @param w http.ResponseWriter
* @param r *http.Request
* @param statusCode int
* @param data interface{}
* @return error
**/
func Stream(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error {
	if data == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(data)
	if err != nil {
		return err
	}

	WriteResponse(w, statusCode, e)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

/**
* HTTPApp
* @param r chi.Router
* @param path string
* @param root http.FileSystem
**/
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
