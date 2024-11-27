package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	tk "github.com/cgalvisleon/et/token"
	"github.com/cgalvisleon/et/utility"
)

/**
* tokenFromAuthorization
* @param authorization string
* @return string
* @return error
**/
func tokenFromAuthorization(authorization string) (string, error) {
	if authorization == "" {
		return "", logs.Alertm("Autorization is required")
	}

	if !strings.HasPrefix(authorization, "Bearer") {
		return "", logs.Alertm("Invalid autorization format")
	}

	l := strings.Split(authorization, " ")
	if len(l) != 2 {
		return "", logs.Alertm("Invalid autorization format")
	}

	return l[1], nil
}

/**
* GetAuthorization
* @param w http.ResponseWriter
* @param r *http.Request
* @return string
* @return error
**/
func GetAuthorization(w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie("auth_token")
	if err == nil {
		return cookie.Value, nil
	}

	authorization := r.Header.Get("Authorization")
	result, err := tokenFromAuthorization(authorization)
	if err != nil {
		return "", logs.Alert(err)
	}

	return result, nil
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

		clm, err := tk.Valid(token)
		if err != nil {
			response.Unauthorized(w, r)
			return
		}

		if clm == nil {
			response.Unauthorized(w, r)
			return
		}

		serviceId := utility.UUID()
		ctx := r.Context()
		ctx = context.WithValue(ctx, tk.ServiceIdKey, serviceId)
		ctx = context.WithValue(ctx, tk.ClientIdKey, clm.ClientId)
		ctx = context.WithValue(ctx, tk.NameKey, clm.Name)
		ctx = context.WithValue(ctx, tk.AppKey, clm.App)
		ctx = context.WithValue(ctx, tk.DeviceKey, clm.Device)
		ctx = context.WithValue(ctx, tk.DuractionKey, clm.Duration)
		ctx = context.WithValue(ctx, tk.TokenKey, token)

		now := utility.Now()
		data := et.Json{
			"serviceId": serviceId,
			"clientId":  clm.ClientId,
			"last_use":  now,
			"host_name": hostName,
			"token":     token,
		}

		go event.TokenLastUse(data)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
