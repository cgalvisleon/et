package response

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/request"
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
* GetStr
* @param r *http.Request
* @return et.Json, error
**/
func GetStr(r *http.Request) (string, error) {
	body, err := request.ReadBody(r.Body)
	if err != nil {
		return "", err
	}

	result := body.ToString()
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
* @param r *http.Request, key string
* @return string
**/
func GetParam(r *http.Request, key string) string {
	return r.PathValue(key)
}

/**
* WriteResponse
* @param w http.ResponseWriter, statusCode int, e []byte
* @return error
**/
func WriteResponse(w http.ResponseWriter, statusCode int, e []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(e)

	return nil
}

/**
* RESULT
* @param w http.ResponseWriter, r *http.Request, statusCode int, data interface{}
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
* @param w http.ResponseWriter, r *http.Request, statusCode int, data interface{}
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
* @param w http.ResponseWriter, r *http.Request, statusCode int, data et.Item
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
* @param w http.ResponseWriter, r *http.Request, statusCode int, data et.Items
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
* @param w http.ResponseWriter, r *http.Request, statusCode int, data et.Json
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
* @param w http.ResponseWriter, r *http.Request, statusCode int, result interface{}
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
* @param w http.ResponseWriter, r *http.Request, statusCode int, message string
* @return error
**/
func HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	msg := et.Json{
		"message": message,
	}

	return JSON(w, r, statusCode, msg)
}

/**
* HTTPAlert
* @param w http.ResponseWriter, r *http.Request, message string
* @return error
**/
func HTTPAlert(w http.ResponseWriter, r *http.Request, message string) error {
	return HTTPError(w, r, http.StatusBadRequest, message)
}

/**
* Unauthorized
* @param w http.ResponseWriter, r *http.Request
**/
func Unauthorized(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, r, http.StatusUnauthorized, "401 Unauthorized")
}

/**
* InternalServerError
* @param w http.ResponseWriter, r *http.Request, err error
**/
func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	HTTPError(w, r, http.StatusInternalServerError, "500 Autentication Server Error - "+err.Error())
}

/**
* Forbidden
* @param w http.ResponseWriter, r *http.Request
**/
func Forbidden(w http.ResponseWriter, r *http.Request) {
	HTTPError(w, r, http.StatusForbidden, "403 Forbidden")
}

type DataFunction func(page, rows int) (et.Items, error)

/**
* Stream
* @param w http.ResponseWriter, r *http.Request, rows int, getData DataFunction
**/
func Stream(w http.ResponseWriter, r *http.Request, rows int, getData DataFunction) {
	page := 1
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("["))

	first := true
	for {
		items, err := getData(page, rows)
		if err != nil {
			HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		if !items.Ok {
			break
		}

		if page > 1 {
			w.Write([]byte(","))
		}

		for _, item := range items.Result {
			if !first {
				w.Write([]byte(",")) // separador entre objetos
			}
			val := item.ToEscapeHTML()
			w.Write([]byte(val))
			first = false
		}

		page++
	}

	w.Write([]byte("]"))
}
