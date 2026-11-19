package server

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// freePort: un puerto TCP libre en localhost
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestStartRunsOnClose: una señal de apagado hace que Start detenga el HTTP, corra los OnClose una vez y vuelva.
func TestStartRunsOnClose(t *testing.T) {
	port := freePort(t)
	srv := New("test", port)
	srv.Get("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) })
	var closed atomic.Int32
	srv.OnClose(func() { closed.Add(1) })

	done := make(chan struct{})
	go func() {
		srv.Start()
		close(done)
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("el servidor no respondió: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	srv.stop <- syscall.SIGTERM
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Start no volvió después de la señal")
	}

	if closed.Load() != 1 {
		t.Fatalf("OnClose corrió %d veces", closed.Load())
	}
	if _, err := http.Get(url); err == nil {
		t.Fatal("el HTTP sigue atendiendo después de Close")
	}

	srv.Close()
	if closed.Load() != 1 {
		t.Fatalf("un segundo Close repitió OnClose: %d", closed.Load())
	}
}
