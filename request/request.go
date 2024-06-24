package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/cgalvisleon/et/et"
)

type Status struct {
	Ok      bool   `json:"ok"`
	Group   string `json:"group"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToJson returns a Json object
func (s Status) ToJson() et.Json {
	return et.Json{
		"ok":      s.Ok,
		"group":   s.Group,
		"code":    s.Code,
		"message": s.Message,
	}
}

// ToString returns a string
func (s Status) ToString() string {
	return s.ToJson().ToString()
}

// ioReadeToJson reads the io.Reader and returns a Json object
func ioReadeToJson(r io.Reader) (et.Json, error) {
	var result et.Json
	err := json.NewDecoder(r).Decode(&result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

// Return true if status code is ok
func StatusValid(status int, message string) Status {
	var group string
	if status < 200 {
		group = "Informational responses"
	} else if status < 300 {
		group = "Successful responses"
	} else if status < 400 {
		group = "Redirection messages"
	} else if status < 500 {
		group = "Client error responses"
	} else {
		group = "Server error responses"
	}

	return Status{
		Ok:      status < http.StatusBadRequest,
		Code:    status,
		Group:   group,
		Message: message,
	}
}

// Request post method
func Post(url string, header, body et.Json) (et.Json, Status) {
	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, StatusValid(res.StatusCode, err.Error())
	}
	defer res.Body.Close()

	result, err := ioReadeToJson(res.Body)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	return result, StatusValid(res.StatusCode, http.StatusText(res.StatusCode))
}

// Request get method
func Get(url string, header et.Json) (et.Json, Status) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, StatusValid(res.StatusCode, err.Error())
	}
	defer res.Body.Close()

	result, err := ioReadeToJson(res.Body)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	return result, StatusValid(res.StatusCode, http.StatusText(res.StatusCode))
}

// Request put method
func Put(url string, header, body et.Json) (et.Json, Status) {
	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, StatusValid(res.StatusCode, err.Error())
	}
	defer res.Body.Close()

	result, err := ioReadeToJson(res.Body)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	return result, StatusValid(res.StatusCode, http.StatusText(res.StatusCode))
}

// Request delete method
func Delete(url string, header et.Json) (et.Json, Status) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, StatusValid(res.StatusCode, err.Error())
	}
	defer res.Body.Close()

	result, err := ioReadeToJson(res.Body)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	return result, StatusValid(res.StatusCode, http.StatusText(res.StatusCode))
}

// Request patch method
func Patch(url string, header, body et.Json) (et.Json, Status) {
	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, StatusValid(res.StatusCode, err.Error())
	}
	defer res.Body.Close()

	result, err := ioReadeToJson(res.Body)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	return result, StatusValid(res.StatusCode, http.StatusText(res.StatusCode))
}

// Request options method
func Options(url string, header et.Json) (et.Json, Status) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return et.Json{}, StatusValid(res.StatusCode, err.Error())
	}
	defer res.Body.Close()

	result, err := ioReadeToJson(res.Body)
	if err != nil {
		return et.Json{}, StatusValid(http.StatusBadRequest, err.Error())
	}

	return result, StatusValid(res.StatusCode, http.StatusText(res.StatusCode))
}
