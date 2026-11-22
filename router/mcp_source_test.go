package router

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* testSource: An MCPSource with two services: X-Role guest only sees eam; MCPCall echoes the request it gets
**/
type testSource struct {
	calls []*http.Request
	fail  bool
}

/**
* MCPTools
* @param r *http.Request
* @return []MCPToolDef, error
**/
func (s *testSource) MCPTools(r *http.Request) ([]MCPToolDef, error) {
	if s.fail {
		return nil, errors.New("catalog down")
	}

	eam := Endpoint{PackageName: "eam", Group: "eam", Method: GET, Path: "/api/eam/eam/{model}", Name: "Get a model", Doc: &Doc{MCP: true, Query: Object(et.Json{"limit": Integer()})}}
	query := Endpoint{PackageName: "kinds", Group: "kinds", Method: POST, Path: "/api/kinds/kinds/query", Name: "Query kinds", Doc: &Doc{MCP: true, Body: Object(et.Json{"search": String()})}}
	remove := Endpoint{PackageName: "kinds", Group: "kinds", Method: DELETE, Path: "/api/kinds/kinds/{id}", Name: "Delete a kind", Doc: &Doc{MCP: true}}
	hidden := Endpoint{PackageName: "kinds", Group: "kinds", Method: POST, Path: "/api/kinds/kinds", Name: "New kind", Doc: &Doc{}}
	result := []MCPToolDef{{Name: "eam_" + MCPToolName(eam, "/api/eam"), Endpoint: eam}}
	if r.Header.Get("X-Role") != "guest" {
		result = append(result, MCPToolDef{Name: "kinds_" + MCPToolName(query, "/api/kinds"), Endpoint: query}, MCPToolDef{Endpoint: remove}, MCPToolDef{Endpoint: hidden})
	}
	return result, nil
}

/**
* MCPCall
* @param req *http.Request
* @return int, []byte, error
**/
func (s *testSource) MCPCall(req *http.Request) (int, []byte, error) {
	s.calls = append(s.calls, req)
	if strings.Contains(req.URL.Path, "down") {
		return 0, nil, errors.New("gateway down")
	}

	data, _ := io.ReadAll(req.Body)
	answer, _ := json.Marshal(et.Json{"ok": true, "result": et.Json{"method": req.Method, "uri": req.URL.RequestURI(), "body": string(data), "auth": req.Header.Get("Authorization")}})
	if strings.Contains(req.URL.Path, "missing") {
		return http.StatusNotFound, answer, nil
	}
	return http.StatusOK, answer, nil
}

/**
* sourceApi: The test Api with its local MCP plus one with testSource at /mcp/all
* @param source *testSource
* @return *Api
**/
func sourceApi(source *testSource) *Api {
	api := mcpApi(MCPConfig{})
	api.MountMCPWith("/mcp/all", MCPConfig{Name: "All", Title: "Every service"}, source)
	return api
}

/**
* sourceCall: POST /api/test/mcp/all with a JSON-RPC body and a token
* @param api *Api, body string, header map[string]string
* @return int, et.Json
**/
func sourceCall(t *testing.T, api *Api, body string, header map[string]string) (int, et.Json) {
	t.Helper()
	if header == nil {
		header = map[string]string{"Authorization": "Bearer ok"}
	}
	rec := call(api, POST, "/api/test/mcp/all", body, header)
	result := et.Json{}
	if rec.Body.Len() > 0 {
		json.Unmarshal(rec.Body.Bytes(), &result)
	}
	return rec.Code, result
}

