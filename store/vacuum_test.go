package store

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// metaFreeHead es la posición de freeHead en la página meta (btree/doc.go: magic, version, root, level, fastRoot y
// fastLevel antes, de 4 B cada uno).
const metaFreeHead = page.HeaderSize + 24

// audit es el inventario de las páginas de un store.
type audit struct {
	blocks   int // bloques del fork principal
	tree     int // páginas del árbol (sin la meta)
	deleted  int // páginas borradas del árbol que esperan reciclarse
	free     int // páginas en la lista de libres
	chains   int // páginas de desborde de las versiones del árbol
	overflow int // páginas marcadas de desborde
	entries  int // entradas de hoja (todas las versiones)
	versions int // versiones que ve un recorrido (las vivas)
}

// orphans son las páginas que no son del árbol (enganchadas o borradas), de la lista de libres ni de una cadena de una
// versión.
func (a audit) orphans() int {
	return a.blocks - 1 - a.tree - a.deleted - a.free - a.chains
}

/**
* readPage copia el bloque blk del store.
* @param t testing.TB, s *Store, blk page.BlockNumber
* @return page.Page
**/
func readPage(t testing.TB, s *Store, blk page.BlockNumber) page.Page {
	t.Helper()
	h, err := s.e.pool.Pin(buffer.Tag{File: s.file, Fork: smgr.ForkMain, Block: blk}, buffer.Normal)
	mustOK(t, err)
	defer h.Release()
	h.Lock(buffer.Share)
	defer h.Unlock()
	return bytes.Clone(h.Page())
}

/**
* auditStore hace el inventario de las páginas del store, sin escrituras en curso, y falla si encuentra páginas
* repetidas: en la lista de libres (o un bucle), una página de la lista que no está libre, una página libre fuera de
* la lista, o una página de una cadena que es de otra cadena, del árbol o libre. Supone que todas las versiones del
* árbol son vivas (tras un Vacuum sin nada abierto); si no, entries > versions y sus cadenas no se cuentan.
* @param t testing.TB, s *Store
* @return audit
**/
func auditStore(t testing.TB, s *Store) audit {
	t.Helper()
	n, err := s.e.sm.NBlocks(s.file, smgr.ForkMain)
	mustOK(t, err)
	a := audit{blocks: int(n)}
	st, err := s.tree.Verify()
	mustOK(t, err)
	a.tree, a.deleted, a.entries = st.Pages, st.Deleted, st.Entries

	owner := map[page.BlockNumber]string{}
	claim := func(blk page.BlockNumber, who string) {
		if prev, ok := owner[blk]; ok {
			t.Fatalf("store %q: block %d is both %s and %s", s.name, blk, prev, who)
		}
		owner[blk] = who
	}
	meta := readPage(t, s, btree.MetaBlock)
	for blk := page.BlockNumber(le32(meta[metaFreeHead:])); blk != 0; a.free++ {
		claim(blk, "free")
		pg := readPage(t, s, blk)
		o, err := pg.Opaque()
		if err != nil || o.Flags&page.OpaqueFree == 0 {
			t.Fatalf("store %q: block %d (position %d in the free list) is not free: flags %#x level %d next %d lsn %d prune %d, %v; blocks %d, redo %d",
				s.name, blk, a.free, o.Flags, o.Level, o.Next, pg.LSN(), pg.PruneXID(), err, n, s.e.log.RedoPoint().LSN)
		}
		blk = o.Next
	}
	for blk := page.BlockNumber(1); blk < n; blk++ {
		o, err := readPage(t, s, blk).Opaque()
		if err != nil {
			continue // página nueva sin iniciar
		}
		if o.Flags&page.OpaqueFree != 0 && owner[blk] != "free" {
			t.Fatalf("store %q: block %d is free but not in the free list", s.name, blk)
		}
		if o.Flags&page.OpaqueOverflow != 0 {
			a.overflow++
		}
	}
	err = s.tree.Scan("", nil, func(v btree.Version) (bool, error) {
		a.versions++
		if v.Flags&btree.EntryOverflow == 0 {
			return true, nil
		}
		blk := v.Overflow.First
		for i := range (int(v.Overflow.Len) + overflow.ChunkSize - 1) / overflow.ChunkSize {
			claim(blk, fmt.Sprintf("page %d of the chain of %.20q", i, v.ID))
			o, err := readPage(t, s, blk).Opaque()
			if err != nil || o.Flags&page.OpaqueOverflow == 0 {
				t.Fatalf("store %q: block %d of the chain of %.20q is not an overflow page", s.name, blk, v.ID)
			}
			a.chains++
			blk = o.Next
		}
		return true, nil
	})
	mustOK(t, err)
	return a
}

