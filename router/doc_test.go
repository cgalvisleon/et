package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
)

// echo: an Item handler that returns its body
func echo(r *http.Request, body et.Json) (et.Item, error) {
	return et.Item{Ok: true, Result: body}, nil
}

// usersQuery: the contract of a query route used by the tests
var usersQuery = Doc{
	Description: "Users of the tenant",
	Query:       Object(et.Json{"limit": Integer("rows per page")}),
	Body: Object(et.Json{
		"search": String("text to look for"),
		"status": Enum("status of the user", "active", "archived"),
		"tags":   Array(String()),
	}, "search"),
	Response: Object(et.Json{"ok": Boolean(), "result": Any()}),
	MCP:      true,
	Validate: true,
}

/**
* testApi: An Api without services (no cache, no events) with two documented routes and one without doc
* @return *Api
**/
func testApi() *Api {
	api := NewApi("test", "/api/test", "localhost", 0, 0, 1)
	api.Protected("users", POST, "/users/query", "Query users", api.Item(http.StatusOK, echo), usersQuery)
	api.Protected("users", GET, "/users/{id}", "Get a user", api.Item(http.StatusOK, echo), Doc{MCP: true})
	api.Protected("users", DELETE, "/users/{id}", "Delete a user", api.Item(http.StatusOK, echo), Doc{MCP: true})
	api.Public("version", GET, "/version", "Version", api.Item(http.StatusOK, echo))
	return api
}

