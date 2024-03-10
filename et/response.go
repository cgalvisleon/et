package et

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Result struct {
	Ok     bool        `json:"ok"`
	Result interface{} `json:"result"`
}

func ScanBody(r io.Reader) (Json, error) {
	var result Json
	err := json.NewDecoder(r).Decode(&result)
	if err != nil {
		return Json{}, err
	}

	return result, nil
}

func ScanStr(value string) (Json, error) {
	return ScanBody(strings.NewReader(value))
}

func ScanJson(value map[string]interface{}) (Json, error) {
	var result Json = value
	return result, nil
}

func GetBody(r *http.Request) (Json, error) {
	var result Json
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		return Json{}, err
	}
	defer r.Body.Close()

	return result, nil
}

func GetQuery(r *http.Request) Json {
	var result Json = Json{}
	values := r.URL.Query()
	for key, value := range values {
		if len(value) > 0 {
			result.Set(key, value[0])
		}
	}

	return result
}

func WriteResponse(w http.ResponseWriter, statusCode int, e []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(e)

	return nil
}

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

func ITEM(w http.ResponseWriter, r *http.Request, statusCode int, dt Item) error {
	if &dt == (&Item{}) {
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

func ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, dt Items) error {
	if &dt == (&Items{}) {
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

func HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	msg := Json{
		"message": message,
	}

	return JSON(w, r, statusCode, msg)
}

func HTTPAlert(w http.ResponseWriter, r *http.Request, message string) error {
	return HTTPError(w, r, http.StatusBadRequest, message)
}

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
