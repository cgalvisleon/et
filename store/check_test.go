package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* fillHealthy llena un motor con lo que tiene uno sano: valores chicos y de desborde, reemplazos, borrados, un lote
* confirmado y uno deshecho, una limpieza (cadenas liberadas, bits del mapa, congelado) y un store vaciado.
* @param t testing.TB, e *Engine
**/
func fillHealthy(t testing.TB, e *Engine) {
	t.Helper()
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	for i := range 500 {
		mustOK(t, a.Put(fmt.Sprintf("k%04d", i), []byte(fmt.Sprintf("value %d", i))))
	}
	for i := range 6 {
		mustOK(t, b.Put(fmt.Sprintf("big%d", i), bytes.Repeat([]byte{byte(i)}, 9_000+i*3_000)))
	}
	for i := range 100 {
		mustOK(t, a.Remove(fmt.Sprintf("k%04d", i*3)))
	}
	_, err := b.Delete("big2")
	mustOK(t, err)
	mustOK(t, b.Put("big3", bytes.Repeat([]byte{9}, 20_000)))
	tx := e.NewBatch()
	mustOK(t, a.PutIn(tx, "batch", []byte("x")))
	mustOK(t, b.PutIn(tx, "bigbatch", bytes.Repeat([]byte{7}, 10_000)))
	mustOK(t, tx.Commit())
	tx = e.NewBatch()
	mustOK(t, b.PutIn(tx, "rolled", bytes.Repeat([]byte{8}, 10_000)))
	mustOK(t, tx.Rollback())
	mustOK(t, e.Vacuum())
	c := mustStore(t, e, "c")
	mustOK(t, c.Put("x", []byte("y")))
	mustOK(t, c.Empty())
	mustOK(t, c.Put("z", bytes.Repeat([]byte{1}, 12_000)))
}

// TestVerifyHealthy: un motor sano da 0 hallazgos en Verify y Repair (antes y después de reabrir), con escrituras en
// curso en otro store mientras tanto.
func TestVerifyHealthy(t *testing.T) {
	base := crashfs.New()
	e := openTest(t, base, testConfig())
	fillHealthy(t, e)
	checkClean(t, e)
	r, err := e.Repair()
	mustOK(t, err)
	if !r.Clean() {
		t.Fatalf("Repair of a healthy engine:\n%s", r)
	}
	mustOK(t, e.Close())
	e = openTest(t, base, testConfig())
	defer e.Close()
	r = checkClean(t, e)
	if r.Stores != 3 || r.LiveIDs != int64(e.mustCount(t)) || r.FreePages == 0 || r.ChainPages == 0 {
		t.Fatalf("totals: %s", r)
	}
	t.Log(r.String())
}

/**
* checkClean corre Verify y falla si hay algún hallazgo.
* @param t testing.TB, e *Engine
* @return Report
**/
func checkClean(t testing.TB, e *Engine) Report {
	t.Helper()
	r, err := e.Verify()
	mustOK(t, err)
	if !r.Clean() {
		t.Fatalf("Verify of a healthy engine:\n%s", r)
	}
	return r
}

/**
* mustCount suma Count de todos los stores.
* @param t testing.TB
* @return int64
**/
func (e *Engine) mustCount(t testing.TB) int64 {
	t.Helper()
	var n int64
	for _, name := range e.Stores() {
		n += mustStore(t, e, name).Count()
	}
	return n
}

// TestVerifyConcurrent: Verify con escrituras sueltas y lotes en otros stores y en el mismo, sin hallazgos.
func TestVerifyConcurrent(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	defer e.Close()
	fillHealthy(t, e)
	a := mustStore(t, e, "a")
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				done <- nil
				return
			default:
			}
			tx := e.NewBatch()
			if err := a.PutIn(tx, fmt.Sprintf("c%03d", i%200), []byte("v")); err != nil {
				done <- err
				return
			}
			if err := tx.Commit(); err != nil {
				done <- err
				return
			}
			if err := a.Put(fmt.Sprintf("d%03d", i%200), bytes.Repeat([]byte{1}, i%3*4_000)); err != nil {
				done <- err
				return
			}
		}
	}()
	for range 5 {
		checkClean(t, e)
	}
	close(stop)
	mustOK(t, <-done)
	checkClean(t, e)
}

// damaged es lo que daña buildDamaged, para comprobar el informe.
type damaged struct {
	okFile, badFile, chainFile smgr.FileNo
	badLeaf, chainPage         page.BlockNumber
	want                       map[string]string // contenido de «ok» tras reparar
}

