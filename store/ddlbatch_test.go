package store

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* dataFiles son los números de archivo que tienen algún archivo en data/.
* @param t testing.TB, fsys vfs.FS
* @return []smgr.FileNo: en orden, sin repetidos
**/
func dataFiles(t testing.TB, fsys vfs.FS) []smgr.FileNo {
	t.Helper()
	names, err := fsys.ReadDir(filepath.Join(testDir, smgr.DataDir))
	mustOK(t, err)
	var files []smgr.FileNo
	for _, name := range names {
		if file, ok := parseDataName(name); ok && !slices.Contains(files, file) {
			files = append(files, file)
		}
	}
	slices.Sort(files)
	return files
}

/**
* catalogFiles son los números de archivo de los stores del catálogo, en orden.
* @param e *Engine
* @return []smgr.FileNo
**/
func catalogFiles(e *Engine) []smgr.FileNo {
	var files []smgr.FileNo
	for _, entry := range e.cat.Catalog().Entries {
		files = append(files, entry.File)
	}
	slices.Sort(files)
	return files
}

/**
* checkNoOrphans exige que data/ tenga solo archivos de stores del catálogo y que Verify no encuentre nada.
* @param t testing.TB, e *Engine, fsys vfs.FS
**/
func checkNoOrphans(t testing.TB, e *Engine, fsys vfs.FS) {
	t.Helper()
	got, want := dataFiles(t, fsys), catalogFiles(e)
	for _, file := range got {
		if !slices.Contains(want, file) {
			t.Fatalf("data/ has files of %d, the catalog has %v", file, want)
		}
	}
	checkClean(t, e)
}

// Un store creado dentro de un lote: nadie lo ve hasta Commit (Stores no lo lista, Store desde fuera espera y da
// ErrStorePending, las escrituras sueltas en él dan ErrStorePending, un snapshot anterior no ve sus datos); al
// confirmar queda para todos, también tras reabrir.
func TestStoreInCommit(t *testing.T) {
	base := crashfs.New()
	cfg := testConfig()
	cfg.lockTimeout = 50 * time.Millisecond
	e := openTest(t, base, cfg)
	mustStore(t, e, "a")
	before := e.Snapshot()
	tx := e.NewBatch()
	s, err := e.StoreIn(tx, "nuevo")
	mustOK(t, err)
	mustOK(t, s.PutIn(tx, "k1", []byte("v1")))
	mustOK(t, s.PutIn(tx, "big", bytes.Repeat([]byte("x"), 20000)))
	if got, err := e.StoreIn(tx, "nuevo"); err != nil || got != s {
		t.Fatalf("StoreIn again = %p, %v; want the same store", got, err)
	}
	if _, found := readString(t, func(id string) ([]byte, bool, error) { return s.ReadIn(tx, id) }, "k1"); !found {
		t.Fatal("the batch does not see its own store")
	}
	if names := e.Stores(); slices.Contains(names, "nuevo") {
		t.Fatalf("Stores lists a store of an open batch: %v", names)
	}
	if _, err := e.Store("nuevo"); !errors.Is(err, ErrStorePending) {
		t.Fatalf("Store from outside = %v, want ErrStorePending", err)
	}
	if err := s.Put("k2", []byte("v")); !errors.Is(err, ErrStorePending) {
		t.Fatalf("single Put on a pending store = %v, want ErrStorePending", err)
	}
	if err := e.DropStore("nuevo"); !errors.Is(err, ErrStorePending) {
		t.Fatalf("DropStore of a pending store = %v, want ErrStorePending", err)
	}
	if _, found := readString(t, s.Read, "k1"); found {
		t.Fatal("a single Read sees an uncommitted store")
	}
	mustOK(t, tx.CommitDeferred()) // con DDL espera el fsync igual

	if !slices.Contains(e.Stores(), "nuevo") || mustStore(t, e, "nuevo") != s || s.Count() != 2 {
		t.Fatalf("after Commit: Stores %v, Count %d", e.Stores(), s.Count())
	}
	if _, found, err := s.ReadAt(before, "k1"); err != nil || found {
		t.Fatalf("a snapshot older than the Commit reads %v, %v", found, err)
	}
	mustOK(t, before.Close())
	mustOK(t, s.Put("k2", []byte("v2")))
	checkNoOrphans(t, e, base)
	mustOK(t, e.Close())

	e = openTest(t, base, cfg)
	defer e.Close()
	checkStoreContent(t, mustStore(t, e, "nuevo"), map[string]string{
		"k1": "v1", "k2": "v2", "big": string(bytes.Repeat([]byte("x"), 20000)),
	})
	checkNoOrphans(t, e, base)
}

