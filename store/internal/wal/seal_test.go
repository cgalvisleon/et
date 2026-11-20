package wal

import (
	"errors"
	"os"
	"testing"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* flipByte invierte un byte del segmento num del motor de dir.
* @param t testing.TB, dir string, num uint64, off int64
**/
func flipByte(t testing.TB, dir string, num uint64, off int64) {
	t.Helper()
	f, err := os.OpenFile(segPath(dir, num), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b := make([]byte, 1)
	if _, err := f.ReadAt(b, off); err != nil {
		t.Fatal(err)
	}
	b[0] ^= 0xff
	if _, err := f.WriteAt(b, off); err != nil {
		t.Fatal(err)
	}
}

/**
* openErr abre el WAL de dir desde Start y devuelve el error de Open (cerrando el WAL si abrió).
* @param dir string, opts Options
* @return int: registros recorridos, error
**/
func openErr(dir string, opts Options) (int, error) {
	n := 0
	w, _, err := Open(vfs.OS(), dir, opts, Start, func(*Record) error { n++; return nil })
	if err == nil {
		w.Close()
	}
	return n, err
}

// Un registro dañado en medio de un segmento sellado con Switch, con el siguiente vacío: Open falla con
// ErrCorrupted y no corta nada (tarea 90). Antes cortaba en silencio y perdía los 6 registros durables siguientes.
func TestSealedCorruptionFails(t *testing.T) {
	for _, prealloc := range []bool{true, false} {
		dir := t.TempDir()
		opts := Options{SegmentSize: 4 * MaxRecordSize, Spares: 1, Preallocate: prealloc}
		w, _, _ := openTest(t, vfs.OS(), dir, opts, Start)
		writeHistory(t, w, 10)
		if err := w.WaitDurable(10); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Switch(); err != nil {
			t.Fatal(err)
		}
		mustClose(t, w)
		size := fileSize(t, segPath(dir, 1))
		flipByte(t, dir, 1, fileSizeOfHistory(t, 4)+HeaderSize+3) // dentro del registro 5
		n, err := openErr(dir, opts)
		if !errors.Is(err, ErrCorrupted) {
			t.Fatalf("prealloc=%v: Open with a damaged sealed segment: %d records, %v; want ErrCorrupted", prealloc, n, err)
		}
		if got := fileSize(t, segPath(dir, 1)); got != size {
			t.Fatalf("prealloc=%v: the failed Open cut segment 1 from %d to %d bytes", prealloc, size, got)
		}
	}
}

// El primer registro dañado de un segmento sellado que sigue a otro entero: el corte cae al final del anterior, pero
// queda un segmento sellado después, así que Open falla.
func TestSealedAfterEndFails(t *testing.T) {
	dir := t.TempDir()
	opts := Options{SegmentSize: 4 * MaxRecordSize, Spares: 1, Preallocate: true}
	w, _, _ := openTest(t, vfs.OS(), dir, opts, Start)
	writeHistory(t, w, 10)
	if _, err := w.Switch(); err != nil {
		t.Fatal(err)
	}
	writeHistoryFrom(t, w, 11, 20)
	if _, err := w.Switch(); err != nil {
		t.Fatal(err)
	}
	mustClose(t, w)
	flipByte(t, dir, 2, HeaderSize+3)
	if n, err := openErr(dir, opts); !errors.Is(err, ErrCorrupted) {
		t.Fatalf("Open with the first record of sealed segment 2 damaged: %d records, %v; want ErrCorrupted", n, err)
	}
}

// Una cola rota en el segmento activo (sin sellar) sigue siendo el final del WAL: Open corta ahí sin error, también
// cuando el activo es un repuesto reciclado que conserva la marca de sellado de su número viejo.
func TestTornTailActiveStillCut(t *testing.T) {
	t.Run("fresh", func(t *testing.T) {
		dir := t.TempDir()
		opts := Options{SegmentSize: 4 * MaxRecordSize, Spares: 1, Preallocate: true}
		w, _, _ := openTest(t, vfs.OS(), dir, opts, Start)
		writeHistory(t, w, 10)
		mustClose(t, w)
		flipByte(t, dir, 1, fileSizeOfHistory(t, 4)+HeaderSize+3)
		if n, err := openErr(dir, opts); err != nil || n != 4 {
			t.Fatalf("Open with a torn tail in the active segment: %d records, %v; want 4, nil", n, err)
		}
	})
	t.Run("recycled", func(t *testing.T) {
		dir := t.TempDir()
		opts := smallOptions(2)
		w, _, _ := openTest(t, vfs.OS(), dir, opts, Start)
		i := 0
		for w.active.num < 3 { // los segmentos 1 y 2 quedan sellados, con su marca
			i++
			appendRecord(t, w, testRecord(i))
		}
		redo, err := w.BeginCheckpoint(nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Recycle(redo); err != nil {
			t.Fatal(err)
		}
		for w.active.num < 4 { // el activo pasa al repuesto 4 (el 1 reciclado, con la marca del número 1)
			i++
			appendRecord(t, w, testRecord(i))
		}
		if _, ok, _ := readTrailer(vfs.OS(), dirOf(dir), 4); ok {
			t.Fatal("a recycled segment kept a valid seal mark")
		}
		start := w.active.size
		appendRecord(t, w, testRecord(i+1))
		appendRecord(t, w, testRecord(i+2))
		mustClose(t, w)
		flipByte(t, dir, 4, start+HeaderSize+3)
		var n int
		w2, end, err := Open(vfs.OS(), dir, opts, redo, func(*Record) error { n++; return nil })
		if err != nil {
			t.Fatalf("Open with a torn tail in a recycled active segment: %v", err)
		}
		defer w2.Close()
		if end.Segment != 4 || end.Offset != start {
			t.Fatalf("end %+v, want segment 4 offset %d", end, start)
		}
	})
}

/**
* dirOf es el directorio del WAL del motor de dir.
* @param dir string
* @return string
**/
func dirOf(dir string) string {
	return dir + string(os.PathSeparator) + Dir
}