/**
* call: A request to the router of an Api
* @param api *Api, method, target, body string, header map[string]string
* @return *httptest.ResponseRecorder
**/
func call(api *Api, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range header {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	api.Router.ServeHTTP(rec, req)
	return rec
}

func TestEndpointsPublishDoc(t *testing.T) {
	api := testApi()
	items, err := api.ListEndpoints(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if items.Count != 4 {
		t.Fatalf("endpoints: got %d, want 4", items.Count)
	}

	query := items.Result[0]
	if query.Str("access") != AccessProtected || query.Str("path") != "/api/test/users/query" {
		t.Fatalf("query endpoint: %v", query)
	}
	doc := query.Json("doc")
	if !doc.Bool("mcp") || !doc.Bool("validate") || doc.Str("description") != "Users of the tenant" {
		t.Fatalf("doc: %v", doc)
	}
	if doc.ValStr("", "body", "properties", "search", "type") != "string" {
		t.Fatalf("doc body: %v", doc.Json("body"))
	}
	if _, ok := doc["header"]; ok {
		t.Fatalf("empty header published: %v", doc)
	}

	version := items.Result[3]
	if _, ok := version["doc"]; ok || version.Str("access") != AccessPublic {
		t.Fatalf("version endpoint: %v", version)
	}
}

func TestValidateBodyAndQuery(t *testing.T) {
	api := testApi()
	cases := []struct {
		name, target, body string
		status             int
		contains           string
	}{
		{"valid", "/api/test/users/query?limit=10", `{"search":"ana","status":"active","tags":["a"]}`, http.StatusOK, `"search":"ana"`},
		{"required", "/api/test/users/query", `{}`, http.StatusBadRequest, "body.search"},
		{"empty body", "/api/test/users/query", ``, http.StatusBadRequest, "body.search"},
		{"type", "/api/test/users/query", `{"search":1}`, http.StatusBadRequest, "body.search"},
		{"enum", "/api/test/users/query", `{"search":"a","status":"other"}`, http.StatusBadRequest, "body.status"},
		{"items", "/api/test/users/query", `{"search":"a","tags":[1]}`, http.StatusBadRequest, "body.tags[0]"},
		{"query", "/api/test/users/query?limit=abc", `{"search":"a"}`, http.StatusBadRequest, "query.limit"},
		{"json", "/api/test/users/query", `{`, http.StatusBadRequest, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(api, POST, tc.target, tc.body, nil)
			if rec.Code != tc.status {
				t.Fatalf("status: got %d, want %d (%s)", rec.Code, tc.status, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("body %s does not contain %s", rec.Body.String(), tc.contains)
			}
		})
	}
}

func TestValidateSchema(t *testing.T) {
	schema := Object(et.Json{
		"page":  Integer(),
		"price": Number(),
		"items": Array(Object(et.Json{"id": String()}, "id")),
		"any":   Any(),
		"kind":  et.Json{"type": []any{"string", "null"}},
	}, "page")
	var value any
	json.Unmarshal([]byte(`{"page":1,"price":1.5,"items":[{"id":"a"}],"any":[1,"x"],"kind":null}`), &value)
	if errs := ValidateSchema(schema, value, "body"); len(errs) != 0 {
		t.Fatalf("valid value: %v", errs)
	}

	json.Unmarshal([]byte(`{"page":1.5,"items":[{}],"kind":3}`), &value)
	errs := ValidateSchema(schema, value, "body")
	want := []string{"body.items[0].id", "body.kind", "body.page"}
	if len(errs) != len(want) {
		t.Fatalf("errors: got %v, want %v", errs, want)
	}
	for i, field := range want {
		if !strings.Contains(errs[i], field) {
			t.Fatalf("error %d: got %s, want %s", i, errs[i], field)
		}
	}
}

func TestPathParams(t *testing.T) {
	got := PathParams("/api/{tenantId}/users/{id:[0-9]+}")
	if strings.Join(got, ",") != "tenantId,id" {
		t.Fatalf("params: %v", got)
	}
}

// RpcService: a net/rpc service for the doc of /rpcs
type RpcService struct{}

func (s *RpcService) Echo(args et.Json, reply *et.Json) error {
	*reply = args
	return nil
}

func TestRpcDoc(t *testing.T) {
	if _, err := jrpc.Mount("localhost", 4999, &RpcService{}, "test"); err != nil {
		t.Fatal(err)
	}

	api := testApi()
	api.DocRpc("RpcService.Echo", Doc{Description: "Returns its arguments", Body: Object(et.Json{"text": String()})})
	items, err := api.ListRpcs(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, item := range items.Result {
		if item.Str("method") == "RpcService.Echo" {
			found = item.ValStr("", "doc", "description") == "Returns its arguments"
		}
	}
	if !found {
		t.Fatalf("rpc doc not published: %v", items.Result)
	}
}

func TestDocParams(t *testing.T) {
	api := NewApi("test", "/api/test", "localhost", 0, 0, 1)
	described := Doc{Params: Object(et.Json{"id": Integer("The id of the user")}), Validate: true}
	api.Protected("users", GET, "/users/{tenantId}/{id}", "Get a user", api.Item(http.StatusOK, echo), described)
	api.Protected("users", DELETE, "/users/{id}", "Delete a user", api.Item(http.StatusOK, echo))
	api.Public("version", GET, "/version", "Version", api.Item(http.StatusOK, echo))

	items, _ := api.ListEndpoints(nil, nil)
	get := items.Result[0].Json("doc")
	if get.ValStr("", "params", "properties", "id", "description") != "The id of the user" || get.ValStr("", "params", "properties", "id", "type") != "integer" {
		t.Fatalf("described param: %v", get.Json("params"))
	}
	if get.ValStr("", "params", "properties", "tenantId", "type") != "string" || get.ValStr("", "params", "properties", "tenantId", "description") == "" {
		t.Fatalf("completed param: %v", get.Json("params"))
	}
	if strings.Join(get.ArrayStr("params", "required"), ",") != "tenantId,id" {
		t.Fatalf("required: %v", get.ArrayStr("params", "required"))
	}
	if len(described.Params["properties"].(et.Json)) != 1 {
		t.Fatalf("the caller's doc was changed: %v", described.Params)
	}

	// A route with params and without doc gets one with its params
	del := items.Result[1].Json("doc")
	if del.ValStr("", "params", "properties", "id", "type") != "string" || strings.Join(del.ArrayStr("params", "required"), ",") != "id" {
		t.Fatalf("doc without params: %v", del)
	}
	if _, ok := items.Result[2]["doc"]; ok {
		t.Fatalf("version without params got a doc: %v", items.Result[2])
	}

	// Validate converts the params by their type
	if rec := call(api, GET, "/api/test/users/t1/7", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("valid params: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(api, GET, "/api/test/users/t1/abc", "", nil); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "params.id") {
		t.Fatalf("typed param: %d %s", rec.Code, rec.Body.String())
	}
}
