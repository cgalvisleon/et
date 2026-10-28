package et

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/timezone"
)

var escapeHTMLPool = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

/**
* Json type
**/
type Json map[string]interface{}

/**
* Scan load rows to a json
* @param src interface{}
* @return error
**/
func (s *Json) Scan(src interface{}) error {
	var text string
	switch v := src.(type) {
	case nil:
		// A NULL column.
		*s = Json{}
		return nil
	case []byte:
		text = string(v)
	case string:
		text = v
	case Json:
		*s = v
		return nil
	case map[string]interface{}:
		*s = Json(v)
		return nil
	default:
		return fmt.Errorf(MSG_FAILED_TO_UNMARSHAL_JSON_VALUE, fmt.Sprintf("%v", src))
	}

	t := map[string]interface{}{}
	if err := parseJSON(text, &t); err != nil {
		return err
	}

	*s = Json(t)

	return nil
}

/**
* ScanRows load rows to a json
* @param rows *sql.Rows
* @return error
**/
func (s *Json) ScanRows(rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(cols))
	pointers := make([]interface{}, len(cols))
	for i := range values {
		pointers[i] = &values[i]
	}

	if err := rows.Scan(pointers...); err != nil {
		return err
	}

	byteVal := func(val interface{}) interface{} {
		switch v := val.(type) {
		case []uint8:
			return string(v)
		default:
			return v
		}
	}

	result := make(Json)
	for i, col := range cols {
		src := values[i]
		switch v := src.(type) {
		case nil:
			result[col] = nil
		case []byte:
			var bt interface{}
			err = decodeJSON(v, &bt)
			if err != nil {
				result[col] = byteVal(v)
				continue
			}
			result[col] = bt
		case string:
			// Drivers without a native JSON/bytea type (e.g. SQLite) return JSON
			// column values as plain strings instead of []byte. Only attempt to
			// decode strings that look like a JSON object or array, so an
			// ordinary text value (e.g. "123" or "true") is left untouched.
			trimmed := strings.TrimSpace(v)
			if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
				result[col] = v
				continue
			}
			var bt interface{}
			if err := parseJSON(v, &bt); err != nil {
				result[col] = v
				continue
			}
			result[col] = bt
		default:
			result[col] = src
		}
	}

	*s = result

	return nil
}

/**
* ToByte convert a json to a []byte
* @return []byte
**/
func (s Json) ToByte() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* ToObject convert a json to a object
* @param dest any
* @return error
**/
func (s Json) ToObject(dest any) error {
	bt, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return json.Unmarshal(bt, dest)
}

/**
* ToString convert a json to a string
* @return string
**/
func (s Json) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	return string(bt)
}

