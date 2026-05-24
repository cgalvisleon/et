package cache

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

type Store struct{}

/**
* LoadStore creates a new Store instance
* @return *Store
 */
func LoadStore() *Store {
	return &Store{}
}

/**
* Set stores a value with a key and expiration time
* @param key string, val interface{}, duration time.Duration
* @return interface{}
 */
func (s *Store) Set(key string, val interface{}, duration time.Duration) interface{} {
	return Set(key, val, duration)
}

/**
* GetItem retrieves an item by key
* @param key string
* @return (et.Item, error)
 */
func (s *Store) GetItem(key string) (et.Item, error) {
	return GetItem(key)
}

/**
* Delete removes an item by key
* @param key string
* @return (int64, error)
 */
func (s *Store) Delete(key string) (int64, error) {
	return Delete(key)
}
