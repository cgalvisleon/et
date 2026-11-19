package btree

import (
	"bytes"
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// testDir es el directorio del motor de prueba; testFile, el store.
const (
	testDir              = "/db"
	testFile smgr.FileNo = 1
)

// testOptions: segmentos de 1 MB, el mínimo (así también se rota).
var testOptions = wal.Options{SegmentSize: wal.MinSegmentSize, Spares: 1}

// lateFlusher es el wal.Flusher de la caché de prueba: mientras se rehace dentro de wal.Open todavía no hay WAL
// abierto, y todo lo que se reproduce ya está en el disco (se está leyendo de ahí), así que da todo por durable.
type lateFlusher struct {
	log atomic.Pointer[wal.WAL]
}

func (f *lateFlusher) Durable() page.LSN {
	if w := f.log.Load(); w != nil {
		return w.Durable()
	}
	return MaxLSN
}

func (f *lateFlusher) WaitDurable(lsn page.LSN) error {
	if w := f.log.Load(); w != nil {
		return w.WaitDurable(lsn)
	}
	return nil
}

// env es un motor de prueba abierto: archivos de datos, WAL, caché real y fsm real.
type env struct {
	fs   vfs.FS
	sm   smgr.Manager
	log  *wal.WAL
	pool buffer.Pool
	free *fsm.Manager
}

/**
* openEnv abre el motor de prueba sobre fsys rehaciendo desde redo los registros de btree y de fsm, con una caché de
* buffers páginas (chica: obliga a expulsar) y sin escritor de fondo.
* @param t testing.TB, fsys vfs.FS, redo wal.Position, buffers int
* @return *env
**/
func openEnv(t testing.TB, fsys vfs.FS, redo wal.Position, buffers int) *env {
	t.Helper()
	return openEnvWith(t, fsys, redo, buffers, -1)
}

/**
* openEnvWith es openEnv con la pausa del escritor de fondo (negativa = sin él).
* @param t testing.TB, fsys vfs.FS, redo wal.Position, buffers int, bgDelay time.Duration
* @return *env
**/
func openEnvWith(t testing.TB, fsys vfs.FS, redo wal.Position, buffers int, bgDelay time.Duration) *env {
	t.Helper()
	sm, err := smgr.Open(fsys, testDir, 16)
	if err != nil {
		t.Fatal(err)
	}
	fl := &lateFlusher{}
	pool, err := buffer.Open(buffer.Config{Smgr: sm, Flusher: fl, Buffers: buffers, BgWriterDelay: bgDelay})
	if err != nil {
		t.Fatal(err)
	}
	log, _, err := wal.Open(fsys, testDir, testOptions, redo, func(rec *wal.Record) error {
		return redoAny(pool, sm, rec)
	})
	if err != nil {
		t.Fatal(err)
	}
	fl.log.Store(log)
	t.Cleanup(func() {
		pool.Close()
		log.Close()
		sm.Close()
	})
	return &env{fs: fsys, sm: sm, log: log, pool: pool, free: fsm.New(pool, sm, log)}
}

/**
* redoAny manda cada registro a su dueño: btree (16–31) o fsm (40–47).
* @param pool buffer.Pool, sm smgr.Manager, rec *wal.Record
* @return error
**/
func redoAny(pool buffer.Pool, sm smgr.Manager, rec *wal.Record) error {
	switch {
	case rec.Kind >= wal.KindLeafWrite && rec.Kind <= wal.KindSetHints:
		return Redo(pool, sm, rec)
	case rec.Kind >= wal.KindFreePush && rec.Kind <= wal.KindVMClear:
		return fsm.Redo(pool, sm, rec)
	}
	return fmt.Errorf("unexpected kind %d", rec.Kind)
}

// config arma la Config del store de prueba.
func (e *env) config() Config {
	return Config{File: testFile, Pool: e.pool, Smgr: e.sm, Free: e.free, Log: e.log}
}

// createTree crea el store de prueba.
func createTree(t testing.TB, e *env) *BTree {
	t.Helper()
	tr, err := Create(e.config())
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// openTree abre el store de prueba ya creado.
func openTree(t testing.TB, e *env) *BTree {
	t.Helper()
	tr, err := Open(e.config())
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

/**
* checkpoint hace lo que el checkpoint del motor hace con las páginas: fija el punto de rehacer, escribe todas las
* sucias (cada una tras el WAL durable hasta su pd_lsn) y hace fsync de los archivos de datos.
* @param t testing.TB, e *env
* @return wal.Position: el punto de rehacer desde donde recuperar
**/
func checkpoint(t testing.TB, e *env) wal.Position {
	t.Helper()
	pos, err := e.log.BeginCheckpoint(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if err := e.sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
	return pos
}

// model es el contenido esperado: id → valor vivo.
type model map[string][]byte

// put escribe id = value como una operación suelta y lo anota en el modelo.
func put(t testing.TB, tr *BTree, m model, id string, value []byte) Result {
	t.Helper()
	res, err := tr.Write(id, Write{XID: 1, Value: value, Commit: true})
	if err != nil {
		t.Fatalf("Write(%q): %v", short(id), err)
	}
	if m != nil {
		m[id] = value
	}
	return res
}

// remove borra id (lápida) y lo quita del modelo.
func remove(t testing.TB, tr *BTree, m model, id string) Result {
	t.Helper()
	res, err := tr.Write(id, Write{XID: 1, Tombstone: true, Commit: true})
	if err != nil {
		t.Fatalf("Remove(%q): %v", short(id), err)
	}
	if _, live := m[id]; live != res.Applied {
		t.Fatalf("Remove(%q) applied %v, live in the model %v", short(id), res.Applied, live)
	}
	delete(m, id)
	return res
}

// short acorta un id largo para los mensajes.
func short(id string) string {
	if len(id) > 24 {
		return fmt.Sprintf("%s…(%d B)", id[:12], len(id))
	}
	return id
}

// sortedIDs devuelve los ids del modelo en orden.
func (m model) sortedIDs() []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

/**
* checkTree verifica la estructura y que el árbol tenga exactamente el modelo: Lookup de cada id (y de algunos que
* no están), Scan entero hacia adelante y hacia atrás.
* @param t testing.TB, tr *BTree, m model
* @return VerifyStats
**/
func checkTree(t testing.TB, tr *BTree, m model) VerifyStats {
	t.Helper()
	st, err := tr.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	t.Logf("árbol: %+v", st)
	ids := m.sortedIDs()
	for _, id := range ids {
		v, ok, err := tr.Lookup(id, nil)
		if err != nil || !ok || !bytes.Equal(v.Value, m[id]) {
			t.Fatalf("Lookup(%q) = %q, %v, %v; se esperaba %q", short(id), v.Value, ok, err, m[id])
		}
	}
	for _, id := range []string{"\x00", "absent-id", "\xff\xff\xff"} {
		if _, ok := m[id]; ok {
			continue
		}
		if _, ok, err := tr.Lookup(id, nil); ok || err != nil {
			t.Fatalf("Lookup(%q) de un id que no está: %v, %v", id, ok, err)
		}
	}
	got := scanAll(t, tr, "", 0)
	if !slices.Equal(got, ids) {
		t.Fatalf("Scan: %d ids, se esperaban %d (primer distinto %s)", len(got), len(ids), firstDiff(got, ids))
	}
	back := scanBack(t, tr, "", 0)
	slices.Reverse(back)
	if !slices.Equal(back, ids) {
		t.Fatalf("ScanBackward: %d ids, se esperaban %d (primer distinto %s)", len(back), len(ids), firstDiff(back, ids))
	}
	return st
}

// firstDiff describe la primera diferencia entre dos listas de ids.
func firstDiff(a, b []string) string {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return fmt.Sprintf("en %d: %q / %q", i, short(a[i]), short(b[i]))
		}
	}
	return fmt.Sprintf("en %d (largo)", min(len(a), len(b)))
}

// scanAll recorre desde from hacia adelante y devuelve hasta limit ids (0 = todos).
func scanAll(t testing.TB, tr *BTree, from string, limit int) []string {
	t.Helper()
	var ids []string
	err := tr.Scan(from, nil, func(v Version) (bool, error) {
		ids = append(ids, v.ID)
		return limit == 0 || len(ids) < limit, nil
	})
	if err != nil {
		t.Fatalf("Scan(%q): %v", short(from), err)
	}
	return ids
}

// scanBack recorre desde to hacia atrás y devuelve hasta limit ids (0 = todos).
func scanBack(t testing.TB, tr *BTree, to string, limit int) []string {
	t.Helper()
	var ids []string
	err := tr.ScanBackward(to, nil, func(v Version) (bool, error) {
		ids = append(ids, v.ID)
		return limit == 0 || len(ids) < limit, nil
	})
	if err != nil {
		t.Fatalf("ScanBackward(%q): %v", short(to), err)
	}
	return ids
}

// randomID arma un id al azar de 1 a maxLen bytes sobre un alfabeto chico (para que haya prefijos comunes).
func randomID(rng *rand.Rand, maxLen int) string {
	b := make([]byte, 1+rng.Intn(maxLen))
	for i := range b {
		b[i] = "abcdxyz"[rng.Intn(7)]
	}
	return string(b)
}

// randomValue arma un valor al azar de hasta maxLen bytes.
func randomValue(rng *rand.Rand, maxLen int) []byte {
	b := make([]byte, rng.Intn(maxLen+1))
	rng.Read(b)
	return b
}
