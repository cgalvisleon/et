package request

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/go-chi/chi/v5"
)

var (
	methods = map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"OPTIONS": true,
	}
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
* ToItem returns an Item object
* @return et.Item
**/
func (b Body) ToItem() (et.Item, error) {
	var result et.Item
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}

/**
* ToItems returns an Items object
* @return et.Items
**/
func (b Body) ToItems() (et.Items, error) {
	var result et.Items
	err := json.Unmarshal(b.Data, &result)
	if err != nil {
		return et.Items{}, err
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
* @return *Body, error
**/
func ReadBody(body io.ReadCloser) (*Body, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	return &Body{Data: bodyBytes}, nil
}

/**
* statusOk
* @param status int
* @return bool
**/
func statusOk(status int) bool {
	return status < http.StatusBadRequest
}

/**
* bodyParams
* @param header, body et.Json
* @return []byte
**/
func bodyParams(header, body et.Json) []byte {
	contentType := header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" {
		data := url.Values{}
		for k, v := range body {
			data.Set(k, v.(string))
		}
		return []byte(data.Encode())
	} else if contentType == "application/json" {
		return []byte(body.ToString())
	} else {
		return []byte(body.ToString())
	}
}

/**
* GetStr
* @param r *http.Request
* @return et.Json, error
**/
func GetStr(r *http.Request) (string, error) {
	body, err := ReadBody(r.Body)
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
	body, err := ReadBody(r.Body)
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
* URLParam
* @param r *http.Request, key string
* @return string
**/
func URLParam(r *http.Request, key string) string {
	result := chi.URLParam(r, key)
	return result
}

/**
* Query
* @param r *http.Request, key string
* @return string
**/
func Query(r *http.Request, key string) string {
	result := r.URL.Query().Get(key)
	return result
}
