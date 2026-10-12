package jsql

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
)

/**
* sqlParse: Replaces $N positional placeholders in sql with their quoted argument values.
* @param sql string
* @param args ...any
* @return string
**/
func sqlParse(sql string, args ...any) string {
	if len(args) == 0 {
		return sql
	}
	// One pass over whole placeholders: $10 is never read as $1 followed by 0, and a "$2" inside
	// an already substituted value is not replaced again.
	return placeholder.ReplaceAllStringFunc(sql, func(match string) string {
		n, err := strconv.Atoi(match[1:])
		if err != nil || n < 1 || n > len(args) {
			return match
		}
		return fmt.Sprintf(`%v`, Quoted(args[n-1]))
	})
}

var placeholder = regexp.MustCompile(`\$\d+`)

/**
* quoted: Returns val formatted as a SQL literal (quoted string, bare number, NULL, etc.).
* @param val any
* @return any
**/
func quoted(val any) any {
	format := `'%v'`
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(format, EscapeSQLString(v))
	case int:
		return v
	case float64:
		return v
	case float32:
		return v
	case int16:
		return v
	case int32:
		return v
	case int64:
		return v
	case bool:
		return v
	case time.Time:
		return fmt.Sprintf(format, v.Format("2006-01-02 15:04:05"))
	case et.Json, map[string]interface{}:
		str, err := JsonString(v)
		if err != nil {
			logs.Errorf("Quote, type:%v, value:%v, error marshalling json: %v", reflect.TypeOf(v), v, err)
			return fmt.Sprintf(format, `{}`)
		}
		return fmt.Sprintf(format, EscapeSQLString(str))
	case []string, []et.Json, []interface{}, []map[string]interface{}:
		str, err := JsonString(v)
		if err != nil {
			logs.Errorf("Quote, type:%v, value:%v, error marshalling array: %v", reflect.TypeOf(v), v, err)
			return strs.Format(format, `[]`)
		}
		return fmt.Sprintf(format, EscapeSQLString(str))
	case []uint8:
		b := []byte(val.([]uint8))
		return fmt.Sprintf("'\\x%s'", hex.EncodeToString(b))
	case nil:
		return fmt.Sprintf(`%s`, "NULL")
	default:
		logs.Errorf("Quote, type:%v, value:%v", reflect.TypeOf(v), v)
		return val
	}
}

/**
* escapeSQLString: Escapes single quotes in s by doubling them (standard SQL
* string-literal escaping), so a value can be safely embedded between the
* surrounding '...' produced by Quoted.
* @param s string
* @return string
**/
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

/**
* jsonString: Serializes val as compact JSON without HTML escaping, so characters
* such as <, > and & are stored as-is instead of \u003c, \u003e and \u0026.
* The result still needs EscapeSQLString before being embedded in a SQL literal.
* @param val any
* @return string, error
**/
func jsonString(val any) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(val); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

/**
* rowsToItems: Scans all rows from a *sql.Rows result set into an et.Items collection.
* @param rows *sql.Rows
* @return et.Items
**/
func rowsToItems(rows *sql.Rows) et.Items {
	defer rows.Close()

	result := et.Items{Result: []et.Json{}}
	append := func(item et.Json) {
		result.Add(item)
	}

	// Every result set is read: MySQL and SQL Server run a command as a batch (DML followed by the
	// SELECT that stands in for RETURNING), whose rows come in a later result set.
	for {
		for rows.Next() {
			var item et.Json
			item.ScanRows(rows)

			if len(item) == 1 {
				for _, v := range item {
					switch val := v.(type) {
					case et.Json:
						append(val)
					case map[string]interface{}:
						append(et.Json(val))
					default:
						append(item)
					}
				}
			} else {
				append(item)
			}
		}
		if !rows.NextResultSet() {
			break
		}
	}

	return result
}