/**
* ToEscapeHTML convert a json to a string without escape html
* @return string
**/
func (s Json) ToEscapeHTML() string {
	buf := escapeHTMLPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer escapeHTMLPool.Put(buf)

	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(s)
	if err != nil {
		return ""
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

/**
* ToMap convert a json to a map
* @return map[string]interface{}
**/
func (s Json) ToMap() map[string]interface{} {
	return s
}

/**
* IsEmpty return if the json is empty
* @return bool
**/
func (s Json) IsEmpty() bool {
	return len(s) == 0
}

/**
* IsExist return if the json has the key
* @param key string
* @return bool
**/
func (s Json) IsExist(key string) bool {
	_, ok := s[key]
	return ok
}

/**
* Clone a json: a shallow copy, nested objects and lists are shared with s (DeepClone copies them too)
* @return Json
**/
func (s Json) Clone() Json {
	result := Json{}
	maps.Copy(result, s)
	return result
}

/**
* DeepClone: A copy of s where nested objects (Json, map[string]interface{}) and lists ([]interface{}, []Json)
* are copied as well; other values are kept as they are
* @return Json
**/
func (s Json) DeepClone() Json {
	if s == nil {
		return nil
	}

	result := make(Json, len(s))
	for k, v := range s {
		result[k] = deepCloneValue(v)
	}
	return result
}

/**
* deepCloneValue: Copies a nested object or list for DeepClone
* @param v interface{}
* @return interface{}
**/
func deepCloneValue(v interface{}) interface{} {
	switch val := v.(type) {
	case Json:
		return val.DeepClone()
	case map[string]interface{}:
		return map[string]interface{}(Json(val).DeepClone())
	case []Json:
		result := make([]Json, len(val))
		for i, item := range val {
			result[i] = item.DeepClone()
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = deepCloneValue(item)
		}
		return result
	default:
		return v
	}
}

/**
* ValAny
* @param def interface{}, atribs ...string
* @return any
**/
func (s Json) ValAny(def interface{}, atribs ...string) (result interface{}) {
	result = def
	if len(atribs) == 0 {
		return
	}

	var src interface{} = s
	var ok bool
	for _, atrib := range atribs {
		switch v := src.(type) {
		case Json:
			result, ok = v[atrib]
			if !ok {
				result = def
				return
			}
			src = result
		case map[string]interface{}:
			result, ok = v[atrib]
			if !ok {
				result = def
				return
			}
			src = result
		case interface{ JsonValue() Json }:
			result, ok = v.JsonValue()[atrib]
			if !ok {
				result = def
				return
			}
			src = result
		default:
			result = def
			return
		}
	}

	return
}

/**
* ValStr return string value of the key
* @param def string, atribs ...string
* @return string
**/
func (s Json) ValStr(def string, atribs ...string) string {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		// Numbers keep their own decimals: 1 → "1", 3.10 → "3.1".
		return stringOf(v)
	}
}

/**
* ValInt return int value of the key
* @param def int, atribs ...string
* @return int
**/
func (s Json) ValInt(def int, atribs ...string) int {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	i, ok := intOf(val)
	if !ok || i > math.MaxInt || i < math.MinInt {
		return def
	}

	return int(i)
}

/**
* ValInt64 return int64 value of the key
* @param def int64, atribs ...string
* @return int64
**/
func (s Json) ValInt64(def int64, atribs ...string) int64 {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	i, ok := intOf(val)
	if !ok {
		return def
	}

	return i
}

/**
* ValNum return float64 value of the key
* @param def float64, atribs ...string
* @return float64
**/
func (s Json) ValNum(def float64, atribs ...string) float64 {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	f, ok := floatOfAny(val)
	if !ok {
		return def
	}

	return f
}

/**
* ValBool return bool value of the key
* @param def bool, atribs ...string
* @return bool
**/
func (s Json) ValBool(def bool, atribs ...string) bool {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	// SQLite has no boolean type: EXISTS and 0/1 columns come back as numbers.
	b, ok := boolOf(val)
	if !ok {
		return def
	}

	return b
}

/**
* ValTime return time value of the key
* @param def time.Time, atribs ...string
* @return time.Time
**/
func (s Json) ValTime(def time.Time, atribs ...string) time.Time {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case string:
		layouts := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05"}
		v = strings.TrimSpace(v)
		for _, layout := range layouts {
			if result, err := time.Parse(layout, v); err == nil {
				return result
			}
		}
		if result, ok := parseDate(v); ok {
			return result
		}
		return def
	case time.Time:
		return v
	default:
		return def
	}
}

/**
* ValDuration
* @param _default time.Duration, atribs ...string
* @return string
**/
func (s Json) ValDuration(_default time.Duration, atribs ...string) time.Duration {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case time.Duration:
		if v == 0 {
			return _default
		}
		return v
	case string:
		result, err := time.ParseDuration(v)
		if err != nil {
			return _default
		}
		return result
	default:
		return _default
	}
}

/**
* ValJson
* @param def Json, atribs ...string
* @return Json
**/
func (s Json) ValJson(def Json, atribs ...string) Json {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case Json:
		return v
	case map[string]interface{}:
		return Json(v)
	case interface{ JsonValue() Json }:
		return v.JsonValue()
	case string:
		var result Json
		if err := parseJSON(v, &result); err != nil || result == nil {
			return def
		}

		return result
	default:
		src, err := json.Marshal(v)
		if err != nil {
			return def
		}

		var result Json
		err = decodeJSON(src, &result)
		if err != nil {
			return def
		}

		return result
	}
}

