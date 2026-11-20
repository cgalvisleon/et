package store

import (
	"errors"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// TestPromote (tarea 85): Promote vuelve primario a una réplica sin cerrarla. Lo confirmado en el primario está; lo
// que seguía en curso (un lote con escrituras y otro que creaba un store) queda abortado; un snapshot abierto antes
// sigue leyendo; ApplyWal da ErrNotReplica, ReplayLSN 0 y Stats.Replica false; se escribe enseguida (suelto y en
// lote); Verify da limpio; y tras un corte, al reabrir con Open, sigue todo lo confirmado después de promover.
func TestPromote(t *testing.T) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	_, err := p.CreateSlot("r")
	mustOK(t, err)
	a := mustStore(t, p, "a")
	mustOK(t, a.Put("k1", []byte("v1")))
	rfs := crashfs.New()
	_, err = p.backup(rfs, testDir)
	mustOK(t, err)
	proc := newProc(rfs)
	r := openTest(t, proc, replicaConfig())

	mustOK(t, a.Put("k2", []byte("v2")))
	open := p.NewBatch()
	mustOK(t, a.PutIn(open, "never", []byte("x")))
	ddl := p.NewBatch()
	_, err = p.StoreIn(ddl, "pending")
	mustOK(t, err)
	catchUp(t, p, r, "r")
	v := r.Snapshot()
	ra := mustStore(t, r, "a")
	mustOK(t, r.Promote())

	if err := r.Promote(); !errors.Is(err, ErrNotReplica) {
		t.Fatalf("second Promote: %v", err)
	}
	if err := r.ApplyWal(nil); !errors.Is(err, ErrNotReplica) {
		t.Fatalf("ApplyWal after Promote: %v", err)
	}
	if st := r.Stats(); st.Replica || r.ReplayLSN() != 0 {
		t.Fatalf("after Promote: replica %v, replay lsn %d", st.Replica, r.ReplayLSN())
	}
	if data, ok, err := ra.ReadAt(v, "k2"); err != nil || !ok || string(data) != "v2" {
		t.Fatalf("snapshot from before: %q %v %v", data, ok, err)
	}
	mustOK(t, v.Close())
	if _, ok, _ := ra.Read("never"); ok {
		t.Fatal("a write of a batch that never committed is visible")
	}
	for _, name := range r.Stores() {
		if name == "pending" {
			t.Fatal("the store of a batch that never committed survived")
		}
	}
	mustOK(t, ra.Put("k3", []byte("v3")))
	b := r.NewBatch()
	mustOK(t, ra.PutIn(b, "never", []byte("mine"))) // el id del lote abortado se escribe sin esperar a nadie
	mustOK(t, ra.PutIn(b, "k4", []byte("v4")))
	mustOK(t, b.Commit())
	checkClean(t, r)

	proc.kill(crashfs.LoseUnsynced, 1)
	r.Close()
	r = openTest(t, newProc(rfs), smallWAL())
	defer r.Close()
	checkContents(t, r, contents{"a": {"k1": "v1", "k2": "v2", "k3": "v3", "k4": "v4", "never": "mine"}})
	checkClean(t, r)
	_ = open.Rollback()
	_ = ddl.Rollback()
}
