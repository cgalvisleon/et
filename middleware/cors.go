package middleware

import (
	"net/http"
	"slices"

	"github.com/rs/cors"
)

func AllowAll(allowedOrigins []string) *cors.Cors {
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
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
}
