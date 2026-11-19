package store

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// versioned: el valor de la versión v de id, de largo variable, que dice de quién es (para detectar lecturas mezcladas).
func versioned(id string, v int) []byte {
	return bytes.Repeat(fmt.Appendf(nil, "%s=%d;", id, v), 1+v%7)
}

// TestReadUsesMap: un registro normal se lee del mapa del segmento (sellado o activo); uno más grande que el tamaño
// de segmento no cabe en el mapa y se lee con pread, hasta que al reabrir su segmento se mapea con su largo. Todos
// dan el valor correcto.
func TestReadUsesMap(t *testing.T) {
	dir := t.TempDir()
	opts := testOptions(1024, false)
	s := openTest(t, dir, opts)
	big := bytes.Repeat([]byte("b"), 5000)
	for i := range 40 {
		s.Put(fmt.Sprintf("k%02d", i), versioned("k", i))
	}
	s.Put("grande", big)
	s.Put("ultimo", []byte("activo"))

	check := func(s *Store, bigMapped bool) {
		t.Helper()
		for id, want := range map[string]bool{"k00": true, "k39": true, "ultimo": true, "grande": bigMapped} {
			r, _ := s.index.get(id)
			if _, mapped := s.segmentAt(r.segment).view(r.offset, int(r.size)); mapped != want {
				t.Fatalf("%s en el mapa = %v, quería %v", id, mapped, want)
			}
		}
		if got, _ := value(t, s, "grande"); !bytes.Equal(got, big) {
			t.Fatalf("grande: %d bytes", len(got))
		}
		if got, _ := value(t, s, "k07"); !bytes.Equal(got, versioned("k", 7)) {
			t.Fatalf("k07 = %q", got)
		}
		if got, _ := value(t, s, "ultimo"); string(got) != "activo" {
			t.Fatalf("ultimo = %q", got)
		}
	}
	if len(s.segmentList()) < 3 {
		t.Fatalf("solo %d segmentos: la prueba necesita rotar", len(s.segmentList()))
	}
	check(s, false)
	s.Close()
	check(openTest(t, dir, opts), true)
}

// TestReadWhileWriting: lectores leen ids al azar del segmento activo mientras escritores los reemplazan y los
// segmentos rotan; cada lectura debe ser una versión completa del id leído, nunca un registro a medias. Al reabrir
// queda la última versión de cada id.
func TestReadWhileWriting(t *testing.T) {
	dir := t.TempDir()
	opts := testOptions(8192, false)
	s := openTest(t, dir, opts)
	const writers, idsPerWriter, versions = 4, 20, 60
	for w := range writers {
		for i := range idsPerWriter {
			s.Put(fmt.Sprintf("w%d-%02d", w, i), versioned(fmt.Sprintf("w%d-%02d", w, i), 0))
		}
	}

	var wg sync.WaitGroup
	var stop atomic.Bool
	for w := range writers {
		wg.Go(func() {
			for v := 1; v <= versions; v++ {
				for i := range idsPerWriter {
					id := fmt.Sprintf("w%d-%02d", w, i)
					if err := s.Put(id, versioned(id, v)); err != nil {
						t.Error(err)
						return
					}
				}
			}
		})
	}
	var readers sync.WaitGroup
	var reads atomic.Int64
	for range 4 {
		readers.Go(func() {
			for !stop.Load() {
				id := fmt.Sprintf("w%d-%02d", rand.IntN(writers), rand.IntN(idsPerWriter))
				got, ok, err := s.Read(id)
				if err != nil || !ok || !bytes.HasPrefix(got, []byte(id+"=")) {
					t.Errorf("%s = %q, %v, %v", id, got, ok, err)
					return
				}
				reads.Add(1)
			}
		})
	}
	wg.Wait()
	stop.Store(true)
	readers.Wait()
	if len(s.segmentList()) < 3 || reads.Load() == 0 {
		t.Fatalf("segmentos %d, lecturas %d: la prueba no ejercitó rotación y lectura", len(s.segmentList()), reads.Load())
	}

	s.Close()
	s = openTest(t, dir, opts)
	for w := range writers {
		for i := range idsPerWriter {
			id := fmt.Sprintf("w%d-%02d", w, i)
			if got, _ := value(t, s, id); !bytes.Equal(got, versioned(id, versions)) {
				t.Fatalf("%s al reabrir = %q", id, got)
			}
		}
	}
}

// TestReadConcurrentWithClose: Close con lectores encima espera a que terminen antes de deshacer los mapas: ninguna
// lectura toca memoria desmapeada (eso tumbaría el proceso) y, después de Close, todas dan MSG_CLOSED.
func TestReadConcurrentWithClose(t *testing.T) {
	for round := range 20 {
		s, err := open(t.TempDir(), testOptions(4096, false))
		if err != nil {
			t.Fatal(err)
		}
		for i := range 200 {
			s.Put(fmt.Sprintf("k%03d", i), versioned("k", i))
		}
		var wg sync.WaitGroup
		var started sync.WaitGroup
		for range 8 {
			started.Add(1)
			wg.Go(func() {
				started.Done()
				for {
					id := fmt.Sprintf("k%03d", rand.IntN(200))
					got, ok, err := s.Read(id)
					if err != nil {
						if err.Error() != MSG_CLOSED {
							t.Errorf("vuelta %d: %v", round, err)
						}
						return
					}
					if !ok || !bytes.HasPrefix(got, []byte("k=")) {
						t.Errorf("vuelta %d: %s = %q, %v", round, id, got, ok)
						return
					}
				}
			})
		}
		started.Wait()
		runtime.Gosched()
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		if _, _, err := s.Read("k001"); err == nil || err.Error() != MSG_CLOSED {
			t.Fatalf("Read tras Close: %v", err)
		}
	}
}

// BenchmarkReadLatency: latencia de Read con todas las goroutines leyendo ids al azar (100.000 ids de 100 B):
// p50, p99 y p999 en nanosegundos. Incluye lo que cuesta medir (time.Now, unos 20 ns).
func BenchmarkReadLatency(b *testing.B) {
	s, err := open(b.TempDir(), testOptions(128<<20, false))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	const n = 100_000
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("k%09d", i)
		s.Put(ids[i], bytes.Repeat([]byte("v"), 100))
	}
	var mu sync.Mutex
	var all []time.Duration
	var seed atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewPCG(seed.Add(1), 2))
		mine := make([]time.Duration, 0, 1<<16)
		for pb.Next() {
			id := ids[rng.IntN(n)]
			start := time.Now()
			_, _, err := s.Read(id)
			mine = append(mine, time.Since(start))
			if err != nil {
				b.Error(err)
				return
			}
		}
		mu.Lock()
		all = append(all, mine...)
		mu.Unlock()
	})
	b.StopTimer()
	slices.Sort(all)
	for _, p := range []struct {
		name string
		q    float64
	}{{"p50-ns", 0.50}, {"p99-ns", 0.99}, {"p999-ns", 0.999}} {
		b.ReportMetric(float64(all[int(p.q*float64(len(all)-1))]), p.name)
	}
}
