package btree

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// bigValue es el largo de un valor que va a desborde en estas pruebas (dos páginas de desborde).
const bigValue = overflow.ChunkSize + 100

/**
* fixedValue arma un valor al azar de n bytes.
* @param rng *rand.Rand, n int
* @return []byte
**/
func fixedValue(rng *rand.Rand, n int) []byte {
	b := make([]byte, n)
	rng.Read(b)
	return b
}

/**
* freeWith arma la función de liberar de VacuumOptions sobre el overflow de prueba.
* @param e *env
* @return func(v Version) error
**/
func freeWith(e *env) func(v Version) error {
	return func(v Version) error {
		_, err := e.ov.FreeTail(testFile, v.Overflow, v.XID, 0)
		return err
	}
}

/**
* writeValue escribe una versión de xid con value (a desborde si no cabe en línea).
* @param t testing.TB, e *env, tr *BTree, xid page.XID, id string, value []byte, tomb bool
* @return Result, error: el de Write
**/
func writeValue(t testing.TB, e *env, tr *BTree, xid page.XID, id string, value []byte, tomb bool) (Result, error) {
	t.Helper()
	w := Write{XID: xid, Tombstone: tomb, Value: value}
	if !tomb && EntryHeaderSize+len(id)+len(value) > MaxEntrySize {
		ref, err := e.ov.Write(testFile, xid, value)
		if err != nil {
			t.Fatal(err)
		}
		w.Value, w.Overflow = nil, &ref
		res, err := tr.Write(id, w)
		if err != nil || !res.Applied {
			if ferr := e.ov.Free(testFile, ref, xid); ferr != nil {
				t.Fatal(ferr)
			}
		}
		return res, err
	}
	return tr.Write(id, w)
}

/**
* readValue lee la versión de id que ve snap, con su valor de desborde.
* @param t testing.TB, e *env, tr *BTree, id string, snap *xact.Snapshot
* @return []byte, bool
**/
func readValue(t testing.TB, e *env, tr *BTree, id string, snap *xact.Snapshot) ([]byte, bool) {
	t.Helper()
	v, ok, err := tr.Lookup(id, snap)
	if err != nil {
		t.Fatalf("Lookup(%q): %v", id, err)
	}
	if !ok || v.Flags&EntryOverflow == 0 {
		return v.Value, ok
	}
	data, err := e.ov.Read(testFile, v.Overflow)
	if err != nil {
		t.Fatalf("Read del desborde de %q: %v", id, err)
	}
	return data, true
}

/**
* treeBlocks devuelve los bloques de todas las páginas del árbol, nivel por nivel.
* @param t testing.TB, tr *BTree
* @return []page.BlockNumber
**/
func treeBlocks(t testing.TB, tr *BTree) []page.BlockNumber {
	t.Helper()
	m, err := tr.readMeta()
	if err != nil {
		t.Fatal(err)
	}
	var out []page.BlockNumber
	for leftmost := m.root; leftmost != 0; {
		var child page.BlockNumber
		for blk := leftmost; blk != 0; {
			h, n, err := tr.lockNode(blk, buffer.Share)
			if err != nil {
				t.Fatal(err)
			}
			if blk == leftmost && !n.leaf() {
				v, _ := n.pivotAt(n.firstData())
				child = v.downlink()
			}
			out = append(out, blk)
			blk = n.o.Next
			h.Release()
		}
		leftmost = child
	}
	return out
}

