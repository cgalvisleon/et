package fsm

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// TestAllocateExtendsWhenEmpty: con la lista vacía Allocate extiende el fork; Free apila (la última liberada sale
// primero) y Allocate saca de la lista una página vacía, bloqueada y con el lsn de su registro.
func TestAllocateExtendsWhenEmpty(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)

	for want := page.BlockNumber(1); want <= 4; want++ {
		if got := allocate(t, e.fsm, 1); got != want {
			t.Fatalf("Allocate = %d, se esperaba %d (extender)", got, want)
		}
	}
	free(t, e.fsm, 1, 2)
	free(t, e.fsm, 1, 4)
	if list := freeList(t, e.pool, 1); !slices.Equal(list, []page.BlockNumber{4, 2}) {
		t.Fatalf("lista = %v, se esperaba [4 2]", list)
	}

	h, err := e.fsm.Allocate(1, 9)
	if err != nil {
		t.Fatal(err)
	}
	p := h.Page()
	o, err := p.Opaque()
	if h.Tag().Block != 4 || err != nil || o != (page.Opaque{}) || p.LSN() != e.log.Published() {
		t.Fatalf("Allocate = bloque %d, opaque %+v (%v), lsn %d; se esperaba 4, vacío, lsn %d",
			h.Tag().Block, o, err, p.LSN(), e.log.Published())
	}
	release(h)
	if list := freeList(t, e.pool, 1); !slices.Equal(list, []page.BlockNumber{2}) {
		t.Fatalf("lista = %v, se esperaba [2]", list)
	}
	if got := allocate(t, e.fsm, 1); got != 2 {
		t.Fatalf("Allocate = %d, se esperaba 2", got)
	}
	if got := allocate(t, e.fsm, 1); got != 5 {
		t.Fatalf("Allocate = %d, se esperaba 5 (lista vacía otra vez)", got)
	}
}

// TestFreeRejects: no se libera la meta ni una página ya libre, y sin meta no hay lista.
func TestFreeRejects(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)
	b := allocate(t, e.fsm, 1)
	free(t, e.fsm, 1, b)

	if err := e.fsm.Free(1, 0, 1); !errors.Is(err, ErrFreeMeta) {
		t.Fatalf("Free(meta) = %v, se esperaba ErrFreeMeta", err)
	}
	before := e.log.Published()
	if err := e.fsm.Free(1, b, 1); !errors.Is(err, ErrDoubleFree) {
		t.Fatalf("Free dos veces = %v, se esperaba ErrDoubleFree", err)
	}
	if e.log.Published() != before {
		t.Fatal("un Free rechazado escribió en el WAL")
	}

	if err := e.sm.Create(2, smgr.ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := e.sm.Extend(2, smgr.ForkMain, 0, make(page.Page, page.Size)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.fsm.Allocate(2, 1); !errors.Is(err, ErrNoMeta) {
		t.Fatalf("Allocate sin meta = %v, se esperaba ErrNoMeta", err)
	}
}

// TestConcurrentAllocateFree: muchas goroutines piden y devuelven páginas a la vez en dos stores; ninguna página se
// entrega dos veces y al final cada página es de alguien o está en la lista, una sola vez.
func TestConcurrentAllocateFree(t *testing.T) {
	e := openEnv(t, crashfs.New())
	files := []smgr.FileNo{1, 2}
	for _, f := range files {
		createStore(t, e.sm, f)
	}

	const workers, rounds = 8, 150
	var mu sync.Mutex
	held := map[smgr.FileNo]map[page.BlockNumber]bool{1: {}, 2: {}}
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			file := files[w%len(files)]
			var mine []page.BlockNumber
			for r := range rounds {
				if r%3 != 2 || len(mine) == 0 {
					h, err := e.fsm.Allocate(file, page.XID(w+1))
					if err != nil {
						errs <- err
						return
					}
					b := h.Tag().Block
					release(h)
					mu.Lock()
					dup := held[file][b]
					held[file][b] = true
					mu.Unlock()
					if dup {
						errs <- errors.New("una página se entregó dos veces")
						return
					}
					mine = append(mine, b)
					continue
				}
				b := mine[r%len(mine)]
				mine = slices.DeleteFunc(mine, func(x page.BlockNumber) bool { return x == b })
				mu.Lock()
				delete(held[file], b)
				mu.Unlock()
				if err := e.fsm.Free(file, b, page.XID(w+1)); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	for _, f := range files {
		list := freeList(t, e.pool, f)
		n, err := e.sm.NBlocks(f, smgr.ForkMain)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[page.BlockNumber]bool{}
		for _, b := range list {
			if held[f][b] {
				t.Fatalf("store %d: bloque %d está en la lista y entregado", f, b)
			}
			seen[b] = true
		}
		if len(seen)+len(held[f]) != int(n)-1 {
			t.Fatalf("store %d: %d libres + %d entregadas != %d páginas sin la meta", f, len(seen), len(held[f]), n-1)
		}
	}
}

// TestRecordsCarryImages: tras un checkpoint, el primer cambio de la meta lleva su imagen entera; la página libre
// (BlockWillInit) nunca la lleva.
func TestRecordsCarryImages(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)
	b := allocate(t, e.fsm, 1)
	free(t, e.fsm, 1, b)
	if _, err := e.log.BeginCheckpoint(nil); err != nil {
		t.Fatal(err)
	}
	redo := e.log.RedoPoint()
	b2 := allocate(t, e.fsm, 1) // saca b: la meta lleva imagen
	free(t, e.fsm, 1, b2)       // la meta ya cambió tras el checkpoint: sin imagen

	var images []bool
	_, err := wal.NewReader(e.fs, testDir).Scan(redo, func(rec *wal.Record) error {
		if rec.Blocks[1].Flags&wal.BlockHasImage != 0 {
			t.Errorf("la página libre del registro %d lleva imagen", rec.LSN)
		}
		images = append(images, rec.Blocks[0].Flags&wal.BlockHasImage != 0)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(images, []bool{true, false}) {
		t.Fatalf("imágenes de la meta = %v, se esperaba [true false]", images)
	}
}
