package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/et"
)

type Status struct {
	Ok      bool   `json:"ok"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToJson returns a Json object
func (s Status) ToJson() et.Json {
	return et.Json{
		"ok":      s.Ok,
		"code":    s.Code,
		"message": s.Message,
	}
}

// ToString returns a string
func (s Status) ToString() string {
	return s.ToJson().ToString()
}

/**
* Body struct to convert the body response
**/
type Body struct {
	Data []byte
}

/**
* ToJson returns a Json object
* @return et.Json
**/
func (b Body) ToJson() (et.Json, error) {
	var result et.Json
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* ToArrayJson returns a Json array object
* @return []et.Json
**/
func (b Body) ToArrayJson() ([]et.Json, error) {
	var result []et.Json
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* ToString returns a string
* @return string
**/
func (b Body) ToString() string {
	return string(b.Data)
}

/**
* ToInt returns an integer
* @return int
**/
func (b Body) ToInt() (int, error) {
	var result int
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* ToInt64 returns an integer
* @return int64
**/
func (b Body) ToInt64() (int64, error) {
	var result int64
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* ToFloat returns a float
* @return float64
**/
func (b Body) ToFloat() (float64, error) {
	var result float64
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* ToBool returns a boolean
* @return bool
**/
func (b Body) ToBool() (bool, error) {
	var result bool
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return false, err
	}

	return result, nil
}

/**
* ToTime returns a time
* @return time.Time
**/
func (b Body) ToTime() (time.Time, error) {
	var result time.Time
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return time.Time{}, err
	}

	return result, nil
}

/**
* ReadBody reads the body response
* @param body io.ReadCloser
* @return *Body
* @return error
**/
func ReadBody(body io.ReadCloser) (*Body, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	return &Body{Data: bodyBytes}, nil
}

// Return true if status code is ok
func statusOk(status int) bool {
	return status < http.StatusBadRequest
}

// Request post method
func Post(url string, header, body et.Json) (*Body, Status) {
	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	return result, Status{Ok: statusOk(res.StatusCode), Code: res.StatusCode, Message: res.Status}
}

// Request get method
func Get(url string, header et.Json) (*Body, Status) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	return result, Status{Ok: statusOk(res.StatusCode), Code: res.StatusCode, Message: res.Status}
}

// Request put method
func Put(url string, header, body et.Json) (*Body, Status) {
	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	return result, Status{Ok: statusOk(res.StatusCode), Code: res.StatusCode, Message: res.Status}
}

// Request delete method
func Delete(url string, header et.Json) (*Body, Status) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	return result, Status{Ok: statusOk(res.StatusCode), Code: res.StatusCode, Message: res.Status}
}

// Request patch method
func Patch(url string, header, body et.Json) (*Body, Status) {
	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	return result, Status{Ok: statusOk(res.StatusCode), Code: res.StatusCode, Message: res.Status}
}

// Request options method
func Options(url string, header et.Json) (*Body, Status) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	return result, Status{Ok: statusOk(res.StatusCode), Code: res.StatusCode, Message: res.Status}
}
