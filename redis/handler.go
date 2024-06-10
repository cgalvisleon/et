package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/redis/go-redis/v9"
)

// Set a key with a value and a duration
func SetCtx(ctx context.Context, key, val string, second time.Duration) error {
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

// Get a key value
func GetCtx(ctx context.Context, key, def string) (string, error) {
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

// Delete a key
func DelCtx(ctx context.Context, key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	intCmd := conn.db.Del(ctx, key)

	return intCmd.Val(), intCmd.Err()
}

// Set a key with a value and a duration
func HSetCtx(ctx context.Context, key string, val map[string]string) error {
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

// Get a key value
func HGetCtx(ctx context.Context, key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	result := conn.db.HGetAll(ctx, key).Val()

	return result, nil
}

// Delete a key
func HDelCtx(ctx context.Context, key, atr string) error {
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
* Not contect functions
**/

// Get a key value
func Get(key, def string) (string, error) {
	if conn == nil {
		return def, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return GetCtx(conn.ctx, key, def)
}

// Delete a key
func Del(key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return DelCtx(conn.ctx, key)
}

// Set a key with a value and a duration
func Set(key, val string, second time.Duration) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return SetCtx(conn.ctx, key, val, second)
}

// Set a key with a value and a day duration
func SetD(key, val string) error {
	return Set(key, val, time.Hour*24)
}

// Set a key with a value and a week duration
func SetW(key, val string) error {
	return Set(key, val, time.Hour*24*7)
}

// Set a key with a value and a month duration
func SetM(key, val string) error {
	return Set(key, val, time.Hour*24*30)
}

// Set a key with a value and a year duration
func SetY(key, val string) error {
	return Set(key, val, time.Hour*24*365)
}

// Empty the cache
func Empty() error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	ctx := context.Background()
	iter := conn.db.Scan(ctx, 0, "*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		DelCtx(ctx, key)
	}

	return nil
}

// Increment a key
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

// Set a key with a map values
func HSet(key string, val map[string]string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, val)
}

// Get a key with a map values
func HGet(key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HGetCtx(conn.ctx, key)
}

// Set a key with a atrib and a value
func HSetAtrib(key, atr, val string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, map[string]string{atr: val})
}

// Get a key with a atrib value
func HGetAtrib(key, atr string) (string, error) {
	if conn == nil {
		return "", logs.Log(ERR_NOT_CACHE_SERVICE)
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
* Verify OTP values
**/

// Delete a key with a atrib
func HDel(key, atr string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HDelCtx(conn.ctx, key, atr)
}

// Increment a key with a atrib
func SetVerify(device, key, val string) error {
	key = strs.Format(`verify:%s/%s`, device, key)
	return Set(key, val, 5*60)
}

// Get a key with a atrib
func GetVerify(device string, key string) (string, error) {
	key = strs.Format(`verify:%s/%s`, device, key)
	return Get(key, "")
}

// Delete a key with a atrib
func DelVerify(device string, key string) (int64, error) {
	key = strs.Format(`verify:%s/%s`, device, key)
	return Del(key)
}

/**
* Other functions
**/

// Get all keys
func AllCache(search string, page, rows int) (et.List, error) {
	if conn == nil {
		return et.List{}, logs.Log(ERR_NOT_CACHE_SERVICE)
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

// Get key and json value
func GetJson(key string) (et.Json, error) {
	if conn == nil {
		return et.Json{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	_default := "{}"
	val, err := Get(key, _default)
	if err != nil {
		return et.Json{}, err
	}

	if val == _default {
		return et.Json{}, nil
	}

	var result et.Json = et.Json{}
	err = result.Scan(val)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

// Set key and json value
func SetJson(key string, val et.Json, second time.Duration) (et.Json, error) {
	if conn == nil {
		return val, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	err := Set(key, val.ToString(), second)
	if err != nil {
		return et.Json{}, err
	}

	return val, nil
}

// Set key and json value with a day duration
func SetJsonD(key string, val et.Json) (et.Json, error) {
	day := time.Hour * 24
	return SetJson(key, val, day)
}

// Set key and json value with a week duration
func SetJsonW(key string, val et.Json) (et.Json, error) {
	week := time.Hour * 24 * 7
	return SetJson(key, val, week)
}

// Set key and json value with a month duration
func SetJsonM(key string, val et.Json) (et.Json, error) {
	month := time.Hour * 24 * 30
	return SetJson(key, val, month)
}

// Set key and json value with a year duration
func SetJsonY(key string, val et.Json) (et.Json, error) {
	year := time.Hour * 24 * 365
	return SetJson(key, val, year)
}

/**
* Item values
**/

// Get a key and item value
func GetItem(key string) (et.Item, error) {
	if conn == nil {
		return et.Item{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	_default := "{}"
	val, err := Get(key, _default)
	if err != nil {
		return et.Item{}, err
	}

	if val == _default {
		return et.Item{}, nil
	}

	var result et.Json = et.Json{}
	err = result.Scan(val)
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok:     true,
		Result: result,
	}, nil
}

// Set a key and item value
func SetItem(key string, val et.Item, second time.Duration) (et.Item, error) {
	if conn == nil {
		return val, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	err := Set(key, val.ToString(), second)
	if err != nil {
		return et.Item{}, err
	}

	return val, nil
}

// Set a key and item value with a day duration
func SetItemD(key string, val et.Item) (et.Item, error) {
	day := time.Hour * 24
	return SetItem(key, val, day)
}

// Set a key and item value with a week duration
func SetItemW(key string, val et.Item) (et.Item, error) {
	week := time.Hour * 24 * 7
	return SetItem(key, val, week)
}

// Set a key and item value with a month duration
func SetItemM(key string, val et.Item) (et.Item, error) {
	month := time.Hour * 24 * 30
	return SetItem(key, val, month)
}

// Set a key and item value with a year duration
func SetItemY(key string, val et.Item) (et.Item, error) {
	year := time.Hour * 24 * 365
	return SetItem(key, val, year)
}

/**
* Items values
**/

// Get a key and items value
func GetItems(key string) (et.Items, error) {
	if conn == nil {
		return et.Items{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	_default := "[]"
	val, err := Get(key, _default)
	if err != nil {
		return et.Items{}, err
	}

	if val == _default {
		return et.Items{}, nil
	}

	var result et.Items = et.Items{}
	err = result.Scan(val)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

// Set a key and items value
func SetItems(key string, val et.Items, second time.Duration) (et.Items, error) {
	if conn == nil {
		return val, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	err := Set(key, val.ToString(), second)
	if err != nil {
		return et.Items{}, err
	}

	return val, nil
}

// Set a key and items value with a day duration
func SetItemsD(key string, val et.Items) (et.Items, error) {
	day := time.Hour * 24
	return SetItems(key, val, day)
}

func SetItemsW(key string, val et.Items) (et.Items, error) {
	week := time.Hour * 24 * 7
	return SetItems(key, val, week)
}

// Set a key and items value with a month duration
func SetItemsM(key string, val et.Items) (et.Items, error) {
	month := time.Hour * 24 * 30
	return SetItems(key, val, month)
}

// Set a key and items value with a year duration
func SetItemsY(key string, val et.Items) (et.Items, error) {
	year := time.Hour * 24 * 365
	return SetItems(key, val, year)
}
