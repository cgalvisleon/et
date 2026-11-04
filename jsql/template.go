package jsql

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* templatePart: A piece of a TEMPLATE: a fixed text or a field ({{path|format:arg|…}}) with its formats.
**/
type templatePart struct {
	text    string
	path    []string // the field and, in an object or a list, the keys inside it (clase.name)
	formats []templateFormat
}

/**
* templateFormat: A format of a field of a TEMPLATE: its name and its argument (without quotes).
**/
type templateFormat struct {
	name string
	arg  string
}

/**
* queryTemplate: A TEMPLATE column a query returns, already parsed.
**/
type queryTemplate struct {
	name  string
	parts []templatePart
}

// templateVar: a {{…}} of a template
var templateVar = regexp.MustCompile(`\{\{(.*?)\}\}`)

// Formats: what each format of a template does with the value of its field (the one before it, in a chain). The
// argument comes without quotes; "" when it has none.
var formats = map[string]func(value any, arg string) any{
	"date":     func(value any, arg string) any { return formatTime(value, arg, "2006-01-02") },
	"datetime": func(value any, arg string) any { return formatTime(value, arg, "2006-01-02 15:04") },
	"number":   func(value any, arg string) any { return formatNumber(value, arg, "") },
	"currency": func(value any, arg string) any { return formatNumber(value, arg, "$ ") },
	"upper":    func(value any, arg string) any { return strings.ToUpper(textOf(value)) },
	"lower":    func(value any, arg string) any { return strings.ToLower(textOf(value)) },
	"default": func(value any, arg string) any {
		if textOf(value) == "" {
			return arg
		}
		return value
	},
}

/**
* parseTemplate: Splits a template into fixed texts and fields with their formats.
* @param template string
* @return []templatePart, error
**/
func parseTemplate(template string) ([]templatePart, error) {
	result := []templatePart{}
	last := 0
	for _, match := range templateVar.FindAllStringSubmatchIndex(template, -1) {
		if match[0] > last {
			result = append(result, templatePart{text: template[last:match[0]]})
		}
		last = match[1]

		pieces := splitFormats(template[match[2]:match[3]])
		field := strings.TrimSpace(pieces[0])
		if field == "" {
			return nil, fmt.Errorf("empty field in %s", template[match[0]:match[1]])
		}
		part := templatePart{path: strings.Split(field, ".")}
		for _, piece := range pieces[1:] {
			name, arg, _ := strings.Cut(piece, ":")
			arg = strings.TrimSpace(arg)
			if unquoted, err := strconv.Unquote(arg); err == nil {
				arg = unquoted
			}
			part.formats = append(part.formats, templateFormat{name: strings.TrimSpace(name), arg: arg})
		}
		result = append(result, part)
	}
	if last < len(template) {
		result = append(result, templatePart{text: template[last:]})
	}

	return result, nil
}

/**
* splitFormats: Splits "field|format:arg|…" by the | that are outside quotes.
* @param text string
* @return []string
**/
func splitFormats(text string) []string {
	result := []string{}
	quoted := false
	start := 0
	for i, char := range text {
		switch {
		case char == '"' && (i == 0 || text[i-1] != '\\'):
			quoted = !quoted
		case char == '|' && !quoted:
			result = append(result, text[start:i])
			start = i + 1
		}
	}
	return append(result, text[start:])
}

/**
* templateOk: The fields of a template are fields of the model (a column, an attribute, a relation) and its formats
* exist; a template cannot use another template
* @param name, template string
* @return error
**/
func (s *Model) templateOk(name, template string) error {
	parts, err := parseTemplate(template)
	if err != nil {
		return fmt.Errorf(MSG_TEMPLATE_INVALID, name, err.Error())
	}

	for _, part := range parts {
		if part.path == nil {
			continue
		}
		field := part.path[0]
		column, ok := s.getColumn(field)
		if !ok || field == name || column.TypeColumn == TEMPLATE {
			return fmt.Errorf(MSG_TEMPLATE_FIELD_NOT_FOUND, name, field)
		}
		for _, format := range part.formats {
			if _, ok := formats[format.name]; !ok {
				return fmt.Errorf(MSG_TEMPLATE_FORMAT_UNKNOWN, name, format.name)
			}
		}
	}

	return nil
}

/**
* renderTemplate: The text of a template for a row
* @param parts []templatePart, item et.Json
* @return string
**/
func renderTemplate(parts []templatePart, item et.Json) string {
	var result strings.Builder
	for _, part := range parts {
		if part.path == nil {
			result.WriteString(part.text)
			continue
		}
		value := valueAt(item, part.path)
		for _, format := range part.formats {
			value = formats[format.name](value, format.arg)
		}
		result.WriteString(textOf(value))
	}
	return result.String()
}

/**
* valueAt: The value of a path in a row: a key of an object or, in a list (a master or a detail), of its first item
* @param item et.Json, path []string
* @return any
**/
func valueAt(item et.Json, path []string) any {
	var value any = item
	for _, key := range path {
		switch current := value.(type) {
		case []et.Json:
			if len(current) == 0 {
				return nil
			}
			value = current[0][key]
		case []any:
			if len(current) == 0 {
				return nil
			}
			value = mapOf(current[0])[key]
		default:
			value = mapOf(current)[key]
		}
	}
	return value
}

/**
* mapOf: A value as an object (nil when it is not one)
* @param value any
* @return map[string]any
**/
func mapOf(value any) map[string]any {
	switch current := value.(type) {
	case et.Json:
		return current
	case map[string]any:
		return current
	}
	return nil
}

