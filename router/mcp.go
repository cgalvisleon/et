package router

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/go-chi/chi/v5"
)

/**
* MCP server of an Api (Streamable HTTP, JSON-RPC 2.0 over POST, responses in application/json): initialize, ping,
* tools/list and tools/call. The tools are the routes whose Doc has MCP; a tools/call runs the route in the same chi
* router with the Authorization (and the other headers) of the MCP request, so it passes through the same
* authentication, authorization and validation as an HTTP call. There is no SSE stream, no session and no progress:
* a stream route gives its whole response. A request with an Origin header that is not in the allowed origins gets 403
* (against DNS rebinding); without Origin (a client that is not a browser) it passes, with its token. MountMCPWith serves
* the same protocol with the tools of an MCPSource (routes of other services, as an aggregating MCP) instead of the local
* routes.
**/

// MCPProtocolVersion: the newest version of the MCP protocol this server speaks
const MCPProtocolVersion = "2025-06-18"

// mcpVersions: the versions of the protocol it accepts from a client (it answers with the client's if it is here)
var mcpVersions = []string{MCPProtocolVersion, "2025-03-26", "2024-11-05"}

// JSON-RPC 2.0 error codes
const (
	mcpParseError     = -32700
	mcpInvalidRequest = -32600
	mcpMethodNotFound = -32601
	mcpInvalidParams  = -32602
	mcpInternalError  = -32603
	mcpForbidden      = -32000
)

// mcpLocalHosts: the hosts allowed by default, with any port and scheme
var mcpLocalHosts = []string{"localhost", "127.0.0.1", "::1"}

// toolName: what a tool name may have (MCP: letters, digits, _ and -, up to 64)
var toolName = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

/**
* MCPConfig: The MCP server of an Api: Name is the name of its route in the endpoint catalog, Title and Instructions go
* to initialize, Destructive also offers the destructive tools (Doc.Destructive or DELETE), out by default, and Origins
* are the allowed values of the Origin header: "*" (all), a full origin ("https://app.example.com:8443", exact) or a host
* ("app.example.com", any scheme and port). Empty: MCP_ORIGINS (comma separated) or, without it, localhost, 127.0.0.1,
* ::1 and the Host of the Api.
**/
type MCPConfig struct {
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Instructions string   `json:"instructions"`
	Destructive  bool     `json:"destructive"`
	Origins      []string `json:"origins"`
}

/**
* mcpTool: A route offered as a tool: its endpoint, the tool name, its path params and its query params
**/
type mcpTool struct {
	endpoint Endpoint
	name     string
	params   []string
	query    []string
	wrapped  bool
}

/**
* mcpMessage: A JSON-RPC 2.0 message from the client; without id it is a notification
**/
type mcpMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

/**
* mcpError: A JSON-RPC 2.0 error
**/
type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

/**
* MCPToolDef: A tool of an MCPSource: the route that serves it (Method, Path with its {params}, Name and Doc: the input
* schema, the annotations and how the arguments go to path, query and body come from it) and its Name (empty:
* MCPToolName of the route without a base). Like a local route, it needs Doc.MCP and, if destructive, MCPConfig.Destructive
**/
type MCPToolDef struct {
	Name     string   `json:"name"`
	Endpoint Endpoint `json:"endpoint"`
}

/**
* MCPSource: Where an MCP server mounted with MountMCPWith takes its tools and runs them: MCPTools gives the tools the
* request can use (tools/list and tools/call look only there) and MCPCall runs the HTTP request of a call (its Path is
* the one of the route, with the params, the query, the JSON body and the headers of the MCP request) and gives the status
* and the body of the answer; an error of MCPCall is a result with isError
**/
type MCPSource interface {
	MCPTools(r *http.Request) ([]MCPToolDef, error)
	MCPCall(req *http.Request) (int, []byte, error)
}

/**
* mcpServer: An MCP server of an Api: its config and its source (nil: the routes of the Api)
**/
type mcpServer struct {
	api    *Api
	config MCPConfig
	source MCPSource
}

