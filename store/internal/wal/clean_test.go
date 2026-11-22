package wal

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Pruebas de Options.CleanEnd (tarea 109): tras un Close limpio, Open no corta la cola ni los repuestos preasignados.

// cleanDir es el directorio del motor en las pruebas de CleanEnd.
const cleanDir = "/db"

/**
* cleanHistory escribe una historia con preasignación que recicla segmentos (los repuestos guardan registros viejos
* con crc válido), deja el siguiente preparado y cierra limpio.
* @param t testing.TB, fsys vfs.FS, opts Options
* @return Position: el punto de rehacer desde donde reabrir, int: registros escritos
**/
func cleanHistory(t testing.TB, fsys vfs.FS, opts Options) (Position, int) {
	t.Helper()
	w, _, _ := openTest(t, fsys, cleanDir, opts, Start)
	const n = 300
	var redo Position
	for i := 1; i <= n; i++ {
		if err := w.Room(); err != nil {
			t.Fatal(err)
		}
		appendRecord(t, w, testRecord(i))
		if i == 250 {
			var err error
			if redo, err = w.BeginCheckpoint(nil); err != nil {
				t.Fatal(err)
			}
			if err := w.Recycle(redo); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Room(); err != nil {
		t.Fatal(err)
	}
	if w.maxNum <= w.active.num {
		t.Fatalf("sin repuestos: activo %d, mayor %d", w.active.num, w.maxNum)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return redo, n
}

/**
* checkFrom comprueba que recs sean los registros from, from+1… de testRecord, con su lsn.
* @param t testing.TB, recs []Record, from int
**/
func checkFrom(t testing.TB, recs []Record, from int) {
	t.Helper()
	for i := range recs {
		want := testRecord(from + i)
		want.LSN = page.LSN(from + i)
		if !sameRecord(&recs[i], &want) {
			t.Fatalf("registro %d distinto: lsn %d", from+i, recs[i].LSN)
		}
	}
}

/**
* segmentSizes es el largo de cada segmento del motor.
* @param t testing.TB, fsys vfs.FS
* @return map[uint64]int64
**/
func segmentSizes(t testing.TB, fsys vfs.FS) map[uint64]int64 {
	t.Helper()
	nums, err := listSegments(fsys, cleanDir+"/"+Dir)
	if err != nil {
		t.Fatal(err)
	}
	sizes := map[uint64]int64{}
	for _, num := range nums {
		if sizes[num], err = segmentSize(fsys, cleanDir+"/"+Dir, num); err != nil {
			t.Fatal(err)
		}
	}
	return sizes
}

// openCost es lo que cuesta en disco reabrir y escribir el primer registro durable.
type openCost struct{ syncs, truncates, written int64 }

/**
* reopenCost reabre el WAL con cleanEnd (0 = como tras un corte), pide lugar, escribe un registro durable y devuelve
* lo que costó en disco.
* @param t testing.TB, cleanEnd page.LSN
* @return openCost, map[uint64]int64: el largo de los segmentos justo después de Open
**/
func reopenCost(t testing.TB, cleanEnd page.LSN) (openCost, map[uint64]int64) {
	t.Helper()
	fsys := &syncCountFS{FS: crashfs.New()}
	opts := preallocOptions(2)
	redo, n := cleanHistory(t, fsys, opts)
	fsys.reset()
	opts.CleanEnd = cleanEnd
	if cleanEnd == 1 { // el final de la historia
		opts.CleanEnd = page.LSN(n + 1)
	}
	w, end, recs := openTest(t, fsys, cleanDir, opts, redo)
	if end.LSN != page.LSN(n+1) || len(recs) != n-int(redo.LSN)+1 {
		t.Fatalf("fin %+v con %d registros", end, len(recs))
	}
	checkFrom(t, recs, int(redo.LSN))
	sizes := segmentSizes(t, fsys)
	if err := w.Room(); err != nil {
		t.Fatal(err)
	}
	lsn := appendRecord(t, w, testRecord(n+1))
	if err := w.WaitDurable(lsn); err != nil {
		t.Fatal(err)
	}
	return openCost{fsys.syncs.Load(), fsys.truncates.Load(), fsys.written.Load()}, sizes
}

// TestCleanEndKeepsPrealloc: con CleanEnd igual al final, Open no trunca nada y el primer registro no vuelve a
// rellenar con ceros el activo ni el siguiente; sin CleanEnd, o con uno que no coincide, se corta como siempre.
func TestCleanEndKeepsPrealloc(t *testing.T) {
	before := segmentSizesAfterClose(t)
	clean, sizes := reopenCost(t, 1)
	for num, size := range sizes {
		if size != before[num] {
			t.Fatalf("segmento %d: medía %d y tras Open %d", num, before[num], size)
		}
	}
	if clean.truncates != 0 || clean.syncs != 1 || clean.written > 4<<10 {
		t.Fatalf("cierre limpio: %+v", clean)
	}
	cut, _ := reopenCost(t, 0)
	mismatch, _ := reopenCost(t, 7) // no es el final: se corta
	opts := preallocOptions(2)
	for _, c := range []openCost{cut, mismatch} {
		if c.truncates < 2 || c.written < opts.SegmentSize/2 {
			t.Fatalf("sin CleanEnd tenía que cortar y rellenar: %+v", c)
		}
	}
	t.Logf("cierre limpio: %+v; cortando: %+v", clean, cut)
}

/**
* segmentSizesAfterClose da el largo de los segmentos de cleanHistory recién cerrado.
* @param t testing.TB
* @return map[uint64]int64
**/
func segmentSizesAfterClose(t testing.TB) map[uint64]int64 {
	t.Helper()
	fsys := crashfs.New()
	cleanHistory(t, fsys, preallocOptions(2))
	return segmentSizes(t, fsys)
}

// TestCleanEndCrash: cierre limpio → Open sin cortar → escribir (con rotaciones a repuestos reciclados, que guardan
// registros viejos con crc válido) → corte en los tres modos → Open (cortando, como tras un corte): un prefijo exacto
// de la historia, con todo lo confirmado, sin registros fantasma ni huecos falsos; y se sigue escribiendo.
func TestCleanEndCrash(t *testing.T) {
	for name, mode := range crashModes {
		for seed := int64(0); seed < 6; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				cleanEndCrash(t, mode, seed)
			})
		}
	}
}