/**
* ValArray
* @param def []interface{}, atribs ...string
* @return []interface{}
**/
func (s Json) ValArray(def []interface{}, atribs ...string) []interface{} {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case []interface{}:
		return v
	case []Json:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []string:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []int:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []int64:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []float32:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []float64:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []bool:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []time.Time:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case string:
		// A list that arrives as JSON text.
		result := []interface{}{}
		if err := parseJSON(v, &result); err != nil {
			return def
		}
		return result
	default:
		result := []interface{}{}
		bt, err := json.Marshal(v)
		if err != nil {
			return def
		}
		if err := decodeJSON(bt, &result); err != nil {
			return def
		}
		return result
	}
}

/**
* Any return any value of the key
* @param def any
* @param atribs ...string
* @return *Any
**/
func (s Json) Any(def interface{}, atribs ...string) interface{} {
	return s.ValAny(def, atribs...)
}

/**
* Str return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) Str(atribs ...string) string {
	return s.ValStr("", atribs...)
}

/**
* String return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) String(atribs ...string) string {
	return s.Str(atribs...)
}

/**
* FromBase64 decode base64 value to string
* @param atribs ...string
* @return string
**/
func (s Json) FromBase64(atribs ...string) string {
	result := s.Str(atribs...)
	bt, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		return result
	}

	// Values encoded before ToBase64 stopped quoting them hold a JSON string.
	var quoted string
	if len(bt) >= 2 && bt[0] == '"' && json.Unmarshal(bt, &quoted) == nil {
		return quoted
	}

	return string(bt)
}

/**
* ToBase64 encode string value to base64
* @param atribs ...string
* @return string
**/
func (s Json) ToBase64(atribs ...string) string {
	return base64.StdEncoding.EncodeToString([]byte(s.Str(atribs...)))
}

/**
* Int return the value of the key
* @param atribs ...string
* @return int
**/
func (s Json) Int(atribs ...string) int {
	return s.ValInt(0, atribs...)
}

/**
* Int64 return the value of the key
* @param atribs ...string
* @return int64
**/
func (s Json) Int64(atribs ...string) int64 {
	return s.ValInt64(0, atribs...)
}

/**
* Num return the value of the key
* @param atribs ...string
* @return float64
**/
func (s Json) Num(atribs ...string) float64 {
	return s.ValNum(0.00, atribs...)
}

/**
* Bool return the value of the key
* @param atribs ...string
* @return bool
**/
func (s Json) Bool(atribs ...string) bool {
	return s.ValBool(false, atribs...)
}

