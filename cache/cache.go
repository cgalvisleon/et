package cache

import "time"

type Cache interface {
	Type() string
	Set(key string, value interface{}, expiration time.Duration) interface{}
	Get(key string, _default interface{}) interface{}
	Del(key string) bool
	Count(key string, expiration time.Duration) int
	Clear()
	Len() int
	Keys() []string
	Values() []interface{}
}