/**
* MountMCP: Mounts the MCP server of the Api at POST <path>/mcp (Session: with token if the Api uses authentication;
* each tool keeps the access of its route)
* @param config ...MCPConfig
**/
func (s *Api) MountMCP(config ...MCPConfig) {
	cfg := MCPConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}
	cfg.Origins = s.mcpOrigins(cfg.Origins)
	s.mcp = &cfg
	server := &mcpServer{api: s, config: cfg}
	name := cmp.Or(cfg.Name, "MCP server")
	s.Session("mcp", POST, "/mcp", name, server.handle, mcpDoc)
}

/**
* MountMCPWith: Mounts at POST <path of the Api><path> an MCP server whose tools come from source instead of the routes of
* the Api (Session: with token if the Api uses authentication); the protocol, the origins and the answers are the ones of
* MountMCP
* @param path string, config MCPConfig, source MCPSource
**/
func (s *Api) MountMCPWith(path string, config MCPConfig, source MCPSource) {
	config.Origins = s.mcpOrigins(config.Origins)
	server := &mcpServer{api: s, config: config, source: source}
	name := cmp.Or(config.Name, "MCP server")
	s.Session("mcp", POST, path, name, server.handle, mcpSourceDoc)
}

/**
* mcpOrigins: The allowed origins: the ones given, MCP_ORIGINS (comma separated) or the local hosts plus the Host of the
* Api; the entries are trimmed, without empty ones and without a final /
* @param origins []string
* @return []string
**/
func (s *Api) mcpOrigins(origins []string) []string {
	if len(origins) == 0 {
		origins = strings.Split(envar.GetStr("MCP_ORIGINS", ""), ",")
	}

	result := []string{}
	for _, origin := range origins {
		origin = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(origin)), "/")
		if origin != "" && !slices.Contains(result, origin) {
			result = append(result, origin)
		}
	}
	if len(result) > 0 {
		return result
	}

	result = append(result, mcpLocalHosts...)
	host := strings.ToLower(strings.TrimSpace(s.Host))
	if host != "" && host != "0.0.0.0" && !slices.Contains(result, host) {
		result = append(result, host)
	}
	return result
}

/**
* OriginAllowed: If an Origin header is in the list: "*", the same origin (scheme, host and port) or the same host;
* an Origin that does not parse (as "null") only passes with "*". The rule of the MCP origins and of the CORS of the
* ettp/v2 gateway
* @param origins []string, origin string
* @return bool
**/
func OriginAllowed(origins []string, origin string) bool {
	if slices.Contains(origins, "*") {
		return true
	}

	origin = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(origin)), "/")
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	host := parsed.Hostname()
	for _, allowed := range origins {
		if strings.Contains(allowed, "://") {
			if allowed == parsed.Scheme+"://"+parsed.Host {
				return true
			}
			continue
		}
		if strings.Trim(allowed, "[]") == host {
			return true
		}
	}
	return false
}

/**
* handle: One JSON-RPC message or a batch; notifications and responses get 202 without body
* @param w http.ResponseWriter, r *http.Request
**/
func (s *mcpServer) handle(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && !OriginAllowed(s.config.Origins, origin) {
		writeMCP(w, http.StatusForbidden, mcpFail(nil, mcpForbidden, fmt.Sprintf(MSG_MCP_ORIGIN, origin)))
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCP(w, http.StatusBadRequest, mcpFail(nil, mcpParseError, MSG_MCP_PARSE_ERROR))
		return
	}

	data = bytes.TrimSpace(data)
	batch := len(data) > 0 && data[0] == '['
	messages := []mcpMessage{}
	if batch {
		err = json.Unmarshal(data, &messages)
	} else {
		message := mcpMessage{}
		err = json.Unmarshal(data, &message)
		messages = append(messages, message)
	}
	if err != nil {
		writeMCP(w, http.StatusBadRequest, mcpFail(nil, mcpParseError, MSG_MCP_PARSE_ERROR))
		return
	}

	responses := []et.Json{}
	for _, message := range messages {
		if response := s.message(r, message); response != nil {
			responses = append(responses, response)
		}
	}

	switch {
	case len(responses) == 0:
		w.WriteHeader(http.StatusAccepted)
	case batch:
		writeMCP(w, http.StatusOK, responses)
	default:
		writeMCP(w, http.StatusOK, responses[0])
	}
}

