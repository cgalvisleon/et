package wal

import (
	"fmt"
	"os"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
)

// crashModes son los tres modos de corte de crashfs, con su nombre para los subtests.
var crashModes = map[string]crashfs.Mode{
	"LoseUnsynced":    crashfs.LoseUnsynced,
	"ReorderUnsynced": crashfs.ReorderUnsynced,
	"TornWrites":      crashfs.TornWrites,
}

// TestDurableSurvivesCrash: lo confirmado con WaitDurable sobrevive a un corte en los tres modos, también a través de
// varias rotaciones de segmento. Lo escrito después sin esperar puede perderse, reordenarse o quedar roto: la
// recuperación devuelve un prefijo exacto de la historia, corta la cola en el primer registro inválido y se sigue
// escribiendo a continuación.
func TestDurableSurvivesCrash(t *testing.T) {
	const dir, confirmed, total = "/db", 150, 300
	for _, buffer := range []int64{0, 4 << 10} { // sin búfer y con búfer (Options.Buffer, tarea 100)
		for name, mode := range crashModes {
			for seed := int64(0); seed < 8; seed++ {
				t.Run(fmt.Sprintf("buffer%d/%s/seed%d", buffer, name, seed), func(t *testing.T) {
					durableCrash(t, dir, confirmed, total, buffer, mode, seed)
				})
			}
		}
	}
}

/**
* durableCrash es un caso de TestDurableSurvivesCrash.
* @param t *testing.T, dir string, confirmed, total int, buffer int64, mode crashfs.Mode, seed int64
**/
func durableCrash(t *testing.T, dir string, confirmed, total int, buffer int64, mode crashfs.Mode, seed int64) {
	opts := smallOptions(0)
	opts.Buffer = buffer
	fsys := crashfs.New()
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	writeHistory(t, w, confirmed)
	if err := w.WaitDurable(page.LSN(confirmed)); err != nil {
		t.Fatal(err)
	}
	for i := confirmed + 1; i <= total; i++ { // sin esperar durabilidad
		appendRecord(t, w, testRecord(i))
	}
	durable := w.Durable()

	fsys.Crash(mode, seed)
	w.Close()
	w, end, recs := openTest(t, fsys, dir, opts, Start)
	if len(recs) < int(durable) || len(recs) > total {
		t.Fatalf("tras el corte: %d registros, durable %d", len(recs), durable)
	}
	checkHistory(t, recs, len(recs))
	t.Logf("recuperados %d de %d (durable %d)", len(recs), total, durable)
	if end.LSN != page.LSN(len(recs)+1) {
		t.Fatalf("fin %+v con %d registros", end, len(recs))
	}
	if size := vfsSize(t, fsys, segPath(dir, end.Segment)); size != end.Offset {
		t.Fatalf("la cola no se cortó: el segmento %d mide %d, fin %d", end.Segment, size, end.Offset)
	}

	// Se sigue escribiendo y lo nuevo también sobrevive a otro corte.
	lsn := appendRecord(t, w, testRecord(len(recs)+1))
	if lsn != end.LSN {
		t.Fatalf("lsn al seguir %d, se esperaba %d", lsn, end.LSN)
	}
	if err := w.WaitDurable(lsn); err != nil {
		t.Fatal(err)
	}
	fsys.Crash(mode, seed+100)
	w.Close()
	_, _, again := openTest(t, fsys, dir, opts, Start)
	if len(again) < int(lsn) {
		t.Fatalf("tras el segundo corte: %d registros, confirmado %d", len(again), lsn)
	}
	checkHistory(t, again, len(again))
}

// vfsSize es el largo de un archivo de fsys.
func vfsSize(t *testing.T, fsys crashfs.FS, path string) int64 {
	t.Helper()
	f, err := fsys.OpenFile(path, os.O_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	size, err := f.Size()
	if err != nil {
		t.Fatal(err)
	}
	return size
}
