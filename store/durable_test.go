package store

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
)

// crashModes son los tres modos de corte de crashfs.
var crashModes = map[string]crashfs.Mode{
	"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
}

/**
* deferredValue es el valor del i-ésimo cambio diferido de las pruebas: uno de cada cinco va a desborde, para que «a
* medias» pueda verse (un valor cortado o una cadena incompleta).
* @param i int
* @return []byte
**/
func deferredValue(i int) []byte {
	if i%5 == 0 {
		return bytes.Repeat([]byte(fmt.Sprintf("big-%d-", i)), 3000)
	}
	return []byte(fmt.Sprintf("v-%d", i))
}

/**
* notDurable dice si el WAL no es durable hasta la última confirmación anotada.
* @param e *Engine
* @return bool
**/
func notDurable(e *Engine) bool {
	return e.log.Durable() < page.LSN(e.lastCommit.Load())
}

// Las escrituras diferidas devuelven lo mismo que las sueltas, se ven enseguida (también desde un lote nuevo y un
// snapshot) aunque no sean durables, y WaitDurable las deja en disco con un solo fsync.
func TestDeferredOps(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "a")
	d := s.Deferred()
	big := deferredValue(0)

	mustOK(t, d.Put("a", []byte("1")))
	mustOK(t, d.Put("big", big))
	checkBool(t, "insert a", false)(d.Insert("a", []byte("x")))
	checkBool(t, "insert b", true)(d.Insert("b", []byte("2")))
	checkBool(t, "update same", false)(d.Update("a", []byte("1")))
	checkBool(t, "update a", true)(d.Update("a", []byte("10")))
	checkBool(t, "update missing", false)(d.Update("zz", []byte("1")))
	checkBool(t, "delete missing", false)(d.Delete("zz"))
	checkBool(t, "delete b", true)(d.Delete("b"))
	mustOK(t, d.Remove("zz"))
	mustOK(t, d.Put("gone", []byte("x")))
	mustOK(t, d.Remove("gone"))
	if !notDurable(e) {
		t.Fatal("deferred writes are already durable without WaitDurable")
	}

	checkContent(t, s, map[string][]byte{"a": []byte("10"), "big": big}, []string{"b", "zz", "gone"})
	v := e.Snapshot()
	defer v.Close()
	if got, found, err := s.ReadAt(v, "a"); err != nil || !found || string(got) != "10" {
		t.Fatalf("ReadAt a = %q, %v, %v", got, found, err)
	}
	tx := e.NewBatch()
	if got, found, err := s.ReadIn(tx, "big"); err != nil || !found || !bytes.Equal(got, big) {
		t.Fatalf("ReadIn big = %d B, %v, %v", len(got), found, err)
	}
	mustOK(t, tx.Rollback())

	mustOK(t, e.WaitDurable())
	if notDurable(e) {
		t.Fatal("WaitDurable returned with the WAL behind the last commit")
	}
	mustOK(t, e.WaitDurable()) // ya durable: no hace nada
}

// WaitDurable con el motor cerrado devuelve ErrClosed; los métodos diferidos, también.
func TestDeferredClosed(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	s := mustStore(t, e, "a")
	mustOK(t, e.Close())
	if err := e.WaitDurable(); !errors.Is(err, ErrClosed) {
		t.Fatalf("WaitDurable after Close = %v", err)
	}
	if err := s.Deferred().Put("a", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("Deferred().Put after Close = %v", err)
	}
	if _, err := s.Deferred().Insert("a", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("Deferred().Insert after Close = %v", err)
	}
}

// Un lote con CommitDeferred se ve enseguida y no es durable hasta WaitDurable; su Count se suma al confirmar.
func TestBatchCommitDeferred(t *testing.T) {
	e := newTestEngine(t, nil)
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	tx := e.NewBatch()
	mustOK(t, a.PutIn(tx, "x", []byte("1")))
	mustOK(t, b.PutIn(tx, "y", deferredValue(0)))
	mustOK(t, tx.CommitDeferred())
	if !notDurable(e) {
		t.Fatal("a deferred batch is durable without WaitDurable")
	}
	checkContent(t, a, map[string][]byte{"x": []byte("1")}, nil)
	checkContent(t, b, map[string][]byte{"y": deferredValue(0)}, nil)
	if err := tx.CommitDeferred(); !errors.Is(err, ErrTxDone) {
		t.Fatalf("second CommitDeferred = %v", err)
	}
	mustOK(t, e.WaitDurable())
	if notDurable(e) {
		t.Fatal("WaitDurable left the batch behind")
	}
}

