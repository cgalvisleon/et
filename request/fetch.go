package request

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

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
		return nil, errors.New("CRT certificate path is required")
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return nil, errors.New("CRT certificate not found")
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
