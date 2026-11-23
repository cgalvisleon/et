package ettp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* serve: A request through the HTTP handler of the server, with the body decoded
* @param s *Server, method, path string
* @return int, et.Json
**/
func serve(t *testing.T, s *Server, method, path string) (int, et.Json) {
	w := httptest.NewRecorder()
	s.httpHandler().ServeHTTP(w, httptest.NewRequest(method, path, nil))
	var body et.Json
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s %s: body %q: %v", method, path, w.Body.String(), err)
	}
	return w.Code, body
}

/**
* newMetricsServer: A gateway with its own routes and two of a service (svc): /api/ok answers 201, /api/fail 503
* @param t *testing.T
* @return *Server
**/
func newMetricsServer(t *testing.T) *Server {
	s := newTestServer(t, &Config{Parent: "/api"})
	if err := s.basicRoutes(); err != nil {
		t.Fatal(err)
	}
	answer := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{}`))
		}
	}
	s.Public(GET, "/ok", answer(http.StatusCreated), "svc")
	s.Public(POST, "/fail", answer(http.StatusServiceUnavailable), "svc")
	return s
}

/**
* TestUsageP95: the p95 is the bound of the bucket of the 95th request, never above the max
**/
func TestUsageP95(t *testing.T) {
	u := usage{}
	for range 95 {
		u.add(200, 3)
	}
	for range 5 {
		u.add(500, 800)
	}
	if got := u.p95(); got != 5 {
		t.Fatalf("p95 = %v, want 5 (bucket of 3 ms)", got)
	}
	u.add(200, 900)
	if got := u.p95(); got != 900 {
		t.Fatalf("p95 = %v, want 900 (bucket of 1000 ms capped at the max)", got)
	}

	one := usage{}
	one.add(200, 0.4)
	if got := one.p95(); got != 0.4 {
		t.Fatalf("p95 of one request = %v, want its latency 0.4 (capped at the max)", got)
	}
	j := u.toJson()
	if j.Int("errors") != 5 || j.Json("status").Int("5xx") != 5 || j.Json("status").Int("2xx") != 96 {
		t.Fatalf("counters: %s", j.ToString())
	}
}

/**
* TestMetricsRecord: each route counts its requests, statuses, last request and last error; the gateway's own routes
* (its package) are marked own and stay out of the totals; a request without route counts as unmatched; DELETE resets
**/
func TestMetricsRecord(t *testing.T) {
	s := newMetricsServer(t)
	for range 3 {
		serve(t, s, GET, "/api/ok?token=secret")
	}
	serve(t, s, POST, "/api/fail")
	serve(t, s, GET, "/api/nothing")
	serve(t, s, GET, "/api/version")

	code, body := serve(t, s, GET, "/api/metrics")
	if code != http.StatusOK {
		t.Fatalf("GET /metrics = %d", code)
	}
	result := body.Json("result")
	totals := result.Json("totals")
	if totals.Int("requests") != 4 || totals.Int("errors") != 1 || totals.Int("unmatched") != 1 {
		t.Fatalf("totals: %s", totals.ToString())
	}
	if n := len(result.ArrayJson("minutes")); n != metricsWindow {
		t.Fatalf("minutes = %d, want %d", n, metricsWindow)
	}
	last := result.ArrayJson("minutes")[metricsWindow-1]
	if last.Int("requests") != 4 {
		t.Fatalf("current minute: %s", last.ToString())
	}

	routes := map[string]et.Json{}
	for _, route := range result.ArrayJson("routes") {
		routes[route.Str("id")] = route
	}
	ok := routes["GET:/api/ok"]
	if ok.Int("requests") != 3 || ok.Json("status").Int("2xx") != 3 || ok.Str("package") != "svc" || ok.Bool("own") {
		t.Fatalf("GET:/api/ok: %s", ok.ToString())
	}
	if ok.Json("last").Str("path") != "/api/ok" || ok.Json("last").Int("status") != http.StatusCreated {
		t.Fatalf("last of GET:/api/ok (without the query): %s", ok.Json("last").ToString())
	}
	fail := routes["POST:/api/fail"]
	if fail.Int("errors") != 1 || fail.Json("last_error").Int("status") != http.StatusServiceUnavailable {
		t.Fatalf("POST:/api/fail: %s", fail.ToString())
	}
	if version := routes["GET:/api/version"]; !version.Bool("own") || version.Int("requests") != 1 {
		t.Fatalf("GET:/api/version: %s", version.ToString())
	}

	packages := map[string]et.Json{}
	for _, pkg := range result.ArrayJson("packages") {
		packages[pkg.Str("package")] = pkg
	}
	if packages["svc"].Int("requests") != 4 || packages["svc"].Int("errors") != 1 || !packages["test"].Bool("own") {
		t.Fatalf("packages: %v", packages)
	}

	if code, _ := serve(t, s, DELETE, "/api/metrics"); code != http.StatusOK {
		t.Fatalf("DELETE /metrics = %d", code)
	}
	_, body = serve(t, s, GET, "/api/metrics")
	result = body.Json("result")
	if result.Json("totals").Int("requests") != 0 || len(result.ArrayJson("routes")) != 1 {
		t.Fatalf("after reset only the DELETE /metrics itself (own) is counted: %s", result.ToString())
	}
}

/**
* TestMetricsConcurrent: requests in parallel with reads and resets (go test -race)
**/
func TestMetricsConcurrent(t *testing.T) {
	s := newMetricsServer(t)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				serve(t, s, GET, "/api/ok")
				if i == 0 {
					s.metrics.ToJson()
				}
			}
		}()
	}
	wg.Wait()
	if got := s.metrics.ToJson().Json("totals").Int("requests"); got != 400 {
		t.Fatalf("requests = %d, want 400", got)
	}
	s.metrics.Reset()
}

/**
* TestWindowRolls: a minute older than the window is dropped from it
**/
func TestWindowRolls(t *testing.T) {
	m := newMetrics()
	old := time.Now().Unix()/60 - metricsWindow
	m.minutes[old%metricsWindow] = minute{at: old, usage: usage{requests: 9}}
	for _, item := range m.window(time.Now()) {
		if item.Int("requests") != 0 {
			t.Fatalf("an old minute is in the window: %s", item.ToString())
		}
	}
}

/**
* TestGetRoutes: GET /routes without params answers the server with its solvers (it was empty: the requests in flight
* embed an *http.Request that json.Marshal rejects)
**/
func TestGetRoutes(t *testing.T) {
	s := newMetricsServer(t)
	code, body := serve(t, s, GET, "/api/routes")
	if code != http.StatusOK {
		t.Fatalf("GET /routes = %d", code)
	}
	result := body.Json("result")
	if result.Json("solvers").Json("GET:/api/ok").Str("package_name") != "svc" {
		t.Fatalf("solvers: %s", result.Json("solvers").ToString())
	}
	if len(result.ArrayJson("requests")) != 1 {
		t.Fatalf("requests in flight (this one): %v", result["requests"])
	}
}
