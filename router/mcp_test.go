package router

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/go-chi/chi/v5"
)

// rawEcho: a handler that answers {ok, result: <body as text>}, for a body that is not an object
func rawEcho(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true,"result":` + strconv.Quote(string(data)) + `}`))
}

/**
* mcpApi: The test Api with a fake authentication (Authorization: Bearer ok) and authorization (X-Role guest cannot
* GET a user), plus its MCP server
* @param config MCPConfig
* @return *Api
**/
func mcpApi(config MCPConfig) *Api {
	api := NewApi("test", "/api/test", "localhost", 0, 0, 1)
	api.UseAuthentication(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer ok" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	api.UseAuthorization(func(method, path string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Role") == "guest" && method == GET {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			})
		}
	})
	api.Protected("users", POST, "/users/query", "Query users", api.Item(http.StatusOK, echo), usersQuery)
	api.Protected("users", GET, "/users/{id}", "Get a user", api.Item(http.StatusOK, func(r *http.Request, body et.Json) (et.Item, error) {
		return et.Item{Ok: true, Result: et.Json{"id": chi.URLParam(r, "id"), "limit": r.URL.Query().Get("limit")}}, nil
	}), Doc{MCP: true, Params: Object(et.Json{"id": String("The id of the user")}), Query: Object(et.Json{"limit": Integer()})})
	api.Protected("users", PUT, "/users/{id}/tags", "Set the tags", rawEcho, Doc{MCP: true, Body: Array(String())})
	api.Protected("users", DELETE, "/users/{id}", "Delete a user", api.Item(http.StatusOK, echo), Doc{MCP: true})
	api.Protected("users", POST, "/users", "Create a user", api.Item(http.StatusOK, echo))
	api.MountMCP(config)
	return api
}

/**
* mcpCallTest: POST /api/test/mcp with a JSON-RPC body
* @param api *Api, body string, header map[string]string
* @return int, et.Json
**/
func mcpCallTest(t *testing.T, api *Api, body string, header map[string]string) (int, et.Json) {
	t.Helper()
	if header == nil {
		header = map[string]string{"Authorization": "Bearer ok"}
	}
	rec := call(api, POST, "/api/test/mcp", body, header)
	if rec.Body.Len() == 0 {
		return rec.Code, et.Json{}
	}
	result := et.Json{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		return rec.Code, et.Json{"raw": rec.Body.String()}
	}
	return rec.Code, result
}

func TestMCPInitialize(t *testing.T) {
	api := mcpApi(MCPConfig{Title: "Test", Instructions: "Use the users tools"})
	status, result := mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`, nil)
	if status != http.StatusOK {
		t.Fatalf("status %d: %v", status, result)
	}
	if result.ValStr("", "result", "protocolVersion") != "2025-03-26" || result.ValStr("", "result", "serverInfo", "name") != "test" {
		t.Fatalf("initialize: %v", result)
	}
	if result.ValStr("", "result", "instructions") != "Use the users tools" || result.ValBool(true, "result", "capabilities", "tools", "listChanged") {
		t.Fatalf("initialize: %v", result)
	}

	status, _ = mcpCallTest(t, api, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, nil)
	if status != http.StatusAccepted {
		t.Fatalf("notification: status %d", status)
	}

	status, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":2,"method":"resources/list"}`, nil)
	if status != http.StatusOK || result.ValInt(0, "error", "code") != mcpMethodNotFound {
		t.Fatalf("unknown method: %v", result)
	}

	status, _ = mcpCallTest(t, api, `{`, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("parse error: status %d", status)
	}
}

func TestMCPWithoutToken(t *testing.T) {
	api := mcpApi(MCPConfig{})
	status, _ := mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, map[string]string{})
	if status != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", status)
	}
}

/**
* toolNames: The names of the tools of a tools/list response
* @param result et.Json
* @return []string
**/
func toolNames(result et.Json) []string {
	names := []string{}
	for _, tool := range result.Json("result").ArrayJson("tools") {
		names = append(names, tool.Str("name"))
	}
	return names
}

