package fsm

import (
	"errors"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// edgeBlocks son bloques en los bordes de un byte y de una página del mapa.
var edgeBlocks = []page.BlockNumber{
	0, 1, 3, 4, 7, 8,
	VMBlocksPerPage - 1, VMBlocksPerPage, VMBlocksPerPage + 1,
	2*VMBlocksPerPage - 1, 2 * VMBlocksPerPage,
}

// TestVMEdges: poner y borrar bits en los bordes de byte y de página no toca a los vecinos; un bloque fuera del mapa
// no tiene bits; congelado implica visible.
func TestVMEdges(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)

	if bits, err := e.fsm.Get(1, 5); err != nil || bits != 0 {
		t.Fatalf("Get sin mapa = %d, %v", bits, err)
	}
	want := map[page.BlockNumber]VMBits{}
	check := func() {
		t.Helper()
		for _, edge := range edgeBlocks {
			for b := max(edge, 2) - 2; b <= edge+2; b++ {
				bits, err := e.fsm.Get(1, b)
				if err != nil {
					t.Fatal(err)
				}
				if bits != want[b] {
					t.Fatalf("Get(%d) = %d, se esperaba %d", b, bits, want[b])
				}
			}
		}
	}

	for i, b := range edgeBlocks {
		bits := VMAllVisible
		if i%2 == 1 {
			bits = VMAllFrozen
		}
		if err := e.fsm.Set(1, b, bits); err != nil {
			t.Fatal(err)
		}
		want[b] = VMAllVisible
		if i%2 == 1 {
			want[b] = VMAllVisible | VMAllFrozen
		}
	}
	check()

	for i, b := range edgeBlocks {
		if i%3 == 0 {
			if err := e.fsm.Clear(1, b); err != nil {
				t.Fatal(err)
			}
			delete(want, b)
		}
	}
	check()

	if n, err := e.sm.NBlocks(1, smgr.ForkVM); err != nil || n != 3 {
		t.Fatalf("páginas del mapa = %d, %v; se esperaban 3", n, err)
	}
	if bits, err := e.fsm.Get(1, 3*VMBlocksPerPage); err != nil || bits != 0 {
		t.Fatalf("Get más allá del mapa = %d, %v", bits, err)
	}
}

// TestVMNoChangeNoRecord: poner bits que ya están o borrar los que no están no escribe en el WAL; bits inválidos
// se rechazan.
func TestVMNoChangeNoRecord(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)
	if err := e.fsm.Set(1, 10, VMAllFrozen); err != nil {
		t.Fatal(err)
	}
	before := e.log.Published()
	if err := e.fsm.Set(1, 10, VMAllVisible); err != nil {
		t.Fatal(err)
	}
	if err := e.fsm.Clear(1, 11); err != nil {
		t.Fatal(err)
	}
	if err := e.fsm.Clear(1, 5*VMBlocksPerPage); err != nil {
		t.Fatal(err)
	}
	if e.log.Published() != before {
		t.Fatalf("lsn %d → %d sin cambios en el mapa", before, e.log.Published())
	}
	for _, bits := range []VMBits{0, 4, 7} {
		if err := e.fsm.Set(1, 10, bits); !errors.Is(err, ErrBits) {
			t.Fatalf("Set(bits %d) = %v, se esperaba ErrBits", bits, err)
		}
	}
}

// TestVMConcurrent: muchas goroutines ponen y borran bits de bloques vecinos (los mismos bytes y páginas) a la vez y
// extienden el mapa al mismo tiempo; al final cada bloque tiene lo que dejó la última operación sobre él.
func TestVMConcurrent(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)

	const workers, perWorker = 8, 40
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perWorker {
				// Los bloques de cada goroutine se intercalan con los de las demás en los mismos bytes, y la
				// mitad cae en la segunda página del mapa.
				b := page.BlockNumber(i*workers + w)
				if i%2 == 1 {
					b += VMBlocksPerPage
				}
				if err := e.fsm.Set(1, b, VMAllFrozen); err != nil {
					errs <- err
					return
				}
				if i%4 == 0 {
					if err := e.fsm.Clear(1, b); err != nil {
						errs <- err
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	for w := range workers {
		for i := range perWorker {
			b := page.BlockNumber(i*workers + w)
			if i%2 == 1 {
				b += VMBlocksPerPage
			}
			want := VMAllVisible | VMAllFrozen
			if i%4 == 0 {
				want = 0
			}
			if bits, err := e.fsm.Get(1, b); err != nil || bits != want {
				t.Fatalf("Get(%d) = %d, %v; se esperaba %d", b, bits, err, want)
			}
		}
	}
	if n, err := e.sm.NBlocks(1, smgr.ForkVM); err != nil || n != 2 {
		t.Fatalf("páginas del mapa = %d, %v; se esperaban 2", n, err)
	}
}