/**
* buildDamaged arma un motor y lo daña a mano. Store «ok» (todo reparable): una cadena huérfana (escrita sin su
* hoja), una página en ceros sin dueño, la lista de libres cortada (sus últimas páginas quedan fuera), un bit del mapa
* en la página meta y Count corrido en 7; un archivo de datos huérfano. Con irreparable, además: store «bad» con una
* hoja ilegible (suma mala) y store «chain» con una página de una cadena viva cambiada de posición.
* @param t testing.TB, base crashfs.FS, irreparable bool
* @return damaged
**/
func buildDamaged(t testing.TB, base crashfs.FS, irreparable bool) damaged {
	t.Helper()
	e := openTest(t, base, testConfig())
	d := damaged{want: map[string]string{}}
	ok := mustStore(t, e, "ok")
	d.okFile = ok.file
	for i := range 300 {
		id, v := fmt.Sprintf("k%04d", i), fmt.Sprintf("value %d", i)
		mustOK(t, ok.Put(id, []byte(v)))
		d.want[id] = v
	}
	xid := e.txs.Begin()
	_, err := e.chains.Write(ok.file, xid, bytes.Repeat([]byte{3}, 20_000)) // huérfana: 3 páginas
	mustOK(t, err)
	e.txs.End(xid)
	for i := range 4 {
		mustOK(t, ok.Put(fmt.Sprintf("big%d", i), bytes.Repeat([]byte{byte(i)}, 12_000)))
	}
	for i := range 3 {
		mustOK(t, ok.Remove(fmt.Sprintf("big%d", i)))
	}
	d.want["big3"] = string(bytes.Repeat([]byte{3}, 12_000))
	mustOK(t, e.Vacuum()) // libera 6 páginas de desborde y pone los bits del mapa
	h, err := e.pool.PinNew(ok.file, smgr.ForkMain)
	mustOK(t, err)
	h.Unlock()
	h.Release()
	e.counts.add(ok.file, 7)

	if irreparable {
		bad := mustStore(t, e, "bad")
		d.badFile = bad.file
		for i := range 600 {
			mustOK(t, bad.Put(fmt.Sprintf("b%04d", i), []byte(fmt.Sprintf("bad value %d", i))))
		}
		d.badLeaf = leafBlocks(t, bad)[1]
		ch := mustStore(t, e, "chain")
		d.chainFile = ch.file
		mustOK(t, ch.Put("c1", bytes.Repeat([]byte{1}, 20_000)))
		v, found, err := ch.tree.Lookup("c1", nil)
		if err != nil || !found {
			t.Fatalf("lookup c1: %v %v", found, err)
		}
		d.chainPage = v.Overflow.First
	}
	gone := mustStore(t, e, "gone").file // su número queda debajo de NextFileNo sin store: el huérfano de abajo
	mustOK(t, e.DropStore("gone"))
	mustOK(t, e.Checkpoint())
	mustOK(t, e.Close())

	cutFreeList(t, base, d.okFile)
	editPage(t, base, vmPath(d.okFile), 0, true, func(p page.Page) { p[page.HeaderSize] |= 1 }) // bits del bloque 0
	writeFile(t, base, filepath.Join(testDir, smgr.DataDir, fmt.Sprintf("%010d.0001", gone)), make([]byte, page.Size))
	// uno con número ≥ NextFileNo es de un alta que nunca llegó a ser durable: Open lo borra (removeUnallocated)
	writeFile(t, base, filepath.Join(testDir, smgr.DataDir, "0000000099.0001"), make([]byte, page.Size))
	if irreparable {
		editPage(t, base, dataPath(d.badFile), d.badLeaf, false, func(p page.Page) { p[4000] ^= 0xff })
		editPage(t, base, dataPath(d.chainFile), d.chainPage, true, func(p page.Page) {
			o, _ := p.Opaque()
			o.Level = 5
			mustOK(t, p.SetOpaque(o))
		})
	}
	return d
}

/**
* leafBlocks da los bloques de las hojas de un store, de izquierda a derecha.
* @param t testing.TB, s *Store
* @return []page.BlockNumber
**/
func leafBlocks(t testing.TB, s *Store) []page.BlockNumber {
	t.Helper()
	var leaves []page.BlockNumber
	_, err := s.tree.VerifyWith(btree.VerifyOptions{Leaf: func(info btree.LeafInfo, _ []btree.Version) error {
		leaves = append(leaves, info.Block)
		return nil
	}})
	mustOK(t, err)
	if len(leaves) < 3 {
		t.Fatalf("store %q has %d leaves", s.name, len(leaves))
	}
	return leaves
}

// dataPath es el primer archivo del fork principal de un store en el disco de prueba.
func dataPath(file smgr.FileNo) string {
	return filepath.Join(testDir, smgr.DataDir, fmt.Sprintf("%010d.0001", file))
}

