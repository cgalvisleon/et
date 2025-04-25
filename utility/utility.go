package utility

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"golang.org/x/exp/slices"
)

const NOT_FOUND = "Not found"
const FOUND = "Found"
const SYSID = "-1"
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
const QUEUE_STACK = "stack"

var locks = make(map[string]*sync.RWMutex)
var count = make(map[string]int64)
var LIST_STATES = []string{ACTIVE, ARCHIVED, CANCELLED, IN_PROCESS, PENDING_APPROVAL, APPROVAL, REFUSED}

/**
* SetStatus
* @param status string
* @return bool
**/
func SetStatus(status string) bool {
	LIST_STATES = append(LIST_STATES, status)

	return slices.Contains(LIST_STATES, status)
}

/**
* DeleteStatus
* @param status string
* @return bool
**/
func DeleteStatus(status string) bool {
	LIST_STATES = slices.Delete(LIST_STATES, slices.Index(LIST_STATES, status), 1)

	return !slices.Contains(LIST_STATES, status)
}

/**
* NowTime
* @return time.Time
**/
func NowTime() time.Time {
	return timezone.NowTime()
}

/**
* Now return the current date
* @return string
**/
func Now() string {
	return timezone.Now()
}

/**
* More return the next value of a serie
* @param tag string
* @return int
**/
func More(tag string, expiration time.Duration) int64 {
	lock := locks[tag]
	if lock == nil {
		lock = &sync.RWMutex{}
		locks[tag] = lock
	}

	lock.Lock()
	defer lock.Unlock()

	n, ok := count[tag]
	if !ok {
		n = 0
	} else {
		n++
	}
	count[tag] = 0

	clean := func() {
		delete(count, tag)
		delete(locks, tag)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return n
}

/**
* UUIndex return the next value of a serie
* @param tag string
* @return int64
**/
func UUIndex(tag string) int64 {
	now := timezone.NowTime()
	result := now.UnixMilli() * 10000
	key := fmt.Sprintf("%s:%d", tag, result)
	n := More(key, 1*time.Second)
	result = result + int64(n)

	return result
}

/**
* Contains return true if the value is in the list
* @param v interface{}
* @param vals ...any
* @return bool
**/
func Contains(v interface{}, vals ...any) bool {
	return slices.Contains(vals, v)
}

/**
* InStr return true if the value is in the list
* @param val string
* @param in []string
* @return bool
**/
func InStr(val string, in []string) bool {
	return slices.Contains(in, val)
}

/**
* InInt return true if the value is in the list
* @param val string
* @param in []string
* @return bool
**/
func InInt(val string, in []string) bool {
	ok := slices.Contains(in, val)

	return ok
}

/**
* TimeDifference return the difference between two dates
* @param dateInt any
* @param dateEnd any
* @return time.Duration
**/
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

/**
* GeneratePortNumber return a random port number
* @return int
**/
func GeneratePortNumber() int {
	rand.New(rand.NewSource(timezone.NowTime().UnixNano()))
	min := 1000
	max := 99999
	port := rand.Intn(max-min+1) + min

	return port
}

/**
* ExtractMencion return the mentions in a string
* @param str string
* @return []string
**/
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

var quotedChar = `'`

/**
* SetQuotedChar
* @param char string
**/
func SetQuotedChar(char string) {
	quotedChar = strs.Format(`%s`, char)
}

/**
* unquote
* @param str string
* @return string
**/
func unquote(str string) string {
	str = strings.ReplaceAll(str, `'`, `"`)
	result, err := strconv.Unquote(str)
	if err != nil {
		result = str
	}

	return result
}

/**
* quote
* @param str string
* @return string
**/
func quote(str string) string {
	result := strconv.Quote(str)
	if quotedChar == `"` {
		return result
	}

	return strings.ReplaceAll(result, `"`, `'`)
}

/**
* Unquote
* @param val interface{}
* @return any
**/
func Unquote(val interface{}) any {
	switch v := val.(type) {
	case string:
		return unquote(v)
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
	case et.Json:
		return strs.Format(`%s`, v.ToString())
	case map[string]interface{}:
		return strs.Format(`%s`, et.Json(v).ToString())
	case time.Time:
		return strs.Format(`%s`, v.Format("2006-01-02 15:04:05"))
	case []string:
		var r string
		for i, _v := range v {
			if i == 0 {
				r = strs.Format(`%s`, unquote(_v))
			} else {
				r = strs.Format(`%s, %s`, r, unquote(_v))
			}
		}
		return strs.Format(`[%s]`, unquote(r))
	case []interface{}:
		var r string
		for i, _v := range v {
			q := Unquote(_v)
			if i == 0 {
				r = strs.Format(`%v`, q)
			} else {
				r = strs.Format(`%s, %v`, r, q)
			}
		}
		return strs.Format(`[%s]`, r)
	case []uint8:
		return strs.Format(`%s`, string(v))
	case nil:
		return strs.Format(`%s`, "NULL")
	default:
		logs.Errorf("Not unquoted type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}

/**
* Quote
* @param val interface{}
* @return any
**/
func Quote(val interface{}) any {
	fmt := `'%s'`
	if quotedChar == `"` {
		fmt = `"%s"`
	}
	switch v := val.(type) {
	case string:
		return quote(v)
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
		return strs.Format(fmt, v.Format("2006-01-02 15:04:05"))
	case et.Json:
		return strs.Format(fmt, v.ToString())
	case map[string]interface{}:
		return strs.Format(fmt, et.Json(v).ToString())
	case []string:
		var r string
		for i, _v := range v {
			if i == 0 {
				r = strs.Format(fmt, unquote(_v))
			} else {
				s := strs.Format(fmt, unquote(_v))
				r = strs.Format(`%s, %s`, r, s)
			}
		}
		r = strs.Format(`[%s]`, r)
		return strs.Format(fmt, r)
	case []interface{}:
		var r string
		for i, _v := range v {
			q := Quote(_v)
			if i == 0 {
				r = strs.Format(`%v`, q)
			} else {
				r = strs.Format(`%s, %v`, r, q)
			}
		}
		r = strs.Format(`[%s]`, r)
		return strs.Format(fmt, r)
	case []uint8:
		return strs.Format(fmt, string(v))
	case nil:
		return strs.Format(`%s`, "NULL")
	default:
		logs.Errorf("Not quoted type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}

/**
* Params return a string with the values replaced
* @param str string
* @param args ...any
* @return string
**/
func Params(str string, args ...any) string {
	var result string = str
	for i, v := range args {
		p := strs.Format(`$%d`, i+1)
		rp := strs.Format(`%v`, v)
		result = strs.Replace(result, p, rp)
	}

	return result
}

/**
* ParamQuote return a string with the values replaced
* @param str string
* @param args ...any
* @return string
**/
func ParamQuote(str string, args ...any) string {
	for i, arg := range args {
		old := strs.Format(`$%d`, i+1)
		new := strs.Format(`%v`, Quote(arg))
		str = strings.ReplaceAll(str, old, new)
	}

	return str
}

/**
* Address return a string with the host and port
* @param host string
* @param port int
* @return string
**/
func Address(host string, port int) string {
	return strs.Format("%s:%d", host, port)
}

/**
* BannerTitle return the value in a string format
* @param name string
* @param size int
* @return string
**/
func BannerTitle(name string, size int) string {
	return strs.Format(`{{ .Title "%s" "" %d }}`, name, size)
}

/**
* GoMod return the value of a go.mod attribute
* @param atrib string
* @return string
* @return error
**/
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

/**
* ToBase64
* @param data string
* @return string
**/
func ToBase64(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

/**
* FromBase64
* @param data string
* @return string
**/
func FromBase64(data string) string {
	result, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ""
	}

	return string(result)
}

/**
* PayloadEncoded
* @param data et.Json
* @return string
**/
func PayloadEncoded(data et.Json) string {
	result := ToBase64(data.ToString())

	return result
}

/**
* PayloadDecoded
* @param token string
* @return et.Json
**/
func PayloadDecoded(token string) (et.Json, error) {
	data := FromBase64(token)
	result, err := et.Object(data)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* SerializeStruct
* @param s interface{}
* @return []byte
* @return error
**/
func SerializeStruct(s interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
