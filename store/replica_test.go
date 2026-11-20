package store

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* replicaConfig es testConfig para una réplica: OpenReplica, con un plazo corto para los conflictos con snapshots.
* @return config
**/
func replicaConfig() config {
	cfg := smallWAL()
	cfg.replica = true
	cfg.conflictDelay = 2 * time.Second
	return cfg
}

/**
* newReplica hace un Backup del primario en un disco nuevo y lo abre como réplica.
* @param t testing.TB, p *Engine
* @return *Engine, crashfs.FS: la réplica y su disco
**/
func newReplica(t testing.TB, p *Engine) (*Engine, crashfs.FS) {
	t.Helper()
	fsys := crashfs.New()
	_, err := p.backup(fsys, testDir)
	mustOK(t, err)
	return openTest(t, fsys, replicaConfig()), fsys
}

/**
* streamOnce copia a la réplica lo durable del primario desde su ReplayLSN, la hace durable y confirma la ranura
* slot (si no es ""). Devuelve cuántos registros aplicó.
* @param p, r *Engine, slot string
* @return int, error
**/
func streamOnce(p, r *Engine, slot string) (int, error) {
	rd, err := p.WalSince(r.ReplayLSN())
	if err != nil {
		return 0, err
	}
	n := 0
	for {
		entries, err := rd.Next(64 << 10)
		if err != nil {
			return n, err
		}
		if len(entries) == 0 {
			break
		}
		if err := r.ApplyWal(entries); err != nil {
			return n, err
		}
		n += len(entries)
	}
	if err := r.WaitDurable(); err != nil {
		return n, err
	}
	if slot != "" {
		return n, p.AdvanceSlot(slot, r.ReplayLSN())
	}
	return n, nil
}

/**
* catchUp hace durable el primario y copia todo a la réplica.
* @param t testing.TB, p, r *Engine, slot string
**/
func catchUp(t testing.TB, p, r *Engine, slot string) {
	t.Helper()
	mustOK(t, p.WaitDurable())
	_, err := streamOnce(p, r, slot)
	mustOK(t, err)
	if r.ReplayLSN() < uint64(p.log.Durable()) {
		t.Fatalf("replica at lsn %d, primary durable %d", r.ReplayLSN(), p.log.Durable())
	}
}

/**
* ddlLoop hace DDL en el primario hasta stop: crea, llena y borra stores sueltos, vacía uno, y crea y borra stores
* dentro de lotes (confirmados y deshechos). ErrBusy (Empty con lotes abiertos) se reintenta en la próxima vuelta.
* @param p *Engine, stop *atomic.Bool, errs chan error
**/
func ddlLoop(p *Engine, stop *atomic.Bool, errs chan error) {
	for i := 0; !stop.Load(); i++ {
		if err := ddlOnce(p, i); err != nil {
			errs <- err
			return
		}
	}
}

/**
* ddlOnce es una vuelta de ddlLoop.
* @param p *Engine, i int
* @return error
**/
func ddlOnce(p *Engine, i int) error {
	tmp := fmt.Sprintf("tmp%d", i%3)
	s, err := p.Store(tmp)
	if err != nil {
		return err
	}
	if err := s.Put("x", []byte(tmp)); err != nil {
		return err
	}
	if i%2 == 0 {
		if err := p.DropStore(tmp); err != nil {
			return err
		}
	}
	empty, err := p.Store("empty")
	if err != nil {
		return err
	}
	if err := empty.Empty(); err != nil && !errors.Is(err, ErrBusy) {
		return err
	}
	if err := empty.Put(fmt.Sprintf("e%d", i), []byte("again")); err != nil {
		return err
	}
	tx := p.NewBatch()
	in, err := p.StoreIn(tx, fmt.Sprintf("lot%d", i%4))
	if err == nil {
		err = in.PutIn(tx, "a", []byte(fmt.Sprint(i)))
	}
	if err == nil && i%3 == 0 {
		err = p.DropStoreIn(tx, fmt.Sprintf("lot%d", (i+1)%4))
		if errors.Is(err, ErrStoreNotFound) {
			err = nil
		}
	}
	if err != nil {
		tx.Rollback()
		if errors.Is(err, ErrSerialization) || errors.Is(err, ErrConflict) || errors.Is(err, ErrStorePending) ||
			errors.Is(err, ErrBusy) || errors.Is(err, ErrStoreDropped) {
			return nil
		}
		return err
	}
	if i%5 == 4 {
		return tx.Rollback()
	}
	return tx.Commit()
}

