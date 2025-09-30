package cache

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

const IsNil = redis.Nil

/**
* SetDuration
* @params key string, val interface{}, expMilisecond int64
* @return interface{}
**/
func SetDuration(key string, val interface{}, expiration time.Duration) interface{} {
	if conn == nil {
		return val
	}

	switch v := val.(type) {
	case et.Json:
		return SetCtx(conn.ctx, key, v.ToString(), expiration)
	case et.Items:
		return SetCtx(conn.ctx, key, v.ToString(), expiration)
	case et.Item:
		return SetCtx(conn.ctx, key, v.ToString(), expiration)
	case int:
		return SetCtx(conn.ctx, key, fmt.Sprintf(`%d`, v), expiration)
	case int64:
		return SetCtx(conn.ctx, key, fmt.Sprintf(`%d`, v), expiration)
	case float64:
		return SetCtx(conn.ctx, key, fmt.Sprintf(`%f`, v), expiration)
	case bool:
		return SetCtx(conn.ctx, key, fmt.Sprintf(`%t`, v), expiration)
	case []byte:
		return SetCtx(conn.ctx, key, string(v), expiration)
	case time.Time:
		return SetCtx(conn.ctx, key, v.Format(time.RFC3339), expiration)
	case time.Duration:
		return SetCtx(conn.ctx, key, v.String(), expiration)
	default:
		s, ok := v.(string)
		if ok {
			return SetCtx(conn.ctx, key, s, expiration)
		}
	}

	return nil
}

/**
* Incr
* @params key string, expiration time.Duration
* @return int64
**/
func IncrDuration(key string, expiration time.Duration) int64 {
	if conn == nil {
		return 0
	}

	return IncrCtx(conn.ctx, key, expiration)
}

/**
* Expire
* @params key string, expSecond int
* @return error
**/
func Expire(key string, expiration time.Duration) error {
	return ExpireCtx(conn.ctx, key, expiration)
}

/**
* Incr
* @params key string, expiration time.Duration
* @return int64
**/
func Incr(key string, expiration time.Duration) int64 {
	return IncrDuration(key, expiration)
}

/**
* Decr
* @params key string
* @return int64
**/
func Decr(key string) int64 {
	if conn == nil {
		return 0
	}

	return DecrCtx(conn.ctx, key)
}

/**
* Set
* @params key string, val interface{}, expiration time.Duration
* @return interface{}
**/
func Set(key string, val interface{}, expiration time.Duration) interface{} {
	if conn == nil {
		return val
	}

	return SetDuration(key, val, expiration)
}

/**
* Get
* @params key string, defaultvalue string
* @return string, error
**/
func Get(key, defaultvalue string) (string, error) {
	if conn == nil {
		return defaultvalue, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return 0, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	return DeleteCtx(conn.ctx, key)
}

/**
* LPush
* @params key string, val string
* @return error
**/
func LPush(key, val string) error {
	if conn == nil {
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	return LPushCtx(conn.ctx, key, val)
}

/**
* LRem
* @params key string, val string
* @return error
**/
func LRem(key, val string) error {
	if conn == nil {
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	return LRemCtx(conn.ctx, key, val)
}

/**
* LRange
* @params key string, start int64, stop int64
* @return []string, error
**/
func LRange(key string, start, stop int64) ([]string, error) {
	if conn == nil {
		return []string{}, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	return LRangeCtx(conn.ctx, key, start, stop)
}

/**
* LTrim
* @params key string, start int64, stop int64
* @return error
**/
func LTrim(key string, start, stop int64) error {
	if conn == nil {
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	return LTrimCtx(conn.ctx, key, start, stop)
}

/**
* SetH
* @params key string, val interface{}, expiration int
* @return interface{}
**/
func SetH(key string, val interface{}, expiration int) interface{} {
	duration := time.Duration(expiration) * time.Hour
	return Set(key, val, duration)
}

/**
* SetD
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetD(key string, val interface{}, expiration int) interface{} {
	return SetH(key, val, expiration*24)
}

/**
* SetW
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetW(key string, val interface{}, expiration int) interface{} {
	return SetH(key, val, expiration*24*7)
}

/**
* SetM
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetM(key string, val interface{}, expiration int) interface{} {
	return SetH(key, val, expiration*24*30)
}

/**
* SetY
* @params key string, val interface{}, expiration int64
* @return interface{}
**/
func SetY(key string, val interface{}, expiration int) interface{} {
	return SetH(key, val, expiration*24*365)
}

/**
* Empty
* @return error
**/
func Empty(match string) error {
	if conn == nil {
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	iter := conn.Scan(conn.ctx, 0, match, 0).Iterator()
	for iter.Next(conn.ctx) {
		key := iter.Val()
		DeleteCtx(conn.ctx, key)
	}

	return nil
}

/**
* HSet
* @params key string, val map[string]string
* @return error
**/
func HSet(key string, val map[string]string) error {
	if conn == nil {
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return map[string]string{}, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return "", fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HDeleteCtx(conn.ctx, key, atr)
}

/**
* genKey
* @params args ...interface{}
* @return string
**/
func genKey(args ...interface{}) string {
	result := reg.GenKey(args...)
	return utility.ToBase64(result)
}

/**
* SetVerify
* @params device string, key string, val string, expiration time.Duration
* @return interface{}
**/
func SetVerify(device, key, val string, expiration time.Duration) interface{} {
	key = genKey("verify", device, key)
	return Set(key, val, expiration)
}

/**
* GetVerify
* @params device string, key string
* @return string, error
**/
func GetVerify(device string, key string) (string, error) {
	key = genKey("verify", device, key)
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
	key = genKey("verify", device, key)
	return Delete(key)
}

/**
* AllCache
* @params search string, page int, rows int
* @return error
**/
func AllCache(search string, page, rows int) (et.List, error) {
	if conn == nil {
		return et.List{}, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return et.Json{}, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return et.Item{}, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
		return et.Items{}, fmt.Errorf(msg.ERR_NOT_CACHE_SERVICE)
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
