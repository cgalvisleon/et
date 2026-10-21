package et

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
)

/**
* maxExactFloat: Largest integer a float64 holds without losing precision (2^53).
**/
const maxExactFloat = 1 << 53

/**
* decodeJSON: Decodes data like json.Unmarshal, but numbers keep their precision: they come back as
* float64 (as json.Unmarshal does) unless they are integers a float64 can't hold exactly, which come
* back as int64 (or json.Number when they don't fit an int64 either).
* @param data []byte, dest any
* @return error
**/
func decodeJSON(data []byte, dest any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid character after top-level value")
	}

	switch v := dest.(type) {
	case *any:
		*v = normalizeNumbers(*v)
	case *map[string]any:
		normalizeMap(*v)
	case *Json:
		normalizeMap(*v)
	case *[]any:
		for i, item := range *v {
			(*v)[i] = normalizeNumbers(item)
		}
	case *[]Json:
		for _, item := range *v {
			normalizeMap(item)
		}
	}

	return nil
}

/**
* normalizeMap: Replaces the json.Number values of a decoded object, in place (see numberValue)
* @param m map[string]any
**/
func normalizeMap(m map[string]any) {
	for k, item := range m {
		m[k] = normalizeNumbers(item)
	}
}

/**
* normalizeNumbers: Replaces the json.Number values of a decoded value (see numberValue)
* @param v any
* @return any
**/
func normalizeNumbers(v any) any {
	switch val := v.(type) {
	case json.Number:
		return numberValue(val)
	case map[string]any:
		normalizeMap(val)
		return val
	case Json:
		normalizeMap(val)
		return val
	case []any:
		for i, item := range val {
			val[i] = normalizeNumbers(item)
		}
		return val
	default:
		return v
	}
}

/**
* numberValue: A JSON number as float64, or as int64 when it is an integer beyond 2^53, or as is
* when it doesn't fit an int64 either
* @param n json.Number
* @return any
**/
func numberValue(n json.Number) any {
	text := string(n)
	if !strings.ContainsAny(text, ".eE") {
		i, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return n
		}
		if i > maxExactFloat || i < -maxExactFloat {
			return i
		}
		return float64(i)
	}
	f, err := n.Float64()
	if err != nil {
		return n
	}
	return f
}

/**
* parseJSON: Decodes a JSON text into dest, also when it arrives damaged in common ways: raw line
* breaks or tabs inside its strings, escaped once more (`{\"a\":1}` or `{\n  \"a\": 1\n}`), or
* encoded twice (`"{\"a\":1}"`).
* @param src string, dest any
* @return error
**/
func parseJSON(src string, dest any) error {
	text := strings.TrimSpace(strings.TrimPrefix(src, "\uFEFF"))
	err := decodeJSON([]byte(text), dest)
	if err == nil {
		return nil
	}

	for range 3 {
		next, ok := repairJSON(text)
		if !ok {
			break
		}
		text = next
		if decodeJSON([]byte(text), dest) == nil {
			return nil
		}
	}

	return err
}

/**
* repairJSON: One repair step for parseJSON: unwraps a JSON string literal, unescapes a text escaped
* once more, or escapes the control characters inside string literals
* @param text string
* @return string, bool
**/
func repairJSON(text string) (string, bool) {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		var inner string
		if err := json.Unmarshal([]byte(text), &inner); err == nil {
			return strings.TrimSpace(inner), true
		}
	}

	if escaped := escapeControlChars(text); escaped != text {
		return escaped, true
	}

	if strings.Contains(text, `\"`) || strings.Contains(text, `\n`) {
		var inner string
		if err := json.Unmarshal([]byte(`"`+text+`"`), &inner); err == nil && inner != text {
			return strings.TrimSpace(inner), true
		}
	}

	return text, false
}

/**
* escapeControlChars: Escapes the raw control characters (\n, \r, \t…) found inside string literals,
* which JSON does not allow; whitespace between tokens is left as is
* @param text string
* @return string
**/
func escapeControlChars(text string) string {
	var buf strings.Builder
	inString, escaped, changed := false, false, false
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case escaped:
			escaped = false
		case inString && c == '\\':
			escaped = true
		case c == '"':
			inString = !inString
		case inString && c < 0x20:
			changed = true
			switch c {
			case '\n':
				buf.WriteString(`\n`)
			case '\r':
				buf.WriteString(`\r`)
			case '\t':
				buf.WriteString(`\t`)
			default:
				fmt.Fprintf(&buf, `\u%04x`, c)
			}
			continue
		}
		buf.WriteByte(c)
	}

	if !changed {
		return text
	}
	return buf.String()
}

/**
* formatFloat: A float as text with its own decimals: 1 → "1", 3.10 → "3.1", 0.1234567 → "0.1234567"
* @param f float64, bitSize int
* @return string
**/
func formatFloat(f float64, bitSize int) string {
	return strconv.FormatFloat(f, 'f', -1, bitSize)
}

/**
* stringOf: A value as text; numbers without added or lost decimals
* @param val any
* @return string
**/
func stringOf(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return string(v)
	case float64:
		return formatFloat(v, 64)
	case float32:
		return formatFloat(float64(v), 32)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf(`%d`, v)
	case bool:
		return strconv.FormatBool(v)
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf(`%v`, v)
	}
}

/**
* intOf: A value as int64: any integer type, a float (truncated, as before), a json.Number, or a text
* holding an integer ("42", " 42 ", "42.0")
* @param val any
* @return int64, bool
**/
func intOf(val any) (int64, bool) {
	switch v := val.(type) {
	case json.Number:
		return intOfText(string(v))
	case string:
		return intOfText(v)
	}

	f, kind, ok := numberToFloat64(val)
	if !ok {
		return 0, false
	}
	if i, ok := numberToInt64(val); ok {
		return i, true
	}
	if isUnsignedIntKind(kind) {
		u := reflect.ValueOf(val)
		if u.Kind() == reflect.Pointer {
			u = u.Elem()
		}
		if u.Uint() > math.MaxInt64 {
			return 0, false
		}
		return int64(u.Uint()), true
	}
	if math.IsNaN(f) || math.IsInf(f, 0) || f > math.MaxInt64 || f < math.MinInt64 {
		return 0, false
	}
	return int64(f), true
}

/**
* intOfText: A text holding an integer, also written with a zero fraction ("42.0") or exponent ("1e3")
* @param s string
* @return int64, bool
**/
func intOfText(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f != math.Trunc(f) || f > math.MaxInt64 || f < math.MinInt64 {
		return 0, false
	}
	return int64(f), true
}

/**
* floatOfAny: A value as float64: any number type, a json.Number or a numeric text
* @param val any
* @return float64, bool
**/
func floatOfAny(val any) (float64, bool) {
	switch v := val.(type) {
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	}

	f, _, ok := numberToFloat64(val)
	return f, ok
}

/**
* boolOf: A value as bool: a bool, the numbers 1 and 0, or the texts true/false, t/f, 1/0, yes/no,
* y/n, si/sí/no, on/off (any case, surrounding spaces ignored); anything else is not a bool
* @param val any
* @return bool, bool
**/
func boolOf(val any) (bool, bool) {
	switch v := val.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "t", "1", "yes", "y", "si", "sí", "on":
			return true, true
		case "false", "f", "0", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	}

	f, ok := floatOfAny(val)
	if !ok {
		return false, false
	}
	switch f {
	case 1:
		return true, true
	case 0:
		return false, true
	default:
		return false, false
	}
}
