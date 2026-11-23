package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* TestReadCachedBench mide Read con los datos en caché (tarea 68 de F-23) sobre el disco real: carga
* STORE_BENCH_KEYS claves (1 millón por omisión) de 100 B en orden, como test/storebench, y lee 1.000 claves
* repartidas por el árbol una y otra vez (ya en la caché). Reporta p50/p99/p999, ns/op y asignaciones por lectura.
* Con STORE_BENCH_CPUPROFILE o STORE_BENCH_MEMPROFILE escribe el perfil de solo las lecturas. STORE_BENCH_DIR elige
* el directorio (se borra al terminar). Solo corre con STORE_BENCH=1.
* @param t *testing.T
**/
func TestReadCachedBench(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	n := 1_000_000
	if v, err := strconv.Atoi(os.Getenv("STORE_BENCH_KEYS")); err == nil && v > 0 {
		n = v
	}
	root := os.Getenv("STORE_BENCH_DIR")
	if root == "" {
		root = t.TempDir()
	}
	dir := filepath.Join(root, "readbench")
	defer os.RemoveAll(dir)
	cfg := testConfig()
	cfg.buffers = 16384 // 128 MB, como STORE_CACHE_SIZE por omisión
	cfg.wal = wal.Options{SegmentSize: 64 << 20, Spares: 2, Preallocate: true, Buffer: 1 << 20}
	e, err := open(vfs.OS(), dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	s := mustStore(t, e, "bench")
	start := time.Now()
	loadKeys(t, e, s, n)
	t.Logf("%d claves cargadas en %v", n, time.Since(start).Round(time.Millisecond))

	const hot = 1000
	ids := make([]string, hot)
	for i := range ids {
		ids[i] = benchKey(i * (n / hot)) // repartidas por todas las hojas
	}
	for _, id := range ids {
		if _, ok, err := s.Read(id); err != nil || !ok {
			t.Fatalf("read %s: %v %v", id, ok, err)
		}
	}

	times := make([]time.Duration, 0, hot*200)
	for i := range hot * 200 {
		t0 := time.Now()
		_, _, err := s.Read(ids[i%hot])
		times = append(times, time.Since(t0))
		if err != nil {
			t.Fatal(err)
		}
	}
	slices.Sort(times)
	p := func(q int) time.Duration { return times[len(times)*q/1000] }
	t.Logf("lectura en caché (%d claves × 200): p50 %v p99 %v p999 %v", hot, p(500), p(990), p(999))

	stopProfiles := startProfiles(t)
	const loops = 2_000_000
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	t0 := time.Now()
	for i := range loops {
		if _, _, err := s.Read(ids[i%hot]); err != nil {
			t.Fatal(err)
		}
	}
	took := time.Since(t0)
	runtime.ReadMemStats(&after)
	stopProfiles()
	t.Logf("%d lecturas: %.0f ns/op, %.2f allocs/op, %.0f B/op", loops, float64(took.Nanoseconds())/loops,
		float64(after.Mallocs-before.Mallocs)/loops, float64(after.TotalAlloc-before.TotalAlloc)/loops)
}

/**
* loadKeys carga n claves de 100 B en orden en lotes de 20.000 sin esperar fsync, y al final espera lo durable y
* hace un checkpoint.
* @param t *testing.T, e *Engine, s *Store, n int
**/
func loadKeys(t *testing.T, e *Engine, s *Store, n int) {
	value := make([]byte, 100)
	for i := range value {
		value[i] = byte(i*31 + 7)
	}
	for from := 0; from < n; from += 20000 {
		b := e.NewBatch()
		for i := from; i < min(from+20000, n); i++ {
			if err := s.PutIn(b, benchKey(i), value); err != nil {
				t.Fatal(err)
			}
		}
		if err := b.CommitDeferred(); err != nil {
			t.Fatal(err)
		}
	}
	mustOK(t, e.WaitDurable())
	mustOK(t, e.Checkpoint())
}

/**
* startProfiles arranca los perfiles pedidos por STORE_BENCH_CPUPROFILE y STORE_BENCH_MEMPROFILE (asignaciones).
* @param t *testing.T
* @return func(): los cierra y escribe
**/
func startProfiles(t *testing.T) func() {
	cpu, mem := os.Getenv("STORE_BENCH_CPUPROFILE"), os.Getenv("STORE_BENCH_MEMPROFILE")
	var cpuFile *os.File
	if cpu != "" {
		f, err := os.Create(cpu)
		mustOK(t, err)
		mustOK(t, pprof.StartCPUProfile(f))
		cpuFile = f
	}
	rate := runtime.MemProfileRate
	if mem != "" {
		runtime.MemProfileRate = 512
	}
	return func() {
		if cpuFile != nil {
			pprof.StopCPUProfile()
			cpuFile.Close()
		}
		if mem != "" {
			f, err := os.Create(mem)
			mustOK(t, err)
			mustOK(t, pprof.Lookup("allocs").WriteTo(f, 0))
			f.Close()
			runtime.MemProfileRate = rate
		}
	}
}

// benchKey es el id i, como el de test/storebench.
func benchKey(i int) string { return fmt.Sprintf("k%010d", i) }
