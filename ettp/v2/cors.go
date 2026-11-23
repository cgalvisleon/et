package ettp

import (
	"net/http"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/router"
	"github.com/rs/cors"
)

/**
* CorsAllowAll: CORS handler that accepts the given origins (any when empty) and every header. An entry is "*" (all), a
* full origin ("https://app.example.com:8443", exact) or a host ("localhost": any scheme and port), the rule of
* router.OriginAllowed; credentials only with an explicit allowlist without "*".
* @param allowedOrigins []string
* @return *cors.Cors
**/
func CorsAllowAll(allowedOrigins []string) *cors.Cors {
	origins := []string{}
	for _, origin := range allowedOrigins {
		origin = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(origin)), "/")
		if origin != "" && !slices.Contains(origins, origin) {
			origins = append(origins, origin)
		}
	}

	return cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			if len(origins) == 0 {
				return true
			}
			return router.OriginAllowed(origins, origin)
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
		// policy (no list, or "*") — that combo lets any site make
		// authenticated requests using the caller's cookies. Only enable it
		// once an explicit allowlist is in effect.
		AllowCredentials: len(origins) > 0 && !slices.Contains(origins, "*"),
	})
}
