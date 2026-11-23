package btree

import (
	"testing"

	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// FuzzApplyDelta: aplicar cualquier delta de cualquier Kind de btree (en cualquiera de los cinco papeles de bloque) a
// una hoja con datos, de la forma 1 (i/5 par) o de la 2 (impar), nunca entra en pánico; o lo rechaza con un error o
// deja una página con punteros sanos (salvo una meta, que no tiene punteros) y, si sigue siendo una hoja, que se lee.
func FuzzApplyDelta(f *testing.F) {
	leaves := []page.Page{samplePage(f), samplePage2(f)}
	entry := encodeEntry([]byte("id"), Write{XID: 3, Value: []byte("value")})
	right := splitRightDelta(page.Opaque{Prev: 1, Flags: page.OpaqueLeaf}, [][]byte{entry}, false)
	f.Add(uint8(wal.KindLeafWrite), uint8(0), itemDelta(1, entry))
	f.Add(uint8(wal.KindSplit), uint8(0), splitLeftDelta(5, 1, encodePivot(key{id: []byte("m"), lsn: MaxLSN}, 0)))
	f.Add(uint8(wal.KindSplit), uint8(1), right)
	f.Add(uint8(wal.KindInsertPivot), uint8(0), itemDelta(2, encodePivot(key{id: []byte("m"), lsn: 9}, 7)))
	f.Add(uint8(wal.KindNewRoot), uint8(0), newRootDelta(1, 2, encodePivot(key{id: []byte("m"), lsn: MaxLSN}, 3)))
	f.Add(uint8(wal.KindLeafVacuum), uint8(0), vacuumDelta(pageVacuum{remove: []page.OffsetNumber{2}, freeze: []page.OffsetNumber{3, 4}, setVisible: true}))
	f.Add(uint8(wal.KindLeafVacuum), uint8(0), vacuumDelta(pageVacuum{setVisible: true}))
	f.Add(uint8(wal.KindSetFrozen), uint8(0), le.AppendUint64(nil, 99))
	f.Add(uint8(wal.KindInitStore), uint8(0), initMetaDelta(1, 42, MetaLeavesForm2))
	f.Add(uint8(wal.KindInsertPivot), uint8(2), metaFastDelta(4, 1))
	f.Add(uint8(wal.KindMarkHalfDead), uint8(0), blockDelta(5))
	f.Add(uint8(wal.KindMarkHalfDead), uint8(1), halfDeadParentDelta(halfDeadPlan{off: 2, child: 3, right: 4}))
	unlink := unlinkTargetDelta(unlinkTarget{prev: 2, next: 3, info: deletedInfo{safe: 8, next: 6}, flags: unlinkHasLeft})
	f.Add(uint8(wal.KindUnlinkPage), uint8(roleTarget), unlink)
	f.Add(uint8(wal.KindUnlinkPage), uint8(roleRight), blockDelta(2))
	f.Add(uint8(wal.KindUnlinkPage), uint8(roleMeta), append(blockDelta(5), metaFastDelta(3, 0)...))
	f.Add(uint8(wal.KindUnlinkPage), uint8(roleLeft), blockDelta(3))
	f.Add(uint8(wal.KindUnlinkPage), uint8(roleLeaf), blockDelta(0))
	f.Add(uint8(wal.KindRecyclePage), uint8(0), le.AppendUint32(blockDelta(0), 5))
	f.Add(uint8(wal.KindRecyclePage), uint8(1), blockDelta(7))
	// Hojas de la forma 2 (i ≥ 5 elige la de la forma 2)
	entry2 := encodeEntry2([]byte("idb"), Write{XID: 3, Value: []byte("value")})
	f.Add(uint8(wal.KindLeafWrite), uint8(5), leafWriteDelta(3, entry2))
	f.Add(uint8(wal.KindLeafWrite), uint8(5), leafWriteDelta(3, encodeEntry2([]byte("zid"), Write{XID: 3, Tombstone: true})))
	f.Add(uint8(wal.KindLeafRewrite), uint8(5), le.AppendUint16(nil, 1))
	f.Add(uint8(wal.KindLeafRewrite), uint8(0), le.AppendUint16(nil, 0))
	f.Add(uint8(wal.KindSplit), uint8(5), splitLeftDelta(5, 2, encodePivot(key{id: []byte("ic"), lsn: MaxLSN}, 0)))
	right2 := splitRightDelta(page.Opaque{Prev: 1, Next: 9, Flags: page.OpaqueLeaf}, [][]byte{
		encodePivot(key{id: []byte("zz"), lsn: MaxLSN}, 0), encodePrefixItem([]byte("i")), encodeEntry2([]byte("d"), Write{XID: 2}),
	}, true)
	f.Add(uint8(wal.KindSplit), uint8(6), right2)
	f.Add(uint8(wal.KindLeafVacuum), uint8(5), vacuumDelta(pageVacuum{remove: []page.OffsetNumber{3}, freeze: []page.OffsetNumber{4, 5}, setVisible: true}))
	f.Add(uint8(wal.KindInitStore), uint8(6), layoutDelta(page.LayoutVersion2))
	f.Add(uint8(wal.KindSetFrozen), uint8(5), le.AppendUint32(le.AppendUint64(nil, 99), uint32(MetaLeavesForm2)))
	f.Fuzz(func(t *testing.T, kind, i uint8, delta []byte) {
		p := append(page.Page(nil), leaves[int(i/5)%2]...)
		if err := applyBlock(wal.Kind(kind), int(i%5), p, delta, 77); err == nil && !isMeta(p) {
			if err := p.CheckItems(); err != nil {
				t.Fatalf("delta aceptado deja la página mal: %v", err)
			}
			if p.LayoutVersion() == page.LayoutVersion2 { // una hoja de la forma 2 que queda tiene que leerse entera
				n, err := readNode(1, p)
				for off := n.firstData(); err == nil && off <= n.maxOff(); off++ {
					_, err = n.entryAt(off)
				}
				if err != nil {
					t.Fatalf("delta aceptado deja una hoja de la forma 2 que no se lee: %v", err)
				}
			}
		}
	})
}

// FuzzReadNode: una página cualquiera con la suma correcta nunca hace entrar en pánico a la lectura del árbol.
func FuzzReadNode(f *testing.F) {
	f.Add([]byte(samplePage(f)))
	deleted := make(page.Page, page.Size)
	if err := initDeletedPage(deleted, unlinkTarget{prev: 2, next: 3, info: deletedInfo{safe: 9}}); err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(deleted))
	half := samplePage(f)
	for half.MaxOffset() > page.FirstOffset {
		if err := half.DeleteItem(half.MaxOffset()); err != nil {
			f.Fatal(err)
		}
	}
	if err := applyHalfDeadLeaf(half, blockDelta(4)); err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(half))
	f.Add([]byte(samplePage2(f)))
	frozen := samplePage2(f)
	if err := applyLeafVacuum(frozen, vacuumDelta(pageVacuum{freeze: []page.OffsetNumber{3, 4}})); err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(frozen))
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
					_ = e.reencode(len(n.pre), true)
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

// samplePage2 arma una hoja de la forma 2 con prefijo «i», clave alta y algunas entradas (una lápida y una con
// desborde) para las semillas: los datos empiezan en el ítem 3.
func samplePage2(t testing.TB) page.Page {
	c := leafContent{o: page.Opaque{Next: 9, Flags: page.OpaqueLeaf}, highKey: encodePivot(key{id: []byte("zz"), lsn: MaxLSN}, 0)}
	for _, w := range []struct {
		id string
		w  Write
	}{{"ia", Write{XID: 1, Value: []byte("v")}}, {"ib", Write{XID: 2, Tombstone: true}},
		{"ic", Write{XID: 3, Overflow: &overflow.Ref{First: 7, Len: 9000}}}, {"id", Write{XID: 4, Value: []byte("dd")}}} {
		e, err := parseEntry2(encodeEntry2([]byte(w.id), w.w), nil)
		if err != nil {
			t.Fatal(err)
		}
		c.entries = append(c.entries, e)
	}
	p := make(page.Page, page.Size)
	if err := c.write2(p, 1); err != nil {
		t.Fatal(err)
	}
	return p
}