func TestMCPSourceInitialize(t *testing.T) {
	api := sourceApi(&testSource{})
	status, result := sourceCall(t, api, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`, nil)
	if status != http.StatusOK || result.ValStr("", "result", "serverInfo", "title") != "Every service" {
		t.Fatalf("initialize: %d %s", status, result.ToString())
	}

	status, _ = sourceCall(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, map[string]string{})
	if status != http.StatusUnauthorized {
		t.Fatalf("without token: got %d, want 401", status)
	}

	status, _ = sourceCall(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, map[string]string{"Authorization": "Bearer ok", "Origin": "http://evil.example.com"})
	if status != http.StatusForbidden {
		t.Fatalf("origin: got %d, want 403", status)
	}

	for _, endpoint := range api.Endpoints() {
		if endpoint.Path == "/api/test/mcp/all" && (endpoint.Access != AccessSession || endpoint.Doc == nil || endpoint.Doc.MCP) {
			t.Fatalf("route: %+v", endpoint)
		}
	}
}

func TestMCPSourceToolsList(t *testing.T) {
	api := sourceApi(&testSource{})
	_, result := sourceCall(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	names := []string{}
	for _, tool := range result.ArrayJson("result", "tools") {
		names = append(names, tool.Str("name"))
	}
	if strings.Join(names, ",") != "eam_eam_get,kinds_kinds_query" {
		t.Fatalf("tools: %v", names)
	}

	tools := result.ArrayJson("result", "tools")
	schema := tools[0].Json("inputSchema")
	if schema.ValStr("", "properties", "model", "type") != "string" || schema.ValStr("", "properties", "limit", "type") != "integer" {
		t.Fatalf("input schema: %s", schema.ToString())
	}
	if !tools[1].ValBool(false, "annotations", "readOnlyHint") {
		t.Fatalf("annotations: %s", tools[1].ToString())
	}

	_, result = sourceCall(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, map[string]string{"Authorization": "Bearer ok", "X-Role": "guest"})
	if len(result.ArrayJson("result", "tools")) != 1 {
		t.Fatalf("guest: %s", result.ToString())
	}

	_, local := mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	if len(local.ArrayJson("result", "tools")) != 3 {
		t.Fatalf("the local /mcp changed: %s", local.ToString())
	}

	failing := sourceApi(&testSource{fail: true})
	_, result = sourceCall(t, failing, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	if result.ValInt(0, "error", "code") != mcpInternalError {
		t.Fatalf("source error: %s", result.ToString())
	}
}

func TestMCPSourceToolsCall(t *testing.T) {
	source := &testSource{}
	api := sourceApi(source)
	_, result := sourceCall(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"eam_eam_get","arguments":{"model":"eam.areas","limit":3}}}`, nil)
	structured := result.Json("result").Json("structuredContent")
	if structured.ValStr("", "result", "uri") != "/api/eam/eam/eam.areas?limit=3" || structured.ValStr("", "result", "auth") != "Bearer ok" || structured.ValStr("", "result", "method") != GET {
		t.Fatalf("call: %s", result.ToString())
	}
	if result.ValBool(true, "result", "isError") {
		t.Fatalf("isError: %s", result.ToString())
	}

	_, result = sourceCall(t, api, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"kinds_kinds_query","arguments":{"search":"ana"}}}`, nil)
	if result.ValStr("", "result", "structuredContent", "result", "body") != `{"search":"ana"}` {
		t.Fatalf("body: %s", result.ToString())
	}

	_, result = sourceCall(t, api, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"eam_eam_get","arguments":{"model":"missing"}}}`, nil)
	if !result.ValBool(false, "result", "isError") {
		t.Fatalf("404 is an error: %s", result.ToString())
	}

	_, result = sourceCall(t, api, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"eam_eam_get","arguments":{"model":"down"}}}`, nil)
	if !result.ValBool(false, "result", "isError") || !strings.Contains(result.ToString(), "gateway down") {
		t.Fatalf("call error: %s", result.ToString())
	}

	calls := len(source.calls)
	for _, body := range []string{
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"kinds_kinds_query","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"kinds_delete","arguments":{"id":"1"}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"eam_eam_get","arguments":{}}}`,
	} {
		guest := map[string]string{"Authorization": "Bearer ok", "X-Role": "guest"}
		_, result = sourceCall(t, api, body, guest)
		if result.ValInt(0, "error", "code") != mcpInvalidParams {
			t.Fatalf("%s: %s", body, result.ToString())
		}
	}
	if len(source.calls) != calls {
		t.Fatalf("a refused call reached the source")
	}
}
