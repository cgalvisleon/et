package store

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* newTestEngine abre un motor nuevo sobre crashfs con la configuración de prueba cambiada por tweak.
* @param t testing.TB, tweak func(*config): puede ser nil
* @return *Engine
**/
func newTestEngine(t testing.TB, tweak func(*config)) *Engine {
	t.Helper()
	cfg := testConfig()
	if tweak != nil {
		tweak(&cfg)
	}
	e := openTest(t, crashfs.New(), cfg)
	t.Cleanup(func() { e.Close() })
	return e
}

/**
* readString lee id y falla la prueba si hay error; "" y false si no existe.
* @param t testing.TB, read func(string) ([]byte, bool, error), id string
* @return string, bool
**/
func readString(t testing.TB, read func(string) ([]byte, bool, error), id string) (string, bool) {
	t.Helper()
	v, found, err := read(id)
	if err != nil {
		t.Fatalf("read %q: %v", id, err)
	}
	return string(v), found
}

// Un lote confirma todo junto en varios stores, ve sus propios cambios (también lápidas y valores en desborde) y
// los demás no ven nada hasta Commit; Count suma lo del lote al confirmar.
func TestBatchCommit(t *testing.T) {
	e := newTestEngine(t, nil)
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	mustOK(t, a.Put("x", []byte("0")))
	mustOK(t, a.Put("gone", []byte("0")))
	big := bytes.Repeat([]byte("big"), 10000)

	tx := e.NewBatch()
	mustOK(t, a.PutIn(tx, "x", []byte("1")))
	mustOK(t, b.PutIn(tx, "y", big))
	mustOK(t, a.RemoveIn(tx, "gone"))
	if ok, err := a.InsertIn(tx, "x", []byte("no")); ok || err != nil {
		t.Fatalf("InsertIn of an id the batch has = %v, %v", ok, err)
	}
	if ok, err := b.InsertIn(tx, "z", big); !ok || err != nil {
		t.Fatalf("InsertIn big = %v, %v", ok, err)
	}
	if ok, err := a.UpdateIn(tx, "x", []byte("1")); ok || err != nil {
		t.Fatalf("UpdateIn same value = %v, %v", ok, err)
	}
	if ok, err := a.UpdateIn(tx, "x", []byte("2")); !ok || err != nil {
		t.Fatalf("UpdateIn = %v, %v", ok, err)
	}
	if ok, err := a.DeleteIn(tx, "gone"); ok || err != nil {
		t.Fatalf("DeleteIn of an id the batch removed = %v, %v", ok, err)
	}

	// El lote ve lo suyo; los demás, nada.
	read := func(s *Store) func(string) ([]byte, bool, error) {
		return func(id string) ([]byte, bool, error) { return s.ReadIn(tx, id) }
	}
	if v, _ := readString(t, read(a), "x"); v != "2" {
		t.Fatalf("ReadIn x = %q", v)
	}
	if _, found := readString(t, read(a), "gone"); found {
		t.Fatal("ReadIn sees the removed id")
	}
	if v, _ := readString(t, read(b), "y"); v != string(big) {
		t.Fatal("ReadIn y: wrong big value")
	}
	if v, _ := readString(t, a.Read, "x"); v != "0" {
		t.Fatalf("Read sees an uncommitted write: %q", v)
	}
	if _, found := readString(t, b.Read, "y"); found || a.Count() != 2 || b.Count() != 0 {
		t.Fatalf("uncommitted batch is visible: y %v, counts %d %d", found, a.Count(), b.Count())
	}

	mustOK(t, tx.Commit())
	if v, _ := readString(t, a.Read, "x"); v != "2" {
		t.Fatalf("after commit x = %q", v)
	}
	if v, _ := readString(t, b.Read, "z"); v != string(big) {
		t.Fatal("after commit z: wrong big value")
	}
	if _, found := readString(t, a.Read, "gone"); found {
		t.Fatal("after commit gone is still there")
	}
	if a.Count() != 1 || b.Count() != 2 {
		t.Fatalf("counts after commit = %d, %d", a.Count(), b.Count())
	}
}

