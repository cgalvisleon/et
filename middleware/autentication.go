package middleware

import (
	"context"
	"net/http"

	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
)

/**
* GetAuthorization
* @param w http.ResponseWriter
* @param r *http.Request
* @return string
* @return error
**/
func GetAuthorization(w http.ResponseWriter, r *http.Request) (string, error) {
	_, ok := r.Header["Authorization"]
	if ok {
		authorization := r.Header.Get("Authorization")
		token := utility.PrefixRemove("Bearer ", authorization)
		if token == "" {
			return "", logs.Alertm("Autorization is required")
		}

		return token, nil
	}

	return "", logs.Alertm("Autorization is required")
}

/**
* Autentication
* @param next http.Handler
**/
func Autentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := GetAuthorization(w, r)
		if err != nil {
			response.Unauthorized(w, r)
			return
		}

		clm, err := jwt.Validate(token)
		if err != nil {
			response.Unauthorized(w, r)
			return
		}

		if clm == nil {
			response.Unauthorized(w, r)
			return
		}

		serviceId := r.Header.Get("ServiceId")
		if serviceId == "" {
			serviceId = utility.UUID()
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, request.ServiceIdKey, serviceId)
		ctx = context.WithValue(ctx, request.DurationKey, clm.Duration)
		ctx = context.WithValue(ctx, request.DeviceKey, clm.Device)
		ctx = context.WithValue(ctx, request.AppKey, clm.App)
		ctx = context.WithValue(ctx, request.UsernameKey, clm.Username)
		ctx = context.WithValue(ctx, request.PayloadKey, clm.Payload)
		data, err := clm.ToJson()
		if err != nil {
			response.Unauthorized(w, r)
			return
		}
		data.Set("service_id", serviceId)
		data.Set("host_name", r.Host)
		now := utility.Now()
		data.Set("date_at", now)
		PushTokenLastUse(data)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
