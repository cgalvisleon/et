package wal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// smallOptions: segmentos del tamaño mínimo que admite Open (MaxRecordSize), para rotar con pocos registros.
func smallOptions(spares int) Options {
	return Options{SegmentSize: MaxRecordSize, Spares: spares}
}

// testRecord es el registro número i de una historia de prueba: su contenido sale de i, así se puede comparar.
func testRecord(i int) Record {
	delta := bytes.Repeat([]byte{byte(i)}, i*37%1500+1)
	return Record{
		XID:   page.XID(i),
		Kind:  KindLeafWrite,
		Flags: FlagCommit,
		Blocks: []BlockRef{{
			File: smgr.FileNo(i%7 + 1), Fork: smgr.ForkMain, Block: page.BlockNumber(i), Flags: BlockWillInit,
			Delta: delta,
		}},
		Main: fmt.Appendf(nil, "main-%d", i),
	}
}

// cloneRecord copia un registro leído (sus bytes son vistas del búfer de Scan).
func cloneRecord(rec *Record) Record {
	out := *rec
	out.Blocks = make([]BlockRef, len(rec.Blocks))
	for i, b := range rec.Blocks {
		b.Image = bytes.Clone(b.Image)
		b.Delta = bytes.Clone(b.Delta)
		out.Blocks[i] = b
	}
	out.Main = bytes.Clone(rec.Main)
	return out
}

// sameRecord compara lo que se escribe de dos registros (PageLSN no se escribe).
func sameRecord(a, b *Record) bool {
	if a.LSN != b.LSN || a.XID != b.XID || a.Kind != b.Kind || a.Flags != b.Flags || !bytes.Equal(a.Main, b.Main) ||
		len(a.Blocks) != len(b.Blocks) {
		return false
	}
	for i := range a.Blocks {
		x, y := a.Blocks[i], b.Blocks[i]
		if x.File != y.File || x.Fork != y.Fork || x.Block != y.Block || blockFlags(&x) != blockFlags(&y) ||
			!bytes.Equal(x.Image, y.Image) || !bytes.Equal(x.Delta, y.Delta) {
			return false
		}
	}
	return true
}

// openTest abre el WAL de dir desde redo y devuelve los registros recorridos (copiados). Lo cierra al terminar.
func openTest(t testing.TB, fsys vfs.FS, dir string, opts Options, redo Position) (*WAL, Position, []Record) {
	t.Helper()
	var recs []Record
	w, end, err := Open(fsys, dir, opts, redo, func(rec *Record) error {
		recs = append(recs, cloneRecord(rec))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	return w, end, recs
}

// appendRecord reserva y escribe rec; devuelve su lsn.
func appendRecord(t testing.TB, w *WAL, rec Record) page.LSN {
	t.Helper()
	r, err := w.Reserve(&rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Write(); err != nil {
		t.Fatal(err)
	}
	return r.LSN()
}

// writeHistory escribe los registros 1..n de testRecord.
func writeHistory(t testing.TB, w *WAL, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		appendRecord(t, w, testRecord(i))
	}
}

// checkHistory comprueba que recs sean los registros 1..n de testRecord, con lsn 1..n.
func checkHistory(t testing.TB, recs []Record, n int) {
	t.Helper()
	if len(recs) != n {
		t.Fatalf("registros: %d, se esperaban %d", len(recs), n)
	}
	for i := range recs {
		want := testRecord(i + 1)
		want.LSN = page.LSN(i + 1)
		if !sameRecord(&recs[i], &want) {
			t.Fatalf("registro %d distinto: %+v", i+1, recs[i])
		}
	}
}

// segPath es la ruta del segmento num del motor de dir.
func segPath(dir string, num uint64) string {
	return segmentPath(filepath.Join(dir, Dir), num)
}

// fileSize es el largo de un archivo.
func fileSize(t testing.TB, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

// segments son los números de los segmentos del motor de dir.
func segments(t testing.TB, dir string) []uint64 {
	t.Helper()
	nums, err := listSegments(vfs.OS(), filepath.Join(dir, Dir))
	if err != nil {
		t.Fatal(err)
	}
	return nums
}

// encoded es rec codificado y sellado con lsn.
func encoded(t testing.TB, rec Record, lsn page.LSN) []byte {
	t.Helper()
	size, err := recordSize(&rec)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, size)
	encodeRecord(&rec, buf)
	sealRecord(buf, lsn)
	return buf
}
