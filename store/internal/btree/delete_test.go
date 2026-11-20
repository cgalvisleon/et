package btree

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// errUnlinkCut simula un corte entre dos pasos del borrado de páginas.
var errUnlinkCut = errors.New("corte antes de KindUnlinkPage")

// longPrefix hace ids largos con un prefijo común: los separadores no se pueden truncar y las internas tienen pocos
// enlaces, así un árbol de pocos miles de ids ya tiene tres niveles.
var longPrefix = strings.Repeat("p", 180)

// longID es el id número i con el prefijo largo.
func longID(i int) string { return fmt.Sprintf("%s%06d", longPrefix, i) }

// fillLong escribe los ids largos [from, to) con valores chicos.
func fillLong(t testing.TB, tr *BTree, m model, from, to int) {
	t.Helper()
	for i := from; i < to; i++ {
		put(t, tr, m, longID(i), []byte(fmt.Sprintf("v%d", i)))
	}
}

// removeLong borra los ids largos [from, to).
func removeLong(t testing.TB, tr *BTree, m model, from, to int) {
	t.Helper()
	for i := from; i < to; i++ {
		remove(t, tr, m, longID(i))
	}
}

// vacuumAt corre una limpieza con todo lo escrito (xid 1) visible para todos.
func vacuumAt(t testing.TB, tr *BTree, horizon page.XID) VacuumStats {
	t.Helper()
	st, err := tr.Vacuum(VacuumOptions{Horizon: horizon})
	if err != nil {
		t.Fatalf("Vacuum(%d): %v", horizon, err)
	}
	t.Logf("limpieza %d: hojas %d, quitadas %d, medio muertas %d, borradas %d, recicladas %d",
		horizon, st.Leaves, st.Removed, st.HalfDead, st.Deleted, st.Recycled)
	return st
}

// nblocks es el largo del fork principal del store de prueba.
func nblocks(t testing.TB, e *env) int {
	t.Helper()
	n, err := e.sm.NBlocks(testFile, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}
	return int(n)
}

// TestPageDeleteRange: borrar un tramo grande del medio vacía hojas que la limpieza borra del árbol (y padres que se
// quedan sin hijos); la pasada siguiente las recicla, el árbol queda válido con exactamente lo que queda, cada bloque
// tiene un dueño, y volver a escribir el tramo reutiliza las páginas sin crecer el archivo.
func TestPageDeleteRange(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	fillLong(t, tr, m, 0, 3000)
	before := checkTree(t, tr, m)
	if before.Levels < 3 {
		t.Fatalf("se esperaban tres niveles o más: %+v", before)
	}
	var err error
	removeLong(t, tr, m, 300, 2700)
	if before, err = tr.Verify(); err != nil {
		t.Fatal(err)
	}
	st := vacuumAt(t, tr, 2)
	if st.Deleted == 0 || st.HalfDead == 0 {
		t.Fatalf("la limpieza no borró páginas: %+v", st)
	}
	after := checkTree(t, tr, m)
	if after.Deleted != st.Deleted || after.HalfDead != 0 || after.Pages+after.Deleted != before.Pages {
		t.Fatalf("antes %+v, después %+v, limpieza %+v", before, after, st)
	}
	checkBlocks(t, e, tr)

	st2 := vacuumAt(t, tr, 3)
	if st2.Recycled != st.Deleted {
		t.Fatalf("recicló %d de %d borradas", st2.Recycled, st.Deleted)
	}
	if v := checkTree(t, tr, m); v.Deleted != 0 {
		t.Fatalf("quedaron borradas sin reciclar: %+v", v)
	}
	checkBlocks(t, e, tr)

	size := nblocks(t, e)
	fillLong(t, tr, m, 300, 2700)
	checkTree(t, tr, m)
	checkBlocks(t, e, tr)
	if grown := nblocks(t, e); grown > size+5 {
		t.Fatalf("volver a escribir el tramo creció el archivo de %d a %d bloques", size, grown)
	}
}

