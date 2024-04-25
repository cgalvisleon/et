package utility

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

const NOT_FOUND = "Not found"
const FOUND = "Found"
const FOR_DELETE = "-2"
const OF_SYSTEM = "-1"
const ACTIVE = "0"
const ARCHIVED = "1"
const CANCELLED = "2"
const IN_PROCESS = "3"
const PENDING_APPROVAL = "4"
const APPROVAL = "5"
const REFUSED = "6"
const STOP = "Stop"
const CACHE_TIME = 60 * 60 * 24 * 1
const DAY_SECOND = 60 * 60 * 24 * 1
const SELECt = "SELECT"
const INSERT = "INSERT"
const UPDATE = "UPDATE"
const DELETE = "DELETE"
const BEFORE_INSERT = "BEFORE_INSERT"
const AFTER_INSERT = "AFTER_INSERT"
const BEFORE_UPDATE = "BEFORE_UPDATE"
const AFTER_UPDATE = "AFTER_UPDATE"
const BEFORE_STATE = "BEFORE_STATE"
const AFTER_STATE = "AFTER_STATE"
const BEFORE_DELETE = "BEFORE_DELETE"
const AFTER_DELETE = "AFTER_DELETE"
const VALUE_NOT_BOOL = "Value is not bolean"
const ROWS = 30

func Now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetCodeVerify(length int) string {
	const charset = "0123456789"
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

func UUID() string {
	return uuid.NewString()
}

func NewId() string {
	return UUID()
}

func GenId(id string) string {
	if map[string]bool{"": true, "*": true, "new": true}[id] {
		return NewId()
	}

	return id
}

func NilId(id string) string {
	if map[string]bool{"": true, "-1": true, "*": true, "new": true}[id] {
		return uuid.NewString()
	}

	return id
}

func Pointer(collection string, id string) string {
	return strs.Format("%s/%s", collection, id)
}

func Contains(v interface{}, vals ...interface{}) bool {
	for _, i := range vals {
		if i == v {
			return true
		}
	}

	return false
}

func InStr(val string, in []string) bool {
	ok := slices.Contains(in, val)

	return ok
}

func InInt(val string, in []string) bool {
	ok := slices.Contains(in, val)

	return ok
}

func TimeDifference(dateInt, dateEnd any) time.Duration {
	var result time.Time
	layout := "2006-01-02T15:04:05.000Z"

	if dateInt == 0 {
		return result.Sub(result)
	}
	if dateEnd == 0 {
		return result.Sub(result)
	}
	_dateInt, err := time.Parse(layout, fmt.Sprint(dateInt))
	if err != nil {
		return result.Sub(result)
	}

	_dateEnd, err := time.Parse(layout, fmt.Sprint(dateEnd))
	if err != nil {
		return result.Sub(result)
	}

	return _dateInt.Sub(_dateEnd)
}

func GeneratePortNumber() int {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	min := 1000
	max := 99999
	port := rand.Intn(max-min+1) + min

	return port
}

func IsJsonBuild(str string) bool {
	result := strings.Contains(str, "[")
	result = result && strings.Contains(str, "]")
	return result
}

func FindIndex(arr []string, valor string) int {
	for i, v := range arr {
		if v == valor {
			return i
		}
	}
	return -1
}

func OkOrNot(condition bool, ok interface{}, not interface{}) interface{} {
	if condition {
		return ok
	} else {
		return not
	}
}

func ExtractMencion(str string) []string {
	patron := `@([a-zA-Z0-9_]+)`
	expresionRegular := regexp.MustCompile(patron)
	mencions := expresionRegular.FindAllString(str, -1)
	unique := make(map[string]bool)
	result := []string{}

	for _, val := range mencions {
		if !unique[val] {
			unique[val] = true
			result = append(result, val)
		}
	}

	return result
}

func Quote(val interface{}) any {
	switch v := val.(type) {
	case string:
		return strs.Format(`'%s'`, v)
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
		return strs.Format(`'%s'`, v.Format("2006-01-02 15:04:05"))
	case []interface{}:
		var r string
		for _, _v := range v {
			q := Quote(_v).(string)
			if len(r) == 0 {
				r = q
			} else {
				r = strs.Format(`%v, %v`, r, q)
			}
		}
		return strs.Format(`'[%s]'`, r)
	case map[string]interface{}:
		var r string
		for k, _v := range v {
			q := Quote(_v).(string)
			if len(r) == 0 {
				r = strs.Format(`"%v": %v`, k, q)
			} else {
				r = strs.Format(`%v, "%v": %v`, r, k, q)
			}
		}
		return strs.Format(`'%s'`, r)
	case []map[string]interface{}:
		var r string
		for _, _v := range v {
			q := Quote(_v).(string)
			if len(r) == 0 {
				r = q
			} else {
				r = strs.Format(`%v, %v`, r, q)
			}
		}
		return strs.Format(`'[%s]'`, r)
	case nil:
		return "NULL"
	default:
		logs.Errorf("Not quote type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}

func Params(str string, args ...any) string {
	var result string = str
	for i, v := range args {
		p := strs.Format(`$%d`, i+1)
		rp := strs.Format(`%v`, v)
		result = strs.Replace(result, p, rp)
	}

	return result
}

func ParamQuote(str string, args ...any) string {
	for i, arg := range args {
		old := strs.Format(`$%d`, i+1)
		new := strs.Format(`%v`, Quote(arg))
		str = strings.ReplaceAll(str, old, new)
	}

	return str
}

func Address(host string, port int) string {
	return strs.Format("%s:%d", host, port)
}

func BannerTitle(name, version string, size int) string {
	return strs.Format(`{{ .Title "%s V%s" "" %d }}`, name, version, size)
}

func GoMod(atrib string) (string, error) {
	var result string
	rutaArchivoGoMod := "./go.mod"

	contenido, err := os.ReadFile(rutaArchivoGoMod)
	if err != nil {
		return "", err
	}

	lineas := strings.Split(string(contenido), "\n")
	for _, linea := range lineas {
		if strings.HasPrefix(linea, atrib) {
			partes := strings.Fields(linea)
			if len(partes) > 1 {
				result = partes[1]
				break
			}
		}
	}

	return result, nil
}