// Un store creado dentro de un lote que se deshace (Rollback, tiempo máximo, o un error de serialización que
// aborta el lote) no existe: ni en el catálogo ni sus archivos, y Verify queda limpio; el nombre queda libre.
func TestStoreInRollback(t *testing.T) {
	for _, how := range []string{"rollback", "timeout", "abort"} {
		t.Run(how, func(t *testing.T) {
			base := crashfs.New()
			cfg := testConfig()
			if how == "timeout" {
				cfg.txTimeout = 100 * time.Millisecond
			}
			e := openTest(t, base, cfg)
			a := mustStore(t, e, "a")
			mustOK(t, a.Put("x", []byte("0")))
			tx := e.NewBatch()
			s, err := e.StoreIn(tx, "tmp")
			mustOK(t, err)
			mustOK(t, s.PutIn(tx, "k", bytes.Repeat([]byte("y"), 30000)))
			switch how {
			case "rollback":
				mustOK(t, tx.Rollback())
			case "timeout":
				time.Sleep(300 * time.Millisecond)
				if err := s.PutIn(tx, "k2", []byte("v")); !errors.Is(err, ErrTimeout) {
					t.Fatalf("PutIn after the time limit = %v, want ErrTimeout", err)
				}
			case "abort":
				mustOK(t, a.Put("x", []byte("2"))) // otro cambia x y confirma después del snapshot del lote
				if err := a.PutIn(tx, "x", []byte("3")); !errors.Is(err, ErrSerialization) {
					t.Fatalf("PutIn = %v, want ErrSerialization", err)
				}
				mustOK(t, tx.Rollback())
			}
			if slices.Contains(e.Stores(), "tmp") {
				t.Fatalf("the store of an undone batch is still listed: %v", e.Stores())
			}
			if err := s.Put("k", []byte("v")); !errors.Is(err, ErrStoreDropped) {
				t.Fatalf("Put on the undone store = %v, want ErrStoreDropped", err)
			}
			checkNoOrphans(t, e, base)
			mustOK(t, mustStore(t, e, "tmp").Put("k", []byte("suelto")))
			mustOK(t, e.Close())

			e = openTest(t, base, cfg)
			defer e.Close()
			checkStoreContent(t, mustStore(t, e, "tmp"), map[string]string{"k": "suelto"})
			checkNoOrphans(t, e, base)
		})
	}
}

// Borrar un store dentro de un lote: sigue entero hasta Commit (lecturas y escrituras sueltas siguen; Rollback lo
// deja como estaba); al confirmar desaparece con sus archivos, también tras reabrir.
func TestDropStoreIn(t *testing.T) {
	for _, commit := range []bool{true, false} {
		t.Run(fmt.Sprintf("commit=%v", commit), func(t *testing.T) {
			base := crashfs.New()
			e := openTest(t, base, testConfig())
			s, a := mustStore(t, e, "viejo"), mustStore(t, e, "a")
			mustOK(t, s.Put("k", bytes.Repeat([]byte("z"), 25000)))
			tx := e.NewBatch()
			mustOK(t, a.PutIn(tx, "k", []byte("lote")))
			mustOK(t, e.DropStoreIn(tx, "viejo"))
			if err := e.DropStoreIn(tx, "viejo"); !errors.Is(err, ErrStoreDropped) {
				t.Fatalf("DropStoreIn twice = %v, want ErrStoreDropped", err)
			}
			if _, err := e.StoreIn(tx, "viejo"); !errors.Is(err, ErrStoreDropped) {
				t.Fatalf("StoreIn of a store the batch drops = %v, want ErrStoreDropped", err)
			}
			if err := e.DropStoreIn(tx, "nada"); !errors.Is(err, ErrStoreNotFound) {
				t.Fatalf("DropStoreIn of a missing store = %v, want ErrStoreNotFound", err)
			}
			mustOK(t, s.Put("k2", []byte("suelto"))) // suelta: confirma antes que el borrado
			if !slices.Contains(e.Stores(), "viejo") || s.Count() != 2 {
				t.Fatalf("before Commit: Stores %v, Count %d", e.Stores(), s.Count())
			}
			if commit {
				mustOK(t, tx.Commit())
			} else {
				mustOK(t, tx.Rollback())
			}
			if got := slices.Contains(e.Stores(), "viejo"); got == commit {
				t.Fatalf("after the batch, listed = %v", got)
			}
			if commit {
				if _, _, err := s.Read("k"); !errors.Is(err, ErrStoreDropped) {
					t.Fatalf("Read of the dropped store = %v, want ErrStoreDropped", err)
				}
			}
			checkNoOrphans(t, e, base)
			mustOK(t, e.Close())

			e = openTest(t, base, testConfig())
			defer e.Close()
			if got := slices.Contains(e.Stores(), "viejo"); got == commit {
				t.Fatalf("after reopening, listed = %v", got)
			}
			if !commit {
				checkStoreContent(t, mustStore(t, e, "viejo"), map[string]string{
					"k": string(bytes.Repeat([]byte("z"), 25000)), "k2": "suelto",
				})
			}
			checkNoOrphans(t, e, base)
		})
	}
}

