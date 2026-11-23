package store

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
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

// TestPromoteFormatV1Backup (tarea 138): una réplica abierta de la copia de referencia del formato 1, cuyo segmento
// activo es del 1 y tiene registros, al promoverse rota el WAL: el activo pasa a uno nuevo con encabezado del
// formato actual, el del 1 queda sellado para archivar, se escriben valores comprimidos y, tras un corte y Open,
// siguen; Verify limpio.
func TestPromoteFormatV1Backup(t *testing.T) {
	ref := readReference(t, formatV1Dir)
	doc := jsonDoc(rand.New(rand.NewPCG(13, 14)), 8192)
	base := writeImage(t, ref.backup)
	proc := newProc(base)
	cfg := formatConfig()
	cfg.replica = true
	cfg.conflictDelay = 2 * time.Second
	r := openTest(t, proc, cfg)
	old := r.log.ActiveSegment()
	if h := activeHeader(t, base, old); h.Version != 1 {
		t.Fatalf("the replica of the v1 copy starts on a segment of version %d", h.Version)
	}
	mustOK(t, r.Promote())
	active := r.log.ActiveSegment()
	if h := activeHeader(t, base, active); active == old || h.Version != wal.FormatVersion {
		t.Fatalf("after Promote: active %d (was %d), version %d", active, old, h.Version)
	}
	pending, err := r.PendingArchive()
	mustOK(t, err)
	found := false
	for _, seg := range pending {
		found = found || seg.Segment == old
	}
	if !found {
		t.Fatalf("the sealed v1 segment %d is not pending to archive: %+v", old, pending)
	}
	s := mustStore(t, r, "small")
	mustOK(t, s.Put("promoted", doc))
	if got := r.log.ActiveSegment(); got != active {
		t.Fatalf("the first write moved the active segment to %d", got)
	}
	checkClean(t, r)
	proc.kill(crashfs.LoseUnsynced, 1)
	r.Close()

	e := openTest(t, newProc(base), formatConfig())
	defer e.Close()
	got, ok, err := mustStore(t, e, "small").Read("promoted")
	mustOK(t, err)
	if !ok || !bytes.Equal(got, doc) {
		t.Fatalf("value written after Promote: found %v, %d bytes", ok, len(got))
	}
	checkClean(t, e)
}

// TestPromoteRotates (tarea 138): una réplica normal (formato actual) también rota al promoverse; sin registros en
// el activo no rota (no hay nada que dejar atrás).
func TestPromoteRotates(t *testing.T) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	a := mustStore(t, p, "a")
	mustOK(t, a.Put("k1", []byte("v1")))
	r, _ := newReplica(t, p)
	defer r.Close()
	before := r.log.ActiveSegment()
	mustOK(t, r.Promote())
	if after := r.log.ActiveSegment(); after != before+1 {
		t.Fatalf("active segment %d after Promote, was %d", after, before)
	}
	mustOK(t, mustStore(t, r, "a").Put("k2", []byte("v2")))
	checkClean(t, r)

	num, err := r.log.StartSegment() // recién rotado y con el checkpoint escrito: sí rota
	mustOK(t, err)
	if num == 0 {
		t.Fatal("StartSegment did not seal an active segment with records")
	}
	if num, err = r.log.StartSegment(); err != nil || num != 0 {
		t.Fatalf("StartSegment of an empty active segment: %d %v", num, err)
	}
}

/**
* activeHeader lee el encabezado del segmento num del WAL del motor en fsys.
* @param t testing.TB, fsys vfs.FS, num uint64
* @return wal.SegmentHeader
**/
func activeHeader(t testing.TB, fsys vfs.FS, num uint64) wal.SegmentHeader {
	t.Helper()
	f, err := fsys.OpenFile(filepath.Join(testDir, wal.Dir, wal.SegmentName(num)), os.O_RDONLY)
	mustOK(t, err)
	defer f.Close()
	buf := make([]byte, wal.SegmentHeaderSize)
	_, err = f.ReadAt(buf, 0)
	mustOK(t, err)
	h, err := wal.ParseSegmentHeader(buf)
	mustOK(t, err)
	return h
}
