package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/token"
	"github.com/cgalvisleon/et/utility"
)

func tokenFromAuthorization(authorization string) (string, error) {
	if authorization == "" {
		return "", logs.Nerror(ERR_AUTORIZATION_IS_REQUIRED)
	}

	if !strings.HasPrefix(authorization, "Bearer") {
		return "", logs.Nerror(ERR_INVALID_AUTORIZATION_FORMAT)
	}

	l := strings.Split(authorization, " ")
	if len(l) != 2 {
		return "", logs.Nerror(ERR_INVALID_AUTORIZATION_FORMAT)
	}

	return l[1], nil
}

func GetAuthorization(w http.ResponseWriter, r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
	result, err := tokenFromAuthorization(authorization)
	if err != nil {
		return "", err
	}

	return result, nil
}

func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metric := NewMetric(r)
		ctx := r.Context()
		tokenString, err := GetAuthorization(w, r)
		if err != nil {
			metric.Unauthorized(w, r)
			return
		}

		c, err := token.Validate(tokenString)
		if err != nil {
			metric.Unauthorized(w, r)
			return
		}

		type contextKey string

		const (
			serviceIdKey contextKey = "serviceId"
			clientIdKey  contextKey = "clientId"
			nameKey      contextKey = "name"
			iatKey       contextKey = "iat"
			expKey       contextKey = "exp"
			appKey       contextKey = "app"
			kindKey      contextKey = "kind"
			deviceKey    contextKey = "device"
			tokenKey     contextKey = "token"
		)

		serviceId := utility.UUID()
		ctx = context.WithValue(ctx, serviceIdKey, serviceId)
		ctx = context.WithValue(ctx, clientIdKey, c.ClientId)
		ctx = context.WithValue(ctx, nameKey, c.Name)
		ctx = context.WithValue(ctx, iatKey, c.Iat)
		ctx = context.WithValue(ctx, expKey, c.Exp)
		ctx = context.WithValue(ctx, appKey, c.App)
		ctx = context.WithValue(ctx, kindKey, c.Kind)
		ctx = context.WithValue(ctx, deviceKey, c.Device)
		ctx = context.WithValue(ctx, tokenKey, tokenString)

		now := utility.Now()
		hostName, _ := os.Hostname()
		data := js.Json{
			"serviceId": serviceId,
			"clientId":  c.ClientId,
			"last_use":  now,
			"host_name": hostName,
			"token":     tokenString,
		}

		go event.TokeLastUse(data)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