/**
* le32 lee un uint32 little-endian.
* @param b []byte
* @return uint32
**/
func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

/**
* auditClean comprueba que el store no tenga versiones muertas ni páginas huérfanas.
* @param t testing.TB, s *Store
* @return audit
**/
func auditClean(t testing.TB, s *Store) audit {
	t.Helper()
	a := auditStore(t, s)
	if a.entries != a.versions {
		t.Fatalf("store %q: %d leaf entries, %d live versions", s.name, a.entries, a.versions)
	}
	if a.orphans() != 0 || a.overflow != a.chains {
		t.Fatalf("store %q: %d orphan blocks (%+v)", s.name, a.orphans(), a)
	}
	return a
}

/**
* sized da un valor de n bytes que depende de la vuelta.
* @param n, round int
* @return []byte
**/
func sized(n, round int) []byte {
	return bytes.Repeat([]byte{byte('a' + round%26)}, n)
}

// TestVacuumBoundsGrowth reemplaza siempre los mismos ids (valores chicos y grandes): sin limpieza el store crece en
// cada vuelta; con Vacuum tras cada vuelta deja de crecer, todo sigue igual al modelo, no quedan versiones muertas ni
// páginas huérfanas y las páginas de desborde liberadas se reutilizan.
func TestVacuumBoundsGrowth(t *testing.T) {
	for _, vacuum := range []bool{false, true} {
		t.Run(fmt.Sprintf("vacuum=%v", vacuum), func(t *testing.T) {
			e := openTest(t, newProc(crashfs.New()), testConfig())
			defer e.Close()
			s := mustStore(t, e, "a")
			size := func(i int) int { return []int{10, 300, 5000, 20000}[i%4] }
			var blocks []int
			for round := range 12 {
				for i := range 40 {
					mustOK(t, s.Put(fmt.Sprintf("id-%02d", i), sized(size(i), round)))
				}
				if vacuum {
					mustOK(t, e.Vacuum())
					auditClean(t, s)
				}
				blocks = append(blocks, auditStore(t, s).blocks)
			}
			for i := range 40 {
				got, found, err := s.Read(fmt.Sprintf("id-%02d", i))
				if err != nil || !found || !bytes.Equal(got, sized(size(i), 11)) {
					t.Fatalf("id-%02d: %v %v (%d B)", i, found, err, len(got))
				}
			}
			grew := blocks[len(blocks)-1] > blocks[3]
			if grew == vacuum {
				t.Fatalf("blocks per round %v", blocks)
			}
			t.Logf("blocks per round %v", blocks)
		})
	}
}

// TestVacuumKeepsSnapshot: un snapshot abierto frena la limpieza de lo que ve (se lee igual después de Vacuum) y, al
// cerrarlo, la limpieza siguiente lo quita.
func TestVacuumKeepsSnapshot(t *testing.T) {
	e := openTest(t, newProc(crashfs.New()), testConfig())
	defer e.Close()
	s := mustStore(t, e, "a")
	ids := []string{"small", "large", "gone"}
	for _, id := range ids {
		mustOK(t, s.Put(id, sized(map[string]int{"large": 30000}[id]+5, 0)))
	}
	v := e.Snapshot()
	for _, id := range ids[:2] {
		mustOK(t, s.Put(id, sized(map[string]int{"large": 30000}[id]+5, 1)))
	}
	mustOK(t, s.Remove("gone"))
	mustOK(t, e.Vacuum())
	for _, id := range ids {
		got, found, err := s.ReadAt(v, id)
		if err != nil || !found || !bytes.Equal(got, sized(map[string]int{"large": 30000}[id]+5, 0)) {
			t.Fatalf("snapshot %s after Vacuum: %v %v (%d B)", id, found, err, len(got))
		}
	}
	if a := auditStore(t, s); a.entries != 6 {
		t.Fatalf("with the snapshot open: %d entries, want 6", a.entries)
	}
	mustOK(t, v.Close())
	mustOK(t, e.Vacuum())
	if a := auditClean(t, s); a.versions != 2 || a.free == 0 {
		t.Fatalf("after closing the snapshot: %+v", a)
	}
}

