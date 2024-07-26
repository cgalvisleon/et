package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/redis/go-redis/v9"
)

/**
* setCtx, set a key with a value and a duration
* @param ctx context.Context
* @param key string
* @param val string
* @param second time.Duration
* @return error
**/
func setCtx(ctx context.Context, key, val string, second time.Duration) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	duration := second * time.Second

	err := conn.db.Set(ctx, key, val, duration).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* getCtx, get a key value
* @param ctx context.Context
* @param key string
* @param def string
* @return string, error
**/
func getCtx(ctx context.Context, key, def string) (string, error) {
	if conn == nil {
		return def, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	result, err := conn.db.Get(ctx, key).Result()
	switch {
	case err == redis.Nil:
		return def, nil
	case err != nil:
		return def, err
	case result == "":
		return result, nil
	default:
		return result, nil
	}
}

/**
* delCtx, delete a key
* @param ctx context.Context
* @param key string
* @return int64, error
**/
func delCtx(ctx context.Context, key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	intCmd := conn.db.Del(ctx, key)

	return intCmd.Val(), intCmd.Err()
}

/**
* hSetCtx, set a key with a map values
* @param ctx context.Context
* @param key string
* @param val map[string]string
* @return error
**/
func hSetCtx(ctx context.Context, key string, val map[string]string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	for k, v := range val {
		err := conn.db.HSet(ctx, key, k, v).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* hGetCtx, get a key with a map values
* @param ctx context.Context
* @param key string
* @return map[string]string, error
**/
func hGetCtx(ctx context.Context, key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	result := conn.db.HGetAll(ctx, key).Val()

	return result, nil
}

/**
*hDelCtx, delete a key with a atrib
* @param ctx context.Context
* @param key string
* @param atr string
* @return error
**/
func hDelCtx(ctx context.Context, key, atr string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	err := conn.db.Do(ctx, "HDEL", key, atr).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* Get a key
* @param key string
* @param def string
* @return string, error
**/
func Get(key, def string) (string, error) {
	if conn == nil {
		return def, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return getCtx(conn.ctx, key, def)
}

/**
* Delete a key
* @param key string
* @return int64, error
**/
func Del(key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return delCtx(conn.ctx, key)
}

/**
* Set a key with a value and a duration
* @param key string
* @param val string
* @param second time.Duration
* @return error
**/
func Set(key string, val interface{}, second time.Duration) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	switch v := val.(type) {
	case js.Json:
		return setCtx(conn.ctx, key, v.ToString(), second)
	case js.Items:
		return setCtx(conn.ctx, key, v.ToString(), second)
	case js.Item:
		return setCtx(conn.ctx, key, v.ToString(), second)
	default:
		val, ok := val.(string)
		if ok {
			return setCtx(conn.ctx, key, val, second)
		}
	}

	return nil
}

/**
* SetH, set a key with a value and a hour duration
* @param key string
* @param val string
* @return error
**/
func SetH(key string, val interface{}) error {
	return Set(key, val, time.Hour*1)
}

/**
* SetD, set a key with a value and a day duration
* @param key string
* @param val string
* @return error
**/
func SetD(key string, val interface{}) error {
	return Set(key, val, time.Hour*24)
}

/**
* SetW, set a key with a value and a week duration
* @param key string
* @param val string
* @return error
**/
func SetW(key string, val interface{}) error {
	return Set(key, val, time.Hour*24*7)
}

/**
* SetM, set a key with a value and a month duration
* @param key string
* @param val string
* @return error
**/
func SetM(key string, val interface{}) error {
	return Set(key, val, time.Hour*24*30)
}

/**
* SetY, set a key with a value and a year duration
* @param key string
* @param val string
* @return error
**/
func SetY(key string, val interface{}) error {
	return Set(key, val, time.Hour*24*365)
}

/**
* Clear the cache
* @param match string
* @return error
**/
func Clear(match string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	ctx := context.Background()
	iter := conn.db.Scan(ctx, 0, match, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		delCtx(ctx, key)
	}

	return nil
}

/**
* Empty the cache
* @return error
**/
func Empty() error {
	return Clear("*")
}

/**
* Increment a key
* @param key string
* @param second time.Duration
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
* Hset a key with a map values
* @param key string
* @param val map[string]string
* @return error
**/
func HSet(key string, val map[string]string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return hSetCtx(conn.ctx, key, val)
}

/**
* HGet a key with a map values
* @param key string
* @return map[string]string, error
**/
func HGet(key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return hGetCtx(conn.ctx, key)
}

/**
* HSetAtrib, set a key with a atrib value
* @param key string
* @param atr string
* @param val string
* @return error
**/
func HSetAtrib(key, atr, val string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return hSetCtx(conn.ctx, key, map[string]string{atr: val})
}

/**
* HGetAtrib, get a key with a atrib value
* @param key string
* @param atr string
* @return string, error
**/
func HGetAtrib(key, atr string) (string, error) {
	if conn == nil {
		return "", logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	atribs, err := hGetCtx(conn.ctx, key)
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
* HDel, delete a key with a atrib
* @param key string
* @param atr string
* @return error
**/
func HDel(key, atr string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return hDelCtx(conn.ctx, key, atr)
}

/**
* AllCache, get all keys
* @param search string
* @param page int
* @param rows int
* @return js.List, error
**/
func AllCache(search string, page, rows int) (js.List, error) {
	if conn == nil {
		return js.List{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	ctx := context.Background()
	var cursor uint64
	var count int64
	var items js.Items = js.Items{}
	offset := (page - 1) * rows
	cursor = uint64(offset)
	count = int64(rows)

	iter := conn.db.Scan(ctx, cursor, search, count).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		items.Result = append(items.Result, js.Json{"key": key})
		items.Count++
	}

	return items.ToList(items.Count, page, rows), nil
}

/**
* GetJson, get a key and json value
* @param key string
* @return js.Json, error
**/
func GetJson(key string) (js.Json, error) {
	if conn == nil {
		return js.Json{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	_default := "{}"
	val, err := Get(key, _default)
	if err != nil {
		return js.Json{}, err
	}

	if val == _default {
		return js.Json{}, nil
	}

	var result js.Json = js.Json{}
	err = result.Scan(val)
	if err != nil {
		return js.Json{}, err
	}

	return result, nil
}

/**
* GetItem, get a key and item value
* @param key string
* @return js.Item, error
**/
func GetItem(key string) (js.Item, error) {
	if conn == nil {
		return js.Item{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	_default := "{}"
	val, err := Get(key, _default)
	if err != nil {
		return js.Item{}, err
	}

	if val == _default {
		return js.Item{}, nil
	}

	var result js.Json = js.Json{}
	err = result.Scan(val)
	if err != nil {
		return js.Item{}, err
	}

	return js.Item{
		Ok:     true,
		Result: result,
	}, nil
}

/**
* GetItems, get a key and items value
* @param key string
* @return js.Items, error
**/
func GetItems(key string) (js.Items, error) {
	if conn == nil {
		return js.Items{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	_default := "[]"
	val, err := Get(key, _default)
	if err != nil {
		return js.Items{}, err
	}

	if val == _default {
		return js.Items{}, nil
	}

	var result js.Items = js.Items{}
	err = result.Scan(val)
	if err != nil {
		return js.Items{}, err
	}

	return result, nil
}
