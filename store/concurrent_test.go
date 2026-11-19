package store

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestConcurrent: muchas goroutines escriben y leen a la vez en varios stores, con ids propios y compartidos, valores
// chicos y grandes y checkpoints de fondo cada pocos milisegundos. Al final cada goroutine encuentra lo suyo, Count
// coincide con el contenido y los árboles pasan Verify, también tras reabrir.
func TestConcurrent(t *testing.T) {
	fsys := crashfs.New()
	cfg := testConfig()
	cfg.checkpointTimeout = 5 * time.Millisecond
	cfg.checkEvery = time.Millisecond
	e := openTest(t, fsys, cfg)
	stores := []*Store{mustStore(t, e, "a"), mustStore(t, e, "b"), mustStore(t, e, "c")}

	const workers, opsEach = 16, 400
	own := make([]map[string]string, workers) // ids propios de cada goroutine → valor final ("" = borrado)
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			own[w] = concurrentWorker(t, stores, w, opsEach)
		}()
	}
	wg.Wait()
	checkOwn(t, stores, own)
	engineState(t, e)
	verifyTrees(t, e)
	mustOK(t, e.Close())

	e = openTest(t, fsys, cfg)
	defer e.Close()
	stores = []*Store{mustStore(t, e, "a"), mustStore(t, e, "b"), mustStore(t, e, "c")}
	checkOwn(t, stores, own)
	engineState(t, e)
}

/**
* concurrentWorker hace operaciones al azar: sobre sus ids propios lleva el modelo; sobre los compartidos solo
* comprueba que no haya errores y que lo leído sea un valor entero de alguien.
* @param t *testing.T, stores []*Store, w int: número de goroutine, n int: operaciones
* @return map[string]string: store/id → valor final de sus ids propios ("" = no existe)
**/
func concurrentWorker(t *testing.T, stores []*Store, w, n int) map[string]string {
	rng := rand.New(rand.NewPCG(uint64(w), 7))
	model := map[string]string{}
	for range n {
		s := stores[rng.IntN(len(stores))]
		shared := rng.IntN(3) == 0
		id := fmt.Sprintf("w%02d-%02d", w, rng.IntN(30))
		if shared {
			id = fmt.Sprintf("shared-%02d", rng.IntN(10))
		}
		value := wholeValue(id, rng)
		key := s.Name() + "/" + id
		var err error
		switch rng.IntN(6) {
		case 0, 1:
			err = s.Put(id, []byte(value))
			model[key] = value
		case 2:
			var ok bool
			if ok, err = s.Insert(id, []byte(value)); ok {
				model[key] = value
			}
		case 3:
			var ok bool
			if ok, err = s.Delete(id); ok || !shared {
				model[key] = ""
			}
		case 4:
			var ok bool
			if ok, err = s.Update(id, []byte(value)); ok {
				model[key] = value
			}
		default:
			var got []byte
			var found bool
			got, found, err = s.Read(id)
			if err == nil && found && !isWhole(id, got) {
				t.Errorf("read %s: torn value %.40q", key, got)
			}
		}
		if err != nil {
			t.Errorf("worker %d %s: %v", w, key, err)
			return nil
		}
	}
	return model
}

/**
* wholeValue arma un valor que se reconoce entero: el id repetido, chico o grande (desborde).
* @param id string, rng *rand.Rand
* @return string
**/
func wholeValue(id string, rng *rand.Rand) string {
	n := 1 + rng.IntN(10)
	if rng.IntN(10) == 0 {
		n = 300 + rng.IntN(2000)
	}
	return strings.Repeat(id+";", n)
}

/**
* isWhole dice si v es un valor de wholeValue para id.
* @param id string, v []byte
* @return bool
**/
func isWhole(id string, v []byte) bool {
	unit := []byte(id + ";")
	return len(v) > 0 && len(v)%len(unit) == 0 && bytes.Equal(v, bytes.Repeat(unit, len(v)/len(unit)))
}

/**
* checkOwn comprueba los ids propios de cada goroutine.
* @param t *testing.T, stores []*Store, own []map[string]string
**/
func checkOwn(t *testing.T, stores []*Store, own []map[string]string) {
	t.Helper()
	byName := map[string]*Store{}
	for _, s := range stores {
		byName[s.Name()] = s
	}
	for _, model := range own {
		for key, want := range model {
			if strings.Contains(key, "shared") {
				continue
			}
			name, id, _ := strings.Cut(key, "/")
			got, found, err := byName[name].Read(id)
			if err != nil || found != (want != "") || string(got) != want {
				t.Fatalf("%s = %v %v (%d B); want %d B", key, found, err, len(got), len(want))
			}
		}
	}
}

// TestInsertRace: muchas goroutines hacen Insert del mismo id a la vez (esperándose entre transacciones): gana
// exactamente una.
func TestInsertRace(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	defer e.Close()
	s := mustStore(t, e, "s")
	for round := range 20 {
		id := fmt.Sprintf("id-%d", round)
		var wins atomic.Int32
		var wg sync.WaitGroup
		for g := range 32 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ok, err := s.Insert(id, []byte(fmt.Sprint(g)))
				if err != nil {
					t.Error(err)
				}
				if ok {
					wins.Add(1)
				}
			}()
		}
		wg.Wait()
		if wins.Load() != 1 {
			t.Fatalf("round %d: %d inserts won", round, wins.Load())
		}
	}
	if s.Count() != 20 {
		t.Fatalf("count = %d", s.Count())
	}
}

// TestMemoryBounded (regla 1): la memoria del motor no crece con las claves. Sobre el disco real (que no ocupa
// montículo) y con una caché fija, escribir diez veces más claves no agrega más que un margen chico al montículo vivo.
func TestMemoryBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	cfg := testConfig()
	cfg.syncOnWrite = false
	e, err := open(vfs.OS(), t.TempDir(), cfg)
	mustOK(t, err)
	defer e.Close()
	s := mustStore(t, e, "s")
	value := bytes.Repeat([]byte("v"), 100)
	write := func(from, to int) {
		for i := from; i < to; i++ {
			mustOK(t, s.Put(fmt.Sprintf("key-%09d", i), value))
		}
		mustOK(t, e.Checkpoint())
	}
	write(0, 5000)
	before := liveHeap()
	write(5000, 50000)
	after := liveHeap()
	if growth := int64(after) - int64(before); growth > 2<<20 {
		t.Fatalf("heap grew %d bytes from 5.000 to 50.000 keys", growth)
	}
	if s.Count() != 50000 || len(e.handles) != 1 || len(e.counts.snapshot()) != 1 {
		t.Fatal("engine structures grew with the keys")
	}
}

/**
* liveHeap es el montículo vivo tras dos recolecciones.
* @return uint64
**/
func liveHeap() uint64 {
	var m runtime.MemStats
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}
