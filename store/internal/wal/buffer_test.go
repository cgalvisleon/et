package wal

import (
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestBufferLazy: con Options.Buffer, Write deja el registro en memoria (el archivo no crece y published no avanza)
// hasta que WaitDurable lo pide; entonces sale todo junto, con un solo pwrite, y es durable.
func TestBufferLazy(t *testing.T) {
	dir, fsys := "/db", crashfs.New()
	opts := smallOptions(0)
	opts.Buffer = 1 << 20
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	writeHistory(t, w, 20)
	if w.Published() != 0 {
		t.Fatalf("published %d antes de pedirlo", w.Published())
	}
	if size := vfsSize(t, fsys, segPath(dir, 1)); size != SegmentHeaderSize {
		t.Fatalf("el segmento mide %d antes de pedirlo", size)
	}
	if err := w.WaitDurable(20); err != nil {
		t.Fatal(err)
	}
	if w.Published() != 20 || w.Durable() != 20 {
		t.Fatalf("published %d durable %d", w.Published(), w.Durable())
	}
	fsys.Crash(crashfs.LoseUnsynced, 1)
	w.Close()
	_, _, recs := openTest(t, fsys, dir, opts, Start)
	checkHistory(t, recs, 20)
}

// TestBufferFull: lo pendiente que llega a Options.Buffer se escribe sin que nadie lo pida.
func TestBufferFull(t *testing.T) {
	dir, fsys := "/db", crashfs.New()
	opts := smallOptions(0)
	opts.Buffer = 2 << 10
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	writeHistory(t, w, 40)
	if w.Published() == 0 {
		t.Fatal("con el búfer lleno no se escribió nada")
	}
	w.Close()
	_, _, recs := openTest(t, fsys, dir, opts, Start)
	checkHistory(t, recs, 40) // Close escribe lo que quedaba
}

// TestBufferConcurrentWriters: escritores concurrentes con búfer, algunos esperando su fsync y otros no, a través de
// varias rotaciones: al cerrar y reabrir están todos, en orden de lsn y sin huecos. Un Write nunca espera a otra
// reserva (escribe lo contiguo y deja lo demás al dueño del hueco).
func TestBufferConcurrentWriters(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	opts := smallOptions(0)
	opts.Buffer = 8 << 10
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	const writers, each = 16, 200
	var mu sync.Mutex
	byLSN := map[page.LSN]Record{}
	var wg sync.WaitGroup
	for g := range writers {
		wg.Go(func() {
			for i := range each {
				rec := testRecord(g*each + i + 1)
				r, err := w.Reserve(&rec, nil)
				if err != nil {
					t.Error(err)
					return
				}
				if err := r.Write(); err != nil {
					t.Error(err)
					return
				}
				if g%2 == 0 && i%10 == 0 {
					if err := w.WaitDurable(r.LSN()); err != nil {
						t.Error(err)
						return
					}
				}
				mu.Lock()
				byLSN[r.LSN()] = rec
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if n := len(segments(t, dir)); n < 3 {
		t.Fatalf("no rotó: %d segmentos", n)
	}
	w.Close()
	_, end, recs := openTest(t, fsys, dir, opts, Start)
	if len(recs) != writers*each || end.LSN != writers*each+1 {
		t.Fatalf("al reabrir: %d registros, fin %+v", len(recs), end)
	}
	for i := range recs {
		want := byLSN[recs[i].LSN]
		if recs[i].LSN != page.LSN(i+1) || !sameRecord(&recs[i], &want) {
			t.Fatalf("registro %d distinto", i+1)
		}
	}
}
