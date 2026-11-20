package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* TestGroupCommitBench mide escritores concurrentes sobre el disco real (tarea 100 de F-23): cada uno confirma un
* lote con un documento y su índice y espera su fsync, como josefina. Reporta ops/s y cuántas confirmaciones cubre
* cada fsync. Solo corre con STORE_BENCH=1 (usa el disco de verdad y tarda unos segundos).
* @param t *testing.T
**/
func TestGroupCommitBench(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	for _, mode := range []string{"durable", "deferred+wait"} {
		for _, writers := range []int{1, 50} {
			t.Run(fmt.Sprintf("%s/w=%d", mode, writers), func(t *testing.T) {
				benchGroupCommit(t, writers, mode == "deferred+wait", 5*time.Second)
			})
		}
	}
}

/**
* benchGroupCommit corre writers escritores durante d y reporta.
* @param t *testing.T, writers int, deferred bool: CommitDeferred + WaitDurable en lugar de Commit, d time.Duration
**/
func benchGroupCommit(t *testing.T, writers int, deferred bool, d time.Duration) {
	cfg := testConfig()
	cfg.buffers = 16384
	cfg.wal = wal.Options{SegmentSize: 64 << 20, Spares: 2, Preallocate: true, Buffer: 1 << 20}
	dir := filepath.Join(t.TempDir(), "db")
	e, err := open(vfs.OS(), dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	docs, idx := mustStore(t, e, "docs"), mustStore(t, e, "idx")
	start := e.log.SyncLatency().Count
	var ops atomic.Int64
	var lat, pre, seq atomic.Int64
	payload := []byte(`{"jid":"b000000000001","n":1,"k":1,"payload":"` + strings.Repeat("x", 200) + `","created_at":"2026-11-20T14:18:19.123456-05:00"}`)
	stop := time.Now().Add(d)
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; time.Now().Before(stop); i++ {
				t0 := time.Now()
				n := seq.Add(1)
				id := fmt.Sprintf("b%012d", n)
				b := e.NewBatch()
				if err := docs.PutIn(b, id, payload); err != nil {
					t.Error(err)
					return
				}
				if err := idx.PutIn(b, fmt.Sprintf("%012d|%s", n, id), nil); err != nil {
					t.Error(err)
					return
				}
				pre.Add(int64(time.Since(t0)))
				if deferred {
					err = b.CommitDeferred()
					if err == nil {
						err = e.WaitDurable()
					}
				} else {
					err = b.Commit()
				}
				if err != nil {
					t.Error(err)
					return
				}
				lat.Add(int64(time.Since(t0)))
				ops.Add(1)
			}
		}()
	}
	wg.Wait()
	n := ops.Load()
	syncs := e.log.SyncLatency().Count - start
	s := e.log.SyncLatency()
	t.Logf("commit %+v", e.Stats().CommitLatency)
	t.Logf("antes del commit %.2f ms", float64(pre.Load())/float64(max(n, 1))/1e6)
	t.Logf("ops/s %.0f, avg %.2f ms, fsyncs %d, commits por fsync %.1f, fsync p50 %v mean %v",
		float64(n)/d.Seconds(), float64(lat.Load())/float64(max(n, 1))/1e6, syncs, float64(n)/float64(max(syncs, 1)), s.P50, s.Mean)
}

/**
* TestManyStoresBench mide crear muchos stores sueltos sobre el disco real (tarea 98 de F-23): 1.590 nombres del
* largo de los de josefina, uno tras otro, y el checkpoint que sigue. Solo corre con STORE_BENCH=1.
* @param t *testing.T
**/
func TestManyStoresBench(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	cfg := testConfig()
	cfg.buffers = 4096
	cfg.wal = wal.Options{SegmentSize: 64 << 20, Spares: 2, Preallocate: true, Buffer: 1 << 20}
	e, err := open(vfs.OS(), filepath.Join(t.TempDir(), "db"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	const n = 1590
	syncs := e.log.SyncLatency().Count
	start := time.Now()
	for i := range n {
		if _, err := e.Store(fmt.Sprintf("josefina/db/schema/model_%04d/btidx_field_name_%04d#u@1", i/16, i)); err != nil {
			t.Fatal(err)
		}
	}
	took := time.Since(start)
	t0 := time.Now()
	if err := e.Checkpoint(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d stores en %v (%.2f ms cada uno, %d fsyncs del WAL), checkpoint %v", n, took,
		float64(took.Microseconds())/n/1000, e.log.SyncLatency().Count-syncs, time.Since(t0))
}