// Crear y borrar el mismo store dentro de un lote lo borra enseguida, sin dejar nada.
func TestStoreInDropOwn(t *testing.T) {
	base := crashfs.New()
	e := openTest(t, base, testConfig())
	defer e.Close()
	tx := e.NewBatch()
	s, err := e.StoreIn(tx, "fugaz")
	mustOK(t, err)
	mustOK(t, s.PutIn(tx, "k", []byte("v")))
	mustOK(t, e.DropStoreIn(tx, "fugaz"))
	mustOK(t, tx.Commit())
	if slices.Contains(e.Stores(), "fugaz") {
		t.Fatal("the store created and dropped in a batch exists")
	}
	checkNoOrphans(t, e, base)
}

// Dos lotes que crean el mismo nombre: el segundo espera al primero; si confirmó recibe ErrSerialization (su
// snapshot no lo veía), si deshizo lo crea él. Un lote que empieza después del Commit lo abre.
func TestStoreInSameName(t *testing.T) {
	for _, commit := range []bool{true, false} {
		t.Run(fmt.Sprintf("commit=%v", commit), func(t *testing.T) {
			e := newTestEngine(t, nil)
			tx1, tx2 := e.NewBatch(), e.NewBatch()
			_, err := e.StoreIn(tx1, "n")
			mustOK(t, err)
			done := make(chan error, 1)
			go func() {
				_, err := e.StoreIn(tx2, "n")
				done <- err
			}()
			time.Sleep(50 * time.Millisecond)
			select {
			case err := <-done:
				t.Fatalf("the second StoreIn did not wait: %v", err)
			default:
			}
			if commit {
				mustOK(t, tx1.Commit())
			} else {
				mustOK(t, tx1.Rollback())
			}
			err = <-done
			switch {
			case commit && !errors.Is(err, ErrSerialization):
				t.Fatalf("second StoreIn = %v, want ErrSerialization", err)
			case !commit && err != nil:
				t.Fatalf("second StoreIn after a Rollback = %v", err)
			}
			if commit {
				_ = tx2.Rollback()
				tx3 := e.NewBatch()
				if _, err := e.StoreIn(tx3, "n"); err != nil {
					t.Fatalf("StoreIn after the Commit: %v", err)
				}
				mustOK(t, tx3.Commit())
				return
			}
			mustOK(t, tx2.Commit())
			if !slices.Contains(e.Stores(), "n") {
				t.Fatal("the store of the second batch is missing")
			}
		})
	}
}

// Borrar un store en el que escribe otro lote abierto da ErrBusy; cuando ese lote termina, sí. Mientras un lote lo
// borra, las escrituras de otros lotes en él dan ErrStorePending y otro DropStoreIn espera: si el primero confirma,
// ErrSerialization.
func TestDropStoreInBusy(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	mustOK(t, s.Put("k", []byte("v")))
	writer, dropper := e.NewBatch(), e.NewBatch()
	mustOK(t, s.PutIn(writer, "w", []byte("1")))
	if err := e.DropStoreIn(dropper, "s"); !errors.Is(err, ErrBusy) {
		t.Fatalf("DropStoreIn of a store another batch wrote = %v, want ErrBusy", err)
	}
	mustOK(t, writer.Commit())
	mustOK(t, e.DropStoreIn(dropper, "s"))

	other := e.NewBatch()
	if err := s.PutIn(other, "o", []byte("1")); !errors.Is(err, ErrStorePending) {
		t.Fatalf("PutIn of another batch = %v, want ErrStorePending", err)
	}
	if _, found := readString(t, func(id string) ([]byte, bool, error) { return s.ReadIn(other, id) }, "w"); !found {
		t.Fatal("another batch cannot read the store being dropped")
	}
	done := make(chan error, 1)
	go func() { done <- e.DropStoreIn(other, "s") }()
	time.Sleep(50 * time.Millisecond)
	mustOK(t, dropper.Commit())
	if err := <-done; !errors.Is(err, ErrSerialization) {
		t.Fatalf("the second DropStoreIn = %v, want ErrSerialization", err)
	}
	_ = other.Rollback()
	if slices.Contains(e.Stores(), "s") {
		t.Fatal("the dropped store is still listed")
	}
}

