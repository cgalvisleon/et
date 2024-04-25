package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/redis/go-redis/v9"
)

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

func DelCtx(ctx context.Context, key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	intCmd := conn.db.Del(ctx, key)

	return intCmd.Val(), intCmd.Err()
}

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

func HGetCtx(ctx context.Context, key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	result := conn.db.HGetAll(ctx, key).Val()

	return result, nil
}

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
*
**/
func Get(key, def string) (string, error) {
	if conn == nil {
		return def, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return GetCtx(conn.ctx, key, def)
}

func Del(key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return DelCtx(conn.ctx, key)
}

func Set(key, val string, second time.Duration) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return SetCtx(conn.ctx, key, val, second)
}

func SetD(key, val string) error {
	return Set(key, val, time.Hour*24)
}

func SetW(key, val string) error {
	return Set(key, val, time.Hour*24*7)
}

func SetM(key, val string) error {
	return Set(key, val, time.Hour*24*30)
}

func SetY(key, val string) error {
	return Set(key, val, time.Hour*24*365)
}

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

func HSet(key string, val map[string]string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, val)
}

func HGet(key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HGetCtx(conn.ctx, key)
}

func HSetAtrib(key, atr, val string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, key, map[string]string{atr: val})
}

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

func HDel(key, atr string) error {
	if conn == nil {
		return logs.Log(ERR_NOT_CACHE_SERVICE)
	}

	return HDelCtx(conn.ctx, key, atr)
}

func SetVerify(device, key, val string) error {
	key = strs.Format(`verify:%s/%s`, device, key)
	return Set(key, val, 5*60)
}

func GetVerify(device string, key string) (string, error) {
	key = strs.Format(`verify:%s/%s`, device, key)
	return Get(key, "")
}

func DelVerify(device string, key string) (int64, error) {
	key = strs.Format(`verify:%s/%s`, device, key)
	return Del(key)
}

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

/**
* Json
**/
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

func SetJsonD(key string, val et.Json) (et.Json, error) {
	day := time.Hour * 24
	return SetJson(key, val, day)
}

func SetJsonW(key string, val et.Json) (et.Json, error) {
	week := time.Hour * 24 * 7
	return SetJson(key, val, week)
}

func SetJsonM(key string, val et.Json) (et.Json, error) {
	month := time.Hour * 24 * 30
	return SetJson(key, val, month)
}

func SetJsonY(key string, val et.Json) (et.Json, error) {
	year := time.Hour * 24 * 365
	return SetJson(key, val, year)
}

/**
* Item
**/
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

func SetItemD(key string, val et.Item) (et.Item, error) {
	day := time.Hour * 24
	return SetItem(key, val, day)
}

func SetItemW(key string, val et.Item) (et.Item, error) {
	week := time.Hour * 24 * 7
	return SetItem(key, val, week)
}

func SetItemM(key string, val et.Item) (et.Item, error) {
	month := time.Hour * 24 * 30
	return SetItem(key, val, month)
}

func SetItemY(key string, val et.Item) (et.Item, error) {
	year := time.Hour * 24 * 365
	return SetItem(key, val, year)
}

/**
* Items
**/
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

func SetItemsD(key string, val et.Items) (et.Items, error) {
	day := time.Hour * 24
	return SetItems(key, val, day)
}

func SetItemsW(key string, val et.Items) (et.Items, error) {
	week := time.Hour * 24 * 7
	return SetItems(key, val, week)
}

func SetItemsM(key string, val et.Items) (et.Items, error) {
	month := time.Hour * 24 * 30
	return SetItems(key, val, month)
}

func SetItemsY(key string, val et.Items) (et.Items, error) {
	year := time.Hour * 24 * 365
	return SetItems(key, val, year)
}