/**
* readLoop lee de la réplica hasta stop, con Read, recorridos y snapshots: no debe fallar salvo por un conflicto
* (ErrRecoveryConflict), un store pendiente o borrado en el medio.
* @param r *Engine, stop *atomic.Bool, errs chan error
**/
func readLoop(r *Engine, stop *atomic.Bool, errs chan error) {
	ok := func(err error) bool {
		return err == nil || errors.Is(err, ErrRecoveryConflict) || errors.Is(err, ErrStorePending) ||
			errors.Is(err, ErrStoreDropped) || errors.Is(err, ErrStoreNotFound)
	}
	for !stop.Load() {
		for _, name := range []string{"a", "b", "empty"} {
			s, err := r.Store(name)
			if !ok(err) {
				errs <- err
				return
			}
			if err != nil {
				continue
			}
			if _, _, err := s.Read("k0001"); !ok(err) {
				errs <- err
				return
			}
			v := r.Snapshot()
			n, err := s.CountAt(v)
			if err == nil {
				var m int64
				err = s.ForEachAt(v, func(string, []byte) (bool, error) { m++; return true, nil }, true, 0, 0)
				if err == nil && m != n {
					err = fmt.Errorf("snapshot not stable: CountAt %d, ForEachAt %d", n, m)
				}
			}
			v.Close()
			if !ok(err) {
				errs <- err
				return
			}
		}
	}
}

/**
* writeN corre n escritores sobre stores, cada uno con ops operaciones de writeOne, y espera a que terminen.
* @param stores []*Store, n, ops int, seed int64
* @return error: el primero
**/
func writeN(stores []*Store, n, ops int, seed int64) error {
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed + int64(i)))
			for range ops {
				if err := writeOne(stores, i, rng); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	return <-errs
}

/**
* smallWAL es testConfig con un WAL chico, así los checkpoints reciclan durante la prueba.
* @return config
**/
func smallWAL() config {
	cfg := testConfig()
	cfg.maxWALSize = 4 << 20
	return cfg
}

// TestReplicaCatchUp: una réplica que arranca de un Backup alcanza al primario mientras este tiene escritores
// concurrentes (sueltos, diferidos, lotes), DDL suelto y en lotes y Empty, checkpoints que reciclan (la ranura
// retiene) y lectores en la réplica (Read, snapshots y recorridos estables); al final coincide: mismo contenido, Count
// y Verify limpio. Reabrir la réplica sigue desde donde iba, y Open sobre su directorio la promueve a primario con el
// mismo contenido.
func TestReplicaCatchUp(t *testing.T) {
	cfg := smallWAL()
	p := openTest(t, crashfs.New(), cfg)
	defer p.Close()
	_, err := p.CreateSlot("r1")
	mustOK(t, err)
	stores := fillStores(t, p, []string{"a", "b", "empty"}, 150)
	r, rfs := newReplica(t, p)

	var stop atomic.Bool
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); ddlLoop(p, &stop, errs) }()
	go func() { defer wg.Done(); readLoop(r, &stop, errs) }()
	go func() {
		defer wg.Done()
		for !stop.Load() {
			n, err := streamOnce(p, r, "r1")
			if err != nil {
				errs <- err
				return
			}
			if n == 0 {
				time.Sleep(time.Millisecond)
			}
			if rand.Intn(50) == 0 {
				if err := r.Checkpoint(); err != nil {
					errs <- err
					return
				}
			}
		}
	}()
	for round := range 6 {
		if err := writeN(stores[:2], 4, 250, int64(11+round*10)); err != nil {
			t.Fatalf("round %d: %v; replica at %d, primary at %d, slots %+v, stats %+v", round, err, r.ReplayLSN(),
				p.log.Published(), p.Slots(), p.Stats())
		}
		mustOK(t, p.Checkpoint())
	}
	stop.Store(true)
	wg.Wait()
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
	if st := p.Stats(); st.Checkpoints < 3 || st.Slots != 1 {
		t.Fatalf("primary stats: %d checkpoints, %d slots", st.Checkpoints, st.Slots)
	}

	catchUp(t, p, r, "r1")
	want := dumpEngine(t, p)
	checkContents(t, r, want)
	checkClean(t, r)
	if _, err := mustStore(t, r, "a").Put("x", nil), error(nil); !errors.Is(r.NewBatch().Commit(), ErrReadOnly) {
		t.Fatalf("a replica accepted a batch: %v", err)
	}
	if err := mustStore(t, r, "a").Put("x", nil); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Put on a replica: %v", err)
	}
	st := r.Stats()
	if !st.Replica || st.ReplayLSN != r.ReplayLSN() {
		t.Fatalf("replica stats %+v", st)
	}
	mustOK(t, r.Close())

	r = openTest(t, rfs, replicaConfig()) // sigue desde donde iba
	checkContents(t, r, want)
	mustOK(t, mustStore(t, p, "a").Put("after", []byte("reopen")))
	catchUp(t, p, r, "r1")
	want = dumpEngine(t, p)
	checkContents(t, r, want)
	checkClean(t, r)
	mustOK(t, r.Close())

	promoted := openTest(t, rfs, cfg) // Open la promueve
	checkContents(t, promoted, want)
	mustOK(t, mustStore(t, promoted, "a").Put("promoted", []byte("yes")))
	checkClean(t, promoted)
	mustOK(t, promoted.Close())
}