func TestMCPToolsList(t *testing.T) {
	api := mcpApi(MCPConfig{})
	_, result := mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	names := toolNames(result)
	if strings.Join(names, ",") != "users_query,users_get,users_tags" {
		t.Fatalf("tools: %v", names)
	}

	tools := result.Json("result").ArrayJson("tools")
	query := tools[0]
	if !slices.Contains(query.ArrayStr("inputSchema", "required"), "search") || query.ValStr("", "inputSchema", "properties", "limit", "type") != "integer" {
		t.Fatalf("query inputSchema: %v", query.Json("inputSchema"))
	}
	if !query.ValBool(false, "annotations", "readOnlyHint") || query.Str("description") != "Query users. Users of the tenant" {
		t.Fatalf("query tool: %v", query)
	}
	get := tools[1]
	if !slices.Contains(get.ArrayStr("inputSchema", "required"), "id") || get.ValStr("", "inputSchema", "properties", "id", "description") != "The id of the user" {
		t.Fatalf("get inputSchema: %v", get.Json("inputSchema"))
	}
	tags := tools[2]
	if tags.ValStr("", "inputSchema", "properties", "body", "type") != "array" {
		t.Fatalf("tags inputSchema: %v", tags.Json("inputSchema"))
	}

	// The guest role cannot GET: users_get is not listed
	_, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, map[string]string{"Authorization": "Bearer ok", "X-Role": "guest"})
	if slices.Contains(toolNames(result), "users_get") {
		t.Fatalf("guest tools: %v", toolNames(result))
	}

	// With Destructive the DELETE route is offered, annotated
	api = mcpApi(MCPConfig{Destructive: true})
	_, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	names = toolNames(result)
	if !slices.Contains(names, "users_delete") {
		t.Fatalf("destructive tools: %v", names)
	}
	for _, tool := range result.Json("result").ArrayJson("tools") {
		if tool.Str("name") == "users_delete" && !tool.ValBool(false, "annotations", "destructiveHint") {
			t.Fatalf("users_delete without destructiveHint: %v", tool)
		}
	}
}

func TestMCPToolsCall(t *testing.T) {
	api := mcpApi(MCPConfig{})
	_, result := mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"users_query","arguments":{"search":"ana","limit":5}}}`, nil)
	if result.ValBool(true, "result", "isError") || result.ValStr("", "result", "structuredContent", "result", "search") != "ana" {
		t.Fatalf("users_query: %v", result)
	}
	content := result.Json("result").ArrayJson("content")
	if len(content) != 1 || content[0].Str("type") != "text" || !strings.Contains(content[0].Str("text"), `"search":"ana"`) {
		t.Fatalf("users_query content: %v", content)
	}
	if result.ValStr("", "result", "structuredContent", "result", "limit") != "" {
		t.Fatalf("limit went to the body: %v", result)
	}

	// The validation of the route applies: search is required
	_, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"users_query","arguments":{"limit":5}}}`, nil)
	if !result.ValBool(false, "result", "isError") {
		t.Fatalf("users_query without search: %v", result)
	}

	// Path and query params
	_, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"users_get","arguments":{"id":"u 1","limit":7}}}`, nil)
	if result.ValStr("", "result", "structuredContent", "result", "id") != "u 1" || result.ValStr("", "result", "structuredContent", "result", "limit") != "7" {
		t.Fatalf("users_get: %v", result)
	}

	// The role of the token: guest cannot GET, the route answers 403
	_, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"users_get","arguments":{"id":"1"}}}`, map[string]string{"Authorization": "Bearer ok", "X-Role": "guest"})
	if !result.ValBool(false, "result", "isError") {
		t.Fatalf("guest users_get: %v", result)
	}

	// Unknown tool, missing path param and a destructive tool not offered: JSON-RPC errors
	for _, body := range []string{
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nope"}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"users_get","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"users_delete","arguments":{"id":"1"}}}`,
	} {
		_, result = mcpCallTest(t, api, body, nil)
		if result.ValInt(0, "error", "code") != mcpInvalidParams {
			t.Fatalf("%s: %v", body, result)
		}
	}

	// A body without properties goes in the argument body
	_, result = mcpCallTest(t, api, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"users_tags","arguments":{"id":"1","body":["a","b"]}}}`, nil)
	if result.ValBool(true, "result", "isError") {
		t.Fatalf("users_tags: %v", result)
	}

	// Batch
	rec := call(api, POST, "/api/test/mcp", `[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"}]`, map[string]string{"Authorization": "Bearer ok"})
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), `[{`) || strings.Count(rec.Body.String(), `"jsonrpc"`) != 1 {
		t.Fatalf("batch: %d %s", rec.Code, rec.Body.String())
	}
}

