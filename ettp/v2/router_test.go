package ettp

import "testing"

/**
* TestRouterDeleteKeepsOthers: deleting a route keeps the routes that share its nodes (before, deleting /a/b also
* dropped /a when /a had its own solver, and deleting /a dropped /a/b)
**/
func TestRouterDeleteKeepsOthers(t *testing.T) {
	root := newRouter(POST)
	paths := []string{"/api/engine/hooks/a", "/api/engine/hooks/a/b", "/api/engine/hooks/c"}
	for _, path := range paths {
		if _, err := root.set(TpApiRest, POST, path, "http://svc"+path, TpReplaceHeader, nil, nil, 1); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := root.delete("/api/engine/hooks/a/b"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := root.find("/api/engine/hooks/a"); err != nil {
		t.Fatalf("deleting /a/b dropped /a: %s", err)
	}
	if _, _, err := root.find("/api/engine/hooks/a/b"); err == nil {
		t.Fatal("/a/b still resolves")
	}

	if _, err := root.set(TpApiRest, POST, "/api/engine/hooks/a/b", "http://svc/b", TpReplaceHeader, nil, nil, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := root.delete("/api/engine/hooks/a"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := root.find("/api/engine/hooks/a/b"); err != nil {
		t.Fatalf("deleting /a dropped /a/b: %s", err)
	}
	if _, _, err := root.find("/api/engine/hooks/a"); err == nil {
		t.Fatal("/a still resolves")
	}

	for _, path := range []string{"/api/engine/hooks/a/b", "/api/engine/hooks/c"} {
		if _, err := root.delete(path); err != nil {
			t.Fatal(err)
		}
	}
	if len(root.Router) != 0 {
		t.Fatalf("empty nodes left: %v", root.ToJson())
	}
}
