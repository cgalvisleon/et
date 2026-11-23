package btree

import (
	"bytes"
	"fmt"
	"math/rand"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// Pruebas de la revisión independiente del formato 2 (tarea 135 de F-23): ids adversos para el prefijo (más de 255 B
// en común, ids que son prefijo de otros, muchas versiones de un mismo id, valores cerca de MaxEntrySize y en
// desborde), congelado de 12 B, rehacer completo comparado byte a byte y hojas mezcladas de las dos formas con cortes.

// reviewLong es un prefijo común más largo que MaxPrefixLen.
var reviewLong = strings.Repeat("L", 300)

/**
* reviewID arma un id adverso para el prefijo de la hoja.
* @param rng *rand.Rand
* @return string
**/
func reviewID(rng *rand.Rand) string {
	switch rng.Intn(6) {
	case 0:
		return reviewLong + fmt.Sprintf("%04d", rng.Intn(400))
	case 1:
		return reviewLong[:rng.Intn(len(reviewLong))+1] // prefijo de los anteriores (y de sí mismos)
	case 2:
		return strings.Repeat("p", 1+rng.Intn(20))
	case 3:
		return "hot" // muchas versiones de un mismo id: sufijo vacío
	case 4:
		return randomID(rng, 8)
	}
	return fmt.Sprintf("g%02d/%s/%05d", rng.Intn(5), strings.Repeat("v", rng.Intn(260)), rng.Intn(3000))
}

/**
* reviewValue arma un valor de largo variado: vacío, corto, cerca de MaxEntrySize o a desborde.
* @param rng *rand.Rand, id string
* @return []byte
**/
func reviewValue(rng *rand.Rand, id string) []byte {
	n := 0
	switch rng.Intn(8) {
	case 0:
	case 1:
		n = MaxEntrySize - EntryHeaderSize - len(id) // justo en línea
	case 2:
		n = MaxEntrySize - EntryHeaderSize - len(id) + 1 // justo a desborde
	case 3:
		n = 3000 + rng.Intn(5000)
	default:
		n = rng.Intn(120)
	}
	if n < 0 {
		n = 0
	}
	b := make([]byte, n)
	rng.Read(b)
	return b
}

// reviewState es el primario de las pruebas: árbol con estados y mapa, el modelo y el próximo xid.
type reviewState struct {
	e      *env
	tr     *BTree
	status *lsnStatus
	m      model
	xid    page.XID

	noVacuum bool // sin limpiezas en step
}

/**
* step hace n operaciones al azar (escrituras, borrados, alguna abortada) y cada tanto una limpieza que congela.
* @param t testing.TB, rng *rand.Rand, n int
**/
func (s *reviewState) step(t testing.TB, rng *rand.Rand, n int) {
	t.Helper()
	for i := range n {
		id := reviewID(rng)
		_, live := s.m[id]
		tomb := live && rng.Intn(5) == 0
		var value []byte
		if !tomb {
			value = reviewValue(rng, id)
		}
		abort := rng.Intn(15) == 0
		res, err := writeValue(t, s.e, s.tr, s.xid, id, value, tomb)
		if err != nil {
			t.Fatalf("op %d Write(%s): %v", i, short(id), err)
		}
		if abort {
			s.status.set(s.xid, xact.StatusAborted, 0)
		} else {
			s.status.set(s.xid, xact.StatusCommitted, 0)
			if res.Applied {
				if tomb {
					delete(s.m, id)
				} else {
					s.m[id] = value
				}
			}
		}
		s.xid++
		if !s.noVacuum && rng.Intn(400) == 0 {
			if _, err := s.tr.Vacuum(VacuumOptions{Horizon: s.xid, FreezeLimit: s.xid, Free: freeWith(s.e)}); err != nil {
				t.Fatalf("Vacuum: %v", err)
			}
		}
	}
}

/**
* checkModel verifica la estructura y que el árbol tenga el modelo (con los valores de desborde).
* @param t testing.TB, e *env, tr *BTree, m model
* @return VerifyStats
**/
func checkModel(t testing.TB, e *env, tr *BTree, m model) VerifyStats {
	t.Helper()
	st, err := tr.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, id := range m.sortedIDs() {
		v, ok := readValue(t, e, tr, id, nil)
		if !ok || !bytes.Equal(v, m[id]) {
			t.Fatalf("Lookup(%s): ok %v, %d B; se esperaban %d B", short(id), ok, len(v), len(m[id]))
		}
	}
	got := scanAll(t, tr, "", 0)
	if ids := m.sortedIDs(); !slices.Equal(got, ids) {
		t.Fatalf("Scan: %d ids, se esperaban %d (%s)", len(got), len(ids), firstDiff(got, ids))
	}
	back := scanBack(t, tr, "", 0)
	slices.Reverse(back)
	if ids := m.sortedIDs(); !slices.Equal(back, ids) {
		t.Fatalf("ScanBackward: %d ids, se esperaban %d (%s)", len(back), len(ids), firstDiff(back, ids))
	}
	return st
}

/**
* mainPages lee del disco todas las páginas del fork principal del store de prueba.
* @param t testing.TB, e *env
* @return []page.Page
**/
func mainPages(t testing.TB, e *env) []page.Page {
	t.Helper()
	n, err := e.sm.NBlocks(testFile, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}
	pages := make([]page.Page, n)
	for b := range n {
		pages[b] = make(page.Page, page.Size)
		if err := e.sm.Read(testFile, smgr.ForkMain, page.BlockNumber(b), pages[b]); err != nil {
			t.Fatal(err)
		}
	}
	return pages
}

// TestReviewReplayMatchesPages: con ids adversos (prefijos de más de 255 B, ids que son prefijo de otros, un id con
// muchas versiones), valores en el límite de en línea y en desborde, abortadas, borrados y limpiezas que congelan (12 B)
// y borran páginas, rehacer el WAL entero sobre un directorio de datos vacío deja las mismas páginas, byte a byte.
func TestReviewReplayMatchesPages(t *testing.T) {
	for seed := range int64(3) {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			fs := crashfs.New()
			e := openEnv(t, fs, wal.Start, 64)
			status := newLSNStatus()
			cfg := e.config()
			cfg.Status, cfg.VM, cfg.FrozenXID, cfg.NoHints = status, e.free, 1, true
			tr, err := Create(cfg)
			if err != nil {
				t.Fatal(err)
			}
			s := &reviewState{e: e, tr: tr, status: status, m: model{}, xid: 10}
			rng := rand.New(rand.NewSource(seed))
			s.step(t, rng, 4000)
			if _, err := tr.Vacuum(VacuumOptions{Horizon: s.xid, FreezeLimit: s.xid, Free: freeWith(e)}); err != nil {
				t.Fatal(err)
			}
			st := checkModel(t, e, tr, s.m)
			t.Logf("árbol %+v, reescritas %d, podadas %d", st, tr.Rewritten(), tr.Pruned())
			if err := e.log.WaitDurable(e.log.Published()); err != nil {
				t.Fatal(err)
			}
			if err := e.pool.FlushAll(nil); err != nil {
				t.Fatal(err)
			}
			if err := e.sm.SyncAll(); err != nil {
				t.Fatal(err)
			}
			want := mainPages(t, e)

			fs.Crash(crashfs.LoseUnsynced, 0)
			names, err := fs.ReadDir(filepath.Join(testDir, smgr.DataDir))
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range names {
				if err := fs.Remove(filepath.Join(testDir, smgr.DataDir, name)); err != nil {
					t.Fatal(err)
				}
			}
			e2 := openEnv(t, fs, wal.Start, 64)
			if err := e2.pool.FlushAll(nil); err != nil {
				t.Fatal(err)
			}
			got := mainPages(t, e2)
			if len(got) != len(want) {
				t.Fatalf("rehacer dejó %d páginas, el primario %d", len(got), len(want))
			}
			for b := range want {
				if !bytes.Equal(got[b], want[b]) {
					i := 0
					for i < page.Size && got[b][i] == want[b][i] {
						i++
					}
					t.Fatalf("página %d distinta tras rehacer (primer byte distinto %d, layout %d/%d)", b, i,
						got[b].LayoutVersion(), want[b].LayoutVersion())
				}
			}
			cfg2 := e2.config()
			cfg2.Status, cfg2.VM = status, e2.free
			tr2, err := Open(cfg2)
			if err != nil {
				t.Fatal(err)
			}
			checkModel(t, e2, tr2, s.m)
		})
	}
}

