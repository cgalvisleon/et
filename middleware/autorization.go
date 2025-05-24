package middleware

import (
	"net/http"
)

var AuthorizationMiddleware func(next http.Handler) http.Handler

/**
* SetAuthorizationMiddleware
* @param f AuthorizationMiddleware
**/
func SetAuthorizationMiddleware(f func(next http.Handler) http.Handler) {
	AuthorizationMiddleware = f
}
