package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
)

/**
* tokenFromAuthorization
* @param authorization string
* @return string
* @return error
**/
func tokenFromAuthorization(authorization, prefix string) (string, error) {
	if authorization == "" {
		return "", logs.Alertm("Autorization is required")
	}

	if !strings.HasPrefix(authorization, prefix) {
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
	_, ok := r.Header["Authorization"]
	if ok {
		authorization := r.Header.Get("Authorization")
		result, err := tokenFromAuthorization(authorization, "Bearer")
		if err != nil {
			return "", logs.Alert(err)
		}

		return result, nil
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

		clm, err := claim.ValidToken(token)
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
		ctx = context.WithValue(ctx, claim.ServiceIdKey, serviceId)
		ctx = context.WithValue(ctx, claim.ClientIdKey, clm.ID)
		ctx = context.WithValue(ctx, claim.AppKey, clm.App)
		ctx = context.WithValue(ctx, claim.DeviceKey, clm.Device)
		ctx = context.WithValue(ctx, claim.NameKey, clm.Name)
		ctx = context.WithValue(ctx, claim.UsernameKey, clm.Username)
		ctx = context.WithValue(ctx, claim.DataKey, clm.Data)
		ctx = context.WithValue(ctx, claim.DurationKey, clm.Duration)
		now := utility.Now()
		hostName := r.Host
		data := et.Json{
			"date_at":   now,
			"serviceId": serviceId,
			"clientId":  clm.ID,
			"app":       clm.App,
			"device":    clm.Device,
			"name":      clm.Name,
			"username":  clm.Username,
			"host_name": hostName,
			"duration":  clm.Duration,
			"token":     token,
		}

		PushTokenLastUse(data)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
