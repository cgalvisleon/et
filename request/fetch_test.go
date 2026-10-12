package request

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cgalvisleon/et/et"
)

// echoServer answers with the method, Content-Type and decoded body it got.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := map[string]any{"method": r.Method, "contentType": r.Header.Get("Content-Type")}
		switch {
		case r.Header.Get("Content-Type") == "application/json":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			got["body"] = body
		case r.ParseMultipartForm(1<<20) == nil:
			got["name"] = r.FormValue("name")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(got)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fetchJson(t *testing.T, method, url string, header, body et.Json) et.Json {
	t.Helper()
	res, status := Fetch(method, url, header, body, 0, nil)
	if !status.Ok {
		t.Fatalf("%s %v: %s", method, header, status.Message)
	}
	out, err := res.ToJson()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Header names are case-insensitive: "content-type" (as brevo sends it)
// must work like "Content-Type".
func TestFetchLowercaseContentType(t *testing.T) {
	srv := echoServer(t)
	out := fetchJson(t, "POST", srv.URL, et.Json{"content-type": "application/json"}, et.Json{"a": "b"})
	if out.Json("body").Str("a") != "b" {
		t.Errorf("body: %v", out)
	}
}

func TestFetchWithoutContentType(t *testing.T) {
	srv := echoServer(t)
	if out := fetchJson(t, "GET", srv.URL, et.Json{}, nil); out.Str("method") != "GET" || out.Str("contentType") != "" {
		t.Errorf("GET without a body: %v", out)
	}
	if out := fetchJson(t, "POST", srv.URL, nil, et.Json{"a": "b"}); out.Json("body").Str("a") != "b" {
		t.Errorf("a body without Content-Type goes as JSON: %v", out)
	}
}

func TestFetchMultipart(t *testing.T) {
	srv := echoServer(t)
	out := fetchJson(t, "POST", srv.URL, et.Json{"Content-Type": "multipart/form-data"}, et.Json{"name": "Ana"})
	if out.Str("name") != "Ana" {
		t.Errorf("multipart: %v", out)
	}
}