// Regla 49 con confirmaciones no durables: la marca de confirmada no se pone en una versión diferida hasta que el WAL
// es durable hasta su lsn, aunque se lea muchas veces; después de WaitDurable, la primera lectura la pone.
func TestDeferredHintWaitsDurable(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "a")
	mustOK(t, s.Deferred().Put("k", []byte("v")))
	flags := func() btree.EntryFlags {
		v, found, err := s.tree.Lookup("k", nil)
		if err != nil || !found {
			t.Fatalf("lookup = %v, %v", found, err)
		}
		return v.Flags
	}
	for range 3 {
		if flags()&btree.EntryXidCommitted != 0 {
			t.Fatal("committed hint set before the commit lsn is durable")
		}
	}
	mustOK(t, e.WaitDurable())
	flags() // esta lectura marca
	if flags()&btree.EntryXidCommitted == 0 {
		t.Fatal("committed hint not set once the commit lsn is durable")
	}
}

// El escritor de fondo del WAL hace durables las confirmaciones diferidas sin que nadie llame a WaitDurable.
func TestWALWriter(t *testing.T) {
	e := newTestEngine(t, func(c *config) { c.walWriterDelay = 2 * time.Millisecond })
	s := mustStore(t, e, "a")
	mustOK(t, s.Deferred().Put("k", []byte("v")))
	deadline := time.Now().Add(5 * time.Second)
	for notDurable(e) {
		if time.Now().After(deadline) {
			t.Fatal("the WAL writer did not make the deferred commit durable")
		}
		time.Sleep(time.Millisecond)
	}
}

// Con SYNC_ON_WRITE=false toda confirmación es diferida: Put no espera el fsync y WaitDurable sí.
func TestSyncOnWriteOff(t *testing.T) {
	e := newTestEngine(t, func(c *config) { c.syncOnWrite = false })
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("k", []byte("v")))
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "j", []byte("w")))
	mustOK(t, tx.Commit())
	if !notDurable(e) {
		t.Fatal("with SYNC_ON_WRITE=false a commit waited for the fsync")
	}
	mustOK(t, e.WaitDurable())
	if notDurable(e) {
		t.Fatal("WaitDurable left commits behind")
	}
}

/**
* deferredRun escribe n cambios diferidos sobre ids distintos del store "a" (el i-ésimo con deferredValue(i)) sobre
* una base durable, corre after (lo que hace durable o no el final) y corta en mode. Devuelve cuántos cambios del
* principio sobrevivieron, tras comprobar que son un prefijo, con su valor entero y Count.
* @param t *testing.T, mode crashfs.Mode, seed int64, n int, after func(e *Engine, s *Store)
* @return int
**/
func deferredRun(t *testing.T, mode crashfs.Mode, seed int64, n int, after func(e *Engine, s *Store)) int {
	t.Helper()
	base := crashfs.New()
	proc := newProc(base)
	e := openTest(t, proc, testConfig())
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("base", []byte("durable")))
	d := s.Deferred()
	for i := range n {
		mustOK(t, d.Put(fmt.Sprintf("k%03d", i), deferredValue(i)))
	}
	after(e, s)
	proc.kill(mode, seed)
	closeWithin(t, e, 10*time.Second)

	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	s = mustStore(t, e, "a")
	if got, found := readString(t, s.Read, "base"); !found || got != "durable" {
		t.Fatalf("the durable write was lost: %q, %v", got, found)
	}
	kept := 0
	for i := range n {
		got, found, err := s.Read(fmt.Sprintf("k%03d", i))
		mustOK(t, err)
		if !found {
			break
		}
		if !bytes.Equal(got, deferredValue(i)) {
			t.Fatalf("k%03d recovered half-written: %d B, want %d B", i, len(got), len(deferredValue(i)))
		}
		kept++
	}
	for i := kept; i < n; i++ {
		if _, found, err := s.Read(fmt.Sprintf("k%03d", i)); err != nil || found {
			t.Fatalf("k%03d survived but k%03d did not: not a WAL prefix (%v)", i, kept, err)
		}
	}
	if s.Count() != int64(1+kept) {
		t.Fatalf("Count = %d, want %d", s.Count(), 1+kept)
	}
	verifyTrees(t, e)
	return kept
}

// Corte tras escrituras diferidas en los tres modos: sin esperar, sobrevive un prefijo de ellas (cada una entera o
// nada); tras WaitDurable, o tras una confirmación durable posterior (orden del WAL), sobreviven todas.
func TestDeferredCrash(t *testing.T) {
	const n = 40
	for name, mode := range crashModes {
		t.Run(name, func(t *testing.T) {
			lost := 0
			for seed := range int64(6) {
				kept := deferredRun(t, mode, seed, n, func(*Engine, *Store) {})
				if kept < n {
					lost++
				}
			}
			t.Logf("without waiting: %d of 6 runs lost some deferred writes", lost)
			if mode == crashfs.LoseUnsynced && lost == 0 {
				t.Fatal("LoseUnsynced never lost a deferred write: they are being synced")
			}
			for seed := range int64(3) {
				if kept := deferredRun(t, mode, seed, n, func(e *Engine, _ *Store) { mustOK(t, e.WaitDurable()) }); kept != n {
					t.Fatalf("after WaitDurable only %d of %d survived", kept, n)
				}
				if kept := deferredRun(t, mode, seed, n, func(_ *Engine, s *Store) { mustOK(t, s.Put("base", []byte("durable"))) }); kept != n {
					t.Fatalf("after a durable commit only %d of %d survived", kept, n)
				}
			}
		})
	}
}