// vmPath es el primer archivo del mapa de visibilidad de un store en el disco de prueba.
func vmPath(file smgr.FileNo) string {
	return filepath.Join(testDir, smgr.DataDir, fmt.Sprintf("%010d_vm.0001", file))
}

/**
* editPage cambia una página en el disco (durable): la lee, aplica edit y, con fixSum, vuelve a calcular su suma (un
* daño lógico); sin fixSum queda ilegible.
* @param t testing.TB, fsys vfs.FS, path string, blk page.BlockNumber, fixSum bool, edit func(page.Page)
**/
func editPage(t testing.TB, fsys vfs.FS, path string, blk page.BlockNumber, fixSum bool, edit func(page.Page)) {
	t.Helper()
	f, err := fsys.OpenFile(path, os.O_RDWR)
	mustOK(t, err)
	defer f.Close()
	p := make(page.Page, page.Size)
	_, err = f.ReadAt(p, int64(blk)*page.Size)
	mustOK(t, err)
	edit(p)
	if fixSum {
		p.SetChecksum(blk)
	}
	_, err = f.WriteAt(p, int64(blk)*page.Size)
	mustOK(t, err)
	mustOK(t, f.Sync())
}

/**
* cutFreeList corta la lista de libres de un store después de su primera página: las demás quedan sin dueño.
* @param t testing.TB, fsys vfs.FS, file smgr.FileNo
**/
func cutFreeList(t testing.TB, fsys vfs.FS, file smgr.FileNo) {
	t.Helper()
	var head page.BlockNumber
	editPage(t, fsys, dataPath(file), btree.MetaBlock, true, func(p page.Page) {
		head = page.BlockNumber(le32(p[page.MetaOffFreeHead:]))
	})
	if head == 0 {
		t.Fatal("the free list is empty")
	}
	editPage(t, fsys, dataPath(file), head, true, func(p page.Page) {
		o, err := p.Opaque()
		mustOK(t, err)
		if o.Next == 0 {
			t.Fatal("the free list has one page")
		}
		o.Next = 0
		mustOK(t, p.SetOpaque(o))
	})
}

/**
* writeFile crea un archivo durable con data.
* @param t testing.TB, fsys vfs.FS, path string, data []byte
**/
func writeFile(t testing.TB, fsys vfs.FS, path string, data []byte) {
	t.Helper()
	f, err := fsys.OpenFile(path, os.O_CREATE|os.O_RDWR)
	mustOK(t, err)
	_, err = f.WriteAt(data, 0)
	mustOK(t, err)
	mustOK(t, f.Sync())
	mustOK(t, f.Close())
	mustOK(t, fsys.SyncDir(filepath.Dir(path)))
}

/**
* kinds cuenta los hallazgos por «store/clase».
* @param r Report
* @return map[string]int
**/
func kinds(r Report) map[string]int {
	m := map[string]int{}
	for _, f := range r.Findings {
		m[f.Store+"/"+string(f.Kind)]++
	}
	return m
}

// TestVerifyRepairDamage: Verify encuentra cada daño de buildDamaged, Repair arregla lo reparable (y deja el resto
// sin tocar) y después Verify solo informa lo irreparable; los datos sanos siguen legibles.
func TestVerifyRepairDamage(t *testing.T) {
	base := crashfs.New()
	d := buildDamaged(t, base, true)
	e := openTest(t, base, testConfig())
	defer e.Close()

	r, err := e.Verify()
	mustOK(t, err)
	t.Logf("Verify:\n%s", r)
	got := kinds(r)
	for _, k := range []string{"/orphan-file", "ok/orphan-chain", "ok/lost-page", "ok/visibility", "ok/count",
		"bad/tree", "chain/chain"} {
		if got[k] == 0 {
			t.Fatalf("Verify did not find %s:\n%s", k, r)
		}
	}
	if r.LostPages < 3+1+5 {
		t.Fatalf("lost pages %d, want at least 9:\n%s", r.LostPages, r)
	}

	r, err = e.Repair()
	mustOK(t, err)
	t.Logf("Repair:\n%s", r)
	for _, f := range r.Findings {
		if f.Repairable != f.Repaired {
			t.Fatalf("repairable %v but repaired %v: %s", f.Repairable, f.Repaired, f)
		}
		if f.Repairable && f.Store != "ok" && f.Store != "" {
			t.Fatalf("repaired something in a damaged store: %s", f)
		}
	}

	r, err = e.Verify()
	mustOK(t, err)
	t.Logf("Verify after Repair:\n%s", r)
	for _, f := range r.Findings {
		if f.Repairable || (f.Store != "bad" && f.Store != "chain") {
			t.Fatalf("left after Repair: %s", f)
		}
	}
	checkStoreContent(t, mustStore(t, e, "ok"), d.want)
	if _, found, err := mustStore(t, e, "bad").Read("b0000"); err != nil || !found {
		t.Fatalf("bad store's first leaf: %v %v", found, err)
	}
	if _, _, err := mustStore(t, e, "chain").Read("c1"); err == nil {
		t.Fatal("the damaged chain reads")
	}
}

