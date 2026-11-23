package btree

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// Pruebas de las hojas de la forma 2 (formato 2, tarea 134 de F-23): prefijo compartido, cabecera corta, la lectura de
// las dos formas y la migración perezosa desde la forma 1.

/**
* encodeForm1 arma la entrada de la forma 1 de una versión leída de cualquier forma (para bajar una hoja a la forma 1
* en las pruebas, como la escribía el formato 1).
* @param e entry
* @return []byte
**/
func encodeForm1(e entry) []byte {
	flags := e.flags()
	id := e.appendID(nil)
	b := make([]byte, EntryHeaderSize, EntryHeaderSize+len(id)+len(e.tail()))
	le.PutUint32(b[OffEntryFlags:], uint32(flags))
	le.PutUint16(b[OffEntryIDLen:], uint16(len(id)))
	le.PutUint64(b[OffEntryLSN:], uint64(e.lsn()))
	le.PutUint64(b[OffEntryXID:], uint64(e.xid()))
	b = append(b, id...)
	switch {
	case flags&EntryTombstone != 0:
	case flags&EntryOverflow != 0:
		ref := e.overflowRef()
		le.PutUint32(b[OffEntryValueLen:], ref.Len)
		b = le.AppendUint32(b, uint32(ref.First))
	default:
		le.PutUint32(b[OffEntryValueLen:], uint32(len(e.inlineValue())))
		b = append(b, e.inlineValue()...)
	}
	return b
}

/**
* downgradeLeaves baja a la forma 1 las hojas del árbol que entran en ella (una llena de la forma 2 no entra: queda
* como está) y borra MetaLeavesForm2 de la meta, sin WAL: deja el árbol como uno del formato 1 a medio migrar (solo
* pruebas; antes de cortar hay que hacer un checkpoint).
* @param t testing.TB, tr *BTree
* @return int: cuántas hojas bajó
**/
func downgradeLeaves(t testing.TB, tr *BTree) int {
	t.Helper()
	h, n, _, err := tr.descend(minusInf, 0, buffer.Share, false)
	if err != nil {
		t.Fatal(err)
	}
	blk := n.blk
	h.Release()
	count := 0
	for blk != 0 {
		h, n, err := tr.lockNode(blk, buffer.Exclusive)
		if err != nil {
			t.Fatal(err)
		}
		if n.v2 {
			old := append(page.Page(nil), n.p...)
			c, err := readLeafContent(old)
			if err != nil {
				t.Fatal(err)
			}
			items := [][]byte{}
			if c.highKey != nil {
				items = append(items, c.highKey)
			}
			for _, e := range c.entries {
				items = append(items, encodeForm1(e))
			}
			if !fitInPage(items) {
				blk = n.o.Next
				h.Release()
				continue
			}
			p := h.Page()
			if err := initTreePage(p, c.o); err != nil {
				t.Fatal(err)
			}
			p.SetFlags(c.flags &^ page.FlagHasFreeLines)
			p.SetPruneXID(c.prune)
			for _, it := range items {
				if _, err := p.AddItem(it); err != nil {
					t.Fatal(err)
				}
			}
			h.MarkDirty(p.LSN())
			count++
		}
		blk = n.o.Next
		h.Release()
	}
	mh, err := tr.pool.Pin(tr.tag(MetaBlock), buffer.Normal)
	if err != nil {
		t.Fatal(err)
	}
	mh.Lock(buffer.Exclusive)
	le.PutUint32(mh.Page()[offMetaFlags:], 0)
	mh.MarkDirty(mh.Page().LSN())
	mh.Release()
	return count
}

// TestForm1LeavesMigrate: un árbol con todas sus hojas en la forma 1 (como lo deja el formato 1) se lee igual;
// escribir en una hoja la reescribe en la forma 2 antes; la limpieza reescribe las que quedan, anota MetaLeavesForm2 y
// después vuelve a saltar hojas por el mapa; Verify cuenta las de la forma 1 en cada paso.
func TestForm1LeavesMigrate(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 256)
	status := newLSNStatus()
	tr := vmTree(t, e, status, 1)
	m := model{}
	rng := rand.New(rand.NewSource(7))
	xid := page.XID(10)
	for range 3000 {
		id := fmt.Sprintf("doc/%06d", rng.Intn(4000))
		value := randomValue(rng, 80)
		writeValue(t, e, tr, xid, id, value, false)
		status.set(xid, xact.StatusCommitted, 0)
		m[id] = value
		xid++
	}
	st := checkTree(t, tr, m)
	if st.LeavesForm1 != 0 || !st.LeavesForm2 {
		t.Fatalf("un árbol nuevo tiene hojas de la forma 1: %+v", st)
	}
	leaves := downgradeLeaves(t, tr)
	st = checkTree(t, tr, m)
	if leaves < 10 || st.LeavesForm1 != leaves || st.LeavesForm2 {
		t.Fatalf("tras bajar %d hojas: %+v", leaves, st)
	}

	for i := range 50 { // escrituras sueltas: cada hoja que tocan pasa a la forma 2
		id := fmt.Sprintf("doc/%06d", rng.Intn(4000))
		value := []byte(fmt.Sprintf("nuevo-%d", i))
		writeValue(t, e, tr, xid, id, value, false)
		status.set(xid, xact.StatusCommitted, 0)
		m[id] = value
		xid++
	}
	st = checkTree(t, tr, m)
	if st.LeavesForm1 == 0 || st.LeavesForm1 >= leaves || tr.Rewritten() == 0 {
		t.Fatalf("las escrituras no reescribieron sus hojas: %+v, reescritas %d", st, tr.Rewritten())
	}

	vs, err := tr.Vacuum(VacuumOptions{Horizon: xid, FreezeLimit: xid, Free: freeWith(e)})
	if err != nil || !vs.LeavesForm2 || vs.Rewritten != st.LeavesForm1 || vs.Skipped != 0 {
		t.Fatalf("la limpieza no terminó la migración: %+v %v (quedaban %d)", vs, err, st.LeavesForm1)
	}
	st = checkTree(t, tr, m)
	if st.LeavesForm1 != 0 || !st.LeavesForm2 {
		t.Fatalf("tras la limpieza: %+v", st)
	}
	vs, err = tr.Vacuum(VacuumOptions{Horizon: xid, FreezeLimit: xid, Free: freeWith(e)})
	if err != nil || vs.Skipped == 0 || vs.Rewritten != 0 {
		t.Fatalf("la pasada siguiente no saltó hojas por el mapa: %+v %v", vs, err)
	}
	checkVMInvariant(t, e, tr, status)
	checkBlocks(t, e, tr)
}

