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
	SessionIDKey ContextKey = "session_id"
	NameKey      ContextKey = "name"
	TokenKey     ContextKey = "token"
)

/**
* Duration
* @param r *http.Request
* @return time.Duration
**/
func Duration(r *http.Request) time.Duration {
	ctx := r.Context()
	return DurationKey.Duration(ctx, 0)
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
* App
* @param r *http.Request
* @return string
**/
func App(r *http.Request) string {
	ctx := r.Context()
	return AppKey.String(ctx, "")
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
* SessionID
* @param r *http.Request
* @return string
**/
func SessionID(r *http.Request) string {
	ctx := r.Context()
	return SessionIDKey.String(ctx, "")
}

/**
* Name
* @param r *http.Request
* @return string
**/
func Name(r *http.Request) string {
	ctx := r.Context()
	return NameKey.String(ctx, "")
}

/**
* SetDuration
* @param ctx context.Context, duration time.Duration
* @return context.Context
**/
func SetDuration(ctx context.Context, duration time.Duration) context.Context {
	return context.WithValue(ctx, DurationKey, duration)
}

/**
* SetPayload
* @param ctx context.Context, payload et.Json
* @return context.Context
**/
func SetPayload(ctx context.Context, payload et.Json) context.Context {
	return context.WithValue(ctx, PayloadKey, payload)
}

/**
* SetServiceId
* @param ctx context.Context, serviceId string
* @return context.Context
**/
func SetServiceId(ctx context.Context, serviceId string) context.Context {
	return context.WithValue(ctx, ServiceIdKey, serviceId)
}

/**
* SetApp
* @param ctx context.Context, app string
* @return context.Context
**/
func SetApp(ctx context.Context, app string) context.Context {
	return context.WithValue(ctx, AppKey, app)
}

/**
* SetDevice
* @param ctx context.Context, device string
* @return context.Context
**/
func SetDevice(ctx context.Context, device string) context.Context {
	return context.WithValue(ctx, DeviceKey, device)
}

/**
* SetSessionID
* @param ctx context.Context, userId string
* @return context.Context
**/
func SetSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

/**
* SetName
* @param ctx context.Context, username string
* @return context.Context
**/
func SetName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, NameKey, name)
}