/**
* checkStoreContent comprueba que el store tenga exactamente want (recorrido entero) y que Count coincida.
* @param t testing.TB, s *Store, want map[string]string
**/
func checkStoreContent(t testing.TB, s *Store, want map[string]string) {
	t.Helper()
	n := 0
	mustOK(t, s.ForEach(func(id string, data []byte) (bool, error) {
		if w, ok := want[id]; !ok || w != string(data) {
			t.Fatalf("store %q: id %q has %.20q", s.name, id, data)
		}
		n++
		return true, nil
	}, true, 0, 0))
	if n != len(want) || s.Count() != int64(n) {
		t.Fatalf("store %q: %d ids, Count %d, want %d", s.name, n, s.Count(), len(want))
	}
}

// TestRepairCrash: un corte (en los tres modos de crashfs, en cada una de sus operaciones de disco) en medio de Repair deja un motor que abre, cuyo Verify
// solo informa daños reparables (lo que Repair no llegó a hacer) y que un segundo Repair deja limpio, sin perder datos.
func TestRepairCrash(t *testing.T) {
	base := crashfs.New()
	d := buildDamaged(t, base, false)
	img := readImage(t, base, testDir)
	total := repairOps(t, img)
	t.Logf("Repair does %d disk operations", total)
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for cut := int64(1); cut <= total; cut++ {
			t.Run(fmt.Sprintf("%s/%d", name, cut), func(t *testing.T) {
				crashRepair(t, img, d, mode, cut, cut)
			})
		}
	}
}

/**
* repairOps cuenta las operaciones de disco de un Repair entero sobre una copia de img.
* @param t testing.TB, img diskImage
* @return int64
**/
func repairOps(t testing.TB, img diskImage) int64 {
	t.Helper()
	proc := newProc(writeImage(t, img))
	e := openTest(t, proc, testConfig())
	proc.killAfter(1<<40, crashfs.LoseUnsynced, 0)
	_, err := e.Repair()
	mustOK(t, err)
	n := 1<<40 - proc.budget.Load()
	mustOK(t, e.Close())
	return n
}

/**
* crashRepair corta un Repair tras cut operaciones de disco, reabre y comprueba que se pueda terminar.
* @param t *testing.T, img diskImage, d damaged, mode crashfs.Mode, cut, seed int64
**/
func crashRepair(t *testing.T, img diskImage, d damaged, mode crashfs.Mode, cut, seed int64) {
	base := writeImage(t, img)
	proc := newProc(base)
	e := openTest(t, proc, testConfig())
	proc.killAfter(cut, mode, seed)
	_, _ = e.Repair()
	if proc.budget.Load() > 0 {
		proc.kill(mode, seed) // Repair terminó antes del corte: el corte llega al final
	}
	_ = e.Close()

	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	r, err := e.Verify()
	mustOK(t, err)
	for _, f := range r.Findings {
		if !f.Repairable {
			t.Fatalf("irreparable after a cut in Repair: %s\n%s", f, r)
		}
	}
	_, err = e.Repair()
	mustOK(t, err)
	checkClean(t, e)
	checkStoreContent(t, mustStore(t, e, "ok"), d.want)
}

/**
* checkAfterCrash corre Verify en un motor recién recuperado: solo puede encontrar lo que explica un corte (páginas
* sin dueño, cadenas huérfanas, archivos huérfanos, borrados de páginas a medias); si hay, Repair y Verify tiene que
* dar limpio.
* @param t testing.TB, e *Engine
**/
func checkAfterCrash(t testing.TB, e *Engine) {
	t.Helper()
	r, err := e.Verify()
	mustOK(t, err)
	for _, f := range r.Findings {
		switch f.Kind {
		case FindLostPage, FindOrphanChain, FindOrphanFile, FindHalfDead:
		default:
			t.Fatalf("Verify after a crash: %s\n%s", f, r)
		}
	}
	if r.Clean() {
		return
	}
	t.Logf("after the crash: %s", strings.ReplaceAll(r.String(), "\n", "; "))
	if _, err := e.Repair(); err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if r, err = e.Verify(); err != nil || !r.Clean() {
		t.Fatalf("Verify after Repair: %v\n%s", err, r)
	}
}