/**
* message: The response to a message; nil for a notification or a response of the client
* @param r *http.Request, message mcpMessage
* @return et.Json
**/
func (s *mcpServer) message(r *http.Request, message mcpMessage) et.Json {
	if len(message.ID) == 0 {
		return nil
	}

	if message.JSONRPC != "2.0" || message.Method == "" {
		return mcpFail(message.ID, mcpInvalidRequest, MSG_MCP_INVALID)
	}

	params := et.Json{}
	if len(message.Params) > 0 && string(message.Params) != "null" {
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return mcpFail(message.ID, mcpInvalidParams, err.Error())
		}
	}

	switch message.Method {
	case "initialize":
		return mcpResult(message.ID, s.initialize(params))
	case "ping":
		return mcpResult(message.ID, et.Json{})
	case "tools/list":
		tools, err := s.toolsList(r)
		if err != nil {
			return mcpFail(message.ID, mcpInternalError, err.Error())
		}
		return mcpResult(message.ID, et.Json{"tools": tools})
	case "tools/call":
		result, err := s.call(r, params)
		if err != nil {
			return err.response(message.ID)
		}
		return mcpResult(message.ID, result)
	}

	return mcpFail(message.ID, mcpMethodNotFound, fmt.Sprintf(MSG_MCP_METHOD, message.Method))
}

/**
* initialize: The protocol version (the client's if it is supported), the capabilities and the server info
* @param params et.Json
* @return et.Json
**/
func (s *mcpServer) initialize(params et.Json) et.Json {
	version := params.Str("protocolVersion")
	if !slices.Contains(mcpVersions, version) {
		version = MCPProtocolVersion
	}

	info := et.Json{"name": s.api.Name, "version": strconv.Itoa(s.api.Version)}
	if s.config.Title != "" {
		info["title"] = s.config.Title
	}
	result := et.Json{
		"protocolVersion": version,
		"capabilities":    et.Json{"tools": et.Json{"listChanged": false}},
		"serverInfo":      info,
	}
	if s.config.Instructions != "" {
		result["instructions"] = s.config.Instructions
	}
	return result
}

/**
* tools: The tools of the server: the source's or, without it, the routes of the Api
* @param r *http.Request
* @return []mcpTool, error
**/
func (s *mcpServer) tools(r *http.Request) ([]mcpTool, error) {
	if s.source == nil {
		return s.api.mcpTools(), nil
	}

	defs, err := s.source.MCPTools(r)
	if err != nil {
		return nil, err
	}
	return mcpToolsOf(defs, s.config.Destructive), nil
}

/**
* toolsList: The tools the request can use (the local ones, those whose route authorization passes a probe)
* @param r *http.Request
* @return []et.Json, error
**/
func (s *mcpServer) toolsList(r *http.Request) ([]et.Json, error) {
	tools, err := s.tools(r)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	for _, tool := range tools {
		if s.source != nil || s.api.mcpAllowed(r, tool.endpoint) {
			result = append(result, tool.toJson())
		}
	}
	return result, nil
}

/**
* MCPTools: The tools the MCP server offers, without checking a role: {name, description, inputSchema, annotations}
* @return []et.Json
**/
func (s *Api) MCPTools() []et.Json {
	result := []et.Json{}
	for _, tool := range s.mcpTools() {
		result = append(result, tool.toJson())
	}
	return result
}

/**
* mcpAllowed: If the authorization of a protected route lets the request pass (it runs the middleware on a copy of the
* request with the method and the path of the route); public and session routes are always allowed
* @param r *http.Request, endpoint Endpoint
* @return bool
**/
func (s *Api) mcpAllowed(r *http.Request, endpoint Endpoint) bool {
	if endpoint.Access != AccessProtected || s.authorization == nil {
		return true
	}

	allowed := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { allowed = true })
	probe := r.Clone(r.Context())
	probe.Method = endpoint.Method
	probe.URL = &url.URL{Path: endpoint.Path}
	probe.Body = http.NoBody
	s.authorization(endpoint.Method, endpoint.Path)(next).ServeHTTP(httptest.NewRecorder(), probe)
	return allowed
}

/**
* mcpDestructive: If a route has effects that cannot be undone (Doc.Destructive or DELETE)
* @param endpoint Endpoint
* @return bool
**/
func mcpDestructive(endpoint Endpoint) bool {
	return endpoint.Method == DELETE || (endpoint.Doc != nil && endpoint.Doc.Destructive)
}

