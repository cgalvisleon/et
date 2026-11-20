package store

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Pruebas de regresión de la segunda parte de la revisión del 2026-11-20 (doc/review-store-2026-11-20.md): C1, C2,
// V1, D1 y la tarea 101. Cada una falla sin su arreglo.

// TestSlotDeferredCommitConsistent (C1, tarea 102): una ranura lógica creada mientras un lote confirma con
// CommitDeferred (su KindCommit queda en el búfer del WAL) no entrega ese lote a medias: su posición consistente
// cubre lo reservado, no solo lo publicado.
func TestSlotDeferredCommitConsistent(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	defer e.Close()
	a := mustStore(t, e, "a")
	o := mustStore(t, e, "o")
	b := e.NewBatch()
	mustOK(t, a.PutIn(b, "k1", []byte("1")))
	mustOK(t, o.Put("sync", []byte("x"))) // publica k1
	var wg sync.WaitGroup
	wg.Add(1)
	var serr error
	go func() {
		defer wg.Done()
		_, serr = e.CreateLogicalSlot("s")
	}()
	time.Sleep(100 * time.Millisecond) // la ranura espera al lote en waitConsistent
	mustOK(t, a.PutIn(b, "k2", []byte("2")))
	mustOK(t, b.CommitDeferred())
	wg.Wait()
	mustOK(t, serr)
	mustOK(t, e.WaitDurable())
	r, err := e.Changes("s")
	mustOK(t, err)
	defer r.Close()
	if got := readChanges(t, r); len(got) != 0 {
		t.Fatalf("a batch committed before the slot was consistent was delivered: %v", got)
	}
	mustOK(t, a.Put("k3", []byte("3")))
	mustOK(t, e.WaitDurable())
	if got := readChanges(t, r); len(got) != 1 {
		t.Fatalf("after the slot: %v", got)
	}
}

// TestChangeHeadOverflowWrite (C2, tarea 103): ChangeHead tomado mientras una escritura suelta escribe su cadena de
// desborde no corta esa escritura: el lector que parte de ahí la entrega entera o no la entrega (antes, ErrCorrupt).
func TestChangeHeadOverflowWrite(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	defer e.Close()
	a := mustStore(t, e, "a")
	big := []byte(strings.Repeat("v", 60_000))
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = a.Put("big", big)
			time.Sleep(200 * time.Microsecond) // crashfs guarda todo en memoria: acotar el WAL que se escribe
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	for i := 0; i < 100; i++ {
		pos, err := e.ChangeHead()
		mustOK(t, err)
		r, err := e.ChangesFrom(pos, ChangeFilter{})
		mustOK(t, err)
		time.Sleep(2 * time.Millisecond)
		mustOK(t, e.WaitDurable())
		_, err = r.Next(0, func(_ TxInfo, c Change) error {
			if c.Op == ChangePut && len(c.Data) != len(big) {
				t.Errorf("value of %d bytes, want %d", len(c.Data), len(big))
			}
			return nil
		})
		r.Close()
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}

// TestReplicaVerifyLockOrder (V1, tarea 104): en una réplica Verify toma ckptMu antes que applyMu, como el
// restartpoint y Promote. Con ckptMu tomado (un restartpoint en curso), un Verify que espera no puede tener applyMu:
// el restartpoint lo toma. Antes se colgaba; la prueba tiene plazo y falla en vez de colgarse.
func TestReplicaVerifyLockOrder(t *testing.T) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	_, err := p.CreateSlot("r")
	mustOK(t, err)
	a := mustStore(t, p, "a")
	mustOK(t, a.Put("k1", []byte("v1")))
	r, _ := newReplica(t, p)
	defer r.Close()
	mustOK(t, a.Put("k2", []byte("v2")))
	catchUp(t, p, r, "r")

	r.ckptMu.Lock() // como el restartpoint, antes de pedir applyMu
	done := make(chan error, 1)
	go func() {
		_, err := r.Verify()
		done <- err
	}()
	time.Sleep(100 * time.Millisecond) // Verify llega a esperar ckptMu
	got := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if r.replica.applyMu.TryLock() {
			got = true
			r.replica.applyMu.Unlock()
			break
		}
	}
	r.ckptMu.Unlock()
	if !got {
		<-done
		t.Fatal("Verify holds applyMu while it waits for ckptMu: it deadlocks with the restartpoint")
	}
	select {
	case err := <-done:
		mustOK(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Verify did not finish")
	}

	// y de punta a punta: Verify, Checkpoint (restartpoint) y ApplyWal a la vez, con plazo
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for _, f := range []func() error{
		func() error { _, err := r.Verify(); return err },
		r.Checkpoint,
		func() error { _, err := streamOnce(p, r, "r"); return err },
	} {
		wg.Add(1)
		go func(f func() error) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				if err := f(); err != nil {
					errs <- err
					return
				}
			}
		}(f)
	}
	for i := 0; i < 20; i++ {
		mustOK(t, a.Put("k", []byte{byte(i)}))
	}
	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(60 * time.Second):
		t.Fatal("Verify, Checkpoint and ApplyWal on a replica deadlocked")
	}
	close(errs)
	for err := range errs {
		mustOK(t, err)
	}
}