/**
* Byte return the value of the key
* @param atribs ...string
* @return []byte
**/
func (s Json) Byte(atribs ...string) ([]byte, error) {
	value := s.ValAny([]byte{}, atribs...)
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		// A text (e.g. JSON saved as text) as it is, not quoted again.
		return []byte(v), nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

/**
* Time return the value of the key
* @param atribs ...string
* @return time.Time
**/
func (s Json) Time(atribs ...string) time.Time {
	return s.ValTime(timezone.Now(), atribs...)
}

/**
* Json return the value of the key
* @param atrib string
* @return Json
**/
func (s Json) Json(atrib string) Json {
	return s.ValJson(Json{}, atrib)
}

/**
* MapStr return the value of the key as a map[string]string
* @param atrib string
* @return map[string]string
**/
func (s Json) MapStr(atrib string) map[string]string {
	result := map[string]string{}
	jsonValue := s.ValJson(Json{}, atrib)
	for k, v := range jsonValue {
		result[k] = stringOf(v)
	}
	return result
}

/**
* MapInt return the value of the key as a map[string]int
* @param atrib string
* @return map[string]int
**/
func (s Json) MapInt(atrib string) map[string]int {
	result := map[string]int{}
	jsonValue := s.ValJson(Json{}, atrib)
	for k, v := range jsonValue {
		if val, ok := intOf(v); ok {
			result[k] = int(val)
		}
	}
	return result
}

/**
* MapFloat return the value of the key as a map[string]float64
* @param atrib string
* @return map[string]float64
**/
func (s Json) MapFloat(atrib string) map[string]float64 {
	result := map[string]float64{}
	jsonValue := s.ValJson(Json{}, atrib)
	for k, v := range jsonValue {
		if val, ok := floatOfAny(v); ok {
			result[k] = val
		}
	}
	return result
}

/**
* Array return the value of the key
* @param atrib string
* @return []Json
**/
func (s Json) Array(atrib ...string) []interface{} {
	return s.ValArray([]interface{}{}, atrib...)
}

/**
* ArrayBytes
* @param atribs ...string
* @return [][]byte
**/
func (s Json) ArrayBytes(atribs ...string) [][]byte {
	vals := s.Array(atribs...)
	result := make([][]byte, 0, len(vals))
	for _, val := range vals {
		result = append(result, []byte(stringOf(val)))
	}

	return result
}

/**
* ArrayStr
* @return []string
**/
func (s Json) ArrayStr(atribs ...string) []string {
	vals := s.Array(atribs...)
	result := make([]string, 0, len(vals))
	for _, val := range vals {
		result = append(result, stringOf(val))
	}

	return result
}

/**
* ArrayInt
* @param atribs ...string
* @return []int
**/
func (s Json) ArrayInt(atribs ...string) []int {
	vals := s.Array(atribs...)
	result := make([]int, 0, len(vals))
	for _, val := range vals {
		if v, ok := intOf(val); ok {
			result = append(result, int(v))
		}
	}

	return result
}

/**
* ArrayInt64
* @param atribs ...string
* @return []int64
**/
func (s Json) ArrayInt64(atribs ...string) []int64 {
	vals := s.Array(atribs...)
	result := make([]int64, 0, len(vals))
	for _, val := range vals {
		if v, ok := intOf(val); ok {
			result = append(result, v)
		}
	}

	return result
}

/**
* ArrayNumber
* @param atribs ...string
* @return []float64
**/
func (s Json) ArrayNumber(atribs ...string) []float64 {
	vals := s.Array(atribs...)
	result := make([]float64, 0, len(vals))
	for _, val := range vals {
		if v, ok := floatOfAny(val); ok {
			result = append(result, v)
		}
	}

	return result
}

/**
* ArrayJson
* @param atribs ...string
* @return []Json
**/
func (s Json) ArrayJson(atribs ...string) []Json {
	var result = []Json{}
	vals := s.Array(atribs...)
	for _, val := range vals {
		switch v := val.(type) {
		case Json:
			result = append(result, v)
		case map[string]interface{}:
			result = append(result, v)
		case interface{ JsonValue() Json }:
			result = append(result, v.JsonValue())
		case string:
			var item Json
			if err := parseJSON(v, &item); err == nil && item != nil {
				result = append(result, item)
				continue
			}
			var items []Json
			if err := parseJSON(v, &items); err == nil {
				result = append(result, items...)
			}
		default:
			// Not an object: skipped, the rest of the list still counts.
			continue
		}
	}

	return result
}

/**
* TrimString trim the string value of the key
* @param atribs ...string
* @return string
**/
func (s Json) TrimString(atribs ...string) string {
	return strings.TrimSpace(s.Str(atribs...))
}

/**
* LowerString convert the trimmed string value of the key to lowercase
* @param atribs ...string
* @return string
**/
func (s Json) LowerString(atribs ...string) string {
	return strings.ToLower(s.TrimString(atribs...))
}

/**
* UpperString convert the trimmed string value of the key to uppercase
* @param atribs ...string
* @return string
**/
func (s Json) UpperString(atribs ...string) string {
	return strings.ToUpper(s.TrimString(atribs...))
}

/**
* Update: This method update s with values in from. If the key exist in s, the value is replaced with the value in from.
* @param fromJson Json
* @return error
**/
func (s Json) Update(from Json) {
	maps.Copy(s, from)
}

/**
* deepEqualValue reports whether a and b represent the same JSON value.
* It compares via their JSON encoding so that nested maps/slices never panic
* on uncomparable types and named vs. unnamed map types (e.g. Json vs.
* map[string]interface{}) don't cause spurious mismatches.
* @param a, b interface{}
* @return bool
**/
func deepEqualValue(a, b interface{}) bool {
	if equal, ok := equalScalar(a, b); ok {
		return equal
	}

	ab, errA := json.Marshal(a)
	bb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return reflect.DeepEqual(a, b)
	}
	return bytes.Equal(ab, bb)
}

