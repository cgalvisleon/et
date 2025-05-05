package cache

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mistake"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

const IsNil = redis.Nil

/**
* SetDuration
* @params key string, val interface{}, millisecond time.Duration
* @return interface{}
**/
func SetDuration(key string, val interface{}, millisecond time.Duration) interface{} {
	if conn == nil {
		return val
	}

	switch v := val.(type) {
	case et.Json:
		return SetCtx(conn.ctx, key, v.ToString(), millisecond)
	case et.Items:
		return SetCtx(conn.ctx, key, v.ToString(), millisecond)
	case et.Item:
		return SetCtx(conn.ctx, key, v.ToString(), millisecond)
	case int:
		return SetCtx(conn.ctx, key, strs.Format(`%d`, v), millisecond)
	case int64:
		return SetCtx(conn.ctx, key, strs.Format(`%d`, v), millisecond)
	case float64:
		return SetCtx(conn.ctx, key, strs.Format(`%f`, v), millisecond)
	case bool:
		return SetCtx(conn.ctx, key, strs.Format(`%t`, v), millisecond)
	case []byte:
		return SetCtx(conn.ctx, key, string(v), millisecond)
	case time.Time:
		return SetCtx(conn.ctx, key, v.Format(time.RFC3339), millisecond)
	case time.Duration:
		return SetCtx(conn.ctx, key, v.String(), millisecond)
	default:
		s, ok := v.(string)
		if ok {
			return SetCtx(conn.ctx, key, s, millisecond)
		}
	}

	return nil
}

/**
* Set
* @params key string, val interface{}, second time.Duration
* @return interface{}
**/
func Set(key string, val interface{}, second time.Duration) interface{} {
	if conn == nil {
		return val
	}

	dur := second * time.Second / time.Millisecond
	return SetDuration(key, val, dur)
}

