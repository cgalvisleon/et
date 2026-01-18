package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/claim"
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

		serviceId := r.Header.Get("ServiceId")
		if serviceId == "" {
			serviceId = utility.UUID()
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, claim.ServiceIdKey, serviceId)
		ctx = context.WithValue(ctx, claim.DurationKey, clm.Duration)
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