// TestReviewMixedFormsCrash: un árbol con hojas de las dos formas (bajado a la forma 1 con checkpoint), escrituras y
// limpiezas que lo migran, y un corte en los tres modos de crashfs: lo rehecho tiene lo durable, la estructura es
// válida (Verify no ve la meta marcada con hojas de la forma 1) y una limpieza termina la migración.
func TestReviewMixedFormsCrash(t *testing.T) {
	modes := []crashfs.Mode{crashfs.LoseUnsynced, crashfs.ReorderUnsynced, crashfs.TornWrites}
	for _, mode := range modes {
		for seed := range int64(2) {
			t.Run(fmt.Sprintf("mode%d/seed%d", mode, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := openEnv(t, fs, wal.Start, 32)
				status := newLSNStatus()
				tr := vmTree(t, e, status, 1)
				s := &reviewState{e: e, tr: tr, status: status, m: model{}, xid: 10}
				rng := rand.New(rand.NewSource(seed + 50))
				s.step(t, rng, 2500)
				if n := downgradeLeaves(t, tr); n == 0 {
					t.Fatal("ninguna hoja bajó a la forma 1")
				}
				redo := checkpoint(t, e)
				s.noVacuum = true
				s.step(t, rng, 600)
				// Una limpieza de migración cortada a mitad (Stop tras unas hojas): la meta no se marca
				leavesSeen := 0
				vs, err := tr.Vacuum(VacuumOptions{Horizon: s.xid, FreezeLimit: s.xid, Free: freeWith(e),
					Stop: func() bool { leavesSeen++; return leavesSeen > 5 }})
				if err == nil || vs.LeavesForm2 {
					t.Fatalf("la limpieza cortada: %+v %v", vs, err)
				}
				if st := checkModel(t, e, tr, s.m); st.LeavesForm2 || st.LeavesForm1 == 0 {
					t.Fatalf("tras la limpieza cortada: %+v", st)
				}
				s.step(t, rng, 600)
				if err := e.log.WaitDurable(e.log.Published()); err != nil {
					t.Fatal(err)
				}
				fs.Crash(mode, seed)
				e2 := openEnv(t, fs, redo, 32)
				tr2 := reopenVM(t, e2, status)
				st := checkModel(t, e2, tr2, s.m)
				t.Logf("tras el corte: %+v", st)
				vs, err = tr2.Vacuum(VacuumOptions{Horizon: s.xid, FreezeLimit: s.xid, Free: freeWith(e2)})
				if err != nil || !vs.LeavesForm2 {
					t.Fatalf("la limpieza no terminó la migración: %+v %v", vs, err)
				}
				if st := checkModel(t, e2, tr2, s.m); st.LeavesForm1 != 0 || !st.LeavesForm2 {
					t.Fatalf("tras migrar: %+v", st)
				}
			})
		}
	}
}