/**
* argWhitAs: Returns an array with the argument and its alias.
* @param arg string
* @return []string, bool
**/
func argWhitAs(arg string) ([]string, bool) {
	pattern := regexp.MustCompile(`^([A-Za-z0-9_.>-]+):([A-Za-z0-9_]+)$`) // field:as, or schema.field:as
	ok := pattern.MatchString(arg)
	if ok {
		matches := pattern.FindStringSubmatch(arg)
		if len(matches) == 3 {
			return []string{matches[1], matches[2]}, true
		}
	}
	return []string{arg}, false
}

/**
* argWhitSchema: Returns an array with the argument and its schema.
* @param arg string
* @return []string, bool
**/
func argWhitSchema(arg string) ([]string, bool) {
	pattern := regexp.MustCompile(`^([A-Za-z0-9_>-]+)\.([A-Za-z0-9_]+)$`) // schema.table
	ok := pattern.MatchString(arg)
	if ok {
		matches := pattern.FindStringSubmatch(arg)
		if len(matches) == 3 {
			return []string{matches[1], matches[2]}, true
		}
	}
	return []string{arg}, false
}

/**
* addAuditLog
* @param userId string, action string
**/
func addAuditLog(auditLog []et.Json, userId string, action string) []et.Json {
	if auditLog == nil {
		auditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	auditLog = append(auditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(auditLog) > maxAuditLog {
		auditLog = auditLog[len(auditLog)-maxAuditLog:]
	}
	return auditLog
}

var variable = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z0-9_]+)*)\s*\}\}`)

/**
* Bind: Applies vars to the {{name}} variables of a json (a jsql query, a command…): a string that is only "{{name}}"
* takes the value as is (number, list, object); inside a longer text it is replaced by its text. In a list, a variable
* whose value is a list is expanded. A name with dots is a path in vars ({{user.id}}). Unknown variables are left as
* they are. Values stay values (jsql sends them as parameters): do not bind raw SQL text with it
* @param source, vars et.Json
* @return et.Json
**/
func Bind(source, vars et.Json) et.Json {
	result, ok := bindValue(source, vars).(et.Json)
	if !ok {
		return et.Json{}
	}
	return result
}

/**
* bindValue: Bind over any value of a json (object, list, text or other)
* @param value any, vars et.Json
* @return any
**/
func bindValue(value any, vars et.Json) any {
	switch v := value.(type) {
	case et.Json:
		result := et.Json{}
		for key, item := range v {
			result[key] = bindValue(item, vars)
		}
		return result
	case map[string]any:
		return bindValue(et.Json(v), vars)
	case []et.Json:
		result := make([]any, 0, len(v))
		for _, item := range v {
			result = append(result, bindValue(item, vars))
		}
		return result
	case []any:
		result := make([]any, 0, len(v))
		for _, item := range v {
			bound := bindValue(item, vars)
			list, isList := bound.([]any)
			if isList && isVariable(item) {
				result = append(result, list...)
				continue
			}
			result = append(result, bound)
		}
		return result
	case string:
		return bindText(v, vars)
	default:
		return v
	}
}

/**
* bindText: A text with variables: only "{{name}}" gives the value as is; otherwise each one is replaced by its text
* @param text string, vars et.Json
* @return any
**/
func bindText(text string, vars et.Json) any {
	if isVariable(text) {
		name := variable.FindStringSubmatch(text)[1]
		val, ok := varOf(vars, name)
		if !ok {
			return text
		}
		return val
	}

	return variable.ReplaceAllStringFunc(text, func(match string) string {
		name := variable.FindStringSubmatch(match)[1]
		val, ok := varOf(vars, name)
		if !ok {
			return match
		}
		return fmt.Sprintf("%v", val)
	})
}

/**
* varOf: The value of a variable; a name with dots is a path (user.id)
* @param vars et.Json, name string
* @return any, bool
**/
func varOf(vars et.Json, name string) (any, bool) {
	missing := &struct{}{}
	path := strings.Split(name, ".")
	val := vars.ValAny(missing, path...)
	if val == missing {
		return nil, false
	}
	return val, true
}

/**
* isVariable: Whether a value is only one variable ("{{name}}")
* @param value any
* @return bool
**/
func isVariable(value any) bool {
	text, ok := value.(string)
	return ok && text != "" && variable.FindString(text) == text
}