// TestForm1LeavesRedo: los registros del formato 1 (entradas de la forma 1 en hojas de la forma 1) y los del 2 se
// rehacen sobre la misma página: un árbol bajado a la forma 1, con checkpoint, escrituras que lo migran y un corte,
// se recupera con el mismo contenido.
func TestForm1LeavesRedo(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	for i := range 2000 {
		put(t, tr, m, fmt.Sprintf("k%05d", i*3), []byte(fmt.Sprintf("v%d", i)))
	}
	if n := downgradeLeaves(t, tr); n == 0 {
		t.Fatal("ninguna hoja entró en la forma 1")
	}
	redo := checkpoint(t, e)
	for i := range 2000 {
		put(t, tr, m, fmt.Sprintf("k%05d", i*3+1), []byte(fmt.Sprintf("w%d", i)))
	}
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 1)
	tr2 := openTree(t, openEnv(t, fs, redo, 64))
	if st := checkTree(t, tr2, m); st.LeavesForm1 != 0 {
		t.Fatalf("quedaron hojas de la forma 1 que recibieron escrituras: %+v", st)
	}
}

// TestPrefixShrink: un id sin el prefijo de su hoja (más chico que todas, o de otro grupo) la reescribe con un prefijo
// más corto antes de escribirse; si así no cabe, la hoja se parte. El árbol queda igual que el modelo.
func TestPrefixShrink(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	long := strings.Repeat("x", 200)
	for i := range 3000 {
		put(t, tr, m, fmt.Sprintf("m/%s/%06d", long, i), []byte("v"))
	}
	before := tr.Rewritten()
	rng := rand.New(rand.NewSource(1))
	for i := range 500 { // ids que no tienen el prefijo de la hoja donde caen
		id := fmt.Sprintf("m/%s/%06d/%c", long[:rng.Intn(len(long))], rng.Intn(3000), 'a'+rune(i%26))
		put(t, tr, m, id, []byte("w"))
	}
	put(t, tr, m, "a", []byte("antes de todo"))
	put(t, tr, m, "z", []byte("después de todo"))
	if tr.Rewritten() == before {
		t.Fatal("ningún prefijo se acortó")
	}
	checkTree(t, tr, m)
}

// TestFrozenEntryShrinks: congelar una versión en línea de la forma 2 le quita el xid (12 B de cabecera); una con
// desborde lo conserva.
func TestFrozenEntryShrinks(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	status := newLSNStatus()
	tr := vmTree(t, e, status, 1)
	writeValue(t, e, tr, 5, "corto", []byte("valor"), false)
	writeValue(t, e, tr, 6, "largo", []byte(strings.Repeat("d", 4000)), false)
	status.set(5, xact.StatusCommitted, 0)
	status.set(6, xact.StatusCommitted, 0)
	if _, err := tr.Vacuum(VacuumOptions{Horizon: 10, FreezeLimit: 10, Free: freeWith(e)}); err != nil {
		t.Fatal(err)
	}
	h, n, _, err := tr.descend(minusInf, 0, buffer.Share, false)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	for off := n.firstData(); off <= n.maxOff(); off++ {
		en, err := n.entryAt(off)
		if err != nil {
			t.Fatal(err)
		}
		over := en.flags()&EntryOverflow != 0
		switch {
		case en.flags()&EntryFrozen == 0:
			t.Fatalf("%s sin congelar", en.idString())
		case !over && (en.hdrLen() != entry2Frozen || en.xid() != 0):
			t.Fatalf("%s congelada con cabecera de %d B", en.idString(), en.hdrLen())
		case over && (en.hdrLen() != entry2Header || en.xid() != 6):
			t.Fatalf("%s con desborde perdió su xid: %d", en.idString(), en.xid())
		}
	}
}

// TestLeafWriteDeltaHasFullID: el delta de KindLeafWrite en una hoja con prefijo lleva el id entero (el flujo lógico lo
// lee sin la página) y la página guarda solo el sufijo.
func TestLeafWriteDeltaHasFullID(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	for i := range 1500 {
		put(t, tr, m, fmt.Sprintf("indice/valor/%06d", i), nil)
	}
	id := "indice/valor/000700x"
	put(t, tr, m, id, []byte("nueva"))
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	var found bool
	if _, err := wal.NewReader(e.fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		if rec.Kind != wal.KindLeafWrite {
			return nil
		}
		v, err := DecodeLeafWrite(rec)
		found = found || err == nil && v.ID == id && string(v.Value) == "nueva"
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("ningún KindLeafWrite decodifica %q", id)
	}
	h, n, _, err := tr.descend(searchKey([]byte(id)), 0, buffer.Share, false)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	if len(n.pre) < len("indice/valor/000") {
		t.Fatalf("la hoja tiene el prefijo %q", n.pre)
	}
	checkTree(t, tr, m)
}
