package cache

import (
	"slices"
	"testing"
	"time"
)

// TestLoadFallsBackToMemory checks that without REDIS_HOST the package-level
// helpers work against the in-memory store, as a token store (et/jwt) needs.
func TestLoadFallsBackToMemory(t *testing.T) {
	t.Setenv("REDIS_HOST", "")
	if err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Cleanup(Close)

	if !IsLoad() || !HealthCheck() {
		t.Fatal("the in-memory cache should count as loaded and healthy")
	}

	Set("token:a", "value", time.Hour)
	got, err := Get("token:a", "")
	if err != nil || got != "value" {
		t.Fatalf("Get = %q, %v; want value", got, err)
	}

	if _, err := Get("token:missing", ""); err != ErrNotFound {
		t.Fatalf("Get of a missing key: err = %v, want ErrNotFound", err)
	}

	if n, _ := Delete("token:a"); n != 1 || Exists("token:a") {
		t.Fatalf("Delete removed %d keys, key still exists: %v", n, Exists("token:a"))
	}
}

func TestMemStoreExpiration(t *testing.T) {
	s := newMemStore()
	defer s.close()

	s.set("short", "v", 20*time.Millisecond)
	s.set("forever", "v", 0)
	time.Sleep(40 * time.Millisecond)

	if _, ok := s.get("short"); ok {
		t.Error("an expired key is still readable")
	}
	if _, ok := s.get("forever"); !ok {
		t.Error("a key with no expiration disappeared")
	}
}

func TestMemStoreCounters(t *testing.T) {
	s := newMemStore()
	defer s.close()

	if n, created := s.incrBy("c", 1); n != 1 || !created {
		t.Fatalf("first incr = %d, created %v", n, created)
	}
	if n, created := s.incrBy("c", 1); n != 2 || created {
		t.Fatalf("second incr = %d, created %v", n, created)
	}
	if n, _ := s.incrBy("c", -1); n != 1 {
		t.Fatalf("decr = %d, want 1", n)
	}
}

func TestMemStoreLists(t *testing.T) {
	s := newMemStore()
	defer s.close()

	for _, v := range []string{"a", "b", "c", "b"} {
		s.lpush("l", v)
	}
	if got := s.lrange("l", 0, -1); !slices.Equal(got, []string{"b", "c", "b", "a"}) {
		t.Fatalf("lrange = %v", got)
	}

	s.lrem("l", 1, "b")
	if got := s.lrange("l", 0, -1); !slices.Equal(got, []string{"c", "b", "a"}) {
		t.Fatalf("after lrem = %v", got)
	}

	s.ltrim("l", 0, 1)
	if got := s.lrange("l", 0, -1); !slices.Equal(got, []string{"c", "b"}) {
		t.Fatalf("after ltrim = %v", got)
	}
}

func TestMemStoreHashesAndKeys(t *testing.T) {
	s := newMemStore()
	defer s.close()

	s.hset("h", map[string]string{"x": "1", "y": "2"})
	s.hdel("h", "x")
	if got := s.hgetall("h"); len(got) != 1 || got["y"] != "2" {
		t.Fatalf("hgetall = %v", got)
	}

	s.set("app:1", "", 0)
	s.set("app:2", "", 0)
	s.set("other", "", 0)
	if got := s.keys("app:*"); !slices.Equal(got, []string{"app:1", "app:2"}) {
		t.Fatalf("keys = %v", got)
	}

	s.del(s.keys("app:*")...)
	if got := s.keys(""); !slices.Equal(got, []string{"h", "other"}) {
		t.Fatalf("keys after del = %v", got)
	}
}