/**
* freeBlocks devuelve la lista de libres del store de prueba desde la meta.
* @param t testing.TB, e *env
* @return []page.BlockNumber
**/
func freeBlocks(t testing.TB, e *env) []page.BlockNumber {
	t.Helper()
	read := func(b page.BlockNumber) page.Page {
		h, err := e.pool.Pin(buffer.Tag{File: testFile, Fork: smgr.ForkMain, Block: b}, buffer.Normal)
		if err != nil {
			t.Fatal(err)
		}
		h.Lock(buffer.Share)
		defer h.Release()
		return append(page.Page(nil), h.Page()...)
	}
	var out []page.BlockNumber
	for b := page.BlockNumber(le.Uint32(read(MetaBlock)[page.HeaderSize+24:])); b != 0; {
		if slices.Contains(out, b) {
			t.Fatalf("ciclo en la lista de libres en %d", b)
		}
		out = append(out, b)
		o, err := read(b).Opaque()
		if err != nil || o.Flags != page.OpaqueFree {
			t.Fatalf("el bloque %d de la lista no es libre", b)
		}
		b = o.Next
	}
	return out
}

/**
* chainOf devuelve los bloques de una cadena de desborde siguiendo next.
* @param t testing.TB, e *env, ref overflow.Ref
* @return []page.BlockNumber
**/
func chainOf(t testing.TB, e *env, ref overflow.Ref) []page.BlockNumber {
	t.Helper()
	var out []page.BlockNumber
	for b, i := ref.First, 0; b != 0 && i < int((ref.Len+overflow.ChunkSize-1)/overflow.ChunkSize); i++ {
		out = append(out, b)
		h, err := e.pool.Pin(buffer.Tag{File: testFile, Fork: smgr.ForkMain, Block: b}, buffer.Normal)
		if err != nil {
			t.Fatal(err)
		}
		h.Lock(buffer.Share)
		o, _ := h.Page().Opaque()
		h.Release()
		b = o.Next
	}
	return out
}

/**
* checkBlocks comprueba que cada bloque del store está exactamente una vez como meta, página del árbol, página libre o
* página de la cadena de una versión que sigue en las hojas: ni huérfanas ni dobles en la lista de libres.
* @param t testing.TB, e *env, tr *BTree
**/
func checkBlocks(t testing.TB, e *env, tr *BTree) {
	t.Helper()
	seen := map[page.BlockNumber]string{MetaBlock: "meta"}
	note := func(b page.BlockNumber, what string) {
		if prev, ok := seen[b]; ok {
			t.Fatalf("el bloque %d está dos veces: %s y %s", b, prev, what)
		}
		seen[b] = what
	}
	for _, b := range treeBlocks(t, tr) {
		note(b, "árbol")
	}
	for _, b := range freeBlocks(t, e) {
		note(b, "libre")
	}
	for _, v := range leafEntries(t, tr) {
		if v.Flags&EntryOverflow != 0 {
			for _, b := range chainOf(t, e, v.Overflow) {
				note(b, "desborde de "+v.ID)
			}
		}
	}
	n, err := e.sm.NBlocks(testFile, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}
	for b := page.BlockNumber(0); b < page.BlockNumber(n); b++ {
		if _, ok := seen[b]; !ok {
			t.Fatalf("el bloque %d es huérfano", b)
		}
	}
}

// mvccVersion es una versión escrita en TestVacuumModel.
type mvccVersion struct {
	id   string
	lsn  page.LSN
	xid  page.XID
	tomb bool
}

/**
* expectedAfter es el modelo de la limpieza: por id, de la más nueva a la más vieja, la de referencia es la primera
* confirmada con xid < horizon; se quitan las abortadas, las más viejas que la de referencia y la de referencia si es
* una lápida (sin nada detrás). Devuelve lo que queda y las lápidas de referencia (que pueden quedar si las versiones
* del id siguen en la hoja de la derecha).
* @param vs []mvccVersion: en orden de escritura, status *lsnStatus, horizon page.XID
* @return map[key]bool: (id, lsn) que quedan, map[key]bool: lápidas de referencia
**/
func expectedAfter(vs []mvccVersion, status *lsnStatus, horizon page.XID) (map[string]bool, map[string]bool) {
	byID := map[string][]mvccVersion{}
	for _, v := range vs {
		byID[v.id] = append([]mvccVersion{v}, byID[v.id]...)
	}
	keep, anchors := map[string]bool{}, map[string]bool{}
	for _, list := range byID {
		anchored := false
		for _, v := range list {
			st := status.get(v.xid)
			k := fmt.Sprintf("%s/%d", v.id, v.lsn)
			switch {
			case anchored || st == xact.StatusAborted:
			case st == xact.StatusCommitted && v.xid < horizon:
				anchored = true
				if v.tomb {
					anchors[k] = true
				} else {
					keep[k] = true
				}
			default:
				keep[k] = true
			}
		}
	}
	return keep, anchors
}