// TestPageDeleteAll: borrar todo deja en cada nivel solo la página de más a la derecha y la raíz rápida en la hoja
// (las bajadas empiezan ahí); volver a llenarlo sube la raíz rápida con las particiones, y todo sobrevive a reabrir.
func TestPageDeleteAll(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	fillLong(t, tr, m, 0, 3000)
	levels := checkTree(t, tr, m).Levels
	removeLong(t, tr, m, 0, 3000)
	vacuumAt(t, tr, 2)
	st := checkTree(t, tr, m)
	if st.Pages != levels || st.Levels != levels {
		t.Fatalf("tras borrar todo quedan %+v; se esperaba una página por nivel en %d niveles", st, levels)
	}
	mt, err := tr.readMeta()
	if err != nil {
		t.Fatal(err)
	}
	if mt.fastLevel != 0 || mt.fastRoot == mt.root {
		t.Fatalf("raíz rápida %d en el nivel %d (raíz %d): se esperaba la hoja", mt.fastRoot, mt.fastLevel, mt.root)
	}
	vacuumAt(t, tr, 3)
	checkBlocks(t, e, tr)

	fillLong(t, tr, m, 5000, 8000)
	checkTree(t, tr, m)
	if mt, err = tr.readMeta(); err != nil {
		t.Fatal(err)
	}
	if mt.fastLevel == 0 {
		t.Fatalf("la raíz rápida no subió al volver a llenar: %+v", mt)
	}
	checkBlocks(t, e, tr)

	redo := checkpoint(t, e)
	removeLong(t, tr, m, 5000, 7990)
	vacuumAt(t, tr, 4)
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 1)
	tr2 := openTree(t, openEnv(t, fs, redo, 64))
	checkTree(t, tr2, m)
}

// TestPageDeleteRightmostChild: la hoja de más a la derecha de un padre con más hijos no se borra (su hermana es de
// otro padre) hasta que sus hermanas izquierdas se borran y queda como único hijo: entonces se va con su padre.
func TestPageDeleteRightmostChild(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	tr := createTree(t, e)
	m := model{}
	fillLong(t, tr, m, 0, 3000)
	checkTree(t, tr, m)
	removeLong(t, tr, m, 0, 1500)
	st := vacuumAt(t, tr, 2)
	if st.Deleted <= st.HalfDead {
		t.Fatalf("ningún padre se borró con su único hijo: %+v", st)
	}
	checkTree(t, tr, m)
	checkBlocks(t, e, tr)
}

// TestPageDeleteCut: un corte del disco antes de cada KindUnlinkPage de un borrado que sube por padres de un solo
// hijo, en los tres modos de crashfs, deja un árbol válido con la hoja medio muerta (Verify la reconoce) y con
// exactamente lo escrito; la próxima limpieza termina el borrado y la siguiente recicla.
func TestPageDeleteCut(t *testing.T) {
	modes := []struct {
		name string
		mode crashfs.Mode
	}{{"LoseUnsynced", crashfs.LoseUnsynced}, {"ReorderUnsynced", crashfs.ReorderUnsynced}, {"TornWrites", crashfs.TornWrites}}
	for _, md := range modes {
		for cut := range 3 {
			for seed := range int64(2) {
				t.Run(fmt.Sprintf("%s/cut%d/seed%d", md.name, cut, seed), func(t *testing.T) {
					pageDeleteCut(t, md.mode, cut, seed)
				})
			}
		}
	}
}

/**
* pageDeleteCut arma el árbol, borra un tramo que vacía subárboles enteros, corta la limpieza antes del cut-ésimo
* KindUnlinkPage, simula el corte del disco y comprueba la recuperación y las limpiezas siguientes.
* @param t *testing.T, mode crashfs.Mode, cut int, seed int64
**/
func pageDeleteCut(t *testing.T, mode crashfs.Mode, cut int, seed int64) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 32)
	tr := createTree(t, e)
	m := model{}
	fillLong(t, tr, m, 0, 3000)
	removeLong(t, tr, m, 0, 1500)
	redo := checkpoint(t, e)
	calls := 0
	tr.beforeUnlink = func(page.BlockNumber) error {
		if calls++; calls > cut {
			return errUnlinkCut
		}
		return nil
	}
	if _, err := tr.Vacuum(VacuumOptions{Horizon: 2}); !errors.Is(err, errUnlinkCut) {
		t.Fatalf("Vacuum: %v, se esperaba el corte", err)
	}
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	fs.Crash(mode, seed)
	e2 := openEnv(t, fs, redo, 32)
	tr2 := openTree(t, e2)
	if st := checkTree(t, tr2, m); st.HalfDead != 1 {
		t.Fatalf("tras el corte: %+v; se esperaba una hoja medio muerta", st)
	}
	vacuumAt(t, tr2, 3)
	if st := checkTree(t, tr2, m); st.HalfDead != 0 {
		t.Fatalf("la limpieza no terminó el borrado: %+v", st)
	}
	vacuumAt(t, tr2, 4)
	checkBlocks(t, e2, tr2)
	fillLong(t, tr2, m, 0, 1500)
	checkTree(t, tr2, m)
	checkBlocks(t, e2, tr2)
}

