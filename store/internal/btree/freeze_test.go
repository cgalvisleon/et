package btree

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// errStatusRead es lo que devuelve noStatus: una lectura del archivo de estados que no debía hacerse.
var errStatusRead = errors.New("test: status read of a frozen version")

// noStatus es un archivo de estados que falla en toda consulta: prueba que una versión congelada no lo consulta.
type noStatus struct{}

func (noStatus) Status(page.XID) (xact.Status, page.LSN, error) { return 0, 0, errStatusRead }

// leafInfo es una hoja leída para las pruebas: su bloque, si tiene page.FlagAllVisible, sus bits del mapa y sus
// entradas.
type leafInfo struct {
	blk     page.BlockNumber
	visible bool
	bits    fsm.VMBits
	entries []Version
}

/**
* vmTree crea el store de prueba con archivo de estados y mapa de visibilidad.
* @param t testing.TB, e *env, status StatusReader, frozen page.XID: el frozenXID de la meta
* @return *BTree
**/
func vmTree(t testing.TB, e *env, status StatusReader, frozen page.XID) *BTree {
	t.Helper()
	cfg := e.config()
	cfg.Status, cfg.VM, cfg.FrozenXID = status, e.free, frozen
	tr, err := Create(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

/**
* reopenVM abre el store de prueba ya creado con archivo de estados y mapa.
* @param t testing.TB, e *env, status StatusReader
* @return *BTree
**/
func reopenVM(t testing.TB, e *env, status StatusReader) *BTree {
	t.Helper()
	cfg := e.config()
	cfg.Status, cfg.VM = status, e.free
	tr, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

/**
* leaves lee todas las hojas de izquierda a derecha con sus bits del mapa.
* @param t testing.TB, e *env, tr *BTree
* @return []leafInfo
**/
func leaves(t testing.TB, e *env, tr *BTree) []leafInfo {
	t.Helper()
	h, n, _, err := tr.descend(minusInf, 0, buffer.Share, false)
	if err != nil {
		t.Fatal(err)
	}
	var out []leafInfo
	for {
		li := leafInfo{blk: n.blk, visible: n.p.Flags()&page.FlagAllVisible != 0}
		for off := n.firstData(); off <= n.maxOff(); off++ {
			en, err := n.entryAt(off)
			if err != nil {
				t.Fatal(err)
			}
			li.entries = append(li.entries, en.version())
		}
		out = append(out, li)
		if n.rightmost() {
			h.Release()
			break
		}
		if h, n, err = tr.stepRight(h, n); err != nil {
			t.Fatal(err)
		}
	}
	for i := range out {
		if out[i].bits, err = e.free.Get(testFile, out[i].blk); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

/**
* checkVMInvariant comprueba lo que la limpieza puede dar por cierto al saltar una hoja: con bit, la hoja tiene
* page.FlagAllVisible; con el bit de todo-congelada, todas sus versiones están congeladas; y ninguna versión congelada
* es de una transacción abortada.
* @param t testing.TB, e *env, tr *BTree, status *lsnStatus
**/
func checkVMInvariant(t testing.TB, e *env, tr *BTree, status *lsnStatus) {
	t.Helper()
	for _, li := range leaves(t, e, tr) {
		if li.bits != 0 && !li.visible {
			t.Fatalf("hoja %d: bits %d sin FlagAllVisible", li.blk, li.bits)
		}
		for _, v := range li.entries {
			if li.bits != 0 && v.Flags&EntryFreeing != 0 {
				t.Fatalf("hoja %d con bits y %s/%d «liberándose»", li.blk, v.ID, v.LSN)
			}
			if li.bits&fsm.VMAllFrozen != 0 && v.Flags&EntryFrozen == 0 {
				t.Fatalf("hoja %d todo-congelada con %s/%d sin congelar", li.blk, v.ID, v.LSN)
			}
			if v.Flags&EntryFrozen != 0 && status.get(v.XID) == xact.StatusAborted {
				t.Fatalf("versión congelada de una transacción abortada: %s xid %d", v.ID, v.XID)
			}
		}
	}
}

/**
* fillCommitted escribe n ids (uno por transacción, confirmadas) desde el xid first.
* @param t testing.TB, tr *BTree, status *lsnStatus, first page.XID, n int, m model
* @return page.XID: el siguiente xid libre
**/
func fillCommitted(t testing.TB, tr *BTree, status *lsnStatus, first page.XID, n int, m model) page.XID {
	t.Helper()
	xid := first
	for i := range n {
		id := fmt.Sprintf("id%04d", i)
		value := []byte(fmt.Sprintf("value-%04d-%s", i, bytes.Repeat([]byte("x"), 60)))
		writeAs(t, tr, xid, id, string(value))
		status.set(xid, xact.StatusCommitted, 0)
		m[id] = value
		xid++
	}
	return xid
}

// TestFreezeSkipsFrozenLeaves: una limpieza con FreezeLimit = Horizon congela todo lo confirmado, pone page.FlagAllVisible
// y el bit todo-congelado de cada hoja, cubre el store entero (Complete, FrozenXID = Horizon) y la siguiente salta
// todas las hojas sin escribir nada en el WAL. Una escritura borra el bit de su hoja (KindVMClear antes del registro de
// la hoja) y la pasada siguiente lee solo esa hoja. Las versiones congeladas se leen sin consultar el archivo de
// estados.
func TestFreezeSkipsFrozenLeaves(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	status := newLSNStatus()
	tr := vmTree(t, e, status, 100)
	m := model{}
	next := fillCommitted(t, tr, status, 100, 900, m)

	st, err := tr.Vacuum(VacuumOptions{Horizon: next, FreezeLimit: next, Free: freeWith(e)})
	if err != nil || !st.Complete || st.FrozenXID != next || st.Frozen != 900 || st.Skipped != 0 {
		t.Fatalf("primera pasada: %+v %v", st, err)
	}
	all := leaves(t, e, tr)
	if len(all) < 4 {
		t.Fatalf("solo %d hojas: la prueba necesita un árbol de dos niveles", len(all))
	}
	for _, li := range all {
		if li.bits != fsm.VMAllFrozen|fsm.VMAllVisible || !li.visible {
			t.Fatalf("hoja %d: bits %d, FlagAllVisible %v", li.blk, li.bits, li.visible)
		}
	}
	checkVMInvariant(t, e, tr, status)

	before := len(walRecords(t, e, wal.KindLeafVacuum)) + len(walRecords(t, e, wal.KindVMSet))
	st, err = tr.Vacuum(VacuumOptions{Horizon: next + 5, FreezeLimit: next + 5, Free: freeWith(e)})
	if err != nil || !st.Complete || st.Leaves != 0 || st.Skipped != len(all) || st.FrozenXID != next+5 {
		t.Fatalf("segunda pasada: %+v %v", st, err)
	}
	if after := len(walRecords(t, e, wal.KindLeafVacuum)) + len(walRecords(t, e, wal.KindVMSet)); after != before {
		t.Fatalf("la pasada que salta todo escribió %d registros", after-before)
	}

	frozen := reopenVM(t, e, noStatus{})
	checkTree(t, frozen, m)

	writeAs(t, tr, next, "id0450", "nuevo")
	status.set(next, xact.StatusCommitted, 0)
	m["id0450"] = []byte("nuevo")
	changed := 0
	for _, li := range leaves(t, e, tr) {
		if li.bits == 0 {
			changed++
			if li.visible {
				t.Fatalf("hoja %d sin bits con FlagAllVisible", li.blk)
			}
		}
	}
	if changed != 1 || len(walRecords(t, e, wal.KindVMClear)) != 1 {
		t.Fatalf("%d hojas sin bits, %d KindVMClear; se esperaba 1 y 1", changed, len(walRecords(t, e, wal.KindVMClear)))
	}
	st, err = tr.Vacuum(VacuumOptions{Horizon: next + 1, FreezeLimit: next + 1, Free: freeWith(e)})
	if err != nil || !st.Complete || st.Leaves != 1 || st.Skipped != len(all)-1 || st.Removed != 1 {
		t.Fatalf("pasada tras la escritura: %+v %v", st, err)
	}
	checkVMInvariant(t, e, tr, status)
	checkTree(t, tr, m)
}

// TestFreezeMinAge: con FreezeLimit < Horizon las versiones entre los dos quedan visibles sin congelar (bit
// todo-visible y FrozenXID = la más vieja de ellas); una pasada normal salta esas hojas y no cubre el store
// (Complete = false), una agresiva las lee, las congela y sí. Las abortadas por debajo del horizonte nunca se
// congelan: se quitan, en línea o en desborde, y las en curso quedan sin congelar.
func TestFreezeMinAge(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 64)
	status := newLSNStatus()
	tr := vmTree(t, e, status, 1)
	m := model{}
	rng := rand.New(rand.NewSource(3))
	next := fillCommitted(t, tr, status, 100, 600, m)
	for i := range 40 { // abortadas por debajo del horizonte, algunas con desborde, en las hojas que se congelan
		id := fmt.Sprintf("id%04d", rng.Intn(300))
		value := randomValue(rng, 50)
		if i%4 == 0 {
			value = fixedValue(rng, bigValue)
		}
		if _, err := writeValue(t, e, tr, next, id, value, false); err != nil {
			t.Fatal(err)
		}
		status.set(next, xact.StatusAborted, 0)
		next++
	}
	horizon := next
	inProgress := horizon
	writeAs(t, tr, inProgress, "id0599", "en curso")

	st, err := tr.Vacuum(VacuumOptions{Horizon: horizon, FreezeLimit: 400, Free: freeWith(e)})
	if err != nil || !st.Complete || st.FrozenXID != 400 {
		t.Fatalf("pasada con edad mínima: %+v %v", st, err)
	}
	visibleOnly := 0
	for _, li := range leaves(t, e, tr) {
		for _, v := range li.entries {
			switch {
			case status.get(v.XID) == xact.StatusAborted:
				t.Fatalf("quedó una versión abortada: %s xid %d", v.ID, v.XID)
			case v.XID < 400 && v.Flags&EntryFrozen == 0:
				t.Fatalf("%s xid %d sin congelar", v.ID, v.XID)
			case v.XID >= 400 && v.Flags&EntryFrozen != 0:
				t.Fatalf("%s xid %d congelada antes de tiempo", v.ID, v.XID)
			}
		}
		if li.bits == fsm.VMAllVisible {
			visibleOnly++
		}
	}
	if visibleOnly == 0 {
		t.Fatal("ninguna hoja quedó solo todo-visible")
	}
	checkVMInvariant(t, e, tr, status)

	st, err = tr.Vacuum(VacuumOptions{Horizon: horizon, FreezeLimit: 400, Free: freeWith(e)})
	if err != nil || st.Complete || st.SkippedVisible != visibleOnly {
		t.Fatalf("pasada normal: %+v %v (se esperaban %d todo-visibles saltadas)", st, err, visibleOnly)
	}
	st, err = tr.Vacuum(VacuumOptions{Horizon: horizon, FreezeLimit: horizon, Aggressive: true, Free: freeWith(e)})
	if err != nil || !st.Complete || st.SkippedVisible != 0 || st.FrozenXID != inProgress {
		t.Fatalf("pasada agresiva: %+v %v", st, err)
	}
	for _, li := range leaves(t, e, tr) {
		for _, v := range li.entries {
			if v.XID != inProgress && v.Flags&EntryFrozen == 0 {
				t.Fatalf("%s xid %d sin congelar tras la agresiva", v.ID, v.XID)
			}
			if v.XID == inProgress && v.Flags&EntryFrozen != 0 {
				t.Fatal("se congeló una versión en curso")
			}
		}
	}
	checkVMInvariant(t, e, tr, status)
	if _, err := tr.Verify(); err != nil {
		t.Fatal(err)
	}
	checkBlocks(t, e, tr)
}

// TestFrozenXIDMeta: el frozenXID nace con Create, solo sube con SetFrozenXID (KindSetFrozen) y sobrevive a un
// corte y al rehacer.
func TestFrozenXIDMeta(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 16)
	tr := vmTree(t, e, newLSNStatus(), 77)
	if got, err := tr.FrozenXID(); err != nil || got != 77 {
		t.Fatalf("FrozenXID = %d, %v; se esperaba 77", got, err)
	}
	for _, x := range []page.XID{50, 77, 900} {
		if err := tr.SetFrozenXID(x); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := tr.FrozenXID(); got != 900 || len(walRecords(t, e, wal.KindSetFrozen)) != 1 {
		t.Fatalf("FrozenXID = %d con %d KindSetFrozen; se esperaba 900 y 1", got, len(walRecords(t, e, wal.KindSetFrozen)))
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e2 := openEnv(t, fs, wal.Start, 16)
	if got, err := reopenVM(t, e2, newLSNStatus()).FrozenXID(); err != nil || got != 900 {
		t.Fatalf("tras el corte FrozenXID = %d, %v", got, err)
	}
}

// TestFreezeCrash corta el disco en medio de una limpieza que congela (el WAL durable hasta una cadena de desborde a
// mitad de la pasada) y en medio de escrituras sobre hojas congeladas, en los tres modos de crashfs: tras rehacer, toda
// hoja con bit tiene page.FlagAllVisible, toda todo-congelada tiene todas sus versiones congeladas, ninguna versión
// congelada es abortada, lo confirmado se lee igual y la pasada siguiente termina el trabajo.
func TestFreezeCrash(t *testing.T) {
	modes := []struct {
		name string
		mode crashfs.Mode
	}{{"lose", crashfs.LoseUnsynced}, {"reorder", crashfs.ReorderUnsynced}, {"torn", crashfs.TornWrites}}
	for _, md := range modes {
		for seed := range int64(3) {
			for _, during := range []string{"vacuum", "writes"} {
				t.Run(fmt.Sprintf("%s/%s/seed%d", md.name, during, seed), func(t *testing.T) {
					freezeCrash(t, md.mode, seed, during)
				})
			}
		}
	}
}

/**
* freezeCrash es un caso de TestFreezeCrash.
* @param t *testing.T, mode crashfs.Mode, seed int64, during string: "vacuum" o "writes"
**/
func freezeCrash(t *testing.T, mode crashfs.Mode, seed int64, during string) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 32)
	status := newLSNStatus()
	tr := vmTree(t, e, status, 1)
	rng := rand.New(rand.NewSource(seed))
	m := model{}
	xid := page.XID(100)
	for i := range 500 {
		id := fmt.Sprintf("id%04d", rng.Intn(400))
		value := randomValue(rng, 90)
		if i%25 == 0 {
			value = fixedValue(rng, bigValue)
		}
		if _, err := writeValue(t, e, tr, xid, id, value, false); err != nil {
			t.Fatal(err)
		}
		if rng.Intn(6) == 0 {
			status.set(xid, xact.StatusAborted, 0)
		} else {
			status.set(xid, xact.StatusCommitted, 0)
			m[id] = value
		}
		xid++
	}
	redo := checkpoint(t, e)
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	cut := 1 + rng.Intn(10)
	free := func(v Version) error {
		if cut--; cut == 0 && during == "vacuum" {
			if err := e.log.WaitDurable(e.log.Published()); err != nil {
				return err
			}
		}
		return freeWith(e)(v)
	}
	if _, err := tr.Vacuum(VacuumOptions{Horizon: xid, FreezeLimit: xid, Free: free}); err != nil {
		t.Fatal(err)
	}
	checkVMInvariant(t, e, tr, status)
	if during == "writes" {
		if err := e.log.WaitDurable(e.log.Published()); err != nil {
			t.Fatal(err)
		}
		for i := range 60 {
			id := fmt.Sprintf("id%04d", rng.Intn(400))
			value := randomValue(rng, 90)
			if _, err := writeValue(t, e, tr, xid, id, value, false); err != nil {
				t.Fatal(err)
			}
			status.set(xid, xact.StatusAborted, 0) // ninguna escritura posterior cuenta: el modelo no cambia
			xid++
			if i == 30 {
				if err := e.log.WaitDurable(e.log.Published()); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	fs.Crash(mode, seed)
	e2 := openEnv(t, fs, redo, 32)
	tr2 := reopenVM(t, e2, status)
	checkVMInvariant(t, e2, tr2, status)
	for _, li := range leaves(t, e2, tr2) {
		for _, v := range li.entries {
			if li.bits != 0 && v.XID >= xid-60 && during == "writes" {
				t.Fatalf("hoja %d con bit y una versión escrita después de congelar (xid %d)", li.blk, v.XID)
			}
		}
	}
	for pass := range 2 { // la segunda pone los bits de las hojas que tenían versiones «liberándose»
		st, err := tr2.Vacuum(VacuumOptions{Horizon: xid, FreezeLimit: xid, Free: freeWith(e2)})
		if err != nil || !st.Complete {
			t.Fatalf("pasada %d tras el corte: %+v %v", pass, st, err)
		}
		checkVMInvariant(t, e2, tr2, status)
	}
	all := leaves(t, e2, tr2)
	for i, li := range all {
		if li.bits != fsm.VMAllFrozen|fsm.VMAllVisible && !sharesID(all, i) {
			t.Fatalf("hoja %d sin el bit todo-congelado tras la pasada completa (bits %d)", li.blk, li.bits)
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
}

/**
* sharesID dice si la hoja i comparte un id con una vecina (la limpieza no le pone bits: no podría saltarla).
* @param all []leafInfo: las hojas en orden, i int
* @return bool
**/
func sharesID(all []leafInfo, i int) bool {
	firstID := func(li leafInfo) string { return li.entries[0].ID }
	lastID := func(li leafInfo) string { return li.entries[len(li.entries)-1].ID }
	li := all[i]
	if len(li.entries) == 0 {
		return false
	}
	left := i > 0 && len(all[i-1].entries) > 0 && lastID(all[i-1]) == firstID(li)
	right := i+1 < len(all) && len(all[i+1].entries) > 0 && firstID(all[i+1]) == lastID(li)
	return left || right
}

// TestFreezeConcurrentSplits: limpiezas con mapa (que saltan hojas) mientras otra goroutine escribe ids nuevos y
// versiones nuevas de los viejos, partiendo hojas durante el recorrido por los padres. Tras cada pasada completa,
// toda versión con xid < FrozenXID está congelada (o «liberándose»): ninguna hoja nueva de una partición quedó sin
// ver, y las hojas con bit cumplen el invariante.
func TestFreezeConcurrentSplits(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 256)
	status := newLSNStatus()
	tr := vmTree(t, e, status, 1)
	next := fillCommitted(t, tr, status, 100, 2500, model{})
	var xid atomic.Uint64
	xid.Store(uint64(next))
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		rng := rand.New(rand.NewSource(9))
		for i := 0; ; i++ {
			select {
			case <-stop:
				done <- nil
				return
			default:
			}
			x := page.XID(xid.Add(1) - 1)
			id := fmt.Sprintf("id%04d", rng.Intn(2500))
			if i%2 == 0 {
				id = fmt.Sprintf("new%06d", rng.Intn(1_000_000))
			}
			if _, err := tr.Write(id, Write{XID: x, Value: randomValue(rng, 120)}); err != nil {
				done <- err
				return
			}
			if rng.Intn(5) == 0 {
				status.set(x, xact.StatusAborted, 0)
			} else {
				status.set(x, xact.StatusCommitted, 0)
			}
		}
	}()
	for pass := range 10 {
		horizon := page.XID(xid.Load()) // todo xid menor ya terminó o todavía no escribió
		st, err := tr.Vacuum(VacuumOptions{Horizon: horizon - 1, FreezeLimit: horizon - 1, Free: freeWith(e)})
		if err != nil || !st.Complete {
			close(stop)
			t.Fatalf("pasada %d: %+v %v", pass, st, err)
		}
		t.Logf("pasada %d: %+v", pass, st)
		for _, v := range leafEntries(t, tr) {
			if v.XID < st.FrozenXID && v.Flags&(EntryFrozen|EntryFreeing) == 0 {
				close(stop)
				t.Fatalf("pasada %d: %s/%d xid %d < FrozenXID %d sin congelar", pass, v.ID, v.LSN, v.XID, st.FrozenXID)
			}
		}
	}
	close(stop)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	checkVMInvariant(t, e, tr, status)
	if _, err := tr.Verify(); err != nil {
		t.Fatal(err)
	}
}
