package btree

import (
	"bytes"
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// lastWrite es la escritura de un id con el mayor lsn: la que gana.
type lastWrite struct {
	lsn   page.LSN
	value []byte
	tomb  bool
}

// TestConcurrentWritersAndReaders: varios escritores sobre ids que se pisan (también ids crecientes, por la
// inserción rápida), lectores con Lookup y Scan hacia los dos lados, y checkpoints (con el vaciado de la caché) a la
// vez, con una caché chica y el escritor de fondo. Al final el árbol tiene, por id, la escritura de mayor lsn: nada se
// pierde ni se duplica.
func TestConcurrentWritersAndReaders(t *testing.T) {
	e := openEnvWith(t, crashfs.New(), wal.Start, 96, time.Millisecond)
	tr := createTree(t, e)

	const writers, perWriter = 8, 400
	var mu sync.Mutex
	final := map[string]lastWrite{}
	record := func(id string, res Result, value []byte, tomb bool) {
		mu.Lock()
		defer mu.Unlock()
		if res.Applied && res.LSN > final[id].lsn {
			final[id] = lastWrite{lsn: res.LSN, value: value, tomb: tomb}
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, writers+4)
	stop := make(chan struct{})
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(w)))
			for i := range perWriter {
				id := fmt.Sprintf("shared-%03d", rng.Intn(300))
				if w%2 == 0 {
					id = fmt.Sprintf("seq-%d-%06d", w, i) // crecientes por escritor
				}
				value := randomValue(rng, 150)
				tomb := rng.Intn(10) == 0
				res, err := tr.Write(id, Write{XID: 1, Value: value, Tombstone: tomb, Commit: true})
				if err != nil {
					errs <- err
					return
				}
				record(id, res, value, tomb)
				if v, _, err := tr.Lookup(id, nil); err != nil || (res.Applied && v.LSN < res.LSN) {
					errs <- fmt.Errorf("Lookup(%s) tras escribir lsn %d: lsn %d, %v", id, res.LSN, v.LSN, err)
					return
				}
			}
		}()
	}
	var readers sync.WaitGroup
	for r := range 3 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if err := scanOrdered(tr, r%2 == 1); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	readers.Add(1)
	go func() { // checkpoints: las escrituras siguientes a cada uno llevan imagen (y alguna repite por ErrNeedImage)
		defer readers.Done()
		for {
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Millisecond):
			}
			if _, err := e.log.BeginCheckpoint(nil); err != nil {
				errs <- err
				return
			}
			if err := e.pool.FlushAll(nil); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(stop)
	readers.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	m := model{}
	for id, w := range final {
		if !w.tomb {
			m[id] = w.value
		}
	}
	checkTree(t, tr, m)
}

/**
* scanOrdered recorre el árbol entero (hacia adelante o atrás) con escritores en curso y comprueba que los ids salgan
* estrictamente ordenados: ni repetidos ni fuera de lugar.
* @param tr *BTree, backward bool
* @return error
**/
func scanOrdered(tr *BTree, backward bool) error {
	var last string
	first := true
	fn := func(v Version) (bool, error) {
		if !first && ((!backward && v.ID <= last) || (backward && v.ID >= last)) {
			return false, fmt.Errorf("recorrido (atrás %v): %q después de %q", backward, v.ID, last)
		}
		last, first = v.ID, false
		return true, nil
	}
	if backward {
		return tr.ScanBackward("", nil, fn)
	}
	return tr.Scan("", nil, fn)
}

// TestRangeScans: recorridos desde puntos al azar, hacia adelante y hacia atrás, con límite, comparados con el
// modelo, en un árbol con borrados y varias versiones por id.
func TestRangeScans(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 24)
	tr := createTree(t, e)
	m := model{}
	rng := rand.New(rand.NewSource(3))
	for range 3000 {
		id := randomID(rng, 6)
		if rng.Intn(6) == 0 {
			if _, live := m[id]; live {
				remove(t, tr, m, id)
			}
			continue
		}
		put(t, tr, m, id, randomValue(rng, 60))
	}
	ids := m.sortedIDs()
	for range 300 {
		from, limit := randomID(rng, 6), 1+rng.Intn(80)
		i, _ := slices.BinarySearch(ids, from)
		want := ids[i:min(i+limit, len(ids))]
		if got := scanAll(t, tr, from, limit); !slices.Equal(got, want) {
			t.Fatalf("Scan(%q, %d): %v, se esperaba %v", from, limit, got, want)
		}
		j, found := slices.BinarySearch(ids, from)
		if found {
			j++
		}
		var back []string
		for k := j - 1; k >= 0 && len(back) < limit; k-- {
			back = append(back, ids[k])
		}
		if got := scanBack(t, tr, from, limit); !slices.Equal(got, back) {
			t.Fatalf("ScanBackward(%q, %d): %v, se esperaba %v", from, limit, got, back)
		}
	}
	// Los valores que entrega Scan son los del modelo.
	err := tr.Scan("", nil, func(v Version) (bool, error) {
		if !bytes.Equal(v.Value, m[v.ID]) {
			return false, fmt.Errorf("Scan: %q = %q, se esperaba %q", v.ID, v.Value, m[v.ID])
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestModelRandomOps: operaciones al azar (escribir, borrar, leer, recorrer, verificar) sobre ids que se repiten
// mucho, comparadas paso a paso con el modelo, con varias semillas.
func TestModelRandomOps(t *testing.T) {
	for seed := range int64(4) {
		t.Run(fmt.Sprint("seed", seed), func(t *testing.T) {
			e := openEnv(t, crashfs.New(), wal.Start, 16)
			tr := createTree(t, e)
			m := model{}
			rng := rand.New(rand.NewSource(seed))
			for range 1500 {
				id := randomID(rng, 3)
				switch op := rng.Intn(20); {
				case op < 10:
					put(t, tr, m, id, randomValue(rng, 300))
				case op < 14:
					remove(t, tr, m, id)
				case op < 18:
					v, ok, err := tr.Lookup(id, nil)
					want, live := m[id]
					if err != nil || ok != live || !bytes.Equal(v.Value, want) {
						t.Fatalf("Lookup(%q) = %q %v %v; se esperaba %q %v", id, v.Value, ok, err, want, live)
					}
				case op < 19:
					ids := m.sortedIDs()
					i, _ := slices.BinarySearch(ids, id)
					if got := scanAll(t, tr, id, 10); !slices.Equal(got, ids[i:min(i+10, len(ids))]) {
						t.Fatalf("Scan(%q): %v", id, got)
					}
				default:
					if _, err := tr.Verify(); err != nil {
						t.Fatal(err)
					}
				}
			}
			checkTree(t, tr, m)
		})
	}
}