/**
* mcpTools: The routes with Doc.MCP as tools, in the order they were mounted; the destructive ones only if the server
* allows them
* @return []mcpTool
**/
func (s *Api) mcpTools() []mcpTool {
	destructive := s.mcp != nil && s.mcp.Destructive
	defs := []MCPToolDef{}
	for _, endpoint := range s.endpoints {
		if endpoint.Doc != nil && endpoint.Doc.MCP {
			defs = append(defs, MCPToolDef{Name: MCPToolName(endpoint, s.Path), Endpoint: endpoint})
		}
	}
	return mcpToolsOf(defs, destructive)
}

/**
* mcpToolsOf: The tools of a list of definitions: only the ones with Doc.MCP and, if destructive is false, without the
* destructive ones; a name is cut to the characters of MCP (up to 64) and a repeated one takes the method as suffix
* @param defs []MCPToolDef, destructive bool
* @return []mcpTool
**/
func mcpToolsOf(defs []MCPToolDef, destructive bool) []mcpTool {
	names := map[string]bool{}
	result := []mcpTool{}
	for _, def := range defs {
		endpoint := def.Endpoint
		if endpoint.Doc == nil || !endpoint.Doc.MCP {
			continue
		}
		if mcpDestructive(endpoint) && !destructive {
			continue
		}

		name := toolName.ReplaceAllString(cmp.Or(def.Name, MCPToolName(endpoint, "")), "_")
		if len(name) > 64 {
			name = name[:64]
		}
		if names[name] {
			name = name + "_" + strings.ToLower(endpoint.Method)
		}
		names[name] = true
		tool := mcpTool{endpoint: endpoint, name: name, params: PathParams(endpoint.Path)}
		for key := range schemaJson(endpoint.Doc.Query["properties"]) {
			tool.query = append(tool.query, key)
		}
		body := endpoint.Doc.Body
		tool.wrapped = len(body) > 0 && len(schemaJson(body["properties"])) == 0
		result = append(result, tool)
	}
	return result
}

