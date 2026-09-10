package ettp

import (
	"net/http"
	"slices"

	"github.com/rs/cors"
)

func CorsAllowAll(allowedOrigins []string) *cors.Cors {
	return cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			if len(allowedOrigins) == 0 {
				return true
			}
			idx := slices.IndexFunc(allowedOrigins, func(e string) bool { return e == origin })
			return idx != -1
		},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders: []string{"*"},
		// Credentials must never be allowed alongside a reflect-any-origin
		// policy (the case below when allowedOrigins is empty) — that combo
		// lets any site make authenticated requests using the caller's
		// cookies. Only enable it once an explicit allowlist is in effect.
		AllowCredentials: len(allowedOrigins) > 0,
	})
}
