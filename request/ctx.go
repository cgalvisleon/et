package request

import (
	"context"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/et"
)

type ContextKey string

/**
* String
* @param ctx context.Context, def string
* @return string
**/
func (s ContextKey) String(ctx context.Context, def string) string {
	val := ctx.Value(s)
	result, ok := val.(string)
	if !ok {
		return def
	}

	return result
}

/**
* Duration
* @param ctx context.Context, def time.Duration
* @return time.Duration
**/
func (s ContextKey) Duration(ctx context.Context, def time.Duration) time.Duration {
	val := ctx.Value(s)
	result, ok := val.(time.Duration)
	if !ok {
		return def
	}

	return result
}

/**
* Json
* @param ctx context.Context, def et.Json
* @return et.Json
**/
func (s ContextKey) Json(ctx context.Context, def et.Json) et.Json {
	val := ctx.Value(s)
	result, ok := val.(et.Json)
	if !ok {
		return def
	}

	return result
}

const (
	DurationKey  ContextKey = "duration"
	PayloadKey   ContextKey = "payload"
	ServiceIdKey ContextKey = "service_id"
	AppKey       ContextKey = "app"
	DeviceKey    ContextKey = "device"
	UsernameKey  ContextKey = "username"
)

/**
* ServiceId
* @param r *http.Request
* @return string
**/
func ServiceId(r *http.Request) string {
	ctx := r.Context()
	return ServiceIdKey.String(ctx, "")
}

/**
* Username
* @param r *http.Request
* @return string
**/
func Username(r *http.Request) string {
	ctx := r.Context()
	return UsernameKey.String(ctx, "Anonimo")
}

/**
* Device
* @param r *http.Request
* @return string
**/
func Device(r *http.Request) string {
	ctx := r.Context()
	return DeviceKey.String(ctx, "")
}

/**
* App
* @param r *http.Request
* @return string
**/
func App(r *http.Request) string {
	ctx := r.Context()
	return AppKey.String(ctx, "")
}

/**
* Payload
* @param r *http.Request
* @return et.Json
**/
func Payload(r *http.Request) et.Json {
	ctx := r.Context()
	return PayloadKey.Json(ctx, et.Json{})
}
