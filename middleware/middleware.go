package middleware

import (
	"context"
	"net/http"
)

/**
* Test
* @param http.Handler
* @return http.Handler
**/
func Test(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		clientId := "SoyElCliente"
		ctx = context.WithValue(ctx, ClientIdKey, clientId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
