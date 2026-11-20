package store

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// slowFS es el disco real con demoras: cada escritura y cada fsync tardan un poco más. Las pruebas de memoria usan el
// disco real (en un directorio temporal, con pocos MB) para que el heap medido sea solo el del motor: crashfs guarda
// los datos en el heap.
type slowFS struct {
	vfs.FS
	write, sync time.Duration
}

// slowFile es un archivo de slowFS.
type slowFile struct {
	vfs.File
	fs *slowFS
}

func (s *slowFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := s.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return slowFile{File: f, fs: s}, nil
}

func (s *slowFS) SyncDir(dir string) error {
	time.Sleep(s.sync)
	return s.FS.SyncDir(dir)
}

func (f slowFile) WriteAt(b []byte, off int64) (int, error) {
	time.Sleep(f.fs.write)
	return f.File.WriteAt(b, off)
}

func (f slowFile) Sync() error {
	time.Sleep(f.fs.sync)
	return f.File.Sync()
}

// TestSlowDisk: muchos escritores sobre un disco lento (fsync de 5 ms más el real). Con confirmaciones durables
// esperan al fsync (group commit: menos fsync que confirmaciones); con diferidas, el freno no deja que lo que no tiene
// fsync pase de maxDeferredLag registros. En los dos casos el heap vivo no crece con lo escrito.
func TestSlowDisk(t *testing.T) {
	if testing.Short() {
		t.Skip("disco real, unos segundos")
	}
	const syncDelay = 5 * time.Millisecond
	for _, deferred := range []bool{false, true} {
		t.Run(fmt.Sprintf("deferred=%v", deferred), func(t *testing.T) {
			cfg := testConfig()
			cfg.maxWALSize = 4 << 20 // checkpoints de fondo por volumen, cada ~2 MB de WAL
			cfg.checkEvery = 50 * time.Millisecond
			fsys := &slowFS{FS: vfs.OS(), write: 20 * time.Microsecond, sync: syncDelay}
			e, err := open(fsys, t.TempDir(), cfg)
			mustOK(t, err)
			defer e.Close()
			s := mustStore(t, e, "s")

			stop := make(chan struct{})
			var wg sync.WaitGroup
			var maxLag atomic.Uint64
			for w := range 16 {
				wg.Go(func() {
					value := bytes.Repeat([]byte{byte('a' + w)}, 100)
					for i := 0; ; i++ {
						select {
						case <-stop:
							return
						default:
						}
						id := fmt.Sprintf("id-%03d", (w*7919+i)%500)
						var err error
						if deferred {
							err = s.Deferred().Put(id, value)
						} else {
							err = s.Put(id, value)
						}
						if err != nil {
							t.Error(err)
							return
						}
						if lag := uint64(e.log.Published() - e.log.Durable()); lag > maxLag.Load() {
							maxLag.Store(lag)
						}
					}
				})
			}
			time.Sleep(300 * time.Millisecond)
			first := liveHeap()
			peak := first
			for range 5 {
				time.Sleep(200 * time.Millisecond)
				peak = max(peak, liveHeap())
			}
			close(stop)
			wg.Wait()
			st := e.Stats()
			t.Logf("commits %d, fsyncs %d, commit p50 %v max %v, heap %d KB → peak %d KB, max lag %d, checkpoints %d",
				st.Commits, st.FsyncLatency.Count, st.CommitLatency.P50, st.CommitLatency.Max, first>>10, peak>>10,
				maxLag.Load(), st.Checkpoints)

			if peak > first+16<<20 {
				t.Fatalf("the live heap grew from %d KB to %d KB", first>>10, peak>>10)
			}
			if deferred {
				if lag := maxLag.Load(); lag > maxDeferredLag+1024 {
					t.Fatalf("%d records without fsync; the brake is at %d", lag, maxDeferredLag)
				}
				return
			}
			if st.CommitLatency.P50 < syncDelay {
				t.Fatalf("commit p50 %v is below the fsync delay %v: writers did not wait", st.CommitLatency.P50, syncDelay)
			}
			if st.FsyncLatency.Count >= st.Commits {
				t.Fatalf("%d fsyncs for %d commits: no group commit", st.FsyncLatency.Count, st.Commits)
			}
		})
	}
}

// TestWALBacklogThrottle: sin checkpoints, cuando el WAL desde el último pasa de 2 × MAX_WAL_SIZE los escritores
// esperan; uno frenado sigue en cuanto termina un checkpoint, y uno que espera más de LOCK_TIMEOUT recibe
// ErrThrottled sin escribir nada.
func TestWALBacklogThrottle(t *testing.T) {
	cfg := testConfig()
	cfg.maxWALSize = 1 << 20
	cfg.checkEvery = time.Hour // sin checkpoints de fondo
	cfg.lockTimeout = 5 * time.Second
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	s := mustStore(t, e, "s")
	limit := cfg.maxWALSize * walBacklogFactor
	value := bytes.Repeat([]byte("x"), 1000)
	i := 0
	for ; e.walBacklog() <= limit; i++ {
		mustOK(t, s.Put(fmt.Sprintf("id-%03d", i%300), value))
	}
	if backlog := e.walBacklog(); backlog > limit+64<<10 {
		t.Fatalf("the WAL went %d bytes past the limit", backlog-limit)
	}

	done := make(chan error, 1)
	go func() { done <- s.Put("waiting", value) }()
	select {
	case err := <-done:
		t.Fatalf("the writer did not wait: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	mustOK(t, e.Checkpoint())
	select {
	case err := <-done:
		mustOK(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the checkpoint did not release the writer")
	}

	for ; e.walBacklog() <= limit; i++ {
		mustOK(t, s.Put(fmt.Sprintf("id-%03d", i%300), value))
	}
	e.cfg.lockTimeout = 50 * time.Millisecond // nadie más lo lee ahora: sin checkpointer ni limpieza de fondo
	if err := s.Put("late", value); !errors.Is(err, ErrThrottled) {
		t.Fatalf("put past the limit = %v; want ErrThrottled", err)
	}
	if _, found, err := s.Read("late"); err != nil || found {
		t.Fatalf("the throttled put wrote: %v %v", found, err)
	}
	if st := e.Stats(); st.Throttled < 2 {
		t.Fatalf("Stats.Throttled = %d", st.Throttled)
	}
	mustOK(t, e.Checkpoint())
	mustOK(t, s.Put("late", value))
}

// TestDeferredBrake: con un disco de fsync lento (crashfs.SetDelay) y sin escritor de fondo del WAL, un escritor de
// confirmaciones diferidas nunca deja más de maxDeferredLag registros sin fsync: el freno lo hace esperar.
func TestDeferredBrake(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	defer e.Close()
	s := mustStore(t, e, "s")
	fsys.SetDelay(0, 20*time.Millisecond)
	before := e.Stats().FsyncLatency.Count
	for i := range maxDeferredLag + 2000 {
		mustOK(t, s.Deferred().Put(fmt.Sprintf("id-%03d", i%200), []byte("v")))
		if lag := e.log.Published() - e.log.Durable(); lag > maxDeferredLag+1 {
			t.Fatalf("put %d: %d records without fsync", i, lag)
		}
	}
	if n := e.Stats().FsyncLatency.Count - before; n < 1 {
		t.Fatalf("%d fsyncs for %d deferred puts: the brake never waited", n, maxDeferredLag+2000)
	}
}