/**
* MCPToolName: The name of the tool of a route: Doc.Tool or <group>_<action>, where the action is the fixed segments of
* the path after base (the path of its Api) and the group or, without them, get (GET with params), list (GET), new
* (POST), update (PUT, PATCH) or delete (DELETE); only letters, digits, _ and -, up to 64
* @param endpoint Endpoint, base string
* @return string
**/
func MCPToolName(endpoint Endpoint, base string) string {
	if endpoint.Doc != nil && endpoint.Doc.Tool != "" {
		return toolName.ReplaceAllString(endpoint.Doc.Tool, "_")
	}

	rel := strings.TrimPrefix(endpoint.Path, strings.TrimSuffix(base, "/"))
	segments := []string{}
	params := false
	for _, segment := range strings.Split(rel, "/") {
		switch {
		case segment == "":
		case strings.HasPrefix(segment, "{"):
			params = true
		default:
			segments = append(segments, segment)
		}
	}
	if len(segments) > 0 && segments[0] == endpoint.Group {
		segments = segments[1:]
	}

	if len(segments) == 0 {
		switch endpoint.Method {
		case GET:
			if params {
				segments = append(segments, "get")
			} else {
				segments = append(segments, "list")
			}
		case POST:
			segments = append(segments, "new")
		case PUT, PATCH:
			segments = append(segments, "update")
		case DELETE:
			segments = append(segments, "delete")
		default:
			segments = append(segments, strings.ToLower(endpoint.Method))
		}
	}

	name := strings.Join(append([]string{endpoint.Group}, segments...), "_")
	name = toolName.ReplaceAllString(name, "_")
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

/**
* inputSchema: The arguments of a tool: its path params (required, with the schemas of Doc.Params), its query params and the properties of its
* body, in one object; a body without properties (an array, any value) goes in the argument body
* @return et.Json
**/
func (s *mcpTool) inputSchema() et.Json {
	properties := et.Json{}
	required := []string{}
	doc := s.endpoint.Doc
	described := schemaJson(doc.Params["properties"])
	for _, param := range s.params {
		schema := schemaJson(described[param])
		if len(schema) == 0 {
			schema = String(fmt.Sprintf(MSG_PATH_PARAM, param))
		}
		properties[param] = schema
		required = append(required, param)
	}

	schemas := []et.Json{doc.Query}
	if !s.wrapped {
		schemas = append(schemas, doc.Body)
	}
	for _, schema := range schemas {
		for key, value := range schemaJson(schema["properties"]) {
			if _, ok := properties[key]; ok {
				continue
			}
			properties[key] = value
		}
		for _, key := range schemaList(schema["required"]) {
			if !slices.Contains(required, key) {
				required = append(required, key)
			}
		}
	}
	if s.wrapped {
		properties["body"] = doc.Body
		required = append(required, "body")
	}

	result := et.Json{"type": "object", "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

/**
* toJson: The tool as tools/list gives it, with its annotations
* @return et.Json
**/
func (s *mcpTool) toJson() et.Json {
	endpoint := s.endpoint
	description := endpoint.Name
	if endpoint.Doc.Description != "" {
		description = endpoint.Name + ". " + endpoint.Doc.Description
	}
	if endpoint.Doc.Stream != "" {
		description += " (stream: " + endpoint.Doc.Stream + ")"
	}
	readOnly := endpoint.Method == GET || strings.HasSuffix(endpoint.Path, "/query")
	annotations := et.Json{
		"title":           endpoint.Name,
		"readOnlyHint":    readOnly,
		"destructiveHint": mcpDestructive(endpoint),
		"idempotentHint":  readOnly || endpoint.Method == PUT || endpoint.Method == DELETE,
		"openWorldHint":   false,
	}
	return et.Json{
		"name":        s.name,
		"title":       endpoint.Name,
		"description": description,
		"inputSchema": s.inputSchema(),
		"annotations": annotations,
	}
}

/**
* mcpCallError: A tools/call that does not reach the route (unknown tool, bad arguments): a JSON-RPC error
**/
type mcpCallError struct {
	code    int
	message string
}

/**
* response
* @param id json.RawMessage
* @return et.Json
**/
func (s *mcpCallError) response(id json.RawMessage) et.Json {
	return mcpFail(id, s.code, s.message)
}

/**
* call: Runs the route of a tool (in the chi router of the Api or with MCPCall of the source) with the headers of the
* request
* @param r *http.Request, params et.Json
* @return et.Json, *mcpCallError
**/
func (s *mcpServer) call(r *http.Request, params et.Json) (et.Json, *mcpCallError) {
	tools, err := s.tools(r)
	if err != nil {
		return nil, &mcpCallError{mcpInternalError, err.Error()}
	}

	name := params.Str("name")
	idx := slices.IndexFunc(tools, func(tool mcpTool) bool { return tool.name == name })
	if idx < 0 {
		return nil, &mcpCallError{mcpInvalidParams, fmt.Sprintf(MSG_MCP_TOOL_NOT_FOUND, name)}
	}
	tool := tools[idx]

	args := params.Json("arguments")
	req, err := tool.request(r, args)
	if err != nil {
		return nil, &mcpCallError{mcpInvalidParams, fmt.Sprintf(MSG_MCP_TOOL_PARAMS, err.Error())}
	}

	if s.source == nil {
		rec := httptest.NewRecorder()
		s.api.Router.ServeHTTP(rec, req)
		return mcpToolResult(rec.Code, rec.Body.Bytes()), nil
	}

	status, body, err := s.source.MCPCall(req)
	if err != nil {
		return et.Json{"content": []et.Json{{"type": "text", "text": err.Error()}}, "isError": true}, nil
	}
	return mcpToolResult(status, body), nil
}

/**
* mcpToolResult: The answer of a route as the result of a tools/call: as text and, if it is a JSON object, as
* structuredContent; isError if the status is 400 or more
* @param status int, body []byte
* @return et.Json
**/
func mcpToolResult(status int, body []byte) et.Json {
	result := et.Json{
		"content": []et.Json{{"type": "text", "text": string(body)}},
		"isError": status >= http.StatusBadRequest,
	}
	structured := et.Json{}
	if json.Unmarshal(body, &structured) == nil {
		result["structuredContent"] = structured
	}
	return result
}

/**
* request: The HTTP request of a tool call: the path params in the path, the query params in the query and the rest
* of the arguments in the body (or the argument body), with the headers and the context of the MCP request (without
* its chi route, so the router resolves the path again)
* @param r *http.Request, args et.Json
* @return *http.Request, error
**/
func (s *mcpTool) request(r *http.Request, args et.Json) (*http.Request, error) {
	endpoint := s.endpoint
	path := endpoint.Path
	rest := et.Json{}
	maps.Copy(rest, args)

	for _, param := range s.params {
		value, ok := rest[param]
		if !ok || value == nil || fmt.Sprint(value) == "" {
			return nil, fmt.Errorf(MSG_SCHEMA_REQUIRED, param)
		}
		delete(rest, param)
		pattern := regexp.MustCompile(`\{` + regexp.QuoteMeta(param) + `(:[^}]*)?\}`)
		path = pattern.ReplaceAllLiteralString(path, url.PathEscape(fmt.Sprint(value)))
	}

	query := url.Values{}
	for _, key := range s.query {
		value, ok := rest[key]
		if !ok || value == nil {
			continue
		}
		delete(rest, key)
		query.Set(key, textOf(value))
	}
	target := path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var body any = rest
	if s.wrapped {
		body = rest["body"]
	}
	var reader io.Reader = http.NoBody
	withBody := endpoint.Method == POST || endpoint.Method == PUT || endpoint.Method == PATCH || len(endpoint.Doc.Body) > 0
	if withBody {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}

	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, nil)
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, target, reader)
	if err != nil {
		return nil, err
	}

	req.Header = r.Header.Clone()
	for _, key := range []string{"Content-Length", "Mcp-Session-Id", "Mcp-Protocol-Version", "Accept-Encoding"} {
		req.Header.Del(key)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.RemoteAddr = r.RemoteAddr
	req.Host = r.Host
	return req, nil
}

/**
* textOf: A value of a query param as text (a string as is, the rest in JSON)
* @param value any
* @return string
**/
func textOf(value any) string {
	if text, ok := value.(string); ok {
		return text
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

/**
* mcpResult
* @param id json.RawMessage, result et.Json
* @return et.Json
**/
func mcpResult(id json.RawMessage, result et.Json) et.Json {
	return et.Json{"jsonrpc": "2.0", "id": id, "result": result}
}

/**
* mcpFail
* @param id json.RawMessage, code int, message string
* @return et.Json
**/
func mcpFail(id json.RawMessage, code int, message string) et.Json {
	var value any = id
	if len(id) == 0 {
		value = nil
	}
	return et.Json{"jsonrpc": "2.0", "id": value, "error": mcpError{Code: code, Message: message}}
}

/**
* writeMCP
* @param w http.ResponseWriter, status int, value any
**/
func writeMCP(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// mcpSourceDoc: el contrato de un MCP montado con MountMCPWith (las herramientas vienen de su fuente)
var mcpSourceDoc = mcpDocWith("Servidor MCP (JSON-RPC 2.0 sobre HTTP): initialize, ping, tools/list y tools/call con las herramientas de su fuente; un mensaje o un lote")

// mcpDoc: el contrato de POST /mcp para el catálogo de endpoints (no es una herramienta: no lleva MCP)
var mcpDoc = mcpDocWith("Servidor MCP (JSON-RPC 2.0 sobre HTTP): initialize, ping, tools/list y tools/call con las rutas marcadas del servicio; un mensaje o un lote")

/**
* mcpDocWith: El contrato de un servidor MCP con su descripción
* @param description string
* @return Doc
**/
func mcpDocWith(description string) Doc {
	return Doc{
		Description: description,
		Header: Object(et.Json{
			"Authorization": String("Bearer <token>: el mismo de las rutas; cada herramienta pasa por su autorización"),
			"Origin":        String("Si viene, debe estar en la lista permitida (MCPConfig.Origins o MCP_ORIGINS); si no, 403"),
		}, "Authorization"),
		Body: Object(et.Json{
			"jsonrpc": Enum("La versión del protocolo", "2.0"),
			"id":      Any("El id de la petición; sin id es una notificación (responde 202)"),
			"method":  Enum("El método", "initialize", "ping", "tools/list", "tools/call", "notifications/initialized"),
			"params":  Describe(Object(et.Json{"name": String("tools/call: la herramienta"), "arguments": Any("tools/call: sus argumentos")}), "Los parámetros del método"),
		}, "jsonrpc", "method"),
		Response: Describe(Object(et.Json{"jsonrpc": String(), "id": Any(), "result": Any("La respuesta del método"), "error": Any("{code, message} si falló")}), "La respuesta JSON-RPC (o un arreglo para un lote)"),
	}
}
