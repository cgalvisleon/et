package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sync"
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
* bufPool: Pool de buffers para reutilizar memoria.
**/
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type HttpResult struct {
	Body   *Body
	Status Status
}

/**
* HttpWithContext: Ejecuta un request HTTP propagando el context del caller.
* @param ctx context.Context, method string, path string, header et.Json, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func HttpWithContext(ctx context.Context, method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	result := make(chan HttpResult, 1)

	go func() {
		if _, ok := methods[method]; !ok {
			result <- HttpResult{
				Body: nil,
				Status: Status{
					Ok:      false,
					Code:    http.StatusBadRequest,
					Message: "Invalid method",
				},
			}
			return
		}

		contentType := header.Str("Content-Type")

		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			result <- HttpResult{
				Body: nil,
				Status: Status{
					Ok:      false,
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
			return
		}

		var ioBody io.Reader
		var buf *bytes.Buffer

		switch mediaType {
		case "multipart/form-data":
			writer := multipart.NewWriter(buf)
			for k := range body {
				v := body.Str(k)
				if err := writer.WriteField(k, v); err != nil {
					result <- HttpResult{
						Body: nil,
						Status: Status{
							Ok:      false,
							Code:    http.StatusBadRequest,
							Message: err.Error(),
						},
					}
					return
				}
			}
			if err := writer.Close(); err != nil {
				result <- HttpResult{
					Body: nil,
					Status: Status{
						Ok:      false,
						Code:    http.StatusBadRequest,
						Message: err.Error(),
					},
				}
				return
			}
			ioBody = buf
		case "application/x-www-form-urlencoded":
			data := url.Values{}
			for k := range body {
				v := body.Str(k)
				data.Set(k, v)
			}
			ioBody = bytes.NewBufferString(data.Encode())
		case "application/json":
			if body != nil {
				buf = bufPool.Get().(*bytes.Buffer)
				buf.Reset()
				buf.Write(bodyParams(header, body))
				ioBody = buf
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, uRL, ioBody)
		if err != nil {
			result <- HttpResult{
				Body: nil,
				Status: Status{
					Ok:      false,
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
			return
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
			result <- HttpResult{
				Body: nil,
				Status: Status{
					Ok:      false,
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
			return
		}
		defer res.Body.Close()

		resultBody, err := ReadBody(res.Body)
		if err != nil {
			result <- HttpResult{
				Body: nil,
				Status: Status{
					Ok:      false,
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
			return
		}

		result <- HttpResult{
			Body: resultBody,
			Status: Status{
				Ok:      IsStatusOk(res.StatusCode),
				Code:    res.StatusCode,
				Message: res.Status,
			},
		}
		return
	}()

	if timeout == 0 {
		select {
		case respuesta := <-result:
			return respuesta.Body, respuesta.Status
		}
	} else {
		select {
		case respuesta := <-result:
			// La función terminó antes de timeout
			return respuesta.Body, respuesta.Status

		case <-time.After(timeout):
			// Se agotaron los timeout
			return newBody(defaultValue), Status{
				Ok:      false,
				Code:    http.StatusRequestTimeout,
				Message: "timeout",
			}
		}
	}
}

/**
* Http: Ejecuta un request HTTP con context.Background().
* @param method string, uRL string, header et.Json, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func Http(method, uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return HttpWithContext(context.Background(), method, uRL, header, body, tlsConfig, 0, nil)
}

/**
* Fetch
* @param method, uRL string, header, body et.Json
* @return *Body, Status
**/
func Fetch(method, uRL string, header, body et.Json) (*Body, Status) {
	return Http(method, uRL, header, body, nil)
}

/**
* Post
* @param uRL string, header, body et.Json
* @return *Body, Status
**/
func Post(uRL string, header, body et.Json) (*Body, Status) {
	return Http("POST", uRL, header, body, nil)
}

/**
* Get
* @param uRL string, header et.Json
* @return *Body, Status
**/
func Get(uRL string, header et.Json) (*Body, Status) {
	return Http("GET", uRL, header, nil, nil)
}

/**
* Put
* @param uRL string, header, body et.Json
* @return *Body, Status
**/
func Put(uRL string, header, body et.Json) (*Body, Status) {
	return Http("PUT", uRL, header, body, nil)
}

/**
* Delete
* @param uRL string, header et.Json
* @return *Body, Status
**/
func Delete(uRL string, header et.Json) (*Body, Status) {
	return Http("DELETE", uRL, header, et.Json{}, nil)
}

