package cache

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/redis/go-redis/v9"
)

const IsNil = redis.Nil

/**
* Set
* @params key string
* @params val interface{}
* @params second time.Duration
* @return error
**/
func Set(key string, val interface{}, second time.Duration) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	switch v := val.(type) {
	case et.Json:
		return SetCtx(conn.ctx, key, v.ToString(), second)
	case et.Items:
		return SetCtx(conn.ctx, key, v.ToString(), second)
	case et.Item:
		return SetCtx(conn.ctx, key, v.ToString(), second)
	default:
		val, ok := val.(string)
		if ok {
			return SetCtx(conn.ctx, key, val, second)
		}
	}

	return nil
}

/**
* Get
* @params key string
* @params def string
* @return string, error
**/
func Get(key, def string) (string, error) {
	if conn == nil {
		return def, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	return GetCtx(conn.ctx, key, def)
}

/**
* Delete
* @params key string
* @return int64, error
**/
func Delete(key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	return DeleteCtx(conn.ctx, key)
}

/**
* Count
* @params key string
* @params expiration time.Duration
* @return int64
**/
func Count(key string, expiration time.Duration) int64 {
	if conn == nil {
		return 0
	}

	def := "-1"
	val, err := Get(key, def)
	if err != nil {
		return 0
	}

	if val == def {
		Set(key, "0", expiration)
		return 0
	}

	num, err := strconv.ParseInt(val, 10, 64)
	if logs.Alert(err) != nil {
		return 0
	}

	num++
	Set(key, val, expiration)

	return num
}

/**
* SetH
* @params key string
* @params val interface{}
* @return error
**/
func SetH(key string, val interface{}) error {
	return Set(key, val, time.Hour*1)
}

/**
* SetD
* @params key string
* @params val interface{}
* @return error
**/
func SetD(key string, val interface{}) error {
	return Set(key, val, time.Hour*24)
}

/**
* SetW
* @params key string
* @params val interface{}
* @return error
**/
func SetW(key string, val interface{}) error {
	return Set(key, val, time.Hour*24*7)
}

/**
* SetM
* @params key string
* @params val interface{}
* @return error
**/
func SetM(key string, val interface{}) error {
	return Set(key, val, time.Hour*24*30)
}

/**
* SetY
* @params key string
* @params val interface{}
* @return error
**/
func SetY(key string, val interface{}) error {
	return Set(key, val, time.Hour*24*365)
}

/**
* Empty
* @return error
**/
func Empty() error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	ctx := context.Background()
	iter := conn.db.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		DeleteCtx(ctx, key)
	}

	return nil
}

/**
* More
* @params key string
* @params second time.Duration
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
* @params key string
* @params val map[string]string
* @return error
**/
func HSet(key string, val map[string]string) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
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
		return map[string]string{}, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HGetCtx(conn.ctx, key)
}

/**
* HSetAtrib
* @params key string
* @params atr string
* @params val string
* @return error
**/
func HSetAtrib(key, atr, val string) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, map[string]string{atr: val})
}

/**
* HGetAtrib
* @params key string
* @params atr string
* @return string, error
**/
func HGetAtrib(key, atr string) (string, error) {
	if conn == nil {
		return "", logs.Log(msg.ERR_NOT_CACHE_SERVICE)
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
* @params key string
* @params atr string
* @return error
**/
func HDelete(key, atr string) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	return HDeleteCtx(conn.ctx, key, atr)
}

/**
* SetVerify
* @params device string
* @params key string
* @params val string
* @params duration time.Duration
* @return error
**/
func SetVerify(device, key, val string, duration time.Duration) error {
	key = strs.Format(`verify:%s:%s`, device, key)
	return Set(key, val, duration)
}

/**
* GetVerify
* @params device string
* @params key string
* @return string, error
**/
func GetVerify(device string, key string) (string, error) {
	key = strs.Format(`verify:%s:%s`, device, key)
	return Get(key, "")
}

/**
* DeleteVerify
* @params device string
* @params key string
* @return int64, error
**/
func DeleteVerify(device string, key string) (int64, error) {
	key = strs.Format(`verify:%s:%s`, device, key)
	return Delete(key)
}

/**
* AllCache
* @params device string
* @params key string
* @params val string
* @return error
**/
func AllCache(search string, page, rows int) (et.List, error) {
	if conn == nil {
		return et.List{}, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	ctx := context.Background()
	var cursor uint64
	var count int64
	var items et.Items = et.Items{}
	offset := (page - 1) * rows
	cursor = uint64(offset)
	count = int64(rows)

	iter := conn.db.Scan(ctx, cursor, search, count).Iterator()
	for iter.Next(ctx) {
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
		return et.Json{}, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	_default := "{}"
	val, err := Get(key, _default)
	if err != nil {
		return et.Json{}, err
	}

	if val == _default {
		return et.Json{}, nil
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
		return et.Item{}, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	_default := "{}"
	val, err := Get(key, _default)
	if err != nil {
		return et.Item{}, err
	}

	if val == _default {
		return et.Item{}, nil
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
		return et.Items{}, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	_default := "[]"
	val, err := Get(key, _default)
	if err != nil {
		return et.Items{}, err
	}

	if val == _default {
		return et.Items{}, nil
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
* @params w http.ResponseWriter
* @params r *http.Request
**/
func HandlerAll(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	search := query.Str("search")
	page := query.ValInt(1, "page")
	rows := query.ValInt(30, "rows")

	result, err := AllCache(search, page, rows)
	if logs.Alert(err) != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HandlerGet
* @params w http.ResponseWriter
* @params r *http.Request
**/
func HandlerGet(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")

	result, err := Get(key, "")
	if logs.Alert(err) != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* HandlerDelete
* @params w http.ResponseWriter
* @params r *http.Request
**/
func HandlerDelete(w http.ResponseWriter, r *http.Request) {
	query := response.GetQuery(r)
	key := query.Str("key")

	result, err := Delete(key)
	if logs.Alert(err) != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}