// TestVacuumModel: con transacciones confirmadas, abortadas y en curso, valores en línea y en desborde, y un
// horizonte dado, la limpieza deja exactamente lo que dice el modelo (salvo lápidas de referencia en el borde de una
// hoja), todo snapshot posible desde el horizonte ve lo mismo que antes, las cadenas de lo quitado vuelven a la lista
// de libres sin huérfanas, Verify pasa, y una segunda pasada no quita nada más.
func TestVacuumModel(t *testing.T) {
	for seed := range int64(4) {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			e := openEnv(t, crashfs.New(), wal.Start, 64)
			status := newLSNStatus()
			tr := treeWith(t, e, status, nil)
			rng := rand.New(rand.NewSource(seed))
			const txs = 900
			horizon := page.XID(100 + txs*2/3)
			var vs []mvccVersion
			var inProgress []page.XID
			for i := range txs {
				xid := page.XID(100 + i)
				for range 1 + rng.Intn(2) {
					id := fmt.Sprintf("id%03d", rng.Intn(120))
					var value []byte
					if rng.Intn(5) == 0 {
						value = fixedValue(rng, bigValue-rng.Intn(50))
					} else {
						value = randomValue(rng, 80)
					}
					tomb := rng.Intn(6) == 0
					res, err := writeValue(t, e, tr, xid, id, value, tomb)
					var we *WaitError
					if errors.As(err, &we) || errors.Is(err, ErrSerialization) {
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					if res.Applied {
						vs = append(vs, mvccVersion{id: id, lsn: res.LSN, xid: xid, tomb: tomb})
					}
				}
				switch r := rng.Intn(10); {
				case xid >= horizon && r == 0:
					inProgress = append(inProgress, xid)
				case r < 2:
					status.set(xid, xact.StatusAborted, 0)
				default:
					status.set(xid, xact.StatusCommitted, 0)
				}
			}
			snapshots := []*xact.Snapshot{nil}
			for _, xmax := range []page.XID{horizon, horizon + 50, horizon + 200, 100 + txs} {
				s := &xact.Snapshot{Xmin: xmax, Xmax: xmax}
				for _, x := range inProgress {
					if x < xmax {
						s.InProgress = append(s.InProgress, x)
					}
				}
				if len(s.InProgress) > 0 {
					s.Xmin = s.InProgress[0]
				}
				snapshots = append(snapshots, s)
			}
			ids := make([]string, 120)
			for i := range ids {
				ids[i] = fmt.Sprintf("id%03d", i)
			}
			before := map[string]string{}
			for i, s := range snapshots {
				for _, id := range ids {
					v, ok := readValue(t, e, tr, id, s)
					before[fmt.Sprintf("%d/%s", i, id)] = fmt.Sprintf("%v %x", ok, v)
				}
			}

			opts := VacuumOptions{Horizon: horizon, XID: 1, Free: freeWith(e)}
			st, err := tr.Vacuum(opts)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%d versiones; limpieza: %+v", len(vs), st)
			keep, anchors := expectedAfter(vs, status, horizon)
			got := map[string]bool{}
			for _, v := range leafEntries(t, tr) {
				k := fmt.Sprintf("%s/%d", v.ID, v.LSN)
				got[k] = true
				if v.Flags&EntryFreeing != 0 {
					t.Fatalf("quedó una versión liberándose: %s", k)
				}
				if !keep[k] && !anchors[k] {
					t.Fatalf("quedó una versión muerta: %s", k)
				}
			}
			for k := range keep {
				if !got[k] {
					t.Fatalf("se quitó una versión viva: %s", k)
				}
			}
			for i, s := range snapshots {
				for _, id := range ids {
					v, ok := readValue(t, e, tr, id, s)
					if now := fmt.Sprintf("%v %x", ok, v); now != before[fmt.Sprintf("%d/%s", i, id)] {
						t.Fatalf("snapshot %d ve otra cosa de %s tras limpiar", i, id)
					}
				}
			}
			if _, err := tr.Verify(); err != nil {
				t.Fatal(err)
			}
			checkBlocks(t, e, tr)
			st2, err := tr.Vacuum(opts)
			if err != nil || st2.Removed != 0 || st2.Freed != 0 {
				t.Fatalf("segunda pasada: %+v %v", st2, err)
			}
		})
	}
}

