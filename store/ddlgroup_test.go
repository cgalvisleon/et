package store

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// TestCreateStoresCrash (tarea 98): altas sueltas desde varias goroutines (comparten el fsync del WAL, el catálogo
// queda para el checkpoint y data/ no se sincroniza en cada alta) y un corte en una operación de disco al azar, en
// los tres modos de crashfs: al reabrir están todos los stores cuyo Store volvió bien antes del corte (con lo que se
// les escribió), los demás no dejan archivos y Verify da limpio.
func TestCreateStoresCrash(t *testing.T) {
	modes := []crashfs.Mode{crashfs.LoseUnsynced, crashfs.ReorderUnsynced, crashfs.TornWrites}
	for i, mode := range modes {
		for seed := range int64(4) {
			t.Run(fmt.Sprintf("%d/%d", i, seed), func(t *testing.T) {
				createStoresCrash(t, mode, seed)
			})
		}
	}
}

/**
* createStoresCrash es un caso de TestCreateStoresCrash.
* @param t *testing.T, mode crashfs.Mode, seed int64
**/
func createStoresCrash(t *testing.T, mode crashfs.Mode, seed int64) {
	base := crashfs.New()
	proc := newProc(base)
	cfg := testConfig()
	e := openTest(t, proc, cfg)
	mustStore(t, e, "first")
	mustOK(t, e.Checkpoint())
	rng := rand.New(rand.NewSource(seed))
	proc.killAfter(int64(20+rng.Intn(300)), mode, seed)
	var mu sync.Mutex
	var created []string
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Go(func() {
			for i := range 40 {
				name := fmt.Sprintf("g%d-%02d", g, i)
				s, err := e.Store(name)
				if err != nil {
					return
				}
				if err := s.Put("k", []byte(name)); err != nil {
					return
				}
				mu.Lock()
				created = append(created, name)
				mu.Unlock()
				if i%10 == 9 && g == 0 {
					if err := e.Checkpoint(); err != nil {
						return
					}
				}
			}
		})
	}
	wg.Wait()
	proc.kill(mode, seed) // si el corte no llegó, al final
	e.Close()

	e = openTest(t, newProc(base), cfg)
	defer e.Close()
	stores := e.Stores()
	for _, name := range created {
		if !slices.Contains(stores, name) {
			t.Fatalf("store %q was created before the cut and is gone", name)
		}
		v, found, err := mustStore(t, e, name).Read("k")
		if err != nil || !found || string(v) != name {
			t.Fatalf("store %q: %q %v %v", name, v, found, err)
		}
	}
	checkClean(t, e)
	t.Logf("%d stores before the cut, %d after reopening", len(created), len(stores))
}

// TestRemoveUnallocated (tarea 98): los archivos de datos con número ≥ NextFileNo (de un alta cuyo registro se
// perdió) se borran al abrir; los de un número ya dado y sin store siguen siendo huérfanos para Verify.
func TestRemoveUnallocated(t *testing.T) {
	base := crashfs.New()
	e := openTest(t, base, testConfig())
	mustStore(t, e, "a")
	mustOK(t, e.Close())
	for _, name := range []string{"0000000040.0001", "0000000040_vm.0001", "0000000041.0001"} {
		writeFile(t, base, filepath.Join(testDir, smgr.DataDir, name), make([]byte, page.Size))
	}
	e = openTest(t, base, testConfig())
	defer e.Close()
	names, err := base.ReadDir(filepath.Join(testDir, smgr.DataDir))
	mustOK(t, err)
	for _, name := range names {
		if file, ok := parseDataName(name); ok && file >= 40 {
			t.Fatalf("%s was not removed", name)
		}
	}
	checkClean(t, e)
}
