package ettp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

/**
* TestRouterWildcard: a route ending in * takes the rest of the path, the prefix itself and nothing outside it; a fixed
* segment or a param of the same node win over the wildcard
**/
func TestRouterWildcard(t *testing.T) {
	root := newRouter(GET)
	for _, path := range []string{"/plugins/eam/1.0.0/*", "/plugins/eam/1.0.0/health", "/*", "/api/tenancy/{id}"} {
		if _, err := root.set(TpStatic, GET, path, "http://origin/*", TpKeepHeader, nil, nil, 1); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		path, solver, rest string
	}{
		{"/plugins/eam/1.0.0/assets/index-BqzK3a1x.js", "/plugins/eam/1.0.0/*", "assets/index-BqzK3a1x.js"},
		{"/plugins/eam/1.0.0/mf-manifest.json", "/plugins/eam/1.0.0/*", "mf-manifest.json"},
		{"/plugins/eam/1.0.0", "/plugins/eam/1.0.0/*", ""},
		{"/plugins/eam/1.0.0/health", "/plugins/eam/1.0.0/health", ""},
		{"/", "/*", ""},
		{"/desk/home", "/*", "desk/home"},
		{"/api/tenancy/abc", "/api/tenancy/{id}", ""},
	} {
		target, params, err := root.find(tc.path)
		if err != nil {
			t.Fatalf("%s: %s", tc.path, err)
		}
		if target.Solver.Path != tc.solver {
			t.Errorf("%s: solved by %s, want %s", tc.path, target.Solver.Path, tc.solver)
		}
		if params[Wildcard] != tc.rest {
			t.Errorf("%s: rest %q, want %q", tc.path, params[Wildcard], tc.rest)
		}
	}

	// No backtracking: /plugins exists, so /plugins/other does not fall to the root wildcard
	if _, _, err := root.find("/plugins/other/1.0.0/x.js"); err == nil {
		t.Error("/plugins/other resolved")
	}
	if _, _, err := root.find("/api/tenancy/abc/def"); err == nil {
		t.Error("/api/tenancy/abc/def resolved")
	}
	if _, err := root.set(TpStatic, GET, "/a/*/b", "http://origin/*", TpKeepHeader, nil, nil, 1); err == nil {
		t.Error("a wildcard in the middle was accepted")
	}

	if _, err := root.delete("/plugins/eam/1.0.0/*"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := root.find("/plugins/eam/1.0.0/x.js"); err == nil {
		t.Error("the deleted wildcard still resolves")
	}
	if _, _, err := root.find("/plugins/eam/1.0.0/health"); err != nil {
		t.Errorf("deleting the wildcard dropped its sibling: %s", err)
	}
}

/**
* newStaticServer: A gateway without listeners (the RPC one on a free port, closed at the end) and an origin of files
* that echoes what it receives
* @param t *testing.T
* @return *Server, *httptest.Server
**/
func newStaticServer(t *testing.T) (*Server, *httptest.Server) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Auth", r.Header.Get("Authorization")+"|"+r.Header.Get("Cookie"))
		w.Header().Set("X-Path", r.URL.RequestURI())
		w.Header().Add("Set-Cookie", "origin=1")
		switch {
		case strings.HasSuffix(r.URL.Path, "/missing.js"):
			http.NotFound(w, r)
		case strings.HasSuffix(r.URL.Path, "/changes"):
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte("data: {}\n\n"))
		case strings.HasSuffix(r.URL.Path, "/fixed-cache.js"):
			w.Header().Set("Cache-Control", "no-store")
			w.Write([]byte("fixed"))
		default:
			w.Write([]byte("file " + r.URL.Path))
		}
	}))
	t.Cleanup(origin.Close)

	s, err := New("test", &Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.pipe.Close() })
	return s, origin
}

/**
* serveStatic: One request through the gateway handler
* @param s *Server, path, accept string
* @return *httptest.ResponseRecorder
**/
func serveStatic(s *Server, path, accept string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(GET, path, nil)
	r.Header.Set("Authorization", "Bearer secret")
	r.Header.Set("Cookie", "session=1")
	if accept != "" {
		r.Header.Set("Accept", accept)
	}
	w := httptest.NewRecorder()
	s.handler(w, r)
	return w
}

