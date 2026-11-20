package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// TestFailures inyecta fallas de escritura (FailAfter) y de fsync (FailSyncAfter) en un momento al azar: la
// operación que la encuentra devuelve ErrFailed, desde ahí todo devuelve ErrFailed y Close no se cuelga. Al reabrir
// sin corte el contenido es el confirmado (o con la operación en curso) y los árboles están sanos; tras un corte, lo
// mismo. El disco lleno, que casi siempre no deja el motor inutilizable, lo prueba diskfull_test.go.
func TestFailures(t *testing.T) {
	faults := map[string]func(fs crashfs.FS, n int){
		"write": func(fs crashfs.FS, n int) { fs.FailAfter(n) },
		"sync":  func(fs crashfs.FS, n int) { fs.FailSyncAfter(n) },
	}
	for name, inject := range faults {
		for seed := range 6 {
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				t.Parallel()
				runFailure(t, inject, uint64(seed))
			})
		}
	}
}

/**
* runFailure escribe un poco, inyecta la falla y sigue hasta que una operación falla; después comprueba el motor
* fallado, reabre sin corte y con corte.
* @param t *testing.T, inject func(crashfs.FS, int), seed uint64
**/
func runFailure(t *testing.T, inject func(crashfs.FS, int), seed uint64) {
	fsys := crashfs.New()
	gen := newGenerator(seed)
	e := openTest(t, fsys, testConfig())
	model := state{}
	for range 200 {
		o := gen.next()
		want := o.apply(model)
		if got, err := o.run(e); err != nil || got != want {
			t.Fatalf("%v = %v, %v; model says %v", o, got, err, want)
		}
	}
	inject(fsys, gen.rng.IntN(300))
	pending := failingOp(t, e, gen, &model)
	checkFailed(t, e)

	fsys.FailAfter(-1)
	fsys.FailSyncAfter(-1)
	fsys.SetCapacity(0)
	e = openTest(t, fsys, testConfig())
	model = recoveredOneOf(t, e, model, pending)
	verifyTrees(t, e)
	o := op{kind: "put", store: "a", id: "after-failure", value: "ok"}
	o.apply(model)
	_, err := o.run(e)
	mustOK(t, err)
	mustOK(t, e.Close())

	fsys.Crash(crashfs.LoseUnsynced, int64(seed))
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	recoveredOneOf(t, e, model, nil)
	verifyTrees(t, e)
}

/**
* failingOp hace operaciones hasta que una falla (con ErrFailed) y devuelve el estado si esa llegó a durable.
* @param t *testing.T, e *Engine, gen *generator, model *state: se actualiza con las confirmadas
* @return *state
**/
func failingOp(t *testing.T, e *Engine, gen *generator, model *state) *state {
	for range 5000 {
		o := gen.next()
		after := model.clone()
		want := o.apply(after)
		got, err := o.run(e)
		if err != nil {
			if !errors.Is(err, ErrFailed) {
				t.Fatalf("%v failed without ErrFailed: %v", o, err)
			}
			return &after
		}
		if got != want {
			t.Fatalf("%v = %v; model says %v", o, got, want)
		}
		*model = after
	}
	t.Fatal("the fault never hit an operation")
	return nil
}