// Rollback no deja rastro: ni los valores, ni Count, ni tras reabrir; el mismo id se puede escribir enseguida.
func TestBatchRollback(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	a := mustStore(t, e, "a")
	mustOK(t, a.Put("x", []byte("0")))
	tx := e.NewBatch()
	mustOK(t, a.PutIn(tx, "x", []byte("1")))
	mustOK(t, a.PutIn(tx, "big", bytes.Repeat([]byte("v"), 50000)))
	mustOK(t, a.RemoveIn(tx, "x"))
	mustOK(t, tx.Rollback())

	check := func(e *Engine) {
		t.Helper()
		a := mustStore(t, e, "a")
		if v, _ := readString(t, a.Read, "x"); v != "0" {
			t.Fatalf("x = %q after rollback", v)
		}
		if _, found := readString(t, a.Read, "big"); found || a.Count() != 1 {
			t.Fatalf("rollback left a trace: big %v, count %d", found, a.Count())
		}
	}
	check(e)
	done := make(chan error, 1)
	go func() { done <- a.Put("x", []byte("2")) }()
	select {
	case err := <-done:
		mustOK(t, err)
	case <-time.After(time.Second):
		t.Fatal("a write waited for a rolled back batch")
	}
	mustOK(t, a.Put("x", []byte("0")))
	mustOK(t, e.Close())
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	check(e)
	verifyTrees(t, e)
}

// Terminar un lote dos veces, usarlo después y usarlo con un store de otro motor dan errores claros.
func TestBatchDone(t *testing.T) {
	e := newTestEngine(t, nil)
	other := newTestEngine(t, nil)
	a := mustStore(t, e, "a")
	tx := e.NewBatch()
	if err := mustStore(t, other, "a").PutIn(tx, "x", nil); !errors.Is(err, ErrOtherEngine) {
		t.Fatalf("other engine: %v", err)
	}
	if _, _, err := a.ReadAt(other.Snapshot(), "x"); !errors.Is(err, ErrOtherEngine) {
		t.Fatalf("snapshot of another engine: %v", err)
	}
	mustOK(t, a.PutIn(tx, "x", []byte("1")))
	mustOK(t, tx.Commit())
	if err := tx.Commit(); !errors.Is(err, ErrTxDone) {
		t.Fatalf("second Commit: %v", err)
	}
	if err := tx.Rollback(); !errors.Is(err, ErrTxDone) {
		t.Fatalf("Rollback after Commit: %v", err)
	}
	if err := a.PutIn(tx, "x", nil); !errors.Is(err, ErrTxDone) {
		t.Fatalf("PutIn after Commit: %v", err)
	}
	if _, _, err := a.ReadIn(tx, "x"); !errors.Is(err, ErrTxDone) {
		t.Fatalf("ReadIn after Commit: %v", err)
	}
	v := e.Snapshot()
	mustOK(t, v.Close())
	mustOK(t, v.Close())
	if _, _, err := a.ReadAt(v, "x"); !errors.Is(err, ErrTxDone) {
		t.Fatalf("ReadAt after Close: %v", err)
	}
	// Un lote vacío confirma sin escribir nada.
	mustOK(t, e.NewBatch().Commit())
	// Tras cerrar el motor, un lote nuevo y uno abierto devuelven ErrClosed.
	open := e.NewBatch()
	mustOK(t, a.PutIn(open, "y", []byte("1")))
	mustOK(t, e.Close())
	if err := e.NewBatch().Commit(); !errors.Is(err, ErrClosed) {
		t.Fatalf("batch of a closed engine: %v", err)
	}
	if err := open.Commit(); !errors.Is(err, ErrClosed) {
		t.Fatalf("open batch after Close: %v", err)
	}
	if _, _, err := a.ReadAt(e.Snapshot(), "x"); !errors.Is(err, ErrClosed) {
		t.Fatalf("snapshot of a closed engine: %v", err)
	}
}