// TestReviewDamagedPrefixUndetected: store/doc.go dice que un prefijo dañado «sale como claves fuera de orden o de sus
// cotas». En la hoja de más a la derecha (sin cota alta) un prefijo cambiado a uno mayor, del mismo largo, deja todas
// las claves en orden y sobre la cota baja: Verify no lo ve y todos los ids de la hoja cambian de nombre.
func TestReviewDamagedPrefixUndetected(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	for i := range 2000 {
		put(t, tr, m, fmt.Sprintf("k/%06d", i), []byte("v"))
	}
	h, n, _, err := tr.descend(searchKey([]byte("\xff")), 0, buffer.Exclusive, true)
	if err != nil {
		t.Fatal(err)
	}
	if !n.rightmost() || !n.v2 || len(n.pre) == 0 {
		h.Release()
		t.Fatalf("hoja de más a la derecha sin prefijo: %q", n.pre)
	}
	before := string(n.pre)
	b, _ := n.p.Item(n.prefixOff())
	b[PivotHeaderSize] = 'z' // «k/…» → «z/…»: mayor que la cota baja, la hoja no tiene cota alta
	h.MarkDirty(n.p.LSN())
	h.Release()
	if _, err := tr.Verify(); err != nil {
		t.Logf("Verify detectó el prefijo dañado: %v", err)
		return
	}
	if _, ok, _ := tr.Lookup(fmt.Sprintf("k/%06d", 1999), nil); ok {
		t.Fatal("el id sigue con su nombre")
	}
	t.Logf("Verify limpio con el prefijo %q cambiado a %q: los ids de la hoja se renombraron", before, "z"+before[1:])
}