// Un lote con DDL sin confirmar cuando el motor cierra o se corta: al reabrir, el store creado no existe ni deja
// archivos y el borrado sigue entero.
func TestDDLBatchOpenAtClose(t *testing.T) {
	for _, how := range []string{"close", "crash"} {
		t.Run(how, func(t *testing.T) {
			base := crashfs.New()
			proc := newProc(base)
			e := openTest(t, proc, testConfig())
			old := mustStore(t, e, "old")
			mustOK(t, old.Put("k", []byte("v")))
			tx := e.NewBatch()
			s, err := e.StoreIn(tx, "new")
			mustOK(t, err)
			for i := range 200 {
				mustOK(t, s.PutIn(tx, fmt.Sprintf("k%03d", i), bytes.Repeat([]byte("n"), 100)))
			}
			mustOK(t, e.DropStoreIn(tx, "old"))
			mustOK(t, e.Checkpoint()) // el catálogo guardado tiene el store pendiente
			if how == "close" {
				mustOK(t, e.Close())
			} else {
				proc.kill(crashfs.LoseUnsynced, 1)
				closeWithin(t, e, 10*time.Second)
			}
			e = openTest(t, newProc(base), testConfig())
			defer e.Close()
			if got := e.Stores(); !slices.Equal(got, []string{"old"}) {
				t.Fatalf("after reopening, stores %v", got)
			}
			checkStoreContent(t, mustStore(t, e, "old"), map[string]string{"k": "v"})
			checkNoOrphans(t, e, base)
		})
	}
}

// ddlBefore y ddlAfter son los estados antes y después del lote de TestDDLBatchCrash.
var (
	ddlBefore = state{"a": {"a1": "1"}, "b": {"b1": "1", "b2": string(bytes.Repeat([]byte("b"), 12000))}}
	ddlAfter  = state{"a": {"a1": "1", "a2": "2"}, "c": {"c1": "1", "c2": string(bytes.Repeat([]byte("c"), 12000))}}
)

/**
* ddlImage arma el disco de partida de TestDDLBatchCrash: los stores a y b con ddlBefore, cerrado.
* @param t testing.TB
* @return diskImage
**/
func ddlImage(t testing.TB) diskImage {
	base := crashfs.New()
	e := openTest(t, base, testConfig())
	for name, ids := range ddlBefore {
		s := mustStore(t, e, name)
		for id, v := range ids {
			mustOK(t, s.Put(id, []byte(v)))
		}
	}
	mustOK(t, e.Close())
	return readImage(t, base, testDir)
}

/**
* runDDLBatch hace el lote de TestDDLBatchCrash: crea c y escribe en él, borra b y escribe en a, y confirma.
* @param e *Engine
* @return error: el primero
**/
func runDDLBatch(e *Engine) error {
	return runDDLBatchWith(e, false)
}

/**
* runDDLBatchWith es runDDLBatch, confirmando con CommitDeferred si deferred.
* @param e *Engine, deferred bool
* @return error: el primero
**/
func runDDLBatchWith(e *Engine, deferred bool) error {
	tx := e.NewBatch()
	defer tx.Rollback()
	c, err := e.StoreIn(tx, "c")
	if err != nil {
		return err
	}
	for id, v := range ddlAfter["c"] {
		if err := c.PutIn(tx, id, []byte(v)); err != nil {
			return err
		}
	}
	if err := e.DropStoreIn(tx, "b"); err != nil {
		return err
	}
	a, err := e.StoreIn(tx, "a")
	if err != nil {
		return err
	}
	if err := a.PutIn(tx, "a2", []byte("2")); err != nil {
		return err
	}
	if deferred {
		return tx.CommitDeferred()
	}
	return tx.Commit()
}

