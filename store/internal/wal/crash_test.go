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
	for name, mode := range crashModes {
		for seed := int64(0); seed < 8; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				fsys := crashfs.New()
				w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
				writeHistory(t, w, confirmed)
				if err := w.WaitDurable(confirmed); err != nil {
					t.Fatal(err)
				}
				for i := confirmed + 1; i <= total; i++ { // sin esperar durabilidad
					appendRecord(t, w, testRecord(i))
				}
				durable := w.Durable()

				fsys.Crash(mode, seed)
				w.Close()
				w, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
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
				_, _, again := openTest(t, fsys, dir, smallOptions(0), Start)
				if len(again) < int(lsn) {
					t.Fatalf("tras el segundo corte: %d registros, confirmado %d", len(again), lsn)
				}
				checkHistory(t, again, len(again))
			})
		}
	}
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