/**
* Get
* @params key string, defaultvalue string
* @return string, error
**/
func Get(key, defaultvalue string) (string, error) {
	if conn == nil {
		return defaultvalue, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return GetCtx(conn.ctx, key, defaultvalue)
}

/**
* Exists
* @params key string
* @return bool
**/
func Exists(key string) bool {
	if conn == nil {
		return false
	}

	return ExistsCtx(conn.ctx, key)
}

/**
* Delete
* @params key string
* @return int64, error
**/
func Delete(key string) (int64, error) {
	if conn == nil {
		return 0, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return DeleteCtx(conn.ctx, key)
}

/**
* Count
* @params key string, expiration time.Duration (second)
* @return int64
**/
func Count(key string, expiration time.Duration) int {
	if conn == nil {
		return 0
	}

	val, err := Get(key, "0")
	if err != nil {
		return 0
	}

	result, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	result++
	Set(key, result, expiration)

	return result
}

/**
* SetH
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetH(key string, val interface{}, expiration int64) interface{} {
	return Set(key, val, time.Duration(expiration)*time.Hour)
}

/**
* SetD
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetD(key string, val interface{}, expiration int64) interface{} {
	return Set(key, val, time.Duration(expiration)*time.Hour*24)
}

/**
* SetW
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetW(key string, val interface{}, expiration int64) interface{} {
	return Set(key, val, time.Duration(expiration)*time.Hour*24*7)
}

/**
* SetM
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetM(key string, val interface{}, expiration int64) interface{} {
	return Set(key, val, time.Duration(expiration)*time.Hour*24*30)
}

/**
* SetY
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetY(key string, val interface{}, expiration int64) interface{} {
	return Set(key, val, time.Duration(expiration)*time.Hour*24*365)
}

/**
* Empty
* @return error
**/
func Empty(match string) error {
	if conn == nil {
		return mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	iter := conn.Scan(conn.ctx, 0, match, 0).Iterator()
	for iter.Next(conn.ctx) {
		key := iter.Val()
		DeleteCtx(conn.ctx, key)
	}

	return nil
}

/**
* More
* @params key string, second time.Duration
* @return int
**/
func More(key string, second time.Duration) int {
	n, err := Get(key, "")
	if err != nil {
		n = "0"
	}

	if n == "" {
		n = "0"
	}

	val, err := strconv.Atoi(n)
	if err != nil {
		return 0
	}

	val++
	Set(key, strs.Format(`%d`, val), second)

	return val
}

/**
* HSet
* @params key string, val map[string]string
* @return error
**/
func HSet(key string, val map[string]string) error {
	if conn == nil {
		return mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, val)
}

/**
* HGet
* @params key string
* @return map[string]string, error
**/
func HGet(key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HGetCtx(conn.ctx, key)
}

/**
* HSetAtrib
* @params key string, atr string, val string
* @return error
**/
func HSetAtrib(key, atr, val string) error {
	if conn == nil {
		return mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, map[string]string{atr: val})
}

/**
* HGetAtrib
* @params key string, atr string
* @return string, error
**/
func HGetAtrib(key, atr string) (string, error) {
	if conn == nil {
		return "", mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	atribs, err := HGetCtx(conn.ctx, key)
	if err != nil {
		return "", err
	}

	for k, v := range atribs {
		if k == atr {
			return v, nil
		}
	}

	return "", nil
}

/**
* HDelete
* @params key string, atr string
* @return error
**/
func HDelete(key, atr string) error {
	if conn == nil {
		return mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HDeleteCtx(conn.ctx, key, atr)
}

/**
* GenId
* @params args ...interface{}
* @return string
**/
func GenId(args ...interface{}) string {
	var keys []string
	for _, arg := range args {
		keys = append(keys, strs.Format(`%v`, arg))
	}

	return strings.Join(keys, ":")
}

/**
* GenKey
* @params args ...interface{}
* @return string
**/
func GenKey(args ...interface{}) string {
	result := GenId(args...)
	return utility.ToBase64(result)
}

/**
* SetVerify
* @params device string, key string, val string, duration time.Duration
* @return interface{}
**/
func SetVerify(device, key, val string, duration time.Duration) interface{} {
	key = GenKey("verify", device, key)
	return Set(key, val, duration)
}

/**
* GetVerify
* @params device string, key string
* @return string, error
**/
func GetVerify(device string, key string) (string, error) {
	key = GenKey("verify", device, key)
	result, err := Get(key, "")
	if err != nil {
		return "", err
	}

	Delete(key)

	return result, nil
}

/**
* DeleteVerify
* @params device string, key string
* @return int64, error
**/
func DeleteVerify(device string, key string) (int64, error) {
	key = GenKey("verify", device, key)
	return Delete(key)
}

/**
* AllCache
* @params search string, page int, rows int
* @return error
**/
func AllCache(search string, page, rows int) (et.List, error) {
	if conn == nil {
		return et.List{}, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	var cursor uint64
	var count int64
	var items et.Items = et.Items{}
	offset := (page - 1) * rows
	cursor = uint64(offset)
	count = int64(rows)

	iter := conn.Scan(conn.ctx, cursor, search, count).Iterator()
	for iter.Next(conn.ctx) {
		key := iter.Val()
		items.Result = append(items.Result, et.Json{"key": key})
		items.Count++
	}

	return items.ToList(items.Count, page, rows), nil
}

/**
* GetJson
* @params key string
* @return Json, error
**/
func GetJson(key string) (et.Json, error) {
	if conn == nil {
		return et.Json{}, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	defaultVal := ""
	val, err := Get(key, defaultVal)
	if err != nil {
		return et.Json{}, err
	}

	if val == defaultVal {
		return et.Json{}, IsNil
	}

	var result et.Json
	err = result.Scan(val)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* GetItem
* @params key string
* @return Item, error
**/
func GetItem(key string) (et.Item, error) {
	if conn == nil {
		return et.Item{}, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	defaultVal := ""
	val, err := Get(key, defaultVal)
	if err != nil {
		return et.Item{}, err
	}

	if val == defaultVal {
		return et.Item{}, IsNil
	}

	var result et.Json
	err = result.Scan(val)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok:     true,
		Result: result,
	}, nil
}

/**
* GetItems
* @params key string
* @return Items, error
**/
func GetItems(key string) (et.Items, error) {
	if conn == nil {
		return et.Items{}, mistake.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	defaultVal := ""
	val, err := Get(key, defaultVal)
	if err != nil {
		return et.Items{}, err
	}

	if val == defaultVal {
		return et.Items{}, IsNil
	}

	var result et.Items
	err = result.Scan(val)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* HandlerAll
* @params w http.ResponseWriter, r *http.Request
**/
func HandlerAll(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	search := query.Str("search")
	page := query.ValInt(1, "page")
	rows := query.ValInt(30, "rows")

	result, err := AllCache(search, page, rows)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HandlerGet
* @params w http.ResponseWriter, r *http.Request
**/
func HandlerGet(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")

	result, err := Get(key, "")
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HandlerDelete
* @params w http.ResponseWriter, r *http.Request
**/
func HandlerDelete(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")

	result, err := Delete(key)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}