/**
* Patch
* @param uRL string, header, body et.Json
* @return *Body, Status
**/
func Patch(uRL string, header, body et.Json) (*Body, Status) {
	return Http("PATCH", uRL, header, body, nil)
}

/**
* Options
* @param uRL string, header et.Json
* @return *Body, Status
**/
func Options(uRL string, header et.Json) (*Body, Status) {
	return Http("OPTIONS", uRL, header, et.Json{}, nil)
}

/**
* PostWithTls
* @param uRL string, header, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func PostWithTls(uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("POST", uRL, header, body, tlsConfig)
}

/**
* GetWithTls
* @param uRL string, header et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func GetWithTls(uRL string, header et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("GET", uRL, header, et.Json{}, tlsConfig)
}

/**
* PutWithTls
* @param uRL string, header, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func PutWithTls(uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("PUT", uRL, header, body, tlsConfig)
}

/**
* DeleteWithTls
* @param uRL string, header et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func DeleteWithTls(uRL string, header et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("DELETE", uRL, header, et.Json{}, tlsConfig)
}

/**
* PatchWithCA
* @param uRL string, header, body et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func PatchWithTls(uRL string, header, body et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("PATCH", uRL, header, body, tlsConfig)
}

/**
* OptionsWithTls
* @param url string, header et.Json, tlsConfig *tls.Config
* @return *Body, Status
**/
func OptionsWithTls(uRL string, header et.Json, tlsConfig *tls.Config) (*Body, Status) {
	return Http("OPTIONS", uRL, header, et.Json{}, tlsConfig)
}

/**
* HttpCtxWithTimeout
* @param ctx context.Context, method string, path string, header et.Json, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func HttpCtxWithTimeout(ctx context.Context, method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(ctx, method, uRL, header, body, tlsConfig, timeout, defaultValue)
}

/**
* HttpWithTimeout
* @param method string, path string, header et.Json, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func HttpWithTimeout(method, uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), method, uRL, header, body, tlsConfig, timeout, defaultValue)
}

/**
* PostWithTlsTimeout
* @param uRL string, header et.Json, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func PostWithTlsTimeout(uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "POST", uRL, header, body, tlsConfig, timeout, defaultValue)
}

/**
* GetWithTlsTimeout
* @param uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func GetWithTlsTimeout(uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "GET", uRL, header, et.Json{}, tlsConfig, timeout, defaultValue)
}

/**
* PutWithTimeout
* @param uRL string, header et.Json, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func PutWithTlsTimeout(uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "PUT", uRL, header, body, tlsConfig, timeout, defaultValue)
}

/**
* DeleteWithTimeout
* @param uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func DeleteWithTlsTimeout(uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "DELETE", uRL, header, et.Json{}, tlsConfig, timeout, defaultValue)
}

/**
* PatchWithTimeout
* @param uRL string, header et.Json, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func PatchWithTlsTimeout(uRL string, header, body et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "PATCH", uRL, header, body, tlsConfig, timeout, defaultValue)
}

/**
* OptionsWithTimeout
* @param uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func OptionsWithTlsTimeout(uRL string, header et.Json, tlsConfig *tls.Config, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "OPTIONS", uRL, header, et.Json{}, tlsConfig, timeout, defaultValue)
}

/**
* PostWithTimeout
* @param uRL string, header et.Json, body et.Json, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func PostWithTimeout(uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "POST", uRL, header, body, nil, timeout, defaultValue)
}

/**
* GetWithTimeout
* @param uRL string, header et.Json, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func GetWithTimeout(uRL string, header et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "GET", uRL, header, et.Json{}, nil, timeout, defaultValue)
}

/**
* PutWithTimeout
* @param uRL string, header et.Json, body et.Json, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func PutWithTimeout(uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "PUT", uRL, header, body, nil, timeout, defaultValue)
}

/**
* DeleteWithTimeout
* @param uRL string, header et.Json, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func DeleteWithTimeout(uRL string, header et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "DELETE", uRL, header, et.Json{}, nil, timeout, defaultValue)
}

/**
* PatchWithTimeout
* @param uRL string, header et.Json, body et.Json, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func PatchWithTimeout(uRL string, header, body et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "PATCH", uRL, header, body, nil, timeout, defaultValue)
}

/**
* OptionsWithTimeout
* @param uRL string, header et.Json, timeout time.Duration, defaultValue []byte
* @return *Body, Status
**/
func OptionsWithTimeout(uRL string, header et.Json, timeout time.Duration, defaultValue []byte) (*Body, Status) {
	return HttpWithContext(context.Background(), "OPTIONS", uRL, header, et.Json{}, nil, timeout, defaultValue)
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
