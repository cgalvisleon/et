package jws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestHubConcurrentAccess drives Subscribe/Unsubscribe/Publish/SendTo concurrently
// against the same Hub while several real WebSocket connections are live. It exists
// to catch the "concurrent map read and map write" / data-race class of bug found in
// et/jws (Hub.Subscribers/Channels and Channel.Subscribers/Turn were previously
// accessed from multiple goroutines with no synchronization). Run with -race.
func TestHubConcurrentAccess(t *testing.T) {
	hub := New()
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := Upgrader(w, r)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}

		username := r.URL.Query().Get("username")
		ctx := context.WithValue(r.Context(), "username", username)
		if _, err := hub.Connect(socket, ctx); err != nil {
			t.Errorf("connect: %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]

	const users = 8
	conns := make([]*websocket.Conn, users)
	names := make([]string, users)
	for i := range users {
		name := "user" + strconv.Itoa(i)
		names[i] = name

		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?username="+name, nil)
		if err != nil {
			t.Fatalf("dial %s: %v", name, err)
		}
		conns[i] = conn

		go func(c *websocket.Conn) {
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return
				}
			}
		}(conn)
	}
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	// let every connection finish registering with the hub.
	time.Sleep(100 * time.Millisecond)

	ch := hub.Topic("race-topic")

	var wg sync.WaitGroup
	const iterations = 200
	for i := range iterations {
		name := names[i%users]

		wg.Add(4)
		go func() {
			defer wg.Done()
			_ = hub.Subscribe(ch.Name, name)
		}()
		go func() {
			defer wg.Done()
			_ = hub.Unsubscribe(ch.Name, name)
		}()
		go func() {
			defer wg.Done()
			_, _ = hub.Publish(ch.Name, NewMessage(nil, nil))
		}()
		go func() {
			defer wg.Done()
			_, _ = hub.SendTo([]string{name}, NewMessage(nil, []string{name}))
		}()
	}
	wg.Wait()
}

// TestHubReconnectDoesNotLeakGoroutines reconnects the same username repeatedly
// without cleanly closing the previous connection first, mimicking a client that
// drops and re-establishes its socket. Before the fix, Hub.Connect's reconnect
// branch never retired the previous connection's read()/write() goroutines nor
// closed its socket, leaking both on every reconnect.
func TestHubReconnectDoesNotLeakGoroutines(t *testing.T) {
	hub := New()
	hub.Start()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := Upgrader(w, r)
		if err != nil {
			return
		}

		ctx := context.WithValue(r.Context(), "username", "reconnector")
		if _, err := hub.Connect(socket, ctx); err != nil {
			t.Errorf("connect: %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):] + "?username=reconnector"

	runtime.GC()
	base := runtime.NumGoroutine()

	const rounds = 20
	for i := range rounds {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial round %d: %v", i, err)
		}

		go func(c *websocket.Conn) {
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return
				}
			}
		}(conn)

		// deliberately not closing the previous connection: the hub's rebind
		// logic, not the client, is what must retire it.
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()

	if after > base+10 {
		t.Errorf("goroutine count grew from %d to %d after %d reconnects; suspect leaked read()/write() goroutines", base, after, rounds)
	}
}