// TestVacuumRolledBackBatches: las cadenas de desborde de lotes deshechos (y de versiones propias que el mismo lote
// reemplazó) quedan hasta la limpieza; tras Vacuum no hay huérfanas, las páginas vuelven a la lista de libres y
// sirven para lo que se escribe después.
func TestVacuumRolledBackBatches(t *testing.T) {
	e := openTest(t, newProc(crashfs.New()), testConfig())
	defer e.Close()
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("keep", sized(20000, 0)))
	for round := range 5 {
		tx := e.NewBatch()
		for i := range 6 {
			id := fmt.Sprintf("id-%d", i%3) // cada id dos veces: el lote reemplaza su propia versión
			mustOK(t, s.PutIn(tx, id, sized(9000+i, round)))
		}
		mustOK(t, tx.Rollback())
	}
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "keep", sized(20000, 1)))
	mustOK(t, s.PutIn(tx, "keep", sized(20000, 2)))
	mustOK(t, tx.Commit())

	before := auditStore(t, s)
	if before.orphans() == 0 {
		t.Fatalf("before Vacuum the dead chains should not be counted: %+v", before)
	}
	mustOK(t, e.Vacuum())
	after := auditClean(t, s)
	if after.versions != 1 || after.free < before.overflow-after.chains {
		t.Fatalf("before %+v, after %+v", before, after)
	}
	mustOK(t, s.Put("other", sized(40000, 3)))
	if grown := auditStore(t, s).blocks; grown != after.blocks {
		t.Fatalf("writing after Vacuum grew the store from %d to %d blocks", after.blocks, grown)
	}
	got, found, err := s.Read("keep")
	if err != nil || !found || !bytes.Equal(got, sized(20000, 2)) {
		t.Fatalf("keep: %v %v", found, err)
	}
}

// TestVacuumBackground: la limpieza de fondo corre sola tras VACUUM_THRESHOLD escrituras (lo que dejan las últimas
// menos de VACUUM_THRESHOLD queda para la próxima), Close la detiene, y un store borrado no se limpia ni se vuelve a
// crear.
func TestVacuumBackground(t *testing.T) {
	cfg := testConfig()
	cfg.vacuumEvery, cfg.vacuumWrites = 2*time.Millisecond, 10
	e := openTest(t, newProc(crashfs.New()), cfg)
	s := mustStore(t, e, "a")
	tmp := mustStore(t, e, "tmp")
	for round := range 4 {
		for i := range 20 {
			mustOK(t, s.Put(fmt.Sprintf("id-%02d", i), sized(3000, round)))
			mustOK(t, tmp.Put(fmt.Sprintf("id-%02d", i), sized(10, round)))
		}
	}
	mustOK(t, e.DropStore("tmp"))
	// Ids nuevos (sin versiones muertas): más que VACUUM_THRESHOLD, así la última limpieza de fondo empieza después de
	// que terminaron todos los reemplazos.
	for i := range 12 {
		mustOK(t, s.Put(fmt.Sprintf("new-%02d", i), sized(10, 0)))
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		s.vacuumMu.Lock() // sin una limpieza a medias
		a := auditStore(t, s)
		s.vacuumMu.Unlock()
		if a.entries == a.versions {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the background vacuum did not run: %+v", a)
		}
		time.Sleep(time.Millisecond)
	}
	if names := e.Stores(); len(names) != 1 || names[0] != "a" {
		t.Fatalf("stores %v", names)
	}
	closeWithin(t, e, 10*time.Second)
	if err := e.Vacuum(); err != ErrClosed {
		t.Fatalf("Vacuum after Close = %v", err)
	}
}
