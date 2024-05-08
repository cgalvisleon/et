package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/pubsub"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/token"
	"github.com/cgalvisleon/et/utility"
)

func tokenFromAuthorization(authorization string) (string, error) {
	if authorization == "" {
		return "", logs.Alertm(ERR_AUTORIZATION_IS_REQUIRED)
	}

	if !strings.HasPrefix(authorization, "Bearer") {
		return "", logs.Alertm(ERR_INVALID_AUTORIZATION_FORMAT)
	}

	l := strings.Split(authorization, " ")
	if len(l) != 2 {
		return "", logs.Alertm(ERR_INVALID_AUTORIZATION_FORMAT)
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
		ctx := r.Context()
		tokenString, err := GetAuthorization(w, r)
		if err != nil {
			response.HTTPError(w, r, http.StatusUnauthorized, "401 Unauthorized")
			return
		}

		c, err := token.Validate(tokenString)
		if err != nil {
			response.HTTPError(w, r, http.StatusUnauthorized, "401 Unauthorized")
			return
		}

		ctx = context.WithValue(ctx, "clientId", c.ClientId)
		ctx = context.WithValue(ctx, "name", c.Name)
		ctx = context.WithValue(ctx, "iat", c.Iat)
		ctx = context.WithValue(ctx, "exp", c.Exp)
		ctx = context.WithValue(ctx, "app", c.App)
		ctx = context.WithValue(ctx, "kins", c.Kind)
		ctx = context.WithValue(ctx, "device", c.Device)
		ctx = context.WithValue(ctx, "token", tokenString)

		now := utility.Now()
		hostName, _ := os.Hostname()
		data := et.Json{
			"clientId":  c.ClientId,
			"last_use":  now,
			"host_name": hostName,
			"token":     tokenString,
		}

		go pubsub.TokeLastUse(data)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