/**
* equalScalar: Compares scalars without encoding them; ok is false for objects, lists and times.
* @param a, b interface{}
* @return equal bool, ok bool
**/
func equalScalar(a, b interface{}) (equal bool, ok bool) {
	if a == nil || b == nil {
		return a == nil && b == nil, true
	}

	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		return ok && va == vb, true
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb, true
	}

	if _, isNum := numberOf(a); isNum {
		if _, isNum := numberOf(b); !isNum {
			return false, true
		}
		c, ok := compareNumbers(a, b)
		return ok && c == 0, true
	}

	return false, false
}

/**
* Compare: This method return a new json with the diferent values between s and from. Also include the keys that not exist in s.
* @param from Json
* @return bool
**/
func (s Json) Compare(from Json) Json {
	result := Json{}
	for key, value := range from {
		if old, ok := s[key]; !ok || !deepEqualValue(old, value) {
			result[key] = value
		}
	}

	return result
}

/**
* Append: This method append the values in from to s. If the key exist in s, the value is not replaced.
* @param from Json
**/
func (s Json) Append(from Json) Json {
	for key, value := range from {
		if _, ok := s[key]; !ok {
			s[key] = value
		}
	}

	return s
}

/**
* IsChanged: This method return true if the values in s are different to the values in from.
* @param from Json
* @return bool
**/
func (s Json) IsChanged(from Json) bool {
	for key, vb := range from {
		va, ok := s[key]
		if !ok || !deepEqualValue(va, vb) {
			return true
		}
	}

	return false
}

/**
* IsDeferent return true if the attribute does not exist or its value differs from val
* @param atrib string
* @param val interface{}
* @return bool
**/
func (s Json) IsDeferent(atrib string, val interface{}) bool {
	current, ok := s[atrib]
	if !ok {
		return true
	}
	return !deepEqualValue(current, val)
}

/**
* Clear: This method clear the values of s.
* @return Json
**/
func (s Json) Clear() Json {
	clear(s)
	return s
}

/**
* Put: Replaces the content of s with val, in the same map; into itself or a nil Json it does nothing
* @param val Json
* @return Json
**/
func (s Json) Put(val Json) Json {
	if s == nil || sameMap(s, val) {
		return s
	}

	s.Clear()
	maps.Copy(s, val)
	return s
}

/**
* Get
* @param keys ...string
* @return interface{}
**/
func (s Json) Get(keys ...string) (result interface{}) {
	if len(keys) == 0 {
		return nil
	}

	var val interface{} = s
	var ok bool
	for _, key := range keys {
		switch v := val.(type) {
		case Json:
			val, ok = v[key]
			if !ok {
				result = nil
				return
			}
			result = val
		case map[string]interface{}:
			val, ok = v[key]
			if !ok {
				result = nil
				return
			}
			result = val
		case interface{ JsonValue() Json }:
			val, ok = v.JsonValue()[key]
			if !ok {
				result = nil
				return
			}
			result = val
		default:
			result = nil
		}
	}

	return
}

/**
* SetNested: Sets value at the path keys, creating the objects on the way; with no keys it does nothing
* @param keys []string, value interface{}
**/
func (j Json) SetNested(keys []string, value interface{}) {
	if len(keys) == 0 {
		return
	}

	m := map[string]interface{}(j)

	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]

		// si no existe lo creamos
		if _, ok := m[k]; !ok {
			m[k] = map[string]interface{}{}
		}

		// aseguramos que sea map
		next, ok := asJson(m[k])
		if !ok {
			next = map[string]interface{}{}
			m[k] = next
		}

		m = next
	}

	m[keys[len(keys)-1]] = value
}