// TestPageDeleteCrashRandom: una ventana deslizante (se escribe al final, se borra al principio y se reemplaza en el
// medio) con limpiezas intercaladas, cortada en un punto al azar en los tres modos: la recuperación deja exactamente lo
// que quedó en el WAL, un árbol válido, y las limpiezas siguientes terminan los borrados a medias y reciclan.
func TestPageDeleteCrashRandom(t *testing.T) {
	modes := []crashfs.Mode{crashfs.LoseUnsynced, crashfs.ReorderUnsynced, crashfs.TornWrites}
	for _, mode := range modes {
		for seed := range int64(4) {
			t.Run(fmt.Sprintf("mode%d/seed%d", mode, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := openEnv(t, fs, wal.Start, 24)
				tr := createTree(t, e)
				redo := checkpoint(t, e)
				rng := rand.New(rand.NewSource(seed))
				live := model{}
				var ops []op
				horizon := page.XID(2)
				const window, total = 600, 3000
				for i := range total {
					ops = append(ops, slideStep(t, tr, live, rng, i, window)...)
					if i%250 == 249 {
						vacuumAt(t, tr, horizon)
						horizon++
					}
					if i == total/2 {
						redo = checkpoint(t, e)
					}
				}
				durable := e.log.Durable()
				fs.Crash(mode, seed)
				e2 := openEnv(t, fs, redo, 24)
				if e2.log.Published() < durable {
					t.Fatalf("se recuperó hasta %d y era durable hasta %d", e2.log.Published(), durable)
				}
				tr2 := openTree(t, e2)
				m := modelUpTo(ops, e2.log.Published())
				checkTree(t, tr2, m)
				vacuumAt(t, tr2, horizon)
				vacuumAt(t, tr2, horizon+1)
				if st := checkTree(t, tr2, m); st.HalfDead != 0 || st.Deleted != 0 {
					t.Fatalf("tras dos limpiezas: %+v", st)
				}
				checkBlocks(t, e2, tr2)
			})
		}
	}
}

/**
* slideStep es un paso de la ventana deslizante: escribe el id i, borra el id i−window y a veces reemplaza uno al azar
* de la ventana. Devuelve sus escrituras (para rearmar el modelo hasta un lsn: todas son sueltas y en orden).
* @param t testing.TB, tr *BTree, live model, rng *rand.Rand, i, window int
* @return []op
**/
func slideStep(t testing.TB, tr *BTree, live model, rng *rand.Rand, i, window int) []op {
	t.Helper()
	write := func(id string) op {
		value := randomValue(rng, 60)
		return op{lsn: put(t, tr, live, id, value).LSN, id: id, value: value}
	}
	ops := []op{write(longID(i))}
	if i >= window {
		if old := longID(i - window); live[old] != nil {
			ops = append(ops, op{lsn: remove(t, tr, live, old).LSN, id: old, tomb: true})
		}
		if rng.Intn(3) == 0 {
			ops = append(ops, write(longID(i-rng.Intn(window-1))))
		}
	}
	return ops
}

// TestPageDeleteConcurrent: lectores (Lookup y recorridos en los dos sentidos) mientras un escritor borra y vuelve a
// escribir tramos enteros y la limpieza borra y recicla páginas: los ids que nunca se borran se ven siempre, una sola
// vez y en orden, y ningún recorrido se cae en una página reciclada. Los recorridos retienen el horizonte (como el
// snapshot de un recorrido del motor).
func TestPageDeleteConcurrent(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 256)
	tr := createTree(t, e)
	const n = 3000
	for i := range n {
		put(t, tr, nil, longID(i), []byte("v"))
	}
	stable := func(i int) bool { return i%50 == 0 }
	var hz horizonModel
	stop := make(chan struct{})
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	run := func(f func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if err := f(); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	rng := rand.New(rand.NewSource(1))
	var deleted, recycled atomic.Int64
	run(func() error { return churn(tr, &hz, rng, n, stable) })
	run(func() error {
		st, err := tr.Vacuum(VacuumOptions{Horizon: hz.horizon(), NextXID: func() page.XID { return page.XID(hz.next.Load()) }})
		deleted.Add(int64(st.Deleted))
		recycled.Add(int64(st.Recycled))
		return err
	})
	for _, back := range []bool{false, true} {
		run(func() error { return scanStable(tr, &hz, back, n, stable) })
	}
	run(func() error { return lookupStable(tr, n, stable) })
	for range 150 {
		if err := scanStable(tr, &hz, false, n, stable); err != nil {
			close(stop)
			wg.Wait()
			t.Fatal(err)
		}
	}
	close(stop)
	wg.Wait()
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
	st, err := tr.Verify()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("al final: %+v; borradas %d, recicladas %d", st, deleted.Load(), recycled.Load())
	if deleted.Load() == 0 || recycled.Load() == 0 {
		t.Fatal("la prueba no borró ni recicló páginas mientras se leía")
	}
}