/**
* cleanEndCrash es un caso de TestCleanEndCrash.
* @param t *testing.T, mode crashfs.Mode, seed int64
**/
func cleanEndCrash(t *testing.T, mode crashfs.Mode, seed int64) {
	fsys := crashfs.New()
	opts := preallocOptions(2)
	redo, n := cleanHistory(t, fsys, opts)
	opts.CleanEnd = page.LSN(n + 1)
	w, end, _ := openTest(t, fsys, cleanDir, opts, redo)
	first := w.active.num
	const more, confirmed = 160, 90
	for i := n + 1; i <= n+more; i++ {
		if err := w.Room(); err != nil {
			t.Fatal(err)
		}
		appendRecord(t, w, testRecord(i))
		if i == n+confirmed {
			if err := w.WaitDurable(page.LSN(i)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if w.active.num < first+2 {
		t.Fatalf("no rotó a los repuestos: activo %d desde %d", w.active.num, first)
	}
	durable := int(w.Durable())
	fsys.Crash(mode, seed)
	w.Close()

	opts.CleanEnd = 0 // tras un corte el control no dice cerrado bien
	w, end, recs := openTest(t, fsys, cleanDir, opts, redo)
	got := int(redo.LSN) + len(recs) - 1
	if got < durable || got > n+more || end.LSN != page.LSN(got+1) {
		t.Fatalf("tras el corte: hasta %d (durable %d, escritos %d), fin %+v", got, durable, n+more, end)
	}
	checkFrom(t, recs, int(redo.LSN))
	for i := got + 1; i <= got+60; i++ { // se sigue escribiendo y se cierra limpio otra vez
		if err := w.Room(); err != nil {
			t.Fatal(err)
		}
		appendRecord(t, w, testRecord(i))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	opts.CleanEnd = page.LSN(got + 61)
	_, end, recs = openTest(t, fsys, cleanDir, opts, redo)
	if end.LSN != opts.CleanEnd {
		t.Fatalf("tras cerrar limpio: fin %+v, se esperaba lsn %d", end, opts.CleanEnd)
	}
	checkFrom(t, recs, int(redo.LSN))
}

// TestCleanEndOpenBench mide en el disco real (STORE_BENCH=1) lo que cuesta reabrir tras un Close limpio y escribir
// el primer registro durable, con y sin CleanEnd, con los valores del motor: segmentos de 128 MB y 4 repuestos.
func TestCleanEndOpenBench(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	opts := Options{SegmentSize: 128 << 20, Spares: 4, Preallocate: true, Buffer: 1 << 20}
	for _, clean := range []bool{false, true, false, true} {
		dir := t.TempDir()
		fsys := &syncCountFS{FS: vfs.OS()}
		w, _, _ := openTest(t, fsys, dir, opts, Start)
		big := testRecord(1)
		big.Blocks[0].Delta = make([]byte, 60<<10)
		for w.active.num <= uint64(opts.Spares) { // Spares segmentos llenos, que el checkpoint recicla como repuestos
			if err := w.Room(); err != nil {
				t.Fatal(err)
			}
			appendRecord(t, w, big)
		}
		redo, err := w.BeginCheckpoint(nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Recycle(redo); err != nil {
			t.Fatal(err)
		}
		if w.maxNum != w.active.num+uint64(opts.Spares) {
			t.Fatalf("repuestos: activo %d, mayor %d", w.active.num, w.maxNum)
		}
		if err := w.Room(); err != nil {
			t.Fatal(err)
		}
		last := w.LastLSN()
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}

		fsys.reset()
		o := opts
		if clean {
			o.CleanEnd = last + 1
		}
		start := time.Now()
		w, _, _ = openTest(t, fsys, dir, o, redo)
		opened := time.Since(start)
		if err := w.Room(); err != nil {
			t.Fatal(err)
		}
		lsn := appendRecord(t, w, testRecord(2))
		if err := w.WaitDurable(lsn); err != nil {
			t.Fatal(err)
		}
		first := time.Since(start)
		t.Logf("CleanEnd %v: Open %v, Open + primer registro durable %v; fsync %d, truncados %d, escritos %d MB",
			clean, opened, first, fsys.syncs.Load(), fsys.truncates.Load(), fsys.written.Load()>>20)
		w.Close()
	}
}
