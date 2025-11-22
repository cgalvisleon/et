package request

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
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
* Http
* @param method, url string, header, body et.Json
* @return *Body, Status
**/
func Http(method, url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	if _, ok := methods[method]; !ok {
		return nil, Status{
			Ok:      false,
			Code:    http.StatusBadRequest,
			Message: "Invalid method",
		}
	}

	var ioBody io.Reader
	if body != nil {
		bodyParams := bodyParams(header, body)
		ioBody = bytes.NewBuffer(bodyParams)
	}
	req, err := http.NewRequest(method, url, ioBody)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	if tlsConfig != nil {
		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, Status{
			Ok:      false,
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}
	defer res.Body.Close()

	result, err := ReadBody(res.Body)
	if err != nil {
		return nil, Status{
			Ok:      false,
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}

	return result, Status{
		Ok:      statusOk(res.StatusCode),
		Code:    res.StatusCode,
		Message: res.Status,
	}
}

/**
* Fetch
* @param method, url string, header, body et.Json
* @return *Body, Status
**/
func Fetch(method, url string, header, body et.Json) (*Body, Status) {
	return Http(method, url, header, body, nil)
}

/**
* Post
* @param url string, header, body et.Json
* @return *Body, Status
**/
func Post(url string, header, body et.Json) (*Body, Status) {
	return Http("POST", url, header, body, nil)
}

/**
* Get
* @param url string, header et.Json
* @return *Body, Status
**/
func Get(url string, header et.Json) (*Body, Status) {
	return Http("GET", url, header, nil, nil)
}

/**
* Put
* @param url string, header, body et.Json
* @return *Body, Status
**/
func Put(url string, header, body et.Json) (*Body, Status) {
	return Http("PUT", url, header, body, nil)
}

/**
* Delete
* @param url string, header et.Json
* @return *Body, Status
**/
func Delete(url string, header et.Json) (*Body, Status) {
	return Http("DELETE", url, header, et.Json{}, nil)
}

/**
* Patch
* @param url string, header, body et.Json
* @return *Body, Status
**/
func Patch(url string, header, body et.Json) (*Body, Status) {
	return Http("PATCH", url, header, body, nil)
}

/**
* Options
* @param url string, header et.Json
* @return *Body, Status
**/
func Options(url string, header et.Json) (*Body, Status) {
	return Http("OPTIONS", url, header, et.Json{}, nil)
}

/**
* PostWithTls
* @param url string, header, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func PostWithTls(url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("POST", url, header, body, tlsConfig)
}

/**
* GetWithTls
* @param url string, header et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func GetWithTls(url string, header et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("GET", url, header, et.Json{}, tlsConfig)
}

/**
* PutWithTls
* @param url string, header, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func PutWithTls(url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("PUT", url, header, body, tlsConfig)
}

/**
* DeleteWithTls
* @param url string, header et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func DeleteWithTls(url string, header et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("DELETE", url, header, et.Json{}, tlsConfig)
}

/**
* PatchWithCA
* @param url string, header, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func PatchWithTls(url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("PATCH", url, header, body, tlsConfig)
}

/**
* OptionsWithTls
* @param url string, header et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func OptionsWithTls(url string, header et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("OPTIONS", url, header, et.Json{}, tlsConfig)
}

/**
* NewTlsConfig
* @param caPath, certPath, keyPath string
* @return *tls.Config, error
**/
func NewTlsConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	if certFile == "" {
		return nil, fmt.Errorf("CRT certificate path is required")
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("CRT certificate not found")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	if !utility.ValidStr(caFile, 0, []string{""}) {
		return tlsConfig, nil
	}

	caCert, err := os.ReadFile(caFile)
	if !os.IsNotExist(err) {
		tlsConfig.RootCAs = x509.NewCertPool()
		tlsConfig.RootCAs.AppendCertsFromPEM(caCert)
	}

	return tlsConfig, nil
}
