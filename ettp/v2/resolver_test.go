package ettp

import (
	"net/http/httptest"
	"testing"
)

/**
* TestResolverQuery: the url of the service keeps the query string of the request (before, it was dropped and a route
* like /maps/{id}/layers?status=all lost its status)
**/
func TestResolverQuery(t *testing.T) {
	for _, tc := range []struct {
		solver, target, want string
	}{
		{"http://svc:3511/api/maps/{id}/layers", "/api/maps/m1/layers?status=all", "http://svc:3511/api/maps/m1/layers?status=all"},
		{"http://svc:3504/api/ias/chat", "/api/ias/chat?owner_id=tab%3Ahome&last=true", "http://svc:3504/api/ias/chat?owner_id=tab%3Ahome&last=true"},
		{"http://svc:3503/api/tenancy/x?fixed=1", "/api/tenancy/x?a=2", "http://svc:3503/api/tenancy/x?fixed=1&a=2"},
		{"http://svc:3503/api/tenancy/{id}", "/api/tenancy/abc", "http://svc:3503/api/tenancy/abc"},
	} {
		r := httptest.NewRequest("GET", tc.target, nil)
		params := map[string]string{}
		if tc.solver == "http://svc:3511/api/maps/{id}/layers" {
			params["{id}"] = "m1"
		}
		if tc.solver == "http://svc:3503/api/tenancy/{id}" {
			params["{id}"] = "abc"
		}
		resolver, err := newResolver(r, &Solver{Solver: tc.solver}, params)
		if err != nil {
			t.Fatal(err)
		}
		if resolver.URL != tc.want {
			t.Errorf("%s: %q, want %q", tc.target, resolver.URL, tc.want)
		}
	}
}
