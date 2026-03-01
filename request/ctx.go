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
	DurationKey   ContextKey = "duration"
	PayloadKey    ContextKey = "payload"
	ServiceIdKey  ContextKey = "service_id"
	AppKey        ContextKey = "app"
	DeviceKey     ContextKey = "device"
	UsernameIdKey ContextKey = "username_id"
	UsernameKey   ContextKey = "username"
	TenantIdKey   ContextKey = "tenant_id"
	ProfileIdKey  ContextKey = "profile_id"
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
* Username
* @param r *http.Request
* @return string
**/
func Username(r *http.Request) string {
	ctx := r.Context()
	return UsernameKey.String(ctx, "Anonimo")
}

/**
* UsernameId
* @param r *http.Request
* @return string
**/
func UsernameId(r *http.Request) string {
	ctx := r.Context()
	return UsernameIdKey.String(ctx, "")
}

/**
* TenantId
* @param r *http.Request
* @return string
**/
func TenantId(r *http.Request) string {
	ctx := r.Context()
	return TenantIdKey.String(ctx, "")
}

/**
* ProfileId
* @param r *http.Request
* @return string
**/
func ProfileId(r *http.Request) string {
	ctx := r.Context()
	return ProfileIdKey.String(ctx, "")
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
* SetUsernameId
* @param ctx context.Context, usernameId string
* @return context.Context
**/
func SetUsernameId(ctx context.Context, usernameId string) context.Context {
	return context.WithValue(ctx, UsernameIdKey, usernameId)
}

/**
* SetUsername
* @param ctx context.Context, username string
* @return context.Context
**/
func SetUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, UsernameKey, username)
}

/**
* SetTenantId
* @param ctx context.Context, tenantId string
* @return context.Context
**/
func SetTenantId(ctx context.Context, tenantId string) context.Context {
	return context.WithValue(ctx, TenantIdKey, tenantId)
}

/**
* SetProfileId
* @param ctx context.Context, profileId string
* @return context.Context
**/
func SetProfileId(ctx context.Context, profileId string) context.Context {
	return context.WithValue(ctx, ProfileIdKey, profileId)
}
