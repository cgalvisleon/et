package cache

import (
	"maps"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cgalvisleon/et/utility"
)

/**
* memItem: One key of the in-memory store. A key holds a string, a list or a
* hash (like Redis, the kind is fixed by the first write) plus an optional
* expiration (zero means it never expires).
**/
type memItem struct {
	str     string
	list    []string
	hash    map[string]string
	expires time.Time
}

/**
* expired: Returns true when the item has an expiration that already passed.
* @param now time.Time
* @return bool
**/
func (s *memItem) expired(now time.Time) bool {
	return !s.expires.IsZero() && !now.Before(s.expires)
}

/**
* memStore: In-process replacement for Redis, used when REDIS_HOST is not set
* (a desktop app or a single-instance service with no Redis around). It keeps
* the subset of commands this package issues — strings, counters, lists,
* hashes, TTLs and SCAN by glob pattern — for one process only: nothing is
* shared between instances or survives a restart.
**/
type memStore struct {
	id    string
	items map[string]*memItem
	mutex sync.Mutex
	stop  chan struct{}
}

/**
* newMemStore: Creates the store and starts the goroutine that purges expired keys.
* @return *memStore
**/
func newMemStore() *memStore {
	result := &memStore{
		id:    utility.UUID(),
		items: make(map[string]*memItem),
		stop:  make(chan struct{}),
	}
	go result.sweep(time.Minute)
	return result
}

/**
* sweep: Deletes expired keys every interval until close is called. Reads also
* skip expired keys, so this only bounds memory.
* @param interval time.Duration
**/
func (s *memStore) sweep(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			s.mutex.Lock()
			for key, item := range s.items {
				if item.expired(now) {
					delete(s.items, key)
				}
			}
			s.mutex.Unlock()
		}
	}
}

/**
* close: Stops the sweeper and drops every key.
**/
func (s *memStore) close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	close(s.stop)
	s.items = make(map[string]*memItem)
}

/**
* item: Returns the live item for key, deleting it if it expired. Callers hold the mutex.
* @param key string
* @return *memItem, bool
**/
func (s *memStore) item(key string) (*memItem, bool) {
	result, ok := s.items[key]
	if !ok {
		return nil, false
	}
	if result.expired(time.Now()) {
		delete(s.items, key)
		return nil, false
	}
	return result, true
}

/**
* itemOrNew: Returns the live item for key, creating an empty one without expiration. Callers hold the mutex.
* @param key string
* @return *memItem
**/
func (s *memStore) itemOrNew(key string) *memItem {
	result, ok := s.item(key)
	if !ok {
		result = &memItem{}
		s.items[key] = result
	}
	return result
}

/**
* set: SET key val PX expiration (expiration <= 0 keeps the key forever).
* @param key, val string, expiration time.Duration
**/
func (s *memStore) set(key, val string, expiration time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result := &memItem{str: val}
	if expiration > 0 {
		result.expires = time.Now().Add(expiration)
	}
	s.items[key] = result
}

/**
* get: GET key.
* @param key string
* @return string, bool
**/
func (s *memStore) get(key string) (string, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.item(key)
	if !ok {
		return "", false
	}
	return result.str, true
}

/**
* incrBy: INCRBY key delta. A key that does not hold an integer counts as 0.
* The second result is true when the key was created by this call.
* @param key string, delta int64
* @return int64, bool
**/
func (s *memStore) incrBy(key string, delta int64) (int64, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, existed := s.item(key)
	result := s.itemOrNew(key)
	n, _ := strconv.ParseInt(result.str, 10, 64)
	n += delta
	result.str = strconv.FormatInt(n, 10)
	return n, !existed
}

/**
* expire: PEXPIRE key expiration. Returns false when the key does not exist.
* @param key string, expiration time.Duration
* @return bool
**/
func (s *memStore) expire(key string, expiration time.Duration) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.item(key)
	if !ok {
		return false
	}
	result.expires = time.Now().Add(expiration)
	return true
}

/**
* exists: EXISTS key.
* @param key string
* @return bool
**/
func (s *memStore) exists(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.item(key)
	return ok
}

/**
* del: DEL keys... Returns how many existed.
* @param keys ...string
* @return int64
**/
func (s *memStore) del(keys ...string) int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var result int64
	for _, key := range keys {
		if _, ok := s.item(key); ok {
			delete(s.items, key)
			result++
		}
	}
	return result
}

/**
* keys: The live keys matching a Redis-style glob pattern ("*", "?", "[...]"),
* sorted. An empty pattern matches everything, like SCAN without MATCH.
* @param pattern string
* @return []string
**/
func (s *memStore) keys(pattern string) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	result := make([]string, 0)
	for key, item := range s.items {
		if item.expired(now) {
			continue
		}
		if pattern != "" {
			if ok, err := path.Match(pattern, key); err != nil || !ok {
				continue
			}
		}
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

/**
* lpush: LPUSH key val.
* @param key, val string
**/
func (s *memStore) lpush(key, val string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result := s.itemOrNew(key)
	result.list = append([]string{val}, result.list...)
}

/**
* lrem: LREM key count val, for count > 0 (removes the first count occurrences).
* @param key string, count int, val string
**/
func (s *memStore) lrem(key string, count int, val string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.item(key)
	if !ok {
		return
	}
	list := result.list[:0]
	for _, v := range result.list {
		if v == val && count > 0 {
			count--
			continue
		}
		list = append(list, v)
	}
	result.list = list
	if len(list) == 0 && result.hash == nil {
		delete(s.items, key)
	}
}

/**
* listRange: Resolves Redis start/stop indexes (negative counts from the end) to a
* half-open [from, to) range over a list of length n.
* @param n int, start, stop int64
* @return int, int
**/
func listRange(n int, start, stop int64) (int, int) {
	size := int64(n)
	if start < 0 {
		start += size
	}
	if stop < 0 {
		stop += size
	}
	if start < 0 {
		start = 0
	}
	if stop >= size {
		stop = size - 1
	}
	if start > stop {
		return 0, 0
	}
	return int(start), int(stop) + 1
}

/**
* lrange: LRANGE key start stop.
* @param key string, start, stop int64
* @return []string
**/
func (s *memStore) lrange(key string, start, stop int64) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.item(key)
	if !ok {
		return []string{}
	}
	from, to := listRange(len(result.list), start, stop)
	return append([]string{}, result.list[from:to]...)
}

/**
* ltrim: LTRIM key start stop.
* @param key string, start, stop int64
**/
func (s *memStore) ltrim(key string, start, stop int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.item(key)
	if !ok {
		return
	}
	from, to := listRange(len(result.list), start, stop)
	result.list = append([]string{}, result.list[from:to]...)
}

/**
* hset: HSET key field val [field val ...].
* @param key string, val map[string]string
**/
func (s *memStore) hset(key string, val map[string]string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result := s.itemOrNew(key)
	if result.hash == nil {
		result.hash = make(map[string]string, len(val))
	}
	maps.Copy(result.hash, val)
}

/**
* hgetall: HGETALL key.
* @param key string
* @return map[string]string
**/
func (s *memStore) hgetall(key string) map[string]string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result := map[string]string{}
	item, ok := s.item(key)
	if !ok {
		return result
	}
	maps.Copy(result, item.hash)
	return result
}

/**
* hdel: HDEL key field.
* @param key, field string
**/
func (s *memStore) hdel(key, field string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.item(key)
	if !ok {
		return
	}
	delete(result.hash, field)
}
