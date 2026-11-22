package router

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
)

// Stream values of a Doc: how a route streams its response
const (
	StreamSSE    = "sse"
	StreamNDJSON = "ndjson"
)

/**
* Doc: The contract of a route, given as the optional last argument of Public, Session and Protected. Params, Query,
* Header, Body and Response are JSON Schema (the subset type, properties, required, description, enum, items). GET /endpoints
* publishes it with the route; MCP offers the route as a tool (MountMCP), Destructive keeps it out unless the MCP server
* allows destructive tools, and Validate checks the query and the body against their schemas before the handler (400).
* Tool and Description override the tool name (<group>_<action>) and add to the route name in the MCP server. Params
* (the {params} of the path, an object with one property each) is completed when the route is mounted: a {param} missing
* in it becomes a required string with a generic description, so give it only to describe or type a param.
**/
type Doc struct {
	Description string  `json:"description,omitempty"`
	Params      et.Json `json:"params,omitempty"`
	Query       et.Json `json:"query,omitempty"`
	Header      et.Json `json:"header,omitempty"`
	Body        et.Json `json:"body,omitempty"`
	Response    et.Json `json:"response,omitempty"`
	Stream      string  `json:"stream,omitempty"`
	MCP         bool    `json:"mcp,omitempty"`
	Tool        string  `json:"tool,omitempty"`
	Destructive bool    `json:"destructive,omitempty"`
	Validate    bool    `json:"validate,omitempty"`
}

/**
* ToJson: The Doc as JSON, without the empty attributes
* @return et.Json
**/
func (s *Doc) ToJson() et.Json {
	result := et.Json{}
	schemas := map[string]et.Json{"params": s.Params, "query": s.Query, "header": s.Header, "body": s.Body, "response": s.Response}
	for key, schema := range schemas {
		if len(schema) > 0 {
			result[key] = schema
		}
	}
	texts := map[string]string{"description": s.Description, "stream": s.Stream, "tool": s.Tool}
	for key, text := range texts {
		if text != "" {
			result[key] = text
		}
	}
	flags := map[string]bool{"mcp": s.MCP, "destructive": s.Destructive, "validate": s.Validate}
	for key, flag := range flags {
		if flag {
			result[key] = true
		}
	}
	return result
}

/**
* docOf: The first Doc of the optional argument of Public, Session and Protected
* @param docs []Doc
* @return *Doc
**/
func docOf(docs []Doc) *Doc {
	if len(docs) == 0 {
		return nil
	}

	doc := docs[0]
	return &doc
}

/**
* docWithParams: The doc of a route with its path params completed: each {param} of path missing in Params becomes a
* required string with a generic description and every one is required; a route with params and no doc gets one
* @param doc *Doc, path string
* @return *Doc
**/
func docWithParams(doc *Doc, path string) *Doc {
	params := PathParams(path)
	if len(params) == 0 {
		return doc
	}
	if doc == nil {
		doc = &Doc{}
	}

	properties := et.Json{}
	maps.Copy(properties, schemaJson(doc.Params["properties"]))
	required := slices.Clone(schemaList(doc.Params["required"]))
	for _, param := range params {
		if _, ok := properties[param]; !ok {
			properties[param] = String(fmt.Sprintf(MSG_PATH_PARAM, param))
		}
		if !slices.Contains(required, param) {
			required = append(required, param)
		}
	}

	result := et.Json{}
	maps.Copy(result, doc.Params)
	result["type"] = "object"
	result["properties"] = properties
	result["required"] = required
	doc.Params = result
	return doc
}

// ---------- Schemas (JSON Schema subset) ----------

/**
* schemaOf: A schema of a type with an optional description
* @param tp string, description []string
* @return et.Json
**/
func schemaOf(tp string, description []string) et.Json {
	result := et.Json{"type": tp}
	if text := strings.Join(description, " "); text != "" {
		result["description"] = text
	}
	return result
}

/**
* String: A string schema
* @param description ...string
* @return et.Json
**/
func String(description ...string) et.Json {
	return schemaOf("string", description)
}

/**
* Integer: An integer schema
* @param description ...string
* @return et.Json
**/
func Integer(description ...string) et.Json {
	return schemaOf("integer", description)
}

/**
* Number: A number schema
* @param description ...string
* @return et.Json
**/
func Number(description ...string) et.Json {
	return schemaOf("number", description)
}

/**
* Boolean: A boolean schema
* @param description ...string
* @return et.Json
**/
func Boolean(description ...string) et.Json {
	return schemaOf("boolean", description)
}

/**
* Any: A schema without type: any value
* @param description ...string
* @return et.Json
**/
func Any(description ...string) et.Json {
	result := et.Json{}
	if text := strings.Join(description, " "); text != "" {
		result["description"] = text
	}
	return result
}

/**
* Array: An array schema whose items follow items (nil: any item)
* @param items et.Json, description ...string
* @return et.Json
**/
func Array(items et.Json, description ...string) et.Json {
	result := schemaOf("array", description)
	if items != nil {
		result["items"] = items
	}
	return result
}

/**
* Enum: A string schema limited to values
* @param description string, values ...string
* @return et.Json
**/
func Enum(description string, values ...string) et.Json {
	result := schemaOf("string", []string{description})
	result["enum"] = values
	return result
}

/**
* Object: An object schema with its properties and the required ones
* @param properties et.Json, required ...string
* @return et.Json
**/
func Object(properties et.Json, required ...string) et.Json {
	result := et.Json{"type": "object", "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

/**
* Describe: The schema with its description
* @param schema et.Json, description string
* @return et.Json
**/
func Describe(schema et.Json, description string) et.Json {
	result := et.Json{}
	maps.Copy(result, schema)
	result["description"] = description
	return result
}

// pathParam: a {param} of a route path
var pathParam = regexp.MustCompile(`\{([^}/]+)\}`)

/**
* PathParams: The {params} of a route path, in order
* @param path string
* @return []string
**/
func PathParams(path string) []string {
	result := []string{}
	for _, match := range pathParam.FindAllStringSubmatch(path, -1) {
		name, _, _ := strings.Cut(match[1], ":")
		result = append(result, name)
	}
	return result
}
