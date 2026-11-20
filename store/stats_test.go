package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// TestStatsCounters: lecturas, escrituras, confirmaciones, lotes, snapshots y latencias suben; las páginas sucias
// bajan a 0 tras un Checkpoint, que se cuenta; Vacuum pone en 0 las escrituras desde la última limpieza.
func TestStatsCounters(t *testing.T) {
	e := openTest(t, newProc(crashfs.New()), testConfig())
	defer e.Close()
	s := mustStore(t, e, "a")

	before := e.Stats()
	if before.CacheSize != 64 || before.Checkpoints < 1 || before.LastCheckpoint.IsZero() {
		t.Fatalf("al abrir: %+v", before)
	}
	for i := range 20 {
		mustOK(t, s.Put(fmt.Sprintf("k%03d", i), []byte("valor")))
	}
	for i := range 10 {
		if _, _, err := s.Read(fmt.Sprintf("k%03d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Exists("k000"); err != nil {
		t.Fatal(err)
	}
	b := e.NewBatch()
	mustOK(t, s.PutIn(b, "lote", []byte("x")))
	v := e.Snapshot()

	st := e.Stats()
	if st.Writes-before.Writes != 21 || st.Commits-before.Commits != 20 || st.Reads-before.Reads != 11 {
		t.Fatalf("operaciones: antes %+v, después %+v", before, st)
	}
	if st.OpenBatches != 1 || st.OpenSnapshots != 1 {
		t.Fatalf("abiertos: lotes %d, snapshots %d", st.OpenBatches, st.OpenSnapshots)
	}
	if st.OldestXID >= st.NextXID || st.FrozenAge != st.OldestXID-st.FrozenXID {
		t.Fatalf("xids: %+v", st)
	}
	if st.CacheDirty == 0 || st.CachePages == 0 || st.CachePages > int64(st.CacheSize) {
		t.Fatalf("caché: %+v", st)
	}
	if st.WALSinceCheckpoint <= before.WALSinceCheckpoint || st.LSN <= before.LSN || st.DurableLSN > st.LSN {
		t.Fatalf("WAL: antes %+v, después %+v", before, st)
	}
	if st.ReadLatency.Count-before.ReadLatency.Count != 11 || st.CommitLatency.Count != st.Commits ||
		st.FsyncLatency.Count == 0 || st.ReadLatency.P99 < st.ReadLatency.P50 || st.ReadLatency.Max < st.ReadLatency.P99 {
		t.Fatalf("latencias: %+v", st)
	}
	if st.WritesSinceVacuum != st.Writes {
		t.Fatalf("escrituras desde la limpieza = %d, escrituras = %d", st.WritesSinceVacuum, st.Writes)
	}

	mustOK(t, b.Commit())
	mustOK(t, v.Close())
	mustOK(t, e.Checkpoint())
	mustOK(t, e.Vacuum())
	after := e.Stats()
	if after.Commits != st.Commits+1 || after.OpenBatches != 0 || after.OpenSnapshots != 0 {
		t.Fatalf("tras Commit: %+v", after)
	}
	if after.Checkpoints != st.Checkpoints+1 || after.LastCheckpointTook <= 0 {
		t.Fatalf("checkpoints: antes %d, después %d (%v)", st.Checkpoints, after.Checkpoints, after.LastCheckpointTook)
	}
	if after.Vacuums != st.Vacuums+1 {
		t.Fatalf("limpiezas: antes %d, después %d", st.Vacuums, after.Vacuums)
	}
	// La limpieza puede ensuciar páginas (congela); un segundo checkpoint las deja limpias.
	mustOK(t, e.Checkpoint())
	if after = e.Stats(); after.CacheDirty != 0 || after.WritesSinceVacuum != 0 {
		t.Fatalf("tras Checkpoint y Vacuum: sucias %d, escrituras desde la limpieza %d", after.CacheDirty, after.WritesSinceVacuum)
	}
}

// TestOnStatsStopsOnClose: OnStats llama periódicamente; tras Close no llama más, y stop detiene otra antes.
func TestOnStatsStopsOnClose(t *testing.T) {
	e := openTest(t, newProc(crashfs.New()), testConfig())
	var calls, stopped atomic.Int64
	got := make(chan Stats, 1)
	e.OnStats(time.Millisecond, func(st Stats) {
		calls.Add(1)
		select {
		case got <- st:
		default:
		}
	})
	var stop func()
	ready := make(chan struct{})
	stop = e.OnStats(time.Millisecond, func(Stats) {
		<-ready
		stopped.Add(1)
		stop() // desde fn: no debe trabarse
	})
	close(ready)

	select {
	case st := <-got:
		if st.CacheSize != 64 {
			t.Fatalf("Stats recibido: %+v", st)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnStats no llamó")
	}
	for calls.Load() < 3 {
		time.Sleep(time.Millisecond)
	}
	mustOK(t, e.Close())
	n := calls.Load()
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != n {
		t.Fatalf("OnStats siguió llamando tras Close: %d → %d", n, calls.Load())
	}
	if s := stopped.Load(); s != 1 {
		t.Fatalf("la detenida desde fn se llamó %d veces", s)
	}
	if stop := e.OnStats(time.Millisecond, func(Stats) { t.Error("llamada con el motor cerrado") }); stop == nil {
		t.Fatal("stop nil")
	}
	time.Sleep(10 * time.Millisecond)
}

// TestStatsConcurrent: Stats y OnStats a la vez que escrituras, lecturas, lotes y checkpoints (para -race).
func TestStatsConcurrent(t *testing.T) {
	e := openTest(t, newProc(crashfs.New()), testConfig())
	s := mustStore(t, e, "a")
	stop := e.OnStats(time.Millisecond, func(st Stats) { _ = st.CommitLatency.P99 })
	defer stop()

	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				id := fmt.Sprintf("g%d-%03d", g, i)
				if err := s.Put(id, []byte(id)); err != nil {
					t.Error(err)
					return
				}
				if _, _, err := s.Read(id); err != nil {
					t.Error(err)
					return
				}
				if i%50 == 0 {
					b := e.NewBatch()
					_ = s.PutIn(b, id+"-b", []byte("x"))
					_ = b.Commit()
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 5 {
			if err := e.Checkpoint(); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	quit, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-quit:
				return
			default:
			}
			st := e.Stats()
			if st.CachePages > int64(st.CacheSize) || st.OpenBatches < 0 {
				t.Errorf("foto imposible: %+v", st)
				return
			}
			time.Sleep(100 * time.Microsecond)
		}
	}()
	wg.Wait()
	close(quit)
	<-done
	st := e.Stats()
	if st.Writes != 816 || st.Reads != 800 {
		t.Fatalf("escrituras %d, lecturas %d", st.Writes, st.Reads)
	}
	mustOK(t, e.Close())
}
