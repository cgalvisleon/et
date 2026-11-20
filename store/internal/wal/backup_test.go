package wal

import (
	"errors"
	"testing"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestForceImages: en modo respaldo, una página cambiada después del punto de rehacer pero antes del inicio del
// respaldo necesita su imagen en el próximo cambio; al soltarlo, vuelve a decidir solo el punto de rehacer. End da
// la posición siguiente al registro.
func TestForceImages(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	writeHistory(t, w, 5)
	leaf := func() Record {
		return Record{Kind: KindLeafWrite, XID: 1, Blocks: []BlockRef{{File: 1, Block: 7, PageLSN: 3, Delta: []byte("d")}}}
	}
	from, release := w.ForceImages()
	if from != 6 || w.Holds() != 1 {
		t.Fatalf("from %d, holds %d", from, w.Holds())
	}
	rec := leaf()
	if _, err := w.Reserve(&rec, nil); !errors.Is(err, ErrNeedImage) {
		t.Fatalf("sin imagen en modo respaldo: %v", err)
	}
	release()
	release() // una sola vez
	if w.Holds() != 0 {
		t.Fatalf("holds %d tras soltar", w.Holds())
	}
	rec = leaf()
	r, err := w.Reserve(&rec, nil)
	if err != nil {
		t.Fatalf("sin modo respaldo: %v", err)
	}
	end := r.End()
	if err := r.Write(); err != nil {
		t.Fatal(err)
	}
	if end.LSN != r.LSN()+1 || end.Segment != w.active.num || end.Offset != w.active.size {
		t.Fatalf("End %+v, activo %d/%d", end, w.active.num, w.active.size)
	}
}

// TestHold: Recycle no toca los segmentos retenidos; al soltar la retención, el próximo Recycle sí.
func TestHold(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	writeHistory(t, w, 400)
	first := segments(t, dir)[0]
	release := w.Hold(first)
	redo, _ := w.BeginCheckpoint(nil)
	if err := w.Recycle(redo); err != nil {
		t.Fatal(err)
	}
	if got := segments(t, dir); got[0] != first {
		t.Fatalf("recicló un segmento retenido: %v", got)
	}
	release()
	if err := w.Recycle(redo); err != nil {
		t.Fatal(err)
	}
	if got := segments(t, dir); got[0] != redo.Segment {
		t.Fatalf("no recicló tras soltar: %v (redo %d)", got, redo.Segment)
	}
}
