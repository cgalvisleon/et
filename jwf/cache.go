package jwf

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/redis/go-redis/v9"
)

// cacheStore: what the cache binding needs of a Redis: store a text, read it (false when the key is missing) and delete it
type cacheStore struct {
	set func(key, text string, expiration time.Duration)
	get func(key string) (string, bool, error)
	del func(key string) error
}

/**
* CacheOf: The cache binding (set, get, json, items, item, delete) over conn, a Redis of its own (cache.New: the one of a
* tenant), or over the package's (cache.Load) when conn is nil
* @param conn *cache.Conn
* @return map[string]interface{}
**/
func CacheOf(conn *cache.Conn) map[string]interface{} {
	if conn == nil {
		return cacheBinding(cacheStore{
			set: func(key, text string, expiration time.Duration) { cache.Set(key, text, expiration) },
			get: func(key string) (string, bool, error) { return cache.Get(key, "") },
			del: func(key string) error {
				_, err := cache.Delete(key)
				return err
			},
		})
	}

	ctx := context.Background()
	return cacheBinding(cacheStore{
		// Like the package's: Redis takes at least one millisecond
		set: func(key, text string, expiration time.Duration) {
			conn.Set(ctx, key, text, max(expiration, time.Millisecond))
		},
		get: func(key string) (string, bool, error) {
			result, err := conn.Get(ctx, key).Result()
			if errors.Is(err, redis.Nil) {
				return "", false, nil
			}
			return result, err == nil, err
		},
		del: func(key string) error { return conn.Del(ctx, key).Err() },
	})
}

/**
* cacheBinding: The cache binding over a store: set keeps a value as text (cache.TextOf; another type is not stored and
* gives null), get its text or the default, json/items/item its JSON (empty when missing or invalid) and delete whether
* it was deleted
* @param store cacheStore
* @return map[string]interface{}
**/
func cacheBinding(store cacheStore) map[string]interface{} {
	parse := func(key string, dest any) bool {
		text, exists, err := store.get(key)
		if err != nil || !exists {
			return false
		}
		return json.Unmarshal([]byte(text), dest) == nil
	}

	return map[string]interface{}{
		"set": func(key string, value interface{}, expiration time.Duration) interface{} {
			text, ok := cache.TextOf(value)
			if !ok {
				return nil
			}
			store.set(key, text, expiration)
			return text
		},
		"get": func(key string, defaultValue string) string {
			result, exists, err := store.get(key)
			if err != nil || !exists {
				return defaultValue
			}
			return result
		},
		"json": func(key string) et.Json {
			result := et.Json{}
			if !parse(key, &result) {
				return et.Json{}
			}
			return result
		},
		"items": func(key string) et.Items {
			result := et.Items{}
			if !parse(key, &result) {
				return et.Items{}
			}
			return result
		},
		"item": func(key string) et.Item {
			result := et.Item{}
			if !parse(key, &result) {
				return et.Item{}
			}
			return result
		},
		"delete": func(key string) bool {
			return store.del(key) == nil
		},
	}
}