/**
* checkFailed comprueba que el motor fallado devuelva ErrFailed en todo y que Close no se cuelgue.
* @param t *testing.T, e *Engine
**/
func checkFailed(t *testing.T, e *Engine) {
	t.Helper()
	if err := e.Checkpoint(); !errors.Is(err, ErrFailed) {
		t.Fatalf("checkpoint after failure: %v", err)
	}
	if _, err := e.Store("other"); !errors.Is(err, ErrFailed) {
		t.Fatalf("store after failure: %v", err)
	}
	for _, name := range e.Stores() {
		s := e.handles[name]
		if s == nil {
			continue
		}
		if err := s.Put("x", nil); !errors.Is(err, ErrFailed) {
			t.Fatalf("put after failure: %v", err)
		}
		if _, _, err := s.Read("x"); !errors.Is(err, ErrFailed) {
			t.Fatalf("read after failure: %v", err)
		}
	}
	closed := make(chan error, 1)
	go func() { closed <- e.Close() }()
	select {
	case err := <-closed:
		if !errors.Is(err, ErrFailed) {
			t.Fatalf("close after failure: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close of the failed engine hung")
	}
}

// TestCutCreate: un Create del control cortado (archivo de control sin copias válidas) con un catálogo vacío es un
// motor nuevo; con stores en el catálogo es corrupción.
func TestCutCreate(t *testing.T) {
	fsys := crashfs.New()
	mustOK(t, fsys.MkdirAll(testDir))
	f, err := fsys.OpenFile(filepath.Join(testDir, control.FileName), os.O_RDWR|os.O_CREATE)
	mustOK(t, err)
	_, err = f.WriteAt(make([]byte, 2*control.CopySize), 0)
	mustOK(t, err)
	mustOK(t, f.Close())
	e := openTest(t, fsys, testConfig())
	mustOK(t, mustStore(t, e, "s").Put("a", []byte("1")))
	mustOK(t, e.Close())

	f, err = fsys.OpenFile(filepath.Join(testDir, control.FileName), os.O_RDWR)
	mustOK(t, err)
	_, err = f.WriteAt(make([]byte, 2*control.CopySize), 0)
	mustOK(t, err)
	mustOK(t, f.Close())
	if _, err := open(fsys, testDir, testConfig()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("open with a broken control and stores: %v", err)
	}
}

// TestRecordsRoundTrip: los formatos de los registros del motor van y vuelven, y los mal formados se rechazan.
func TestRecordsRoundTrip(t *testing.T) {
	info := checkpointInfo{redo: wal.Position{Segment: 3, Offset: 77, LSN: 900}, nextXID: 12, oldest: 5, time: 42}
	if got, ok := decodeCheckpoint(encodeCheckpoint(info)); !ok || got != info {
		t.Fatalf("checkpoint = %+v %v", got, ok)
	}
	if file, name, ok := decodeStoreMain(encodeStoreMain(9, "ñandú")); !ok || file != 9 || name != "ñandú" {
		t.Fatalf("store main = %d %q %v", file, name, ok)
	}
	deltas := map[smgr.FileNo]int64{1: 3, 7: -2}
	got, err := decodeCommit(encodeCommit(5, commitInfo{deltas: deltas}), 0)
	if err != nil || len(got.deltas) != 2 || got.deltas[1] != 3 || got.deltas[7] != -2 || got.creates != nil || got.drops != nil {
		t.Fatalf("commit = %v %v", got, err)
	}
	ddl := encodeCommit(5, commitInfo{deltas: deltas, creates: []smgr.FileNo{4}, drops: []smgr.FileNo{2, 3}})
	got, err = decodeCommit(ddl, 0)
	if err != nil || len(got.deltas) != 2 || !slices.Equal(got.creates, []smgr.FileNo{4}) ||
		!slices.Equal(got.drops, []smgr.FileNo{2, 3}) {
		t.Fatalf("commit with ddl = %+v %v", got, err)
	}
	for _, cut := range []int{len(ddl) - 1, len(ddl) - 4, 12 + 24 + 4} {
		if _, err := decodeCommit(ddl[:cut], 0); err == nil {
			t.Fatalf("decodeCommit accepted a ddl tail cut at %d", cut)
		}
	}
	for _, bad := range [][]byte{nil, {1, 0, 0, 0, 0, 0}, encodeStoreMain(0, "x")} {
		if _, _, ok := decodeStoreMain(bad); ok {
			t.Fatalf("decodeStoreMain accepted %v", bad)
		}
	}
	if _, ok := decodeCheckpoint([]byte{1}); ok {
		t.Fatal("decodeCheckpoint accepted a short record")
	}
	if _, err := decodeCommit(encodeCommit(5, commitInfo{deltas: deltas})[:20], 0); err == nil {
		t.Fatal("decodeCommit accepted a short record")
	}
	// Con origen (tarea 83): va al final, también tras la cola de DDL, y lo marca wal.FlagOrigin
	for _, c := range []commitInfo{{deltas: deltas, origin: 7}, {deltas: deltas, creates: []smgr.FileNo{4}, origin: 65535}} {
		b := encodeCommit(5, c)
		got, err := decodeCommit(b, commitFlags(c))
		if err != nil || got.origin != c.origin || len(got.deltas) != 2 || !slices.Equal(got.creates, c.creates) {
			t.Fatalf("commit with origin = %+v %v", got, err)
		}
		if _, err := decodeCommit(b, 0); err == nil {
			t.Fatal("decodeCommit read an origin tail without FlagOrigin")
		}
	}
	if commitFlags(commitInfo{deltas: deltas}) != 0 {
		t.Fatal("FlagOrigin without an origin")
	}
	if _, err := decodeCommit([]byte{1, 2}, wal.FlagOrigin); err == nil {
		t.Fatal("decodeCommit accepted a short record with an origin")
	}
}
