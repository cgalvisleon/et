package jwsp

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
)

// Valores por defecto del cliente
const (
	DefaultBaseURL    = "https://graph.facebook.com"
	DefaultVersion    = "v25.0"
	DefaultTimeout    = 30 * time.Second
	DefaultRetries    = 2
	DefaultRetryDelay = 500 * time.Millisecond
	// maxRetryDelay: la espera más larga entre intentos (también acota Retry-After)
	maxRetryDelay = 10 * time.Second
	// maxResponse: el cuerpo de respuesta más grande que se lee de la Graph API (no de una descarga de media)
	maxResponse = 10 << 20
)

// Client: un cliente de la Graph API de WhatsApp Cloud. PhoneNumberId envía y sube media; WabaId maneja las plantillas
// y la suscripción de la app. Retries son los reintentos ante 429, 5xx o un límite de Meta (un POST sin respuesta no se
// reintenta); RetryDelay, la primera espera (se duplica en cada intento). HTTPClient es opcional (sin él, uno con Timeout)
type Client struct {
	BaseURL       string
	Version       string
	Token         string
	PhoneNumberId string
	WabaId        string
	Timeout       time.Duration
	Retries       int
	RetryDelay    time.Duration
	HTTPClient    *http.Client
}

/**
* New: un cliente con los valores por defecto (graph.facebook.com, v25.0, 30 s, 2 reintentos)
* @param token, phoneNumberId string
* @return *Client
**/
func New(token, phoneNumberId string) *Client {
	return &Client{
		BaseURL:       DefaultBaseURL,
		Version:       DefaultVersion,
		Token:         token,
		PhoneNumberId: phoneNumberId,
		Timeout:       DefaultTimeout,
		Retries:       DefaultRetries,
		RetryDelay:    DefaultRetryDelay,
	}
}

/**
* URL: la base con la versión ({base}/{version})
* @return string
**/
func (s *Client) URL() string {
	base := strings.TrimRight(cmp.Or(s.BaseURL, DefaultBaseURL), "/")
	return base + "/" + strings.Trim(cmp.Or(s.Version, DefaultVersion), "/")
}

/**
* httpClient
* @return *http.Client
**/
func (s *Client) httpClient() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: cmp.Or(s.Timeout, DefaultTimeout)}
}

/**
* phonePath: /{phone_number_id}{suffix}; sin número, MSG_PHONE_REQUIRED
* @param suffix string
* @return string, error
**/
func (s *Client) phonePath(suffix string) (string, error) {
	if s.PhoneNumberId == "" {
		return "", errors.New(MSG_PHONE_REQUIRED)
	}
	return "/" + url.PathEscape(s.PhoneNumberId) + suffix, nil
}

/**
* wabaPath: /{waba_id}{suffix}; sin cuenta, MSG_WABA_REQUIRED
* @param suffix string
* @return string, error
**/
func (s *Client) wabaPath(suffix string) (string, error) {
	if s.WabaId == "" {
		return "", errors.New(MSG_WABA_REQUIRED)
	}
	return "/" + url.PathEscape(s.WabaId) + suffix, nil
}

/**
* Call: una llamada a la Graph API con body en JSON (nil: sin cuerpo). Un estado no 2xx o un {error} en la respuesta es un *Error
* @param ctx context.Context, method, path string, query url.Values, body any
* @return et.Json, error
**/
func (s *Client) Call(ctx context.Context, method, path string, query url.Values, body any) (et.Json, error) {
	var raw []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return et.Json{}, err
		}
		raw = data
	}

	return s.do(ctx, method, path, query, "application/json", raw)
}

/**
* do: la llamada con reintentos acotados; raw se vuelve a mandar en cada intento
* @param ctx context.Context, method, path string, query url.Values, contentType string, raw []byte
* @return et.Json, error
**/
func (s *Client) do(ctx context.Context, method, path string, query url.Values, contentType string, raw []byte) (et.Json, error) {
	if s.Token == "" {
		return et.Json{}, errors.New(MSG_TOKEN_REQUIRED)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	target := s.URL() + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	delay := cmp.Or(s.RetryDelay, DefaultRetryDelay)
	for attempt := 0; ; attempt++ {
		result, wait, err := s.once(ctx, method, target, contentType, raw)
		if err == nil {
			return result, nil
		}
		if attempt >= s.Retries || !retryable(method, err) {
			return et.Json{}, err
		}

		pause := min(max(delay<<attempt, wait), maxRetryDelay)
		select {
		case <-ctx.Done():
			return et.Json{}, err
		case <-time.After(pause):
		}
	}
}

/**
* once: un intento; devuelve la espera de Retry-After si Meta la manda
* @param ctx context.Context, method, target, contentType string, raw []byte
* @return et.Json, time.Duration, error
**/
func (s *Client) once(ctx context.Context, method, target, contentType string, raw []byte) (et.Json, time.Duration, error) {
	var reader io.Reader
	if raw != nil {
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return et.Json{}, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/json")
	if raw != nil {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := s.httpClient().Do(req)
	if err != nil {
		return et.Json{}, 0, &transportError{err: err}
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, maxResponse))
	if err != nil {
		return et.Json{}, 0, &transportError{err: err}
	}

	wait := retryAfterOf(res.Header.Get("Retry-After"))
	output := et.Json{}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && json.Unmarshal(trimmed, &output) != nil {
		output = nil
	}
	if res.StatusCode < 200 || res.StatusCode > 299 || output == nil || len(output.Json("error")) > 0 {
		if res.StatusCode >= 200 && res.StatusCode <= 299 && output == nil {
			return et.Json{}, 0, &Error{Status: res.StatusCode, Message: string(trimmed), Body: string(trimmed)}
		}
		return et.Json{}, wait, errorOf(res.StatusCode, trimmed)
	}

	return output, 0, nil
}

// transportError: la llamada no tuvo respuesta (red, timeout)
type transportError struct {
	err error
}

/**
* Error
* @return string
**/
func (s *transportError) Error() string {
	return s.err.Error()
}

/**
* Unwrap
* @return error
**/
func (s *transportError) Unwrap() error {
	return s.err
}

/**
* retryable: un *Error temporal; sin respuesta, solo GET y DELETE (un POST pudo haber llegado a Meta)
* @param method string, err error
* @return bool
**/
func retryable(method string, err error) bool {
	if failure, ok := AsError(err); ok {
		return failure.IsTemporary()
	}
	var transport *transportError
	if errors.As(err, &transport) {
		if errors.Is(err, context.Canceled) {
			return false
		}
		return method == http.MethodGet || method == http.MethodDelete
	}
	return false
}

/**
* retryAfterOf: Retry-After en segundos (0 si no viene o no es un número)
* @param value string
* @return time.Duration
**/
func retryAfterOf(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

/**
* SubscribeApp: suscribe la app del token a la cuenta (POST /{waba_id}/subscribed_apps): los webhooks van a la URL de callback de la app
* @param ctx context.Context
* @return et.Json, error
**/
func (s *Client) SubscribeApp(ctx context.Context) (et.Json, error) {
	path, err := s.wabaPath("/subscribed_apps")
	if err != nil {
		return et.Json{}, err
	}
	return s.Call(ctx, http.MethodPost, path, nil, et.Json{})
}