/**
* mcpOriginTest: The status of a tools/list with token and the given Origin ("" without the header)
* @param t *testing.T, api *Api, origin string
* @return int, et.Json
**/
func mcpOriginTest(t *testing.T, api *Api, origin string) (int, et.Json) {
	t.Helper()
	header := map[string]string{"Authorization": "Bearer ok"}
	if origin != "" {
		header["Origin"] = origin
	}
	return mcpCallTest(t, api, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, header)
}

func TestMCPOriginDefault(t *testing.T) {
	api := mcpApi(MCPConfig{})
	for _, origin := range []string{"", "http://localhost:5600", "http://127.0.0.1:3000", "http://[::1]:8080", "https://localhost"} {
		if status, result := mcpOriginTest(t, api, origin); status != http.StatusOK {
			t.Fatalf("origin %q: status %d %v", origin, status, result)
		}
	}
	for _, origin := range []string{"http://evil.example.com", "http://localhost.evil.com", "null"} {
		status, result := mcpOriginTest(t, api, origin)
		if status != http.StatusForbidden || result.ValInt(0, "error", "code") != mcpForbidden {
			t.Fatalf("origin %q: status %d %v, want 403", origin, status, result)
		}
	}
}

func TestMCPOriginHost(t *testing.T) {
	api := &Api{Host: "Studio.Local"}
	origins := api.mcpOrigins(nil)
	if !OriginAllowed(origins, "https://studio.local:8443") || !OriginAllowed(origins, "http://localhost:5600") {
		t.Fatalf("host of the Api: %v", origins)
	}
	if OriginAllowed(origins, "https://other.local") {
		t.Fatalf("other host allowed: %v", origins)
	}
}

func TestMCPOriginList(t *testing.T) {
	api := mcpApi(MCPConfig{Origins: []string{" https://App.example.com/ ", "intranet", ""}})
	for _, origin := range []string{"https://app.example.com", "http://intranet:9000"} {
		if status, result := mcpOriginTest(t, api, origin); status != http.StatusOK {
			t.Fatalf("origin %q: status %d %v", origin, status, result)
		}
	}
	for _, origin := range []string{"http://app.example.com", "https://app.example.com:8443", "http://localhost:5600"} {
		if status, _ := mcpOriginTest(t, api, origin); status != http.StatusForbidden {
			t.Fatalf("origin %q: status %d, want 403", origin, status)
		}
	}

	all := mcpApi(MCPConfig{Origins: []string{"*"}})
	if status, _ := mcpOriginTest(t, all, "http://evil.example.com"); status != http.StatusOK {
		t.Fatalf("*: status %d", status)
	}
}

func TestMCPOriginEnv(t *testing.T) {
	envar.Set("MCP_ORIGINS", " https://a.example.com , b.example.com")
	t.Cleanup(func() { envar.Set("MCP_ORIGINS", "") })

	api := mcpApi(MCPConfig{})
	for _, origin := range []string{"https://a.example.com", "http://b.example.com:81"} {
		if status, result := mcpOriginTest(t, api, origin); status != http.StatusOK {
			t.Fatalf("origin %q: status %d %v", origin, status, result)
		}
	}
	if status, _ := mcpOriginTest(t, api, "http://localhost:5600"); status != http.StatusForbidden {
		t.Fatalf("localhost with MCP_ORIGINS: status %d, want 403", status)
	}

	own := mcpApi(MCPConfig{Origins: []string{"c.example.com"}})
	if status, _ := mcpOriginTest(t, own, "https://a.example.com"); status != http.StatusForbidden {
		t.Fatalf("Origins over MCP_ORIGINS: status %d, want 403", status)
	}
}