// horizonModel emula el horizonte del motor: un xid por escritura y el más viejo de los recorridos abiertos.
type horizonModel struct {
	next  atomic.Uint64
	mu    sync.Mutex
	scans map[int]page.XID
	seq   int
}

// begin anota un recorrido que empieza (su xmin es el próximo xid) y devuelve con qué cerrarlo.
func (h *horizonModel) begin() func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.scans == nil {
		h.scans = map[int]page.XID{}
	}
	h.seq++
	id := h.seq
	h.scans[id] = page.XID(h.next.Load())
	return func() {
		h.mu.Lock()
		delete(h.scans, id)
		h.mu.Unlock()
	}
}

// horizon es el xid más viejo que alguien puede necesitar.
func (h *horizonModel) horizon() page.XID {
	h.mu.Lock()
	defer h.mu.Unlock()
	hz := page.XID(h.next.Load())
	for _, x := range h.scans {
		hz = min(hz, x)
	}
	return hz
}

/**
* churn borra un tramo al azar de ids que no son estables y lo vuelve a escribir, con un xid nuevo por escritura.
* @param tr *BTree, hz *horizonModel, rng *rand.Rand, n int, stable func(int) bool
* @return error
**/
func churn(tr *BTree, hz *horizonModel, rng *rand.Rand, n int, stable func(int) bool) error {
	from := rng.Intn(n - 400)
	for _, tomb := range []bool{true, false} {
		for i := from; i < from+400; i++ {
			if stable(i) {
				continue
			}
			xid := page.XID(hz.next.Add(1))
			if _, err := tr.Write(longID(i), Write{XID: xid, Tombstone: tomb, Value: []byte("w"), Commit: true}); err != nil {
				return err
			}
		}
	}
	return nil
}

/**
* scanStable recorre todo (reteniendo el horizonte) y comprueba que los ids salgan estrictamente en orden y que estén
* todos los estables.
* @param tr *BTree, hz *horizonModel, back bool, n int, stable func(int) bool
* @return error
**/
func scanStable(tr *BTree, hz *horizonModel, back bool, n int, stable func(int) bool) error {
	done := hz.begin()
	defer done()
	var ids []string
	err := tr.scanWith(ScanOptions{Backward: back, KeysOnly: true}, func(v Version) (bool, error) {
		ids = append(ids, v.ID)
		return true, nil
	})
	if err != nil {
		return err
	}
	if back {
		slices.Reverse(ids)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			return fmt.Errorf("recorrido (atrás %v): %q no va antes de %q", back, short(ids[i-1]), short(ids[i]))
		}
	}
	for i := 0; i < n; i += 50 {
		if stable(i) {
			if _, found := slices.BinarySearch(ids, longID(i)); !found {
				return fmt.Errorf("recorrido (atrás %v): falta el id estable %d", back, i)
			}
		}
	}
	return nil
}

/**
* lookupStable lee los ids estables uno por uno.
* @param tr *BTree, n int, stable func(int) bool
* @return error
**/
func lookupStable(tr *BTree, n int, stable func(int) bool) error {
	for i := 0; i < n; i += 50 {
		if !stable(i) {
			continue
		}
		v, ok, err := tr.Lookup(longID(i), nil)
		if err != nil || !ok || !bytes.Equal(v.Value, []byte("v")) {
			return fmt.Errorf("Lookup del id estable %d: %v %v %q", i, ok, err, v.Value)
		}
	}
	return nil
}

