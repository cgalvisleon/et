package btree

import (
	"fmt"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// benchIDs son ids de índice de josefina (valor + jid): grupos de prefijo compartido.
func benchID(i int) string { return fmt.Sprintf("btidx_city/c%03d|%020d", i%100, i) }

// BenchmarkTreeLookup: Lookup de un id al azar en un árbol de 100.000 ids (tarea 134: lectura con las dos formas).
func BenchmarkTreeLookup(b *testing.B) {
	e := openEnv(b, crashfs.New(), wal.Start, 4096)
	tr := createTree(b, e)
	const n = 100000
	for i := range n {
		if _, err := tr.Write(benchID(i), Write{Value: []byte("v"), Commit: true}); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok, err := tr.Lookup(benchID((i*7919)%n), nil); err != nil || !ok {
			b.Fatal(err)
		}
	}
}

// BenchmarkTreeWrite: Write suelto de ids nuevos de índice (en grupos, al final de cada uno).
func BenchmarkTreeWrite(b *testing.B) {
	e := openEnv(b, crashfs.New(), wal.Start, 4096)
	tr := createTree(b, e)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := tr.Write(benchID(i), Write{Value: []byte("v"), Commit: true}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTreeScan: un recorrido entero de 100.000 ids.
func BenchmarkTreeScan(b *testing.B) {
	e := openEnv(b, crashfs.New(), wal.Start, 4096)
	tr := createTree(b, e)
	const n = 100000
	for i := range n {
		if _, err := tr.Write(benchID(i), Write{Value: []byte("v"), Commit: true}); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		if err := tr.Scan("", nil, func(v Version) (bool, error) { count++; return true, nil }); err != nil || count != n {
			b.Fatal(err, count)
		}
	}
}