// Un corte en cada operación de disco de un lote con DDL, en los tres modos de crashfs: al reabrir el estado es el
// de antes o el de después entero (el de después si Commit volvió bien), Count coincide, data/ no tiene archivos
// huérfanos y Verify queda limpio.
//
// La variante «nosync» (SYNC_ON_WRITE=false y CommitDeferred) exige lo mismo: un lote con DDL siempre espera el
// fsync de su KindCommit, porque el catálogo y el borrado de archivos que le siguen ya son durables (tarea 88).
func TestDDLBatchCrash(t *testing.T) {
	img := ddlImage(t)
	for _, nosync := range []bool{false, true} {
		cfg := testConfig()
		cfg.syncOnWrite = !nosync
		proc := newProc(writeImage(t, img))
		e := openTest(t, proc, cfg)
		proc.killAfter(1<<40, crashfs.LoseUnsynced, 0)
		mustOK(t, runDDLBatchWith(e, nosync))
		total := 1<<40 - proc.budget.Load()
		mustOK(t, e.Close())
		t.Logf("nosync=%v: the DDL batch does %d disk operations", nosync, total)
		modes := map[string]crashfs.Mode{
			"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
		}
		for name, mode := range modes {
			for cut := int64(1); cut <= total+1; cut++ {
				t.Run(fmt.Sprintf("nosync=%v/%s/%d", nosync, name, cut), func(t *testing.T) {
					crashDDLBatch(t, img, cfg, nosync, mode, cut)
				})
			}
		}
	}
}

/**
* crashDDLBatch corta el lote de TestDDLBatchCrash tras cut operaciones de disco, reabre y comprueba el estado.
* @param t *testing.T, img diskImage, cfg config, deferred bool: confirmar con CommitDeferred, mode crashfs.Mode,
* cut int64
**/
func crashDDLBatch(t *testing.T, img diskImage, cfg config, deferred bool, mode crashfs.Mode, cut int64) {
	base := writeImage(t, img)
	proc := newProc(base)
	e := openTest(t, proc, cfg)
	proc.killAfter(cut, mode, cut)
	err := runDDLBatchWith(e, deferred)
	if proc.budget.Load() > 0 {
		proc.kill(mode, cut) // el lote terminó antes del corte: el corte llega al final
	}
	closeWithin(t, e, 10*time.Second)

	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	got := engineState(t, e)
	switch {
	case err == nil && !got.equal(ddlAfter):
		t.Fatalf("Commit returned nil but the state is %v", unionNames(got, state{}))
	case !got.equal(ddlBefore) && !got.equal(ddlAfter):
		t.Fatalf("after a cut the state is neither before nor after: stores %v", unionNames(got, state{}))
	}
	checkNoOrphans(t, e, base)
}

// Varios lotes crean, escriben y borran los mismos stores a la vez, con operaciones sueltas y checkpoints: nada se
// cuelga, los errores son los de las reglas de concurrencia y al final Verify queda limpio, también tras reabrir.
func TestDDLBatchConcurrent(t *testing.T) {
	base := crashfs.New()
	cfg := testConfig()
	cfg.lockTimeout = 20 * time.Millisecond
	e := openTest(t, base, cfg)
	allowed := []error{ErrSerialization, ErrConflict, ErrBusy, ErrStorePending, ErrStoreDropped, ErrStoreNotFound}
	ok := func(err error) bool {
		return err == nil || slices.ContainsFunc(allowed, func(want error) bool { return errors.Is(err, want) })
	}
	names := []string{"p", "q", "r"}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for w := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 60 {
				name := names[(w+i)%len(names)]
				if err := concurrentDDL(e, name, w, i); !ok(err) {
					errs <- fmt.Errorf("worker %d op %d on %s: %w", w, i, name, err)
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 30 {
			if s, err := e.Store(names[i%len(names)]); err == nil {
				_ = s.Put("suelto", []byte("v"))
			} else if !ok(err) {
				errs <- err
				return
			}
			if err := e.Checkpoint(); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	checkNoOrphans(t, e, base)
	engineState(t, e)
	mustOK(t, e.Close())
	e = openTest(t, base, cfg)
	defer e.Close()
	checkNoOrphans(t, e, base)
	engineState(t, e)
}

/**
* concurrentDDL hace un lote al azar de TestDDLBatchConcurrent: crea o abre name y escribe, a veces lo borra, y
* confirma o deshace.
* @param e *Engine, name string, w, i int
* @return error
**/
func concurrentDDL(e *Engine, name string, w, i int) error {
	tx := e.NewBatch()
	defer tx.Rollback()
	if (w+i)%5 == 0 {
		if err := e.DropStoreIn(tx, name); err != nil {
			return err
		}
	} else {
		s, err := e.StoreIn(tx, name)
		if err != nil {
			return err
		}
		if err := s.PutIn(tx, fmt.Sprintf("w%d-%d", w, i), []byte("x")); err != nil {
			return err
		}
	}
	if i%3 == 0 {
		return tx.Rollback()
	}
	return tx.Commit()
}
