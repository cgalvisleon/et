package store

import (
	"errors"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// Las esperas a otra transacción (txs.Wait) se hacen sin Store.mu, closeMu ni Batch.mu (tarea 89): con
// LOCK_TIMEOUT=0 (sin límite) ninguna de estas combinaciones se cuelga. Cada caso tiene un plazo propio de la prueba,
// que falla en vez de colgarse.

/**
* lockWaitEngine abre un motor con LOCK_TIMEOUT=0 y el store s con x escrito.
* @param t *testing.T
* @return *Engine, *Store
**/
func lockWaitEngine(t *testing.T) (*Engine, *Store) {
	t.Helper()
	cfg := testConfig()
	cfg.lockTimeout = 0
	e := openTest(t, crashfs.New(), cfg)
	s := mustStore(t, e, "s")
	mustOK(t, s.Put("x", []byte("0")))
	return e, s
}

/**
* goErr corre fn en otra goroutine y devuelve el canal con su error.
* @param fn func() error
* @return chan error
**/
func goErr(fn func() error) chan error {
	ch := make(chan error, 1)
	go func() { ch <- fn() }()
	return ch
}

/**
* waitErr espera el error de ch hasta limit; si no llega, la prueba falla (el motor quedó trabado).
* @param t *testing.T, what string, ch chan error, limit time.Duration
* @return error
**/
func waitErr(t *testing.T, what string, ch chan error, limit time.Duration) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(limit):
		t.Fatalf("%s hung (deadlock)", what)
		return nil
	}
}

/**
* settle deja que la goroutine recién lanzada llegue a su espera.
**/
func settle() { time.Sleep(100 * time.Millisecond) }

// (a) Una escritura suelta espera al lote B; B borra el store con DropStoreIn y confirma.
func TestLockWaitDropStoreIn(t *testing.T) {
	e, s := lockWaitEngine(t)
	defer e.Close()
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "x", []byte("b")))
	w := goErr(func() error { return s.Put("x", []byte("w")) })
	settle()
	drop := goErr(func() error { return e.DropStoreIn(tx, "s") })
	mustOK(t, waitErr(t, "DropStoreIn", drop, 5*time.Second))
	mustOK(t, waitErr(t, "Commit", goErr(tx.Commit), 5*time.Second))
	if err := waitErr(t, "Put", w, 5*time.Second); !errors.Is(err, ErrStoreDropped) {
		t.Fatalf("Put after the batch dropped the store: %v, want ErrStoreDropped", err)
	}
}

// (b) Una escritura suelta espera al lote B mientras Close espera closeMu; B confirma.
func TestLockWaitClose(t *testing.T) {
	e, s := lockWaitEngine(t)
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "x", []byte("b")))
	w := goErr(func() error { return s.Put("x", []byte("w")) })
	settle()
	closed := goErr(e.Close)
	settle()
	commit := goErr(tx.Commit)
	if err := waitErr(t, "Commit", commit, 5*time.Second); err != nil && !errors.Is(err, ErrClosed) {
		t.Fatalf("Commit: %v", err)
	}
	mustOK(t, waitErr(t, "Close", closed, 5*time.Second))
	if err := waitErr(t, "Put", w, 5*time.Second); !errors.Is(err, ErrClosed) {
		t.Fatalf("Put after Close: %v, want ErrClosed", err)
	}
}

// (c) Una escritura suelta espera al lote B (con DDL); un DropStore suelto espera el store; B confirma.
func TestLockWaitDropStore(t *testing.T) {
	e, s := lockWaitEngine(t)
	defer e.Close()
	tx := e.NewBatch()
	if _, err := e.StoreIn(tx, "t"); err != nil {
		t.Fatal(err)
	}
	mustOK(t, s.PutIn(tx, "x", []byte("b")))
	w := goErr(func() error { return s.Put("x", []byte("w")) })
	settle()
	drop := goErr(func() error { return e.DropStore("s") })
	settle()
	mustOK(t, waitErr(t, "Commit", goErr(tx.Commit), 5*time.Second))
	mustOK(t, waitErr(t, "DropStore", drop, 5*time.Second))
	if err := waitErr(t, "Put", w, 5*time.Second); !errors.Is(err, ErrStoreDropped) {
		t.Fatalf("Put after DropStore: %v, want ErrStoreDropped", err)
	}
}

// (d) Un lote espera a otro lote mientras Close espera closeMu; el primero confirma.
func TestLockWaitBatchClose(t *testing.T) {
	e, s := lockWaitEngine(t)
	first, second := e.NewBatch(), e.NewBatch()
	mustOK(t, s.PutIn(first, "x", []byte("1")))
	w := goErr(func() error { return s.PutIn(second, "x", []byte("2")) })
	settle()
	closed := goErr(e.Close)
	settle()
	if err := waitErr(t, "Commit", goErr(first.Commit), 5*time.Second); err != nil && !errors.Is(err, ErrClosed) {
		t.Fatalf("Commit: %v", err)
	}
	mustOK(t, waitErr(t, "Close", closed, 5*time.Second))
	if err := waitErr(t, "PutIn", w, 5*time.Second); err == nil {
		t.Fatal("PutIn after Close returned nil")
	}
}

// (e) Rollback de un lote cuya escritura espera a otro lote no queda detrás de la espera (b.mu va suelto), y la
// escritura que esperaba termina con ErrTxDone.
func TestLockWaitBatchRollback(t *testing.T) {
	e, s := lockWaitEngine(t)
	defer e.Close()
	first, second := e.NewBatch(), e.NewBatch()
	mustOK(t, s.PutIn(first, "x", []byte("1")))
	w := goErr(func() error { return s.PutIn(second, "x", []byte("2")) })
	settle()
	mustOK(t, waitErr(t, "Rollback", goErr(second.Rollback), time.Second))
	mustOK(t, first.Rollback())
	if err := waitErr(t, "PutIn", w, 5*time.Second); !errors.Is(err, ErrTxDone) {
		t.Fatalf("PutIn of a rolled back batch: %v, want ErrTxDone", err)
	}
}
