package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
)

/**
* getBearerToken
* @param r *http.Request
* @return string, error
**/
func getBearerToken(r *http.Request) (string, error) {
	_, ok := r.Header["Authorization"]
	if !ok {
		return "", logs.Alertm("Autorization is required")
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		return "", logs.Alertm("Autorization is required")
	}

	if !strings.HasPrefix(token, "Bearer ") {
		return "", logs.Alertm("Autorization is required")
	}

	token = strings.TrimPrefix(token, "Bearer ")
	return token, nil
}

/**
* BearerToken
* @param next http.Handler
**/
func BearerToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := getBearerToken(r)
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
		ctx = context.WithValue(ctx, request.UserIdKey, clm.UserId)
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
