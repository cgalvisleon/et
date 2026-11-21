package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
)

/**
* ValidateSchema: The errors of value against a JSON Schema (the subset type, properties, required, enum, items); empty
* if it follows it. name is how the value is called in the messages ("body", "query")
* @param schema et.Json, value any, name string
* @return []string
**/
func ValidateSchema(schema et.Json, value any, name string) []string {
	errs := []string{}
	validateValue(schema, value, name, &errs)
	return errs
}

/**
* validateValue
* @param schema et.Json, value any, name string, errs *[]string
**/
func validateValue(schema et.Json, value any, name string, errs *[]string) {
	if len(schema) == 0 {
		return
	}

	types := schemaTypes(schema)
	if len(types) > 0 && !slices.ContainsFunc(types, func(tp string) bool { return isType(tp, value) }) {
		*errs = append(*errs, fmt.Sprintf(MSG_SCHEMA_TYPE, name, strings.Join(types, " | ")))
		return
	}

	if enum := schemaList(schema["enum"]); len(enum) > 0 {
		text := fmt.Sprint(value)
		if !slices.Contains(enum, text) {
			*errs = append(*errs, fmt.Sprintf(MSG_SCHEMA_ENUM, name, strings.Join(enum, ", ")))
			return
		}
	}

	switch v := value.(type) {
	case map[string]any:
		validateObject(schema, v, name, errs)
	case et.Json:
		validateObject(schema, v, name, errs)
	case []any:
		items := schemaJson(schema["items"])
		for i, item := range v {
			validateValue(items, item, fmt.Sprintf("%s[%d]", name, i), errs)
		}
	}
}

/**
* validateObject: The required properties and each declared property of an object
* @param schema et.Json, value map[string]any, name string, errs *[]string
**/
func validateObject(schema et.Json, value map[string]any, name string, errs *[]string) {
	for _, key := range schemaList(schema["required"]) {
		if item, ok := value[key]; !ok || item == nil {
			*errs = append(*errs, fmt.Sprintf(MSG_SCHEMA_REQUIRED, name+"."+key))
		}
	}

	properties := schemaJson(schema["properties"])
	for _, key := range slices.Sorted(maps.Keys(properties)) {
		item, ok := value[key]
		if !ok || item == nil {
			continue
		}
		validateValue(schemaJson(properties[key]), item, name+"."+key, errs)
	}
}

/**
* isType: If value is of a JSON Schema type
* @param tp string, value any
* @return bool
**/
func isType(tp string, value any) bool {
	switch tp {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	case "object":
		switch value.(type) {
		case map[string]any, et.Json:
			return true
		}
		return false
	case "array":
		_, ok := value.([]any)
		return ok
	case "number":
		_, ok := numberOf(value)
		return ok
	case "integer":
		n, ok := numberOf(value)
		return ok && n == math.Trunc(n)
	}
	return true
}

/**
* numberOf: The value as float64 if it is a number
* @param value any
* @return float64, bool
**/
func numberOf(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	}
	return 0, false
}

/**
* schemaTypes: The type of a schema as a list ("string" or ["string", "null"])
* @param schema et.Json
* @return []string
**/
func schemaTypes(schema et.Json) []string {
	return schemaList(schema["type"])
}

/**
* schemaList: A string or a list of values of a schema as []string
* @param value any
* @return []string
**/
func schemaList(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return []string{v}
	case []string:
		return v
	case []any:
		result := []string{}
		for _, item := range v {
			result = append(result, fmt.Sprint(item))
		}
		return result
	}
	return nil
}

/**
* schemaJson: An object of a schema (properties, items) as et.Json
* @param value any
* @return et.Json
**/
func schemaJson(value any) et.Json {
	switch v := value.(type) {
	case et.Json:
		return v
	case map[string]any:
		return et.Json(v)
	}
	return et.Json{}
}

/**
* queryValues: The query params a schema declares, converted to their type (integer, number, boolean; array with every
* value); a value that does not convert stays as text, so the validation reports it
* @param schema et.Json, r *http.Request
* @return et.Json
**/
func queryValues(schema et.Json, r *http.Request) et.Json {
	result := et.Json{}
	query := r.URL.Query()
	properties := schemaJson(schema["properties"])
	for key, value := range properties {
		values, ok := query[key]
		if !ok || len(values) == 0 {
			continue
		}

		property := schemaJson(value)
		tp := strings.Join(schemaTypes(property), "")
		if tp == "array" {
			items := schemaJson(property["items"])
			list := []any{}
			for _, text := range values {
				list = append(list, queryValue(strings.Join(schemaTypes(items), ""), text))
			}
			result[key] = list
			continue
		}
		result[key] = queryValue(tp, values[0])
	}
	return result
}

/**
* queryValue: A query value converted to its type
* @param tp, text string
* @return any
**/
func queryValue(tp, text string) any {
	switch tp {
	case "integer", "number":
		if n, err := strconv.ParseFloat(text, 64); err == nil {
			return n
		}
	case "boolean":
		if b, err := strconv.ParseBool(text); err == nil {
			return b
		}
	}
	return text
}

/**
* validateRequest: The errors of the query and the body of a request against the schemas of doc; it leaves the body to
* be read again
* @param doc *Doc, r *http.Request
* @return []string, error
**/
func validateRequest(doc *Doc, r *http.Request) ([]string, error) {
	errs := []string{}
	if len(doc.Query) > 0 {
		query := queryValues(doc.Query, r)
		errs = append(errs, ValidateSchema(doc.Query, map[string]any(query), "query")...)
	}

	if len(doc.Body) == 0 {
		return errs, nil
	}

	data := []byte{}
	if r.Body != nil {
		var err error
		data, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()
	}
	r.Body = io.NopCloser(bytes.NewReader(data))

	var body any = map[string]any{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &body); err != nil {
			return nil, err
		}
	}
	errs = append(errs, ValidateSchema(doc.Body, body, "body")...)
	return errs, nil
}

/**
* validateMiddleware: Responds 400 with {message, errors} when the query or the body do not follow the schemas of doc
* @param doc *Doc
* @return func(http.Handler) http.Handler
**/
func validateMiddleware(doc *Doc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs, err := validateRequest(doc, r)
			if err != nil {
				response.HTTPError(w, r, http.StatusBadRequest, fmt.Sprintf(MSG_SCHEMA_INVALID, err.Error()))
				return
			}

			if len(errs) > 0 {
				message := fmt.Sprintf(MSG_SCHEMA_INVALID, strings.Join(errs, "; "))
				response.JSON(w, r, http.StatusBadRequest, et.Json{"message": message, "errors": errs})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