// TestReplicaNotFromScratch: OpenReplica de un directorio sin motor da ErrNotReplica, y ApplyWal en un primario
// también.
func TestReplicaNotFromScratch(t *testing.T) {
	if _, err := open(crashfs.New(), testDir, replicaConfig()); !errors.Is(err, ErrNotReplica) {
		t.Fatalf("OpenReplica of an empty directory: %v", err)
	}
	p := openTest(t, vfs.FS(crashfs.New()), testConfig())
	defer p.Close()
	if err := p.ApplyWal(nil); !errors.Is(err, ErrNotReplica) {
		t.Fatalf("ApplyWal on a primary: %v", err)
	}
}

// TestReplicaCrash: la réplica se corta en medio de aplicar (en los tres modos de crashfs, en un punto al azar de sus
// operaciones de disco: escribir el WAL, rehacer, un restartpoint); al reabrirla con OpenReplica sigue desde lo que
// quedó durable, alcanza al primario y coincide, con Verify limpio.
func TestReplicaCrash(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for seed := int64(1); seed <= 5; seed++ {
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				replicaCrashRound(t, mode, seed)
			})
		}
	}
}

/**
* replicaCrashRound es una vuelta de TestReplicaCrash.
* @param t *testing.T, mode crashfs.Mode, seed int64
**/
func replicaCrashRound(t *testing.T, mode crashfs.Mode, seed int64) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	_, err := p.CreateSlot("r1")
	mustOK(t, err)
	stores := fillStores(t, p, []string{"a", "b", "empty"}, 100)
	rbase := crashfs.New()
	_, err = p.backup(rbase, testDir)
	mustOK(t, err)
	proc := newProc(rbase)
	r := openTest(t, proc, replicaConfig())

	rng := rand.New(rand.NewSource(seed))
	cut := rng.Intn(8)
	var streamErr error
	for round := 0; round < 8 && streamErr == nil; round++ {
		mustOK(t, writeN(stores[:2], 3, 120, seed*100+int64(round)))
		mustOK(t, ddlOnce(p, round))
		mustOK(t, p.WaitDurable())
		if round == cut {
			proc.killAfter(int64(1+rng.Intn(400)), mode, seed)
		}
		_, streamErr = streamOnce(p, r, "r1")
		if streamErr == nil && round%3 == 2 {
			streamErr = r.Checkpoint()
		}
	}
	t.Logf("cut in round %d: %v, replica at %d", cut, streamErr, r.ReplayLSN())
	if streamErr == nil {
		proc.kill(mode, seed) // el corte no llegó: cortar al final
	}
	r.Close()

	r = openTest(t, newProc(rbase), replicaConfig())
	defer r.Close()
	catchUp(t, p, r, "r1")
	checkContents(t, r, dumpEngine(t, p))
	checkClean(t, r)
}