// TestVacuumSpaceReuse: una carga que reemplaza siempre los mismos ids (en línea y en desborde), con una limpieza
// después de cada vuelta, deja de hacer crecer el store: el espacio de las hojas y las páginas de desborde se
// reutilizan. Sin limpiar, crece en cada vuelta.
func TestVacuumSpaceReuse(t *testing.T) {
	for _, vacuum := range []bool{true, false} {
		t.Run(fmt.Sprintf("vacuum=%v", vacuum), func(t *testing.T) {
			e := openEnv(t, crashfs.New(), wal.Start, 64)
			tr := createTree(t, e)
			rng := rand.New(rand.NewSource(3))
			m := map[string][]byte{}
			xid := page.XID(10)
			var sizes []int
			for range 12 {
				for i := range 150 {
					id := fmt.Sprintf("id%03d", i)
					value := randomValue(rng, 100)
					if i%10 == 0 {
						value = fixedValue(rng, bigValue-1)
					}
					xid++
					if _, err := writeValue(t, e, tr, xid, id, value, false); err != nil {
						t.Fatal(err)
					}
					m[id] = value
				}
				if vacuum {
					if _, err := tr.Vacuum(VacuumOptions{Horizon: xid + 1, Free: freeWith(e)}); err != nil {
						t.Fatal(err)
					}
				}
				n, err := e.sm.NBlocks(testFile, smgr.ForkMain)
				if err != nil {
					t.Fatal(err)
				}
				sizes = append(sizes, int(n))
			}
			t.Logf("bloques por vuelta: %v", sizes)
			last, third := sizes[len(sizes)-1], sizes[2]
			if vacuum && last > third+2 {
				t.Fatalf("con limpieza el store creció de %d a %d bloques", third, last)
			}
			if !vacuum && last <= third+10 {
				t.Fatalf("sin limpieza el store no creció (%v): la prueba no mide nada", sizes)
			}
			for id, want := range m {
				if got, ok := readValue(t, e, tr, id, nil); !ok || !bytes.Equal(got, want) {
					t.Fatalf("%s no se lee igual", id)
				}
			}
			if _, err := tr.Verify(); err != nil {
				t.Fatal(err)
			}
			if vacuum {
				checkBlocks(t, e, tr)
			}
		})
	}
}