// TestDropUnlinkAfterSyncAll (D1): un DropStore cuyo KindDropStore queda antes del punto de rehacer y cuyo unlink
// llega después del SyncAll del checkpoint no deja, tras un corte, un archivo huérfano: el paso 3 hace el fsync de
// data/ con ddlMu.
func TestDropUnlinkAfterSyncAll(t *testing.T) {
	base := crashfs.New()
	fsys := &hookFS{procFS: newProc(base)}
	cfg := testConfig()
	e, err := open(fsys, testDir, cfg)
	mustOK(t, err)
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("k", []byte("v")))
	mustOK(t, e.Checkpoint())

	release := make(chan struct{})
	blocked := make(chan struct{})
	var once sync.Once
	fsys.onRemove = func(name string) {
		if strings.Contains(name, "/data/") {
			once.Do(func() { close(blocked); <-release })
		}
	}
	e.afterStep = func(step int) {
		if step == 2 {
			select {
			case <-release:
			default:
				close(release)
			}
		}
	}
	done := make(chan error, 1)
	go func() { done <- e.DropStore("a") }()
	<-blocked
	ck := make(chan error, 1)
	go func() { ck <- e.Checkpoint() }()
	mustOK(t, <-done)
	mustOK(t, <-ck)
	fsys.onRemove = nil
	fsys.kill(crashfs.LoseUnsynced, 1)
	closeWithin(t, e, 10*time.Second)

	e2, err := open(newProc(base), testDir, cfg)
	mustOK(t, err)
	defer e2.Close()
	rep, err := e2.Verify()
	mustOK(t, err)
	if rep.OrphanFiles > 0 || !rep.Clean() {
		t.Fatalf("after the crash:\n%s", rep.String())
	}
}

// TestSealedWALDamageIsErrCorrupt (tarea 101): el daño en un segmento sellado del WAL hace fallar Open con un error
// que cumple errors.Is(err, ErrCorrupt) (y sigue cumpliendo el de wal).
func TestSealedWALDamageIsErrCorrupt(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, smallWAL())
	s := mustStore(t, e, "a")
	for i := 0; i < 50; i++ {
		mustOK(t, s.Put(strings.Repeat("k", 10)+string(rune('a'+i%26)), []byte(strings.Repeat("v", 200))))
	}
	_, err := e.SwitchWAL()
	mustOK(t, err)
	mustOK(t, s.Put("after", []byte("x")))
	mustOK(t, e.WaitDurable())
	img := readImage(t, fsys, testDir) // sin cerrar: el punto de rehacer queda antes del segmento sellado
	mustOK(t, e.Close())

	var segs []string
	for _, p := range img.paths {
		if filepath.Base(filepath.Dir(p)) == wal.Dir {
			segs = append(segs, p)
		}
	}
	sort.Strings(segs)
	if len(segs) < 2 {
		t.Fatalf("want a sealed segment and an active one, got %v", segs)
	}
	seg := img.files[segs[0]]
	seg[2000] ^= 0xff // dentro de los registros, lejos del encabezado y del sellado
	_, err = open(writeImage(t, img), testDir, smallWAL())
	if !errors.Is(err, ErrCorrupt) || !errors.Is(err, wal.ErrCorrupted) {
		t.Fatalf("open with a damaged sealed segment: %v", err)
	}
}