/**
* batchDeferredRun confirma con CommitDeferred un lote de 30 cambios en dos stores (con desborde y un borrado), corre
* after y corta en mode. Comprueba que el lote quedó entero o nada (Count incluido) y devuelve si quedó.
* @param t *testing.T, mode crashfs.Mode, seed int64, after func(e *Engine)
* @return bool
**/
func batchDeferredRun(t *testing.T, mode crashfs.Mode, seed int64, after func(e *Engine)) bool {
	t.Helper()
	base := crashfs.New()
	proc := newProc(base)
	e := openTest(t, proc, testConfig())
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	mustOK(t, a.Put("old", []byte("x")))
	tx := e.NewBatch()
	for i := range 15 {
		mustOK(t, a.PutIn(tx, fmt.Sprintf("a%02d", i), deferredValue(i)))
		mustOK(t, b.PutIn(tx, fmt.Sprintf("b%02d", i), deferredValue(i+1)))
	}
	mustOK(t, a.RemoveIn(tx, "old"))
	mustOK(t, tx.CommitDeferred())
	after(e)
	proc.kill(mode, seed)
	closeWithin(t, e, 10*time.Second)

	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	a, b = mustStore(t, e, "a"), mustStore(t, e, "b")
	_, oldFound := readString(t, a.Read, "old")
	committed := !oldFound
	wantA, wantB := map[string][]byte{}, map[string][]byte{}
	var missA, missB []string
	for i := range 15 {
		ida, idb := fmt.Sprintf("a%02d", i), fmt.Sprintf("b%02d", i)
		if committed {
			wantA[ida], wantB[idb] = deferredValue(i), deferredValue(i+1)
		} else {
			missA, missB = append(missA, ida), append(missB, idb)
		}
	}
	if !committed {
		wantA["old"] = []byte("x")
	}
	checkContent(t, a, wantA, missA)
	checkContent(t, b, wantB, missB)
	verifyTrees(t, e)
	return committed
}

// Un lote diferido es todo o nada tras un corte en los tres modos; tras WaitDurable o una confirmación durable
// posterior, queda siempre.
func TestBatchDeferredCrash(t *testing.T) {
	for name, mode := range crashModes {
		t.Run(name, func(t *testing.T) {
			kept := 0
			for seed := range int64(6) {
				if batchDeferredRun(t, mode, seed, func(*Engine) {}) {
					kept++
				}
			}
			t.Logf("without waiting: the batch survived %d of 6 crashes", kept)
			for seed := range int64(3) {
				if !batchDeferredRun(t, mode, seed, func(e *Engine) { mustOK(t, e.WaitDurable()) }) {
					t.Fatal("the batch was lost after WaitDurable")
				}
				if !batchDeferredRun(t, mode, seed, func(e *Engine) { mustOK(t, mustStore(t, e, "c").Put("x", nil)) }) {
					t.Fatal("the batch was lost after a later durable commit")
				}
			}
		})
	}
}

// Varios escritores diferidos a la vez, cada uno con su WaitDurable de vez en cuando (group commit): tras un corte,
// todo lo que cada uno escribió antes de su último WaitDurable sigue ahí.
func TestDeferredConcurrentWaitDurable(t *testing.T) {
	for name, mode := range crashModes {
		t.Run(name, func(t *testing.T) {
			base := crashfs.New()
			proc := newProc(base)
			e := openTest(t, proc, testConfig())
			s := mustStore(t, e, "a")
			const writers, each = 8, 60
			waited := make([]int, writers) // cuántos de cada escritor quedaron cubiertos por un WaitDurable
			var wg sync.WaitGroup
			for w := range writers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					d := s.Deferred()
					for i := range each {
						if err := d.Put(fmt.Sprintf("w%d-%03d", w, i), deferredValue(i)); err != nil {
							t.Errorf("put: %v", err)
							return
						}
						if i%7 == 6 {
							if err := e.WaitDurable(); err != nil {
								t.Errorf("WaitDurable: %v", err)
								return
							}
							waited[w] = i + 1
						}
					}
				}()
			}
			wg.Wait()
			proc.kill(mode, 1)
			closeWithin(t, e, 10*time.Second)

			e = openTest(t, newProc(base), testConfig())
			defer e.Close()
			s = mustStore(t, e, "a")
			for w := range writers {
				for i := range waited[w] {
					got, found, err := s.Read(fmt.Sprintf("w%d-%03d", w, i))
					if err != nil || !found || !bytes.Equal(got, deferredValue(i)) {
						t.Fatalf("w%d-%03d covered by WaitDurable was lost: %v, %v", w, i, found, err)
					}
				}
			}
			verifyTrees(t, e)
		})
	}
}