// TestVacuumRedoTwice: rehacer el WAL de una carga con limpiezas (KindLeafVacuum y liberaciones) desde el principio
// deja las mismas hojas; rehacerlo otra vez encima, también; y sobre el disco al día, también.
func TestVacuumRedoTwice(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 24)
	tr := createTree(t, e)
	rng := rand.New(rand.NewSource(5))
	xid := page.XID(10)
	for round := range 4 {
		for i := range 120 {
			xid++
			id := fmt.Sprintf("id%03d", rng.Intn(80))
			value := randomValue(rng, 120)
			if i%9 == 0 {
				value = fixedValue(rng, bigValue-1)
			}
			if _, err := writeValue(t, e, tr, xid, id, value, rng.Intn(7) == 0); err != nil {
				t.Fatal(err)
			}
		}
		if round < 3 {
			if _, err := tr.Vacuum(VacuumOptions{Horizon: xid + 1, Free: freeWith(e)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	want := leafEntries(t, tr)
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	same := func(tr *BTree) {
		t.Helper()
		got := leafEntries(t, tr)
		if len(got) != len(want) {
			t.Fatalf("%d entradas, se esperaban %d", len(got), len(want))
		}
		for i := range got {
			if got[i].ID != want[i].ID || got[i].LSN != want[i].LSN || got[i].Flags != want[i].Flags {
				t.Fatalf("entrada %d: %+v, se esperaba %+v", i, got[i], want[i])
			}
		}
		if _, err := tr.Verify(); err != nil {
			t.Fatal(err)
		}
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e2 := openEnv(t, fs, wal.Start, 24)
	tr2 := openTree(t, e2)
	same(tr2)
	checkBlocks(t, e2, tr2)

	if _, err := wal.NewReader(fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		return redoAny(e2.pool, e2.sm, rec)
	}); err != nil {
		t.Fatal(err)
	}
	same(tr2)

	if err := e2.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if err := e2.sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e3 := openEnv(t, fs, wal.Start, 24)
	same(openTree(t, e3))
}

// errVacuumCut simula un corte durante la liberación de una cadena.
var errVacuumCut = errors.New("corte durante la liberación")

// TestVacuumResumeFree: una limpieza cortada antes de liberar una cadena, a mitad (con la cola ya liberada, como
// FreeTail) o al terminarla (sin quitar todavía la versión), con un corte del disco en los tres modos de crashfs,
// se completa con la pasada siguiente: no queda ninguna versión «liberándose», las cadenas de lo quitado están en la
// lista de libres una sola vez, no hay huérfanas y lo vivo se lee igual.
func TestVacuumResumeFree(t *testing.T) {
	modes := []struct {
		name string
		mode crashfs.Mode
	}{{"LoseUnsynced", crashfs.LoseUnsynced}, {"ReorderUnsynced", crashfs.ReorderUnsynced}, {"TornWrites", crashfs.TornWrites}}
	points := []string{"before", "middle", "end"}
	for _, md := range modes {
		for _, point := range points {
			for seed := range int64(3) {
				t.Run(fmt.Sprintf("%s/%s/seed%d", md.name, point, seed), func(t *testing.T) {
					fs := crashfs.New()
					e := openEnv(t, fs, wal.Start, 32)
					tr := createTree(t, e)
					rng := rand.New(rand.NewSource(seed))
					m := map[string][]byte{}
					xid := page.XID(10)
					for round := range 3 {
						for i := range 40 {
							id := fmt.Sprintf("id%02d", i)
							value := fixedValue(rng, 2*overflow.ChunkSize+rng.Intn(500))
							if round == 0 && i%2 == 0 {
								value = randomValue(rng, 60)
							}
							xid++
							if _, err := writeValue(t, e, tr, xid, id, value, false); err != nil {
								t.Fatal(err)
							}
							m[id] = value
						}
					}
					redo := checkpoint(t, e)
					cuts := 0
					free := func(v Version) error {
						if cuts > 0 {
							return freeWith(e)(v)
						}
						cuts++
						switch point {
						case "middle":
							blocks := chainOf(t, e, v.Overflow)
							for i := len(blocks) - 1; i >= len(blocks)/2; i-- {
								if err := e.free.Free(testFile, blocks[i], 0); err != nil {
									return err
								}
							}
						case "end":
							if err := freeWith(e)(v); err != nil {
								return err
							}
						}
						return errVacuumCut
					}
					if _, err := tr.Vacuum(VacuumOptions{Horizon: xid + 1, Free: free}); !errors.Is(err, errVacuumCut) {
						t.Fatalf("la limpieza cortada: %v", err)
					}
					if seed != 1 {
						if err := e.log.WaitDurable(e.log.Published()); err != nil {
							t.Fatal(err)
						}
					}

					fs.Crash(md.mode, seed)
					e2 := openEnv(t, fs, redo, 32)
					tr2 := openTree(t, e2)
					if seed == 2 { // escrituras nuevas toman páginas liberadas antes de la pasada siguiente
						for i := range 5 {
							id := fmt.Sprintf("new%02d", i)
							value := fixedValue(rng, 2*overflow.ChunkSize)
							xid++
							if _, err := writeValue(t, e2, tr2, xid, id, value, false); err != nil {
								t.Fatal(err)
							}
							m[id] = value
						}
					}
					if _, err := tr2.Vacuum(VacuumOptions{Horizon: xid + 1, Free: freeWith(e2)}); err != nil {
						t.Fatalf("la pasada siguiente: %v", err)
					}
					for _, v := range leafEntries(t, tr2) {
						if v.Flags&EntryFreeing != 0 {
							t.Fatalf("quedó %s liberándose", v.ID)
						}
					}
					for id, want := range m {
						if got, ok := readValue(t, e2, tr2, id, nil); !ok || !bytes.Equal(got, want) {
							t.Fatalf("%s no se lee igual tras el corte", id)
						}
					}
					if _, err := tr2.Verify(); err != nil {
						t.Fatal(err)
					}
					checkBlocks(t, e2, tr2)
				})
			}
		}
	}
}

// TestVacuumRedoRejects: KindLeafVacuum y KindSetHints con deltas inválidos no tocan páginas.
func TestVacuumRedoRejects(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 16)
	tr := createTree(t, e)
	put(t, tr, model{}, "a", []byte("1"))
	put(t, tr, model{}, "b", []byte("2"))
	block := func(delta []byte) []wal.BlockRef {
		return []wal.BlockRef{{File: testFile, Fork: smgr.ForkMain, Block: rootBlock, Delta: delta}}
	}
	u16 := func(vs ...uint16) []byte {
		var b []byte
		for _, v := range vs {
			b = le.AppendUint16(b, v)
		}
		return b
	}
	bad := []wal.Record{
		{Kind: wal.KindLeafVacuum, Blocks: block(u16(0, 0))},                      // nada que hacer
		{Kind: wal.KindLeafVacuum, Blocks: block(u16(0, 2, 2, 1))},                // fuera de orden
		{Kind: wal.KindLeafVacuum, Blocks: block(u16(1, 1, 1, 1))},                // marca y quita el mismo
		{Kind: wal.KindLeafVacuum, Blocks: block(u16(0, 1, 9))},                   // no existe
		{Kind: wal.KindLeafVacuum, Blocks: block(u16(1, 0, 1))},                   // marca una versión sin desborde
		{Kind: wal.KindLeafVacuum, Blocks: block(u16(0, 1))},                      // corto
		{Kind: wal.KindSetHints, Blocks: block(u16(1, 1, 0, 0))},                  // marca vacía
		{Kind: wal.KindSetHints, Blocks: block(append(u16(1, 1), 0x10, 0, 0, 0))}, // no es marca de visibilidad
		{Kind: wal.KindSetHints, Blocks: block(append(u16(1, 7), 0x4, 0, 0, 0))},  // no existe
		{Kind: wal.KindSetHints, Blocks: []wal.BlockRef{{File: testFile, Fork: smgr.ForkMain, Block: MetaBlock}}},
	}
	for i, rec := range bad {
		rec.LSN = 1 << 40
		if err := Redo(e.pool, e.sm, &rec); !errors.Is(err, ErrBadRecord) {
			t.Fatalf("registro %d: %v, se esperaba ErrBadRecord", i, err)
		}
	}
	checkTree(t, openTree(t, e), model{"a": []byte("1"), "b": []byte("2")})
}
