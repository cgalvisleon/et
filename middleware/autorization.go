package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/sesion"
)

const (
	PERMISION_READ    = "PERMISION.READ"
	PERMISION_WRITE   = "PERMISION.WRITE"
	PERMISION_DELETE  = "PERMISION.DELETE"
	PERMISION_UPDATE  = "PERMISION.UPDATE"
	PERMISION_EXECUTE = "PERMISION.EXECUTE"
)

var PERMISION_ALL = map[string]bool{
	PERMISION_READ:    true,
	PERMISION_WRITE:   true,
	PERMISION_DELETE:  true,
	PERMISION_UPDATE:  true,
	PERMISION_EXECUTE: true,
}

func MethodAutorized(p map[string]bool, r *http.Request) bool {
	method := r.Method
	switch method {
	case "GET":
		return p[PERMISION_READ]
	case "POST":
		return p[PERMISION_WRITE]
	case "PUT":
		return p[PERMISION_UPDATE]
	case "DELETE":
		return p[PERMISION_DELETE]
	case "PATCH":
		return p[PERMISION_EXECUTE]
	default:
		return false
	}
}

/**
* NewPermisions
* @param data string
* @return map[string]bool, error
**/
func NewPermisions(data string) (map[string]bool, error) {
	var result = make(map[string]bool)
	if data == "" {
		return result, nil
	}

	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* PermisionsToStr
* @param p map[string]bool
* @return string
**/
func PermisionsToStr(p map[string]bool) string {
	data, _ := json.Marshal(p)
	return string(data)
}

type AuthorizationFunc func(profile et.Json) (map[string]bool, error)

var authorizationFunc AuthorizationFunc

/**
* SetAuthorizationFunc
* @param f AuthorizationFunc
**/
func SetAuthorizationFunc(f AuthorizationFunc) {
	authorizationFunc = f
}

/**
* Authorization
* @param next http.Handler
* @return http.Handler
**/
func Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if authorizationFunc == nil {
			response.InternalServerError(w, r, errors.New("AuthorizationFunc not set"))
			return
		}

		profileStr := r.Header.Get("Profile")
		profile, err := et.Object(profileStr)
		if err != nil {
			response.InternalServerError(w, r, err)
			return
		}

		ClientId := sesion.ClientId(r)
		profile["client_id"] = ClientId
		permisions, err := authorizationFunc(profile)
		if err != nil {
			response.InternalServerError(w, r, err)
			return
		}

		ok := MethodAutorized(permisions, r)
		if !ok {
			response.Forbidden(w, r)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