// TestPageDeleteRedoRejects: los registros nuevos con forma o deltas inválidos no tocan páginas.
func TestPageDeleteRedoRejects(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 16)
	tr := createTree(t, e)
	m := model{}
	fillLong(t, tr, m, 0, 200)
	block := func(blk page.BlockNumber, flags wal.BlockFlags, delta []byte) wal.BlockRef {
		return wal.BlockRef{File: testFile, Fork: smgr.ForkMain, Block: blk, Flags: flags, Delta: delta}
	}
	target := unlinkTargetDelta(unlinkTarget{prev: 0, next: 3, level: 0})
	bad := []wal.Record{
		{Kind: wal.KindMarkHalfDead, Blocks: []wal.BlockRef{block(2, 0, blockDelta(0))}},
		{Kind: wal.KindMarkHalfDead, Blocks: []wal.BlockRef{block(2, 0, blockDelta(0)), block(4, 0, []byte{1, 0})}},
		{Kind: wal.KindUnlinkPage, Blocks: []wal.BlockRef{block(2, wal.BlockWillInit, target), block(3, 0, blockDelta(0))}},
		{Kind: wal.KindUnlinkPage, Blocks: []wal.BlockRef{block(2, 0, target), block(3, 0, blockDelta(0)), block(MetaBlock, 0, blockDelta(2))}},
		{Kind: wal.KindUnlinkPage, Blocks: []wal.BlockRef{block(2, wal.BlockWillInit, target), block(4, 0, blockDelta(0)), block(MetaBlock, 0, blockDelta(2))}},
		{Kind: wal.KindUnlinkPage, Blocks: []wal.BlockRef{block(2, wal.BlockWillInit, target), block(3, 0, blockDelta(0)), block(5, 0, blockDelta(2))}},
		{Kind: wal.KindRecyclePage, Blocks: []wal.BlockRef{block(MetaBlock, 0, le.AppendUint32(blockDelta(0), 7)), block(6, wal.BlockWillInit, blockDelta(0))}},
		{Kind: wal.KindRecyclePage, Blocks: []wal.BlockRef{block(MetaBlock, 0, le.AppendUint32(blockDelta(0), 6)), block(6, 0, blockDelta(0))}},
		{Kind: wal.KindInsertPivot, Blocks: []wal.BlockRef{block(2, 0, itemDelta(2, encodePivot(key{id: []byte("m"), lsn: MaxLSN}, 3))), block(3, 0, nil), block(4, 0, metaFastDelta(4, 1))}},
	}
	for i, rec := range bad {
		rec.LSN = 1 << 40
		if err := Redo(e.pool, e.sm, &rec); !errors.Is(err, ErrBadRecord) {
			t.Fatalf("registro %d: %v, se esperaba ErrBadRecord", i, err)
		}
	}
	checkTree(t, openTree(t, e), m)
}

// TestReadDeletedPages: una página borrada y una medio muerta se leen como páginas del árbol que se saltan (sin
// entradas); una borrada sin sus datos es un daño.
func TestReadDeletedPages(t *testing.T) {
	p := make(page.Page, page.Size)
	if err := initDeletedPage(p, unlinkTarget{prev: 2, next: 3, info: deletedInfo{safe: 9, next: 4}}); err != nil {
		t.Fatal(err)
	}
	n, err := readNode(5, p)
	if err != nil || !n.deleted() || !n.ignore() || n.dataCount() != 0 || deletedInfoOf(n) != (deletedInfo{safe: 9, next: 4}) {
		t.Fatalf("borrada: %v %+v", err, deletedInfoOf(n))
	}
	if err := p.DeleteItem(page.FirstOffset); err != nil {
		t.Fatal(err)
	}
	if _, err := readNode(5, p); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("borrada sin datos: %v", err)
	}
	half := samplePage(t)
	for half.MaxOffset() > page.FirstOffset {
		if err := half.DeleteItem(half.MaxOffset()); err != nil {
			t.Fatal(err)
		}
	}
	if err := applyHalfDeadLeaf(half, blockDelta(7)); err != nil {
		t.Fatal(err)
	}
	if n, err := readNode(5, half); err != nil || !n.halfDead() || leafTop(n) != 7 {
		t.Fatalf("medio muerta: %v", err)
	}
	if err := applyHalfDeadLeaf(samplePage(t), blockDelta(0)); !errors.Is(err, ErrBadRecord) {
		t.Fatalf("medio muerta con entradas: %v", err)
	}
}