// Lecturas repetibles en un lote y en un snapshot: lo que otros confirman después no se ve; un snapshot tomado antes
// de un Commit no lo ve, uno tomado después sí.
func TestRepeatableRead(t *testing.T) {
	e := newTestEngine(t, nil)
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	mustOK(t, a.Put("x", []byte("old")))
	tx := e.NewBatch()
	v := e.Snapshot()
	defer v.Close()
	inTx := func(id string) ([]byte, bool, error) { return a.ReadIn(tx, id) }
	atV := func(id string) ([]byte, bool, error) { return a.ReadAt(v, id) }
	if s, _ := readString(t, inTx, "x"); s != "old" {
		t.Fatalf("ReadIn = %q", s)
	}
	mustOK(t, a.Put("x", []byte("new")))
	mustOK(t, a.Put("n", []byte("new")))
	mustOK(t, b.Put("n", []byte("new")))
	for name, read := range map[string]func(string) ([]byte, bool, error){"batch": inTx, "snapshot": atV} {
		if s, _ := readString(t, read, "x"); s != "old" {
			t.Fatalf("%s: x = %q, want old", name, s)
		}
		if _, found := readString(t, read, "n"); found {
			t.Fatalf("%s sees an id created after it started", name)
		}
	}
	if _, found, _ := b.ReadAt(v, "n"); found {
		t.Fatal("snapshot sees a later write in another store")
	}
	mustOK(t, tx.Rollback())

	w := e.NewBatch()
	mustOK(t, b.PutIn(w, "m", []byte("1")))
	before := e.Snapshot()
	mustOK(t, w.Commit())
	after := e.Snapshot()
	if _, found, _ := b.ReadAt(before, "m"); found {
		t.Fatal("a snapshot taken before Commit sees the batch")
	}
	if _, found, _ := b.ReadAt(after, "m"); !found {
		t.Fatal("a snapshot taken after Commit does not see the batch")
	}
	before.Close()
	after.Close()
}

// Escribir en un lote un id que otra transacción confirmó después de que el lote empezó da ErrSerialization, también
// con Put y Remove, y el lote queda abortado: no retiene nada, Commit devuelve el error y Rollback lo acepta.
func TestBatchSerialization(t *testing.T) {
	for _, kind := range []string{"put", "remove", "insert", "insert-big", "update", "delete"} {
		t.Run(kind, func(t *testing.T) {
			e := newTestEngine(t, nil)
			a := mustStore(t, e, "a")
			mustOK(t, a.Put("x", []byte("0")))
			tx := e.NewBatch()
			mustOK(t, a.PutIn(tx, "mine", []byte("1")))
			mustOK(t, a.Put("x", []byte("1")))
			var err error
			switch kind {
			case "put":
				err = a.PutIn(tx, "x", []byte("2"))
			case "remove":
				err = a.RemoveIn(tx, "x")
			case "insert":
				_, err = a.InsertIn(tx, "x", []byte("2"))
			case "insert-big":
				_, err = a.InsertIn(tx, "x", bytes.Repeat([]byte("2"), 20000))
			case "update":
				_, err = a.UpdateIn(tx, "x", []byte("2"))
			case "delete":
				_, err = a.DeleteIn(tx, "x")
			}
			if !errors.Is(err, ErrSerialization) {
				t.Fatalf("%s: %v, want ErrSerialization", kind, err)
			}
			if err := a.PutIn(tx, "other", nil); !errors.Is(err, ErrSerialization) {
				t.Fatalf("after the error: %v", err)
			}
			mustOK(t, a.Put("mine", []byte("free"))) // el lote ya no retiene sus ids
			if err := tx.Commit(); !errors.Is(err, ErrSerialization) {
				t.Fatalf("Commit of an aborted batch: %v", err)
			}
			if err := tx.Rollback(); !errors.Is(err, ErrTxDone) {
				t.Fatalf("Rollback after that Commit: %v", err)
			}
			if v, _ := readString(t, a.Read, "x"); v != "1" {
				t.Fatalf("x = %q", v)
			}
		})
	}
}