/**
* Set a value in the key ("a->b" sets a nested one); a nil Json cannot be written.
* @param key string
* @param val interface{}
**/
func (s Json) Set(key string, val interface{}) {
	if key == "" {
		return
	}

	keys := strings.Split(key, "->")
	if len(keys) == 1 {
		s[key] = val
		return
	}

	s.SetNested(keys, val)
}

/**
* Delete a value in the key
* @param key string
* @return bool
**/
func (s Json) Delete(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	result := false
	for _, key := range keys {
		if _, ok := s[key]; ok {
			delete(s, key)
			result = true
		}
	}

	return result
}

/**
* Exist return if the key exist
* @param key string
* @return bool
**/
func (s Json) Exist(key string) bool {
	_, ok := s[key]
	return ok
}

/**
* Remove
* @param keys ...string
**/
func (s Json) Remove(keys ...string) {
	for _, key := range keys {
		delete(s, key)
	}
}

/**
* Select
* @param keys []string
* @return Json
**/
func (s Json) Select(keys []string) Json {
	result := Json{}
	for _, key := range keys {
		val, ok := s[key]
		if ok {
			result[key] = val
		}
	}

	return result
}

/**
* Hidden
* @param keys []string
* @return Json
**/
func (s Json) Hidden(keys []string) Json {
	hidden := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		hidden[k] = struct{}{}
	}

	result := Json{}
	for key, value := range s {
		if _, ok := hidden[key]; !ok {
			result[key] = value
		}
	}

	return result
}

/**
* From
* @param as string
* @return *Where
**/
func (s Json) From(as string) *Wheres {
	return From([]Json{s}, as)
}

/**
* Query
* @param condition *Condition
* @return bool
**/
func (s Json) Query(condition *Condition) bool {
	return condition.ApplyToObject(s)
}

/**
* GetItem
* @param items []Json, index int
* @return Json
**/
func (s Json) GetItem(items []Json, index int) (Json, bool) {
	if index < 0 || index >= len(items) {
		return Json{}, false
	}

	return items[index], true
}

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
	err := decodeFresh([]byte(text), dest)
	if err == nil {
		return nil
	}

	for range 3 {
		next, ok := repairJSON(text)
		if !ok {
			break
		}
		text = next
		if decodeFresh([]byte(text), dest) == nil {
			return nil
		}
	}

	return err
}

/**
* decodeFresh: decodeJSON into a new value; dest only changes when it decodes
* @param data []byte, dest any
* @return error
**/
func decodeFresh(data []byte, dest any) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return decodeJSON(data, dest)
	}

	fresh := reflect.New(rv.Elem().Type())
	if err := decodeJSON(data, fresh.Interface()); err != nil {
		return err
	}
	rv.Elem().Set(fresh.Elem())
	return nil
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

	n, ok := numberOf(val)
	if !ok {
		return 0, false
	}
	switch n.kind {
	case numSigned:
		return n.i, true
	case numUnsigned:
		if n.u > math.MaxInt64 {
			return 0, false
		}
		return int64(n.u), true
	}
	f := n.f
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

	n, ok := numberOf(val)
	return n.float(), ok
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

/**
* Serialize convert a interface to a []byte
* @param src any
* @return []byte, error
**/
func Serialize(src any) ([]byte, error) {
	result, err := json.Marshal(src)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson convert a string to a json
* @param src string
* @return Json, error
**/
func ToJson(src string) (Json, error) {
	var result map[string]interface{}
	if err := parseJSON(src, &result); err != nil {
		return result, err
	}

	return result, nil
}

/**
* ToString
* @param vals []Json
* @return string
**/
func ToString(vals interface{}) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(vals); err != nil {
		return ""
	}

	return strings.TrimSuffix(buf.String(), "\n")
}
