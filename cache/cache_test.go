package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestConn() *Conn {
	return &Conn{
		Client:   redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
		ctx:      context.Background(),
		channels: make(map[string]*redis.PubSub),
		mutex:    &sync.RWMutex{},
	}
}

// TestConnCloseDoesNotRecurse guards against Conn.Close() calling itself
// instead of the embedded *redis.Client's Close() — an infinite recursion
// that crashed the process with a stack overflow on every call, including via
// the package-level cache.Close() wrapper.
func TestConnCloseDoesNotRecurse(t *testing.T) {
	c := newTestConn()

	done := make(chan struct{})
	go func() {
		c.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not return in time — possible infinite recursion")
	}
}

// TestConnHealthCheckUsesReceiverNotGlobal guards against HealthCheck reading
// the package-level `conn` variable instead of its own receiver. With the
// package-level conn left nil (as it is before Load() is ever called), the
// old code panicked on any *Conn built directly via New(), instead of
// checking the connection it was actually called on.
func TestConnHealthCheckUsesReceiverNotGlobal(t *testing.T) {
	saved := conn
	conn = nil
	defer func() { conn = saved }()

	c := newTestConn()

	if c.HealthCheck() {
		t.Fatal("expected HealthCheck to report unhealthy for an unreachable address")
	}
}

// TestExpireRequiresLoadedConn guards against Expire dereferencing the nil
// package-level conn (via conn.ctx) when called before Load() — every sibling
// wrapper in handler.go already guards this, Expire was the one exception.
func TestExpireRequiresLoadedConn(t *testing.T) {
	saved := conn
	conn = nil
	defer func() { conn = saved }()

	if err := Expire("some-key", time.Second); err == nil {
		t.Fatal("expected an error when conn is not loaded")
	}
}
