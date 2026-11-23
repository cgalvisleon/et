package router

import "testing"

/**
* TestPushStatic: the route of a prefix ends in /* with its resolve, kind static, mode and fallback, and is kept to be
* published again on a reset; RemoveApiGateway(StaticId) drops it
**/
func TestPushStatic(t *testing.T) {
	id := PushStatic("/plugins/eam/1.0.0/", "http://localhost:5603/plugins/eam/1.0.0/", ModeRedirect, "", 1, "frontend")
	if id != "GET:/plugins/eam/1.0.0/*" || id != StaticId("/plugins/eam/1.0.0") {
		t.Fatalf("id %q", id)
	}

	route := GetRoutes()[id]
	if route.Str("path") != "/plugins/eam/1.0.0/*" || route.Str("resolve") != "http://localhost:5603/plugins/eam/1.0.0/*" {
		t.Errorf("route %s", route.ToString())
	}
	if route.Str("kind") != "static" || route.Str("mode") != ModeRedirect {
		t.Errorf("route %s", route.ToString())
	}

	root := PushStatic("/", "http://localhost:5600", ModeProxy, "index.html", 1, "frontend")
	if root != "GET:/*" || GetRoutes()[root].Str("resolve") != "http://localhost:5600/*" {
		t.Errorf("root %q %s", root, GetRoutes()[root].ToString())
	}

	RemoveApiGateway(id)
	RemoveApiGateway(root)
	if _, ok := GetRoutes()[id]; ok {
		t.Error("the route was not removed")
	}
}
