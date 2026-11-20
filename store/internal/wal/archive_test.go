package wal

import (
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestSwitchSegmentInfo: Switch sella el activo aunque no esté lleno (con el activo vacío no hace nada); SegmentInfo
// da rangos de lsn contiguos y el final válido de cada segmento sellado, también tras reabrir (recorriendo los que
// quedaron antes del punto de rehacer); un recorrido sigue de un segmento sellado con Switch al siguiente.
func TestSwitchSegmentInfo(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	opts := Options{SegmentSize: 4 * MaxRecordSize, Spares: 1, Preallocate: true}
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	writeHistory(t, w, 30)
	sealed, err := w.Switch()
	if err != nil || sealed != 1 || w.ActiveSegment() != 2 {
		t.Fatalf("Switch: %d %v, active %d", sealed, err, w.ActiveSegment())
	}
	if again, err := w.Switch(); err != nil || again != 0 {
		t.Fatalf("Switch with an empty active segment: %d %v", again, err)
	}
	for i := 31; i <= 200; i++ {
		appendRecord(t, w, testRecord(i))
	}
	if _, err := w.SegmentInfo(w.ActiveSegment()); err == nil {
		t.Fatal("SegmentInfo of the active segment")
	}
	check := func(w *WAL) {
		t.Helper()
		next := page.LSN(1)
		for num := uint64(1); num < w.ActiveSegment(); num++ {
			info, err := w.SegmentInfo(num)
			if err != nil {
				t.Fatal(err)
			}
			if info.First != next || info.End.LSN <= info.First || info.End.Offset <= 0 || info.End.Offset > opts.SegmentSize {
				t.Fatalf("segment %d: %+v, want first %d", num, info, next)
			}
			next = info.End.LSN
		}
	}
	check(w)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	info1 := func(w *WAL) SegmentInfo {
		info, err := w.SegmentInfo(1)
		if err != nil {
			t.Fatal(err)
		}
		return info
	}
	w, _, recs := openTest(t, fsys, dir, opts, Start)
	checkHistory(t, recs, 200)
	if info := info1(w); info.End.LSN != 31 || info.End.Offset != fileSizeOfHistory(t, 30) {
		t.Fatalf("segment 1 after reopening: %+v", info)
	}
	check(w)
	mustClose(t, w)

	// Desde un punto de rehacer posterior: el segmento 1 no se recorrió al abrir y SegmentInfo lo lee entero.
	second := firstOf(t, fsys, dir, 2)
	w, _, _ = openTest(t, fsys, dir, opts, Position{Segment: 2, Offset: 0, LSN: second})
	if _, known := w.sizes[1]; known {
		t.Fatal("segment 1 should not be known")
	}
	if info := info1(w); info.End.LSN != 31 {
		t.Fatalf("segment 1 read whole: %+v", info)
	}
	check(w)
}

// TestStopLSN: Open con StopLSN rehace hasta ese lsn, corta lo que sigue y escribe a continuación.
func TestStopLSN(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	opts := Options{SegmentSize: 4 * MaxRecordSize, Spares: 1, Preallocate: true}
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	writeHistory(t, w, 150)
	mustClose(t, w)

	stopped := opts
	stopped.StopLSN = 70
	w, end, recs := openTest(t, fsys, dir, stopped, Start)
	checkHistory(t, recs, 70)
	if end.LSN != 71 {
		t.Fatalf("end %+v", end)
	}
	if lsn := appendRecord(t, w, testRecord(71)); lsn != 71 {
		t.Fatalf("next lsn %d", lsn)
	}
	mustClose(t, w)
	_, _, recs = openTest(t, fsys, dir, opts, Start)
	checkHistory(t, recs, 71)
}

/**
* fileSizeOfHistory es el largo de los registros 1..n de testRecord codificados.
* @param t testing.TB, n int
* @return int64
**/
func fileSizeOfHistory(t testing.TB, n int) int64 {
	t.Helper()
	var total int64
	for i := 1; i <= n; i++ {
		total += int64(len(encoded(t, testRecord(i), page.LSN(i))))
	}
	return total
}

/**
* firstOf lee el lsn del primer registro del segmento num del motor de dir.
* @param t testing.TB, fsys vfs.FS, dir string, num uint64
* @return page.LSN
**/
func firstOf(t testing.TB, fsys vfs.FS, dir string, num uint64) page.LSN {
	t.Helper()
	lsn, ok, err := newSegmentScanner().firstLSN(fsys, dir+"/"+Dir, num)
	if err != nil || !ok {
		t.Fatalf("first record of segment %d: %v %v", num, ok, err)
	}
	return lsn
}

/**
* mustClose cierra el WAL y falla la prueba si no puede.
* @param t testing.TB, w *WAL
**/
func mustClose(t testing.TB, w *WAL) {
	t.Helper()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
