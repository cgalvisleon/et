package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/go-chi/chi/v5"
)

/**
* serve: Runs one request through a chi router with the Logger middleware and
* returns the telemetry lines written in the given format and filter.
**/
func serve(t *testing.T, format, filter, path string, handler http.HandlerFunc) []string {
	t.Helper()
	var buf bytes.Buffer
	SetTelemetryOutput(&buf, format, filter)
	t.Cleanup(func() { current.Store(nil) })

	r := chi.NewRouter()
	r.Use(Logger, Recoverer)
	r.Get("/users/{id}", handler)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))

	text := strings.TrimSpace(buf.String())
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func TestTelemetryJson(t *testing.T) {
	lines := serve(t, FormatJson, LogAll, "/users/123?token=secret", func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(MetricKey) != GetMetrics(r) {
			t.Error("GetMetrics should reuse the Metrics of the middleware")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	})
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1: %q", len(lines), lines)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("not a JSON line: %v\n%s", err, lines[0])
	}
	want := map[string]any{
		"level":                     "INFO",
		"msg":                       "http.request",
		"http.request.method":       "GET",
		"http.route":                "/users/{id}",
		"url.path":                  "/users/123",
		"http.response.status_code": float64(201),
		"http.response.body.size":   float64(2),
		"http.request.body.size":    float64(0),
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v, want %v", key, got[key], value)
		}
	}
	if _, ok := got["duration_ms"].(float64); !ok {
		t.Errorf("duration_ms = %v, want a number", got["duration_ms"])
	}
	if strings.Contains(lines[0], "secret") {
		t.Error("the query string must not reach the output")
	}
}

func TestTelemetryOneLinePerRequest(t *testing.T) {
	lines := serve(t, FormatJson, LogAll, "/users/1", func(w http.ResponseWriter, r *http.Request) {
		GetMetrics(r).ITEM(w, r, http.StatusOK, et.Item{Ok: true})
	})
	if len(lines) != 1 {
		t.Fatalf("a handler that finishes the metric and the middleware gave %d lines, want 1", len(lines))
	}
}

func TestTelemetryLevel(t *testing.T) {
	lines := serve(t, FormatJson, LogAll, "/users/1", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	if len(lines) != 1 || !strings.Contains(lines[0], `"level":"ERROR"`) || !strings.Contains(lines[0], `"http.response.status_code":500`) {
		t.Fatalf("a panic should log one ERROR line with status 500: %q", lines)
	}
}

func TestTelemetryTextAndOff(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }

	lines := serve(t, FormatText, LogAll, "/users/7", ok)
	if len(lines) != 1 || !strings.Contains(lines[0], "/users/7") || strings.HasPrefix(lines[0], "{") {
		t.Fatalf("text format should write the colored line: %q", lines)
	}

	if lines := serve(t, FormatOff, LogAll, "/users/7", ok); len(lines) != 0 {
		t.Fatalf("off format wrote %q", lines)
	}
}

func TestTelemetryErrorsOnly(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }
	if lines := serve(t, FormatJson, LogErrors, "/users/1", ok); len(lines) != 0 {
		t.Fatalf("errors only wrote a 200: %q", lines)
	}

	notFound := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }
	if lines := serve(t, FormatJson, LogErrors, "/users/1", notFound); len(lines) != 0 {
		t.Fatalf("errors only wrote a 404: %q", lines)
	}

	failed := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }
	if lines := serve(t, FormatJson, LogErrors, "/users/1", failed); len(lines) != 1 {
		t.Fatalf("errors only should write a 502: %q", lines)
	}

	envar.SetInt("LATENCY_LIMIT", 0) // read when the output is built
	t.Cleanup(func() { envar.SetInt("LATENCY_LIMIT", 1000) })
	if lines := serve(t, FormatJson, LogErrors, "/users/1", ok); len(lines) != 1 {
		t.Fatalf("errors only should write a slow request: %q", lines)
	}
}

func TestTelemetryOffSkipsTheMetric(t *testing.T) {
	serve(t, FormatOff, LogAll, "/users/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(MetricKey) != nil {
			t.Error("with the output off and no NATS the middleware should not build a Metrics")
		}
	})
}

func TestResponseWriterWrapper(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &ResponseWriterWrapper{ResponseWriter: rec}
	rw.Write([]byte("x"))
	if rw.StatusCode != http.StatusOK {
		t.Errorf("a write without WriteHeader gave status %d, want 200", rw.StatusCode)
	}
	if err := http.NewResponseController(rw).Flush(); err != nil || !rec.Flushed {
		t.Errorf("Flush did not reach the recorder: %v", err)
	}
	if _, _, err := rw.Hijack(); err == nil {
		t.Error("Hijack on a writer without it should fail")
	}

	// Behind Logger, a handler can take the connection (the WebSocket upgrade).
	SetTelemetryOutput(&bytes.Buffer{}, FormatJson, LogErrors)
	t.Cleanup(func() { current.Store(nil) })
	srv := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, buf, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Errorf("Hijack behind Logger: %v", err)
			return
		}
		defer conn.Close()
		buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nhi")
		buf.Flush()
	})))
	defer srv.Close()

	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := bufio.NewReader(res.Body).ReadString('\n')
	if body != "hi" {
		t.Errorf("hijacked body = %q, want hi", body)
	}
}

func BenchmarkTelemetry(b *testing.B) {
	b.Setenv("REDIS_HOST", "") // the in-memory cache, as a node without Redis has
	cache.Load()
	defer cache.Close()

	ok := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }
	plain := chi.NewRouter()
	plain.Get("/users/{id}", ok)
	logged := chi.NewRouter()
	logged.Use(Logger)
	logged.Get("/users/{id}", ok)
	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)

	b.Run("none", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			plain.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
	for _, c := range []struct{ format, filter string }{
		{FormatText, LogAll}, {FormatJson, LogAll}, {FormatJson, LogErrors}, {FormatOff, LogAll},
	} {
		b.Run(c.format+"-"+c.filter, func(b *testing.B) {
			SetTelemetryOutput(io.Discard, c.format, c.filter)
			b.ReportAllocs()
			for b.Loop() {
				logged.ServeHTTP(httptest.NewRecorder(), req)
			}
		})
	}
	current.Store(nil)
}
