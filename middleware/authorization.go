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

type contextKey string

const (
	ServiceIdKey contextKey = "serviceId"
	ClientIdKey  contextKey = "clientId"
	NameKey      contextKey = "name"
	IatKey       contextKey = "iat"
	ExpKey       contextKey = "exp"
	AppKey       contextKey = "app"
	KindKey      contextKey = "kind"
	DeviceKey    contextKey = "device"
	TokenKey     contextKey = "token"
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

		serviceId := utility.UUID()
		ctx = context.WithValue(ctx, ServiceIdKey, serviceId)
		ctx = context.WithValue(ctx, ClientIdKey, c.ClientId)
		ctx = context.WithValue(ctx, NameKey, c.Name)
		ctx = context.WithValue(ctx, IatKey, c.Iat)
		ctx = context.WithValue(ctx, ExpKey, c.Exp)
		ctx = context.WithValue(ctx, AppKey, c.App)
		ctx = context.WithValue(ctx, KindKey, c.Kind)
		ctx = context.WithValue(ctx, DeviceKey, c.Device)
		ctx = context.WithValue(ctx, TokenKey, tokenString)

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