/**
* TestStaticProxy: proxy mode copies the file without the token or the cookies (both ways), keeps the query, sets the
* cache policy (entry no-cache, hashed immutable, the origin's otherwise), passes a 404 and streams an event stream
**/
func TestStaticProxy(t *testing.T) {
	s, origin := newStaticServer(t)
	if _, err := s.SetStatic("/plugins/eam/1.0.0/*", origin.URL+"/plugins/eam/1.0.0/*", ModeProxy, "", 1, "frontend", false); err != nil {
		t.Fatal(err)
	}

	w := serveStatic(s, "/plugins/eam/1.0.0/assets/index-BqzK3a1x.js?v=2", "*/*")
	if w.Code != http.StatusOK || w.Body.String() != "file /plugins/eam/1.0.0/assets/index-BqzK3a1x.js" {
		t.Fatalf("hashed: %d %q", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Path"); got != "/plugins/eam/1.0.0/assets/index-BqzK3a1x.js?v=2" {
		t.Errorf("origin got %q", got)
	}
	if got := w.Header().Get("X-Auth"); got != "|" {
		t.Errorf("the origin received the token or the cookies: %q", got)
	}
	if got := w.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("the origin set a cookie: %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != CacheImmutable {
		t.Errorf("hashed cache %q", got)
	}

	w = serveStatic(s, "/plugins/eam/1.0.0/mf-manifest.json", "")
	if got := w.Header().Get("Cache-Control"); got != CacheNoCache {
		t.Errorf("manifest cache %q", got)
	}

	w = serveStatic(s, "/plugins/eam/1.0.0/fixed-cache.js", "")
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("the origin's cache was not kept: %q", got)
	}

	w = serveStatic(s, "/plugins/eam/1.0.0/missing.js", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("missing: %d", w.Code)
	}

	w = serveStatic(s, "/plugins/eam/1.0.0/__core_studio/changes", "text/event-stream")
	if !w.Flushed || w.Body.String() != "data: {}\n\n" {
		t.Errorf("event stream: flushed %v, %q", w.Flushed, w.Body.String())
	}
}

/**
* TestStaticRedirectAndFallback: redirect mode answers 302 to the origin except for the entry files; a SPA answers its
* fallback for the prefix and the pages that ask html, not for a file or a fetch
**/
func TestStaticRedirectAndFallback(t *testing.T) {
	s, origin := newStaticServer(t)
	if _, err := s.SetStatic("/*", origin.URL+"/web/2.0.0/*", ModeRedirect, "index.html", 1, "frontend", false); err != nil {
		t.Fatal(err)
	}

	w := serveStatic(s, "/assets/index-BqzK3a1x.js?v=1", "*/*")
	if w.Code != http.StatusFound || w.Header().Get("Location") != origin.URL+"/web/2.0.0/assets/index-BqzK3a1x.js?v=1" {
		t.Fatalf("redirect: %d %q", w.Code, w.Header().Get("Location"))
	}

	for _, page := range []string{"/", "/desk/home"} {
		w = serveStatic(s, page, "text/html,application/xhtml+xml")
		if w.Code != http.StatusOK || w.Body.String() != "file /web/2.0.0/index.html" {
			t.Errorf("%s: %d %q", page, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Cache-Control"); got != CacheNoCache {
			t.Errorf("%s cache %q", page, got)
		}
	}

	// A module without an extension (/@vite/client) is not a page
	w = serveStatic(s, "/@vite/client", "*/*")
	if w.Code != http.StatusFound {
		t.Errorf("/@vite/client: %d", w.Code)
	}

	if _, err := s.SetStatic("/client", origin.URL+"/*", ModeProxy, "", 1, "frontend", false); err == nil {
		t.Error("a static route without /* was accepted")
	}
	if _, err := s.SetStatic("/client/*", origin.URL+"/*", "copy", "", 1, "frontend", false); err == nil {
		t.Error("an unknown mode was accepted")
	}
}
