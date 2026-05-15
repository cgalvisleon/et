package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

/**
* defaultTransport: Transport compartido que reutiliza conexiones TCP entre requests.
**/
var defaultTransport = &http.Transport{
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   10,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

/**
* defaultClient: Cliente HTTP para requests normales con timeout end-to-end de 15 s.
**/
var defaultClient = &http.Client{
	Transport: defaultTransport,
	Timeout:   15 * time.Second,
}

/**
* StreamClient: Cliente HTTP para respuestas streaming. Sin timeout de body;
* solo aplica ResponseHeaderTimeout para no colgar en servers lentos.
**/
var StreamClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	},
}

/**
* HttpWithContext: Ejecuta un request HTTP propagando el context del caller.
* @param ctx context.Context
* @param method string
* @param url string
* @param header et.Json
* @param body et.Json
* @param tlsConfig *tls.Config
* @return *Body, Status
**/
func HttpWithContext(ctx context.Context, method, url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	if _, ok := methods[method]; !ok {
		return nil, Status{
			Ok:      false,
			Code:    http.StatusBadRequest,
			Message: "Invalid method",
		}
	}

	var ioBody io.Reader
	if body != nil {
		bodyBytes := bodyParams(header, body)
		ioBody = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, ioBody)
	if err != nil {
		return nil, Status{Ok: false, Code: http.StatusBadRequest, Message: err.Error()}
	}

	for k, v := range header {
		req.Header.Set(k, v.(string))
	}

	client := defaultClient
	if tlsConfig != nil {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:       tlsConfig,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Timeout: 15 * time.Second,
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
* Http: Ejecuta un request HTTP con context.Background().
* @param method string
* @param url string
* @param header et.Json
* @param body et.Json
* @param tlsConfig *tls.Config
* @return *Body, Status
**/
func Http(method, url string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return HttpWithContext(context.Background(), method, url, header, body, tlsConfig)
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