/**
* textOf: A value as the text a template shows ("" for nil)
* @param value any
* @return string
**/
func textOf(value any) string {
	switch current := value.(type) {
	case nil:
		return ""
	case string:
		return current
	case float64:
		return strconv.FormatFloat(current, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(current), 'f', -1, 32)
	case time.Time:
		return current.Format("2006-01-02 15:04:05")
	case et.Json, map[string]any, []et.Json, []any:
		text, err := JsonString(current)
		if err != nil {
			return ""
		}
		return text
	}
	return fmt.Sprintf("%v", value)
}

// timeLayouts: how a date can come as text from a driver
var timeLayouts = []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02"}

/**
* formatTime: A date with the Go layout of arg (layout when arg is ""); a value that is not a date stays as it is
* @param value any, arg, layout string
* @return any
**/
func formatTime(value any, arg, layout string) any {
	if arg != "" {
		layout = arg
	}
	switch current := value.(type) {
	case time.Time:
		return current.Format(layout)
	case string:
		for _, from := range timeLayouts {
			if date, err := time.Parse(from, current); err == nil {
				return date.Format(layout)
			}
		}
	}
	return value
}

/**
* formatNumber: A number with arg decimals (0 when arg is ""), "." between thousands and "," before the decimals, after
* prefix; a value that is not a number stays as it is
* @param value any, arg, prefix string
* @return any
**/
func formatNumber(value any, arg, prefix string) any {
	number, ok := numberOf(value)
	if !ok {
		return value
	}
	decimals, _ := strconv.Atoi(arg)
	decimals = max(decimals, 0)

	sign := ""
	if number < 0 {
		sign = "-"
		number = -number
	}
	// Half away from zero (1234.5 → 1235), not to even as FormatFloat does
	scale := math.Pow(10, float64(decimals))
	text := strconv.FormatFloat(math.Round(number*scale)/scale, 'f', decimals, 64)
	whole, fraction, _ := strings.Cut(text, ".")
	groups := []string{}
	for len(whole) > 3 {
		groups = append(groups, whole[len(whole)-3:])
		whole = whole[:len(whole)-3]
	}
	groups = append(groups, whole)
	slices.Reverse(groups)
	result := sign + prefix + strings.Join(groups, ".")
	if fraction != "" {
		result += "," + fraction
	}
	return result
}

/**
* numberOf: A value as a number, also a numeric text
* @param value any
* @return float64, bool
**/
func numberOf(value any) (float64, bool) {
	switch current := value.(type) {
	case float64:
		return current, !math.IsNaN(current)
	case float32:
		return float64(current), true
	case int:
		return float64(current), true
	case int32:
		return float64(current), true
	case int64:
		return float64(current), true
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(current), 64)
		return number, err == nil
	}
	return 0, false
}

/**
* prepareTemplates: The TEMPLATE columns the query returns (all of the model without selects or with "*", or the ones
* in selects) and the fields they use that the query does not bring: those are added to selects (a relation is resolved
* by its driver as when it is selected) and removed from the rows after building the texts. A where or an order by on a
* template is an error. Only for a query of one model (no joins).
* @return error
**/
func (s *Query) prepareTemplates() error {
	// A query run again starts from what it was asked: without the fields added the last time
	s.Selects = slices.DeleteFunc(s.Selects, func(field string) bool { return slices.Contains(s.templateKeys, field) })
	if s.templateStar {
		s.Selects = slices.DeleteFunc(s.Selects, func(field string) bool { return field == STAR })
	}
	s.templates = nil
	s.templateKeys = nil
	s.templateStar = false
	if len(s.Froms) != 1 || s.Froms[0].Model == nil {
		return nil
	}

	model := s.Froms[0].Model
	columns := slices.DeleteFunc(slices.Clone(model.Columns), func(column *Column) bool { return column.TypeColumn != TEMPLATE })
	if len(columns) == 0 {
		return nil
	}

	isTemplate := func(field string) bool {
		def, ok := et.ToField(field)
		return ok && slices.ContainsFunc(columns, func(column *Column) bool { return column.Name == def.Name })
	}
	for _, cond := range s.Conditions {
		if field, ok := cond.Field.Value.(string); ok && isTemplate(field) {
			return fmt.Errorf(MSG_TEMPLATE_NOT_FILTERABLE, field)
		}
	}
	for _, order := range s.OrdersBy {
		if isTemplate(order.Name) {
			return fmt.Errorf(MSG_TEMPLATE_NOT_FILTERABLE, order.Name)
		}
	}

	all := len(s.Selects) == 0 || s.HasStar()
	for _, column := range columns {
		if !all && !slices.Contains(s.Selects, column.Name) {
			continue
		}
		parts, err := parseTemplate(column.Template)
		if err != nil {
			return fmt.Errorf(MSG_TEMPLATE_INVALID, column.Name, err.Error())
		}
		s.templates = append(s.templates, &queryTemplate{name: column.Name, parts: parts})

		for _, part := range parts {
			if part.path == nil {
				continue
			}
			field := part.path[0]
			used, ok := model.getColumn(field)
			if !ok || slices.Contains(s.Selects, field) || slices.Contains(s.templateKeys, field) {
				continue
			}
			// Without selects the columns and attributes come anyway; a relation has to be asked for
			if all && (used.TypeColumn == COLUMN || used.TypeColumn == ATTRIB) {
				continue
			}
			if len(s.Selects) == 0 {
				s.Selects = append(s.Selects, STAR)
				s.templateStar = true
			}
			s.Selects = append(s.Selects, field)
			s.templateKeys = append(s.templateKeys, field)
		}
	}

	return nil
}

/**
* setTemplates: Puts the text of each template of the query in the row and removes the fields added only for them.
* @param item et.Json
**/
func (s *Query) setTemplates(item et.Json) {
	for _, template := range s.templates {
		item[template.name] = renderTemplate(template.parts, item)
	}
	for _, key := range s.templateKeys {
		delete(item, key)
	}
}
