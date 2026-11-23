package ettp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
)

/**
* newTestServer: A gateway without listeners (the RPC one on a free port, closed at the end)
* @param t *testing.T, cnf *Config
* @return *Server
**/
func newTestServer(t *testing.T, cnf *Config) *Server {
	s, err := New("test", cnf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.pipe.Close() })
	return s
}

/**
* preflight: An OPTIONS of a browser from origin through the HTTP handler of the server
* @param s *Server, origin string
* @return *httptest.ResponseRecorder
**/
func preflight(s *Server, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodOptions, "/api/tenancy/plugins/active", nil)
	r.Header.Set("Origin", origin)
	r.Header.Set("Access-Control-Request-Method", http.MethodGet)
	r.Header.Set("Access-Control-Request-Headers", "authorization")
	w := httptest.NewRecorder()
	s.httpHandler().ServeHTTP(w, r)
	return w
}

/**
* TestCorsOrigins: a host entry allows that host with any port (the dev servers), a full origin only itself, the rest
* gets no CORS headers; credentials with a list, not with "*" nor without a list
**/
func TestCorsOrigins(t *testing.T) {
	s := newTestServer(t, &Config{AllowOrigin: []string{" Localhost ", "https://studio.example.com/"}})
	for _, tc := range []struct {
		origin string
		ok     bool
	}{
		{"http://localhost:5600", true},
		{"http://localhost:5603", true},
		{"https://studio.example.com", true},
		{"https://studio.example.com:8443", false},
		{"http://127.0.0.1:5600", false},
		{"https://evil.example.com", false},
	} {
		w := preflight(s, tc.origin)
		got := w.Header().Get("Access-Control-Allow-Origin")
		if tc.ok && got != tc.origin {
			t.Errorf("%s: Allow-Origin %q, want the origin", tc.origin, got)
		}
		if !tc.ok && got != "" {
			t.Errorf("%s: Allow-Origin %q, want none", tc.origin, got)
		}
		if tc.ok && w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("%s: without credentials", tc.origin)
		}
	}

	for _, origins := range [][]string{nil, {"*"}} {
		s := newTestServer(t, &Config{AllowOrigin: origins})
		w := preflight(s, "https://any.example.com")
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Errorf("%v: any origin not allowed", origins)
		}
		if w.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("%v: credentials with any origin", origins)
		}
	}
}

/**
* TestUseMiddlewares: the middlewares of Use wrap every request of the HTTP handler, also the ones given after New
**/
func TestUseMiddlewares(t *testing.T) {
	s := newTestServer(t, &Config{})
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "1")
			next.ServeHTTP(w, r)
		})
	})

	w := httptest.NewRecorder()
	s.httpHandler().ServeHTTP(w, httptest.NewRequest(GET, "/api/unknown", nil))
	if w.Header().Get("X-Middleware") != "1" {
		t.Fatal("the middleware of Use did not run")
	}
}

/**
* TestEventReset: a refresh (keep) of another replica keeps the routes; a reset leaves only the routes of the gateway
**/
func TestEventReset(t *testing.T) {
	s := newTestServer(t, &Config{Parent: "/api"})
	if err := s.basicRoutes(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetRouter(GET, "/api/tenancy/version", "http://localhost:3503/api/tenancy/version", 0, et.Json{}, nil, 1, "tenancy", false); err != nil {
		t.Fatal(err)
	}
	route := "GET:/api/tenancy/version"

	s.eventReset(event.Message{Data: et.Json{"keep": true}})
	if _, ok := s.Solvers[route]; !ok {
		t.Fatal("a refresh dropped the routes")
	}

	s.eventReset(event.Message{Data: et.Json{}})
	if _, ok := s.Solvers[route]; ok {
		t.Fatal("a reset kept the routes of the services")
	}
	if _, ok := s.Solvers["GET:/api/version"]; !ok {
		t.Fatal("a reset dropped the routes of the gateway")
	}
}