// Dos lotes sobre el mismo id: el segundo espera al primero; si el primero confirma, el segundo da ErrSerialization;
// si lo deshace, el segundo sigue y confirma.
func TestBatchWaits(t *testing.T) {
	for _, first := range []string{"commit", "rollback"} {
		t.Run(first, func(t *testing.T) {
			e := newTestEngine(t, nil)
			a := mustStore(t, e, "a")
			t1, t2 := e.NewBatch(), e.NewBatch()
			mustOK(t, a.PutIn(t1, "x", []byte("1")))
			done := make(chan error, 1)
			go func() { done <- a.PutIn(t2, "x", []byte("2")) }()
			select {
			case err := <-done:
				t.Fatalf("second batch did not wait: %v", err)
			case <-time.After(50 * time.Millisecond):
			}
			if first == "commit" {
				mustOK(t, t1.Commit())
				if err := <-done; !errors.Is(err, ErrSerialization) {
					t.Fatalf("after the first committed: %v", err)
				}
				t2.Rollback()
				return
			}
			mustOK(t, t1.Rollback())
			mustOK(t, <-done)
			mustOK(t, t2.Commit())
			if v, _ := readString(t, a.Read, "x"); v != "2" {
				t.Fatalf("x = %q", v)
			}
		})
	}
}

// LOCK_TIMEOUT: un lote o una operación suelta que espera a otra transacción más de ese tiempo da ErrConflict; el
// lote queda abortado.
func TestBatchLockTimeout(t *testing.T) {
	e := newTestEngine(t, func(c *config) { c.lockTimeout = 50 * time.Millisecond })
	a := mustStore(t, e, "a")
	holder := e.NewBatch()
	mustOK(t, a.PutIn(holder, "x", []byte("1")))
	start := time.Now()
	if err := a.Put("x", []byte("2")); !errors.Is(err, ErrConflict) {
		t.Fatalf("single Put: %v", err)
	}
	tx := e.NewBatch()
	if err := a.PutIn(tx, "x", []byte("3")); !errors.Is(err, ErrConflict) {
		t.Fatalf("PutIn: %v", err)
	}
	if took := time.Since(start); took < 100*time.Millisecond || took > 5*time.Second {
		t.Fatalf("waited %v", took)
	}
	if err := tx.Commit(); !errors.Is(err, ErrConflict) {
		t.Fatalf("Commit after the conflict: %v", err)
	}
	mustOK(t, holder.Commit())
}

// TRANSACTION_TIMEOUT: un lote que pasa su tiempo queda abortado (sin retener sus ids) y devuelve ErrTimeout; un
// snapshot con SnapshotFor se cierra solo, y uno con 0 no vence.
func TestTransactionTimeout(t *testing.T) {
	e := newTestEngine(t, func(c *config) { c.txTimeout = 50 * time.Millisecond })
	a := mustStore(t, e, "a")
	tx := e.NewBatch()
	mustOK(t, a.PutIn(tx, "x", []byte("1")))
	short, forever, def := e.SnapshotFor(30*time.Millisecond), e.SnapshotFor(0), e.Snapshot()
	time.Sleep(120 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- a.Put("x", []byte("2")) }()
	select {
	case err := <-done:
		mustOK(t, err)
	case <-time.After(time.Second):
		t.Fatal("the expired batch still holds x")
	}
	if err := a.PutIn(tx, "y", nil); !errors.Is(err, ErrTimeout) {
		t.Fatalf("PutIn after the timeout: %v", err)
	}
	if err := tx.Commit(); !errors.Is(err, ErrTimeout) {
		t.Fatalf("Commit after the timeout: %v", err)
	}
	for name, v := range map[string]*Snapshot{"SnapshotFor(30ms)": short, "Snapshot()": def} {
		if _, _, err := a.ReadAt(v, "x"); !errors.Is(err, ErrTimeout) {
			t.Fatalf("%s: %v", name, err)
		}
		mustOK(t, v.Close())
	}
	if _, found, err := a.ReadAt(forever, "x"); err != nil || found {
		t.Fatalf("SnapshotFor(0) = %v, %v", found, err)
	}
	mustOK(t, forever.Close())

	// Un lote que espera a otra transacción no pasa su propio tiempo esperando.
	holder := e.NewBatch()
	mustOK(t, a.PutIn(holder, "z", nil))
	waiter := e.NewBatch()
	if err := a.PutIn(waiter, "z", nil); !errors.Is(err, ErrTimeout) {
		t.Fatalf("waiting past the batch time: %v", err)
	}
}

