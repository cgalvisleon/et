package btree

import (
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// FuzzApplyDelta: aplicar cualquier delta de cualquier Kind de btree a una hoja con datos nunca entra en pánico; o
// lo rechaza con un error o deja una página con punteros sanos (salvo una meta, que no tiene punteros).
func FuzzApplyDelta(f *testing.F) {
	leaf := samplePage(f)
	entry := encodeEntry([]byte("id"), Write{XID: 3, Value: []byte("value")})
	right := splitRightDelta(page.Opaque{Prev: 1, Flags: page.OpaqueLeaf}, [][]byte{entry})
	f.Add(uint8(wal.KindLeafWrite), uint8(0), itemDelta(1, entry))
	f.Add(uint8(wal.KindSplit), uint8(0), splitLeftDelta(5, 1, encodePivot(key{id: []byte("m"), lsn: MaxLSN}, 0)))
	f.Add(uint8(wal.KindSplit), uint8(1), right)
	f.Add(uint8(wal.KindInsertPivot), uint8(0), itemDelta(2, encodePivot(key{id: []byte("m"), lsn: 9}, 7)))
	f.Add(uint8(wal.KindNewRoot), uint8(0), newRootDelta(1, 2, encodePivot(key{id: []byte("m"), lsn: MaxLSN}, 3)))
	f.Fuzz(func(t *testing.T, kind, i uint8, delta []byte) {
		p := append(page.Page(nil), leaf...)
		if err := applyBlock(wal.Kind(kind), int(i%3), p, delta, 77); err == nil && !isMeta(p) {
			if err := p.CheckItems(); err != nil {
				t.Fatalf("delta aceptado deja la página mal: %v", err)
			}
		}
	})
}

// FuzzReadNode: una página cualquiera con la suma correcta nunca hace entrar en pánico a la lectura del árbol.
func FuzzReadNode(f *testing.F) {
	f.Add([]byte(samplePage(f)))
	f.Fuzz(func(t *testing.T, b []byte) {
		p := make(page.Page, page.Size)
		copy(p, b)
		n, err := readNode(1, p)
		if err != nil {
			return
		}
		k := searchKey([]byte("id"))
		_, _ = n.lowerBound(k)
		_, _ = n.pastHighKey(k)
		if n.leaf() {
			for off := n.firstData(); off <= n.maxOff(); off++ {
				if e, err := n.entryAt(off); err == nil {
					_ = e.version()
				}
			}
		} else {
			_, _ = n.childFor(k)
		}
	})
}

// isMeta dice si la página es una meta.
func isMeta(p page.Page) bool {
	o, err := p.Opaque()
	return err == nil && o.Flags&page.OpaqueMeta != 0
}

// samplePage arma una hoja con algunas entradas (y clave alta) para las semillas.
func samplePage(t testing.TB) page.Page {
	p := make(page.Page, page.Size)
	if err := initTreePage(p, page.Opaque{Next: 9, Flags: page.OpaqueLeaf}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddItem(encodePivot(key{id: []byte("zz"), lsn: MaxLSN}, 0)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b", "c"} {
		if _, err := p.AddItem(encodeEntry([]byte(id), Write{XID: 1, Value: []byte(id + id)})); err != nil {
			t.Fatal(err)
		}
	}
	return p
}
