package request

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

type Response struct {
	Body   *Body
	Status Status
	Header et.Json
}

/**
* ToJson returns a Json object
* @return et.Json
**/
func (r *Response) ToJson() et.Json {
	return et.Json{
		"body":   r.Body,
		"status": r.Status.ToJson(),
		"header": r.Header,
	}
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

/**
* statusOk
* @param status int
* @return bool
**/
func statusOk(status int) bool {
	return status < http.StatusBadRequest
}

/**
* FetchWithTls
* @param method, url string, header, body et.Json, tlsConfig *tls.Config
* @return *Response, error
**/
func FetchWithTls(method, url string, header, body et.Json, tlsConfig *tls.Config) (*Response, error) {
	if map[string]string{
		"GET":     "GET",
		"POST":    "POST",
		"PUT":     "PUT",
		"DELETE":  "DELETE",
		"PATCH":   "PATCH",
		"OPTIONS": "OPTIONS",
	}[method] == "" {
		return nil, fmt.Errorf("invalid method: %s", method)
	}

	bodyParams := []byte(body.ToString())
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyParams))
	if err != nil {
		return nil, err
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	if tlsConfig != nil {
		transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	client := &http.Client{
		Transport: transport,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bodyResponse, err := ReadBody(res.Body)
	if err != nil {
		return nil, err
	}

	headerResponse := et.Json{}
	for k, v := range res.Header {
		headerResponse[k] = v[0]
	}

	return &Response{
		Body: bodyResponse,
		Status: Status{
			Ok:      statusOk(res.StatusCode),
			Code:    res.StatusCode,
			Message: res.Status,
		},
		Header: headerResponse,
	}, nil
}

/**
* Fetch
* @param method, url string, header, body et.Json
* @return *Response, error
**/
func Fetch(method, url string, header, body et.Json) (*Response, error) {
	return FetchWithTls(method, url, header, body, nil)
}

/**
* NewTlsConfig
* @param caPath, certPath, keyPath string
* @return *tls.Config, error
**/
func NewTlsConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	if caPath == "" {
		return nil, fmt.Errorf("CA certificate path is required")
	}

	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("CA certificate not found")
	}

	caCert, _ := os.ReadFile(caPath)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}, nil
}
