package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
	"github.com/redis/go-redis/v9"
)

const IsNil = redis.Nil

/**
* Load
* @param cfg *config.Config
* @return error
**/
func Load(cfg *config.Config) error {
	if conn != nil {
		return nil
	}

	var err error
	conn, err = New(cfg)
	if err != nil {
		return err
	}
	return nil
}

/**
* FromId
* @return string
**/
func FromId() string {
	if conn == nil {
		return ""
	}

	return conn.Id
}

/**
* IsLoad
* @return bool
**/
func IsLoad() bool {
	return conn != nil
}

/**
* Close terminates the Redis connection.
**/
func Close() {
	if conn == nil {
		return
	}

	conn.Close()
}

/**
* HealthCheck
* @return bool
**/
func HealthCheck() bool {
	if conn == nil {
		return false
	}

	return conn.HealthCheck()
}

/**
* SetWithDuration
* @params key string, val interface{}, expMilisecond int64
* @return interface{}
**/
func SetWithDuration(key string, val interface{}, expiration time.Duration) interface{} {
	if conn == nil {
		return val
	}

	switch v := val.(type) {
	case et.Json:
		return SetCtx(conn.ctx, key, v.ToString(), expiration)
	case et.Item:
		return SetCtx(conn.ctx, key, v.ToString(), expiration)
	case et.Items:
		return SetCtx(conn.ctx, key, v.ToString(), expiration)
	case et.List:
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

	return SetWithDuration(key, val, expiration)
}

/**
* Get
* @params key string, defaultvalue string
* @return string, error
**/
func Get(key, defaultvalue string) (string, error) {
	if conn == nil {
		return defaultvalue, errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	return GetCtx(conn.ctx, key, defaultvalue)
}

/**
* GetObject
* @params key string, dest any
* @return error
**/
func GetObject(key string, dest any) (bool, error) {
	result, err := Get(key, "")
	if err != nil {
		return false, err
	}

	err = json.Unmarshal([]byte(result), dest)
	if err != nil {
		return false, err
	}

	return true, nil
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
		return 0, errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
		return []string{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	iter := conn.Scan(conn.ctx, 0, match, 0).Iterator()
	for iter.Next(conn.ctx) {
		key := iter.Val()
		DeleteCtx(conn.ctx, key)
	}

	return nil
}

/**
* ColectionSet
* @params key string, val map[string]string
* @return error
**/
func CollectionSet(name string, val map[string]string) error {
	if conn == nil {
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, name, val)
}

/**
* CollectionGet
* @params name string
* @return map[string]string, error
**/
func CollectionGet(name string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	return HGetCtx(conn.ctx, name)
}

/**
* CollectionDelete
* @params name string, key string
* @return error
**/
func CollectionDelete(name, key string) error {
	if conn == nil {
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	return HDeleteCtx(conn.ctx, name, key)
}

/**
* CollectionPut
* @params name string, key string, val string
* @return error
**/
func CollectionPut(name, key, val string) error {
	if conn == nil {
		return errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	return HSetCtx(conn.ctx, name, map[string]string{key: val})
}

/**
* CollectionFind
* @params name string, key string
* @return string, error
**/
func CollectionFind(name, key string) (string, error) {
	if conn == nil {
		return "", errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	atribs, err := HGetCtx(conn.ctx, name)
	if err != nil {
		return "", err
	}

	for k, v := range atribs {
		if k == key {
			return v, nil
		}
	}

	return "", nil
}

/**
* Objets
* @params name string
* @return []et.Json, error
**/
func ObjetAll(name string) ([]et.Json, error) {
	result := []et.Json{}
	items, err := CollectionGet(name)
	if err != nil {
		return result, err
	}

	for _, item := range items {
		var obj et.Json
		err = json.Unmarshal([]byte(item), &obj)
		if err != nil {
			return result, err
		}

		result = append(result, obj)
	}

	return result, nil
}

/**
* ObjetSet
* @params name string, key string, val []byte
* @return error
**/
func ObjetSet(name string, key string, obj et.Json) error {
	val := obj.ToString()
	return CollectionPut(name, key, val)
}

/**
* ObjetGet
* @params name string, key string, v any
* @return error
**/
func ObjetGet(name, key string) (et.Json, error) {
	scr, err := CollectionFind(name, key)
	if err != nil {
		return et.Json{}, err
	}

	var result et.Json
	err = json.Unmarshal([]byte(scr), &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* ObjetDelete
* @params name string, key string
* @return error
**/
func ObjetDelete(name, key string) error {
	return CollectionDelete(name, key)
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
* AllCache: Scans all keys matching search and returns the requested page.
* Redis SCAN cursor is opaque — offset-based pagination is done in memory after a full scan.
* @param search string
* @param page int
* @param rows int
* @return et.List, error
**/
func AllCache(search string, page, rows int) (et.List, error) {
	if conn == nil {
		return et.List{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	var cursor uint64
	var keys []string
	for {
		var batch []string
		var err error
		batch, cursor, err = conn.Scan(conn.ctx, cursor, search, 100).Result()
		if err != nil {
			return et.List{}, err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	total := len(keys)
	offset := (page - 1) * rows
	end := offset + rows
	if offset > total {
		offset = total
	}
	if end > total {
		end = total
	}

	var items et.Items
	for _, key := range keys[offset:end] {
		items.Result = append(items.Result, et.Json{"key": key})
		items.Count++
	}

	return items.ToList(total, page, rows), nil
}

/**
* GetJson
* @params key string
* @return Json, error
**/
func GetJson(key string) (et.Json, error) {
	if conn == nil {
		return et.Json{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
	err = json.Unmarshal([]byte(val), &result)
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
		return et.Item{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	defaultVal := ""
	val, err := Get(key, defaultVal)
	if err != nil {
		return et.Item{}, err
	}

	if val == defaultVal {
		return et.Item{}, IsNil
	}

	var result et.Item
	err = json.Unmarshal([]byte(val), &result)
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}

/**
* GetItems
* @params key string
* @return Items, error
**/
func GetItems(key string) (et.Items, error) {
	if conn == nil {
		return et.Items{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
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
	err = json.Unmarshal([]byte(val), &result)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}