// Los snapshots abiertos y los lotes cuentan en el horizonte de la limpieza hasta que terminan.
func TestSnapshotHorizon(t *testing.T) {
	e := newTestEngine(t, nil)
	a := mustStore(t, e, "a")
	mustOK(t, a.Put("x", nil))
	v := e.Snapshot()
	tx := e.NewBatch()
	horizon := v.snap.Xmin
	for range 5 {
		mustOK(t, a.Put("x", nil))
	}
	if got := e.txs.OldestActive(); got != horizon {
		t.Fatalf("OldestActive = %d, want the snapshot's Xmin %d", got, horizon)
	}
	mustOK(t, v.Close())
	if got := e.txs.OldestActive(); got != tx.xid {
		t.Fatalf("OldestActive = %d, want the batch %d", got, tx.xid)
	}
	mustOK(t, tx.Rollback())
	if got := e.txs.OldestActive(); got != e.txs.NextXID() {
		t.Fatalf("OldestActive = %d with nothing open, next %d", got, e.txs.NextXID())
	}
}

// Sin cambios perdidos: muchos lotes concurrentes leen un contador, lo suben y lo escriben; los que chocan reintentan
// y al final el contador es exacto. Corre con lecturas sueltas y snapshots a la vez.
func TestBatchNoLostUpdates(t *testing.T) {
	e := newTestEngine(t, nil)
	a := mustStore(t, e, "a")
	mustOK(t, a.Put("n", []byte("0")))
	const workers, per = 8, 25
	var wg sync.WaitGroup
	var retries atomic.Int64
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range per {
				for {
					err := incrementIn(e, a, "n")
					if err == nil {
						break
					}
					if !errors.Is(err, ErrSerialization) && !errors.Is(err, ErrConflict) {
						t.Error(err)
						return
					}
					retries.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if v, _ := readString(t, a.Read, "n"); v != strconv.Itoa(workers*per) {
		t.Fatalf("counter = %s, want %d", v, workers*per)
	}
	t.Logf("%d retries", retries.Load())
}

/**
* incrementIn sube en uno el contador id dentro de un lote nuevo.
* @param e *Engine, s *Store, id string
* @return error: el del lote (ErrSerialization si chocó)
**/
func incrementIn(e *Engine, s *Store, id string) error {
	tx := e.NewBatch()
	v, _, err := s.ReadIn(tx, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	n, _ := strconv.Atoi(string(v))
	if err := s.PutIn(tx, id, []byte(strconv.Itoa(n+1))); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Atomicidad para los lectores: un escritor confirma (y a veces deshace) lotes que escriben el número i en el id k
// de tres stores, en orden a, b, c, con i creciente. Un snapshot ve siempre los tres iguales; lecturas sueltas en el
// mismo orden a, b, c nunca ven un número menor después de uno mayor (si el Commit no apareciera de una vez, a
// podría mostrar i y c todavía i-1).
func TestBatchAtomicForReaders(t *testing.T) {
	e := newTestEngine(t, nil)
	stores := []*Store{mustStore(t, e, "a"), mustStore(t, e, "b"), mustStore(t, e, "c")}
	var stop atomic.Bool
	writer := make(chan error, 1)
	go func() {
		for i := 1; !stop.Load(); i++ {
			tx := e.NewBatch()
			for _, s := range stores {
				if err := s.PutIn(tx, "k", []byte(strconv.Itoa(i))); err != nil {
					writer <- err
					return
				}
			}
			var err error
			if i%5 == 0 {
				err = tx.Rollback()
			} else {
				err = tx.Commit()
			}
			if err != nil {
				writer <- err
				return
			}
		}
		writer <- nil
	}()
	number := func(v []byte) int {
		n, _ := strconv.Atoi(string(v))
		return n
	}
	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		v := e.Snapshot()
		var seen []int
		for _, s := range stores {
			val, _, err := s.ReadAt(v, "k")
			mustOK(t, err)
			seen = append(seen, number(val))
		}
		v.Close()
		if seen[0] != seen[1] || seen[1] != seen[2] {
			t.Fatalf("snapshot sees a partial batch: %v", seen)
		}
		last := 0
		for i, s := range stores {
			val, _, err := s.Read("k")
			mustOK(t, err)
			if n := number(val); n < last {
				t.Fatalf("store %d shows %d after a previous store showed %d", i, n, last)
			} else {
				last = n
			}
		}
	}
	stop.Store(true)
	mustOK(t, <-writer)
}

// Un lote grande (miles de escrituras) no hace crecer la RAM con sus cambios: sus versiones van al árbol y a la
// caché de tamaño fijo. Usa el disco real, porque crashfs guarda los archivos en memoria.
func TestBatchLargeMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("writes several MB to disk")
	}
	cfg := testConfig()
	cfg.syncOnWrite = false
	e, err := open(vfs.OS(), t.TempDir(), cfg)
	mustOK(t, err)
	defer e.Close()
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	value := bytes.Repeat([]byte("v"), 500)
	tx := e.NewBatch()
	write := func(from, to int) {
		for i := from; i < to; i++ {
			s := a
			if i%2 == 1 {
				s = b
			}
			mustOK(t, s.PutIn(tx, fmt.Sprintf("id-%08d", i), value))
		}
	}
	heap := func() uint64 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	write(0, 1000)
	before := heap()
	write(1000, 9000) // 4 MB de valores
	growth := int64(heap()) - int64(before)
	if growth > 1<<20 {
		t.Fatalf("the heap grew %d bytes during 8000 batch writes", growth)
	}
	mustOK(t, tx.Commit())
	if a.Count()+b.Count() != 9000 {
		t.Fatalf("count = %d", a.Count()+b.Count())
	}
	t.Logf("heap growth during 8000 writes: %d bytes", growth)
}

// Un lote abierto durante un checkpoint: si confirma después, está entero tras un corte; si no confirma, no está.
// En los tres modos de crashfs, con un lote largo en varios stores y valores en desborde.
func TestBatchCrash(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for _, commit := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/commit=%v", name, commit), func(t *testing.T) {
				t.Parallel()
				base := crashfs.New()
				proc := newProc(base)
				e := openTest(t, proc, testConfig())
				a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
				mustOK(t, a.Put("before", []byte("1")))
				tx := e.NewBatch()
				for i := range 300 {
					s, v := a, []byte(strconv.Itoa(i))
					if i%2 == 1 {
						s = b
					}
					if i%50 == 0 {
						v = bytes.Repeat(v, 3000)
					}
					mustOK(t, s.PutIn(tx, fmt.Sprintf("k%04d", i), v))
					if i == 150 {
						mustOK(t, e.Checkpoint())
					}
				}
				mustOK(t, a.RemoveIn(tx, "before"))
				if commit {
					mustOK(t, tx.Commit())
				}
				mustOK(t, a.Put("after", []byte("1"))) // suelta, confirmada: tiene que estar
				proc.kill(mode, int64(mode))
				closeWithin(t, e, 10*time.Second)

				e = openTest(t, newProc(base), testConfig())
				defer e.Close()
				a, b = mustStore(t, e, "a"), mustStore(t, e, "b")
				got := engineState(t, e)
				want := 2 // before y after
				if commit {
					want = 301 // 300 del lote + after, sin before
				}
				if n := len(got["a"]) + len(got["b"]); n != want {
					t.Fatalf("after the crash there are %d ids, want %d", n, want)
				}
				if _, found := readString(t, a.Read, "after"); !found {
					t.Fatal("the committed single write is lost")
				}
				verifyTrees(t, e)
				// El motor sigue aceptando lotes sobre los mismos ids.
				tx = e.NewBatch()
				mustOK(t, a.PutIn(tx, "k0000", []byte("x")))
				mustOK(t, tx.Commit())
			})
		}
	}
}
