package store

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// contents es todo lo que tiene un motor: store → id → valor.
type contents map[string]map[string]string

/**
* dumpEngine lee todos los stores del motor con ForEach.
* @param t testing.TB, e *Engine
* @return contents
**/
func dumpEngine(t testing.TB, e *Engine) contents {
	t.Helper()
	c := contents{}
	for _, name := range e.Stores() {
		s := mustStore(t, e, name)
		m := map[string]string{}
		mustOK(t, s.ForEach(func(id string, data []byte) (bool, error) {
			m[id] = string(data)
			return true, nil
		}, true, 0, 0))
		c[name] = m
	}
	return c
}

/**
* checkContents compara el contenido y Count de cada store con want.
* @param t testing.TB, e *Engine, want contents
**/
func checkContents(t testing.TB, e *Engine, want contents) {
	t.Helper()
	got := dumpEngine(t, e)
	if len(got) != len(want) {
		t.Fatalf("stores %v, want %v", slices.Sorted(maps.Keys(got)), slices.Sorted(maps.Keys(want)))
	}
	for name, w := range want {
		g, ok := got[name]
		if !ok {
			t.Fatalf("store %q missing", name)
		}
		if len(g) != len(w) {
			t.Fatalf("store %q has %d ids, want %d", name, len(g), len(w))
		}
		for id, v := range w {
			if g[id] != v {
				t.Fatalf("store %q id %q: got %.20q, want %.20q", name, id, g[id], v)
			}
		}
		if n := mustStore(t, e, name).Count(); n != int64(len(w)) {
			t.Fatalf("store %q Count %d, want %d", name, n, len(w))
		}
	}
}

/**
* restore abre la copia de dst como un motor (la restaura), exige Verify limpio y que el marcador ya no esté.
* @param t testing.TB, dst vfs.FS, cfg config
* @return *Engine
**/
func restore(t testing.TB, dst vfs.FS, cfg config) *Engine {
	t.Helper()
	r := openTest(t, dst, cfg)
	checkClean(t, r)
	names, err := dst.ReadDir(testDir)
	mustOK(t, err)
	if slices.Contains(names, backupLabelName) {
		t.Fatal("backup_label still there after the restore")
	}
	return r
}

/**
* checkReleased exige que el motor no retenga nada de un respaldo: ni modo respaldo ni retención del WAL, ni copia
* de datos en curso, ni borrados pendientes, ni retención del archivo de estados.
* @param t testing.TB, e *Engine
**/
func checkReleased(t testing.TB, e *Engine) {
	t.Helper()
	e.ddlMu.Lock()
	copying, deferred := e.backups.copying, len(e.backups.deferred)
	e.ddlMu.Unlock()
	e.backups.mu.Lock()
	frozen := len(e.backups.frozen)
	e.backups.mu.Unlock()
	if h := e.log.Holds(); h != 0 || copying != 0 || deferred != 0 || frozen != 0 {
		t.Fatalf("backup left holds %d, copying %d, deferred %d, frozen %d", h, copying, deferred, frozen)
	}
}

// backupWriter escribe al azar en stores mientras no lo detengan: sueltas, diferidas o en lotes (con Commit o
// CommitDeferred). Los errores de conflicto o serialización de un lote se deshacen y siguen.
type backupWriter struct {
	stop atomic.Bool
	wg   sync.WaitGroup
	errs chan error
}

/**
* startWriters arranca n escritores sobre stores.
* @param stores []*Store, n int, seed int64
* @return *backupWriter
**/
func startWriters(stores []*Store, n int, seed int64) *backupWriter {
	w := &backupWriter{errs: make(chan error, n)}
	for i := range n {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			rng := rand.New(rand.NewSource(seed + int64(i)))
			for !w.stop.Load() {
				if err := writeOne(stores, i, rng); err != nil {
					w.errs <- err
					return
				}
			}
		}()
	}
	return w
}

/**
* halt detiene a los escritores, espera a que terminen su operación y falla si alguno dio un error.
* @param t testing.TB
**/
func (w *backupWriter) halt(t testing.TB) {
	t.Helper()
	w.stop.Store(true)
	w.wg.Wait()
	select {
	case err := <-w.errs:
		t.Fatal(err)
	default:
	}
}

/**
* writeOne hace una operación al azar del escritor w: Put o Remove sueltos, un Put diferido, o un lote de tres Put
* (Commit o CommitDeferred). Algunos valores van a desborde.
* @param stores []*Store, w int, rng *rand.Rand
* @return error
**/
func writeOne(stores []*Store, w int, rng *rand.Rand) error {
	s := stores[rng.Intn(len(stores))]
	id := fmt.Sprintf("w%d-%03d", w, rng.Intn(150))
	value := []byte(fmt.Sprintf("%s=%d", id, rng.Int63()))
	if rng.Intn(10) == 0 {
		value = append(value, bytes.Repeat([]byte{'x'}, 3000+rng.Intn(6000))...)
	}
	switch rng.Intn(5) {
	case 0:
		return s.Remove(id)
	case 1:
		return s.Deferred().Put(id, value)
	case 2, 3:
		tx := s.e.NewBatch()
		for j := range 3 {
			if err := s.PutIn(tx, fmt.Sprintf("%s-%d", id, j), value); err != nil {
				tx.Rollback()
				if errors.Is(err, ErrSerialization) || errors.Is(err, ErrConflict) {
					return nil
				}
				return err
			}
		}
		if rng.Intn(2) == 0 {
			return tx.CommitDeferred()
		}
		return tx.Commit()
	}
	return s.Put(id, value)
}

/**
* fillStores crea los stores names con n ids cada uno (algunos valores grandes).
* @param t testing.TB, e *Engine, names []string, n int
* @return []*Store
**/
func fillStores(t testing.TB, e *Engine, names []string, n int) []*Store {
	t.Helper()
	var stores []*Store
	for _, name := range names {
		s := mustStore(t, e, name)
		for i := range n {
			v := fmt.Sprintf("%s/%d", name, i)
			if i%25 == 0 {
				v += strings.Repeat("y", 9000)
			}
			mustOK(t, s.Put(fmt.Sprintf("k%04d", i), []byte(v)))
		}
		stores = append(stores, s)
	}
	return stores
}

// TestBackupConcurrent: un respaldo con escritores concurrentes (sueltos, diferidos y lotes) y checkpoints y ráfagas de
// escrituras entre archivo y archivo; la copia restaurada tiene Verify limpio y exactamente lo confirmado hasta el
// fin del respaldo (los escritores se detienen al llegar al fin y vuelven después: lo de después no aparece).
func TestBackupConcurrent(t *testing.T) {
	base, dst := crashfs.New(), crashfs.New()
	cfg := testConfig()
	e := openTest(t, base, cfg)
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b", "c"}, 200)
	w := startWriters(stores, 4, 1)
	var want contents
	rng := rand.New(rand.NewSource(7))
	e.afterBackupStep = func(step string) {
		switch {
		case step == "end":
			w.halt(t)
			want = dumpEngine(t, e)
		case strings.HasPrefix(step, "data/") || strings.HasPrefix(step, "xact/"):
			for range 40 {
				mustOK(t, writeOne(stores, 9, rng))
			}
			mustOK(t, e.Checkpoint())
		}
	}
	info, err := e.backup(dst, testDir)
	mustOK(t, err)
	t.Logf("backup: %+v", info)
	if info.EndLSN <= info.CheckpointLSN || info.CheckpointLSN < info.StartLSN || info.Files < 8 {
		t.Fatalf("info %+v", info)
	}
	checkReleased(t, e)

	w = startWriters(stores, 2, 100) // lo de después del fin no está en la copia
	for range 100 {
		mustOK(t, writeOne(stores, 8, rng))
	}
	w.halt(t)
	checkClean(t, e)

	r := restore(t, dst, cfg)
	checkContents(t, r, want)
	mustOK(t, r.Close())
	r = openTest(t, dst, cfg) // reabrir la copia restaurada es un motor más
	checkContents(t, r, want)
	mustOK(t, r.Close())
}

// TestBackupDDL: crear, borrar y vaciar stores, sueltos y en lotes (confirmados y deshechos), mientras se copian los
// archivos. El original no informa huérfanos durante la copia (los borrados esperan) ni después (se hicieron al
// terminar), y la copia restaurada tiene lo del fin del respaldo, sin huérfanos.
func TestBackupDDL(t *testing.T) {
	base, dst := crashfs.New(), crashfs.New()
	cfg := testConfig()
	e := openTest(t, base, cfg)
	defer e.Close()
	fillStores(t, e, []string{"drop", "empty", "keep", "lot"}, 150)
	var want contents
	done := false
	e.afterBackupStep = func(step string) {
		switch {
		case step == "end":
			want = dumpEngine(t, e)
		case strings.HasPrefix(step, "data/") && !done:
			done = true
			mustOK(t, e.DropStore("drop"))
			mustOK(t, mustStore(t, e, "empty").Empty())
			mustOK(t, mustStore(t, e, "empty").Put("after", []byte("empty again")))
			fillStores(t, e, []string{"new"}, 50)
			tmp := mustStore(t, e, "tmp")
			mustOK(t, tmp.Put("x", []byte("y")))
			mustOK(t, e.DropStore("tmp"))

			tx := e.NewBatch()
			s, err := e.StoreIn(tx, "inbatch")
			mustOK(t, err)
			mustOK(t, s.PutIn(tx, "a", []byte("b")))
			mustOK(t, e.DropStoreIn(tx, "lot"))
			mustOK(t, tx.Commit())

			tx = e.NewBatch()
			s, err = e.StoreIn(tx, "undone")
			mustOK(t, err)
			mustOK(t, s.PutIn(tx, "a", []byte("b")))
			mustOK(t, tx.Rollback())

			mustOK(t, e.Checkpoint())
			checkClean(t, e) // los archivos que esperan no son huérfanos
		}
	}
	_, err := e.backup(dst, testDir)
	mustOK(t, err)
	checkReleased(t, e)
	checkNoOrphans(t, e, base)
	if !done || slices.Contains(e.Stores(), "drop") || want["empty"]["after"] != "empty again" {
		t.Fatalf("the DDL did not run: done %v, stores %v", done, e.Stores())
	}

	r := restore(t, dst, cfg)
	defer r.Close()
	checkContents(t, r, want)
	checkNoOrphans(t, r, dst)
}

// TestBackupRecycle: con un WAL chico (MAX_WAL_SIZE bajo) los checkpoints del medio reciclarían los segmentos desde el
// inicio del respaldo; la retención los guarda hasta copiarlos, la copia se restaura y, al terminar, el próximo
// checkpoint los recicla.
func TestBackupRecycle(t *testing.T) {
	base, dst := crashfs.New(), crashfs.New()
	cfg := testConfig()
	cfg.maxWALSize = 512 << 10
	cfg.wal.SegmentSize = 256 << 10
	cfg.wal.Spares = 1
	e := openTest(t, base, cfg)
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b"}, 100)
	rng := rand.New(rand.NewSource(3))
	var want contents
	var first uint64
	e.afterBackupStep = func(step string) {
		switch {
		case step == "start":
			first = e.log.RedoPoint().Segment
		case step == "end":
			if !slices.Contains(walSegments(t, base), first) {
				t.Fatalf("segment %d of the backup start was recycled", first)
			}
			want = dumpEngine(t, e)
		case strings.HasPrefix(step, "data/") || strings.HasPrefix(step, "xact/"):
			for range 150 { // unos 700 KB de WAL
				mustOK(t, writeOne(stores, 1, rng))
			}
			mustOK(t, e.Checkpoint())
		}
	}
	info, err := e.backup(dst, testDir)
	mustOK(t, err)
	segs := walSegments(t, base)
	t.Logf("backup %+v, segments %d..%d", info, first, segs[len(segs)-1])
	if segs[len(segs)-1]-first < 3 {
		t.Fatalf("the backup spans too few segments: %d..%v", first, segs)
	}
	checkReleased(t, e)
	mustOK(t, e.Checkpoint())
	if slices.Contains(walSegments(t, base), first) {
		t.Fatalf("segment %d still there after the backup", first)
	}

	r := restore(t, dst, cfg)
	defer r.Close()
	checkContents(t, r, want)
}

/**
* walSegments da los números de los segmentos del WAL del motor de prueba.
* @param t testing.TB, fsys vfs.FS
* @return []uint64
**/
func walSegments(t testing.TB, fsys vfs.FS) []uint64 {
	t.Helper()
	names, err := fsys.ReadDir(filepath.Join(testDir, wal.Dir))
	mustOK(t, err)
	var nums []uint64
	for _, name := range names {
		var n uint64
		if _, err := fmt.Sscanf(name, "%010d.seg", &n); err == nil {
			nums = append(nums, n)
		}
	}
	return nums
}

// TestBackupIncomplete: una copia cuyo WAL no llega al registro de fin (cortado antes), o sin terminar (sin fin en el
// marcador), no se abre: ErrBackupIncomplete, sin cambiar nada; con el WAL entero se abre. Un destino que no está
// vacío da ErrBackupDir.
func TestBackupIncomplete(t *testing.T) {
	base, dst := crashfs.New(), crashfs.New()
	cfg := testConfig()
	e := openTest(t, base, cfg)
	defer e.Close()
	fillStores(t, e, []string{"a"}, 300)
	info, err := e.backup(dst, testDir)
	mustOK(t, err)
	if _, err := e.backup(dst, testDir); !errors.Is(err, ErrBackupDir) {
		t.Fatalf("backup to a used directory: %v", err)
	}
	want := dumpEngine(t, e)

	segs := walSegments(t, dst)
	last := filepath.Join(testDir, wal.Dir, fmt.Sprintf("%010d.seg", segs[len(segs)-1]))
	full := readAll(t, dst, last)
	replaceFile(t, dst, last, full[:len(full)-10]) // el registro de fin queda roto
	if _, err := open(dst, testDir, cfg); !errors.Is(err, ErrBackupIncomplete) {
		t.Fatalf("open with the WAL cut before the end: %v", err)
	}
	if got := readAll(t, dst, last); len(got) != len(full)-10 {
		t.Fatalf("the failed open changed the WAL: %d bytes, want %d", len(got), len(full)-10)
	}
	replaceFile(t, dst, last, full)
	r := restore(t, dst, cfg)
	checkContents(t, r, want)
	mustOK(t, r.Close())
	t.Logf("backup %+v", info)

	// Una copia que no terminó: el marcador sigue «en curso».
	half := crashfs.New()
	e.afterBackupStep = func(step string) {
		if step == "end" {
			half.SetCapacity(1) // el destino se llena
		}
	}
	if _, err := e.backup(half, testDir); !errors.Is(err, ErrDiskFull) {
		t.Fatalf("backup to a full disk: %v", err)
	}
	e.afterBackupStep = nil
	half.SetCapacity(0)
	if _, err := open(half, testDir, cfg); !errors.Is(err, ErrBackupIncomplete) {
		t.Fatalf("open of an unfinished backup: %v", err)
	}
}

/**
* readAll lee un archivo entero del disco de prueba.
* @param t testing.TB, fsys vfs.FS, path string
* @return []byte
**/
func readAll(t testing.TB, fsys vfs.FS, path string) []byte {
	t.Helper()
	f, err := fsys.OpenFile(path, os.O_RDONLY)
	mustOK(t, err)
	defer f.Close()
	size, err := f.Size()
	mustOK(t, err)
	b := make([]byte, size)
	_, err = f.ReadAt(b, 0)
	mustOK(t, err)
	return b
}

/**
* replaceFile deja en path exactamente data (corta lo que sobre), con fsync.
* @param t testing.TB, fsys vfs.FS, path string, data []byte
**/
func replaceFile(t testing.TB, fsys vfs.FS, path string, data []byte) {
	t.Helper()
	f, err := fsys.OpenFile(path, os.O_RDWR)
	mustOK(t, err)
	mustOK(t, f.Truncate(int64(len(data))))
	_, err = f.WriteAt(data, 0)
	mustOK(t, err)
	mustOK(t, f.Sync())
	mustOK(t, f.Close())
}

// TestBackupDestFull: el destino se llena a mitad de la copia: ErrDiskFull, el original sigue sano (Verify limpio,
// escribe), no queda nada retenido y el próximo checkpoint recicla y trunca como siempre; otro respaldo funciona.
func TestBackupDestFull(t *testing.T) {
	base := crashfs.New()
	cfg := testConfig()
	e := openTest(t, base, cfg)
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b"}, 300)
	dst := crashfs.New()
	dst.SetCapacity(200 << 10)
	if _, err := e.backup(dst, testDir); !errors.Is(err, ErrDiskFull) || errors.Is(err, ErrFailed) {
		t.Fatalf("backup to a small disk: %v", err)
	}
	checkReleased(t, e)
	checkClean(t, e)
	rng := rand.New(rand.NewSource(5))
	for range 100 {
		mustOK(t, writeOne(stores, 0, rng))
	}
	mustOK(t, e.Checkpoint())
	if segs := walSegments(t, base); len(segs) > 1+cfg.wal.Spares+1 {
		t.Fatalf("WAL segments %v after the failed backup", segs)
	}

	ok := crashfs.New()
	_, err := e.backup(ok, testDir)
	mustOK(t, err)
	want := dumpEngine(t, e)
	r := restore(t, ok, cfg)
	defer r.Close()
	checkContents(t, r, want)
}

// TestBackupCrash: el motor original se corta en medio de un respaldo (tras un DropStore y un checkpoint durante la
// copia, en los tres modos de crashfs). El respaldo devuelve error; el original se recupera con todo lo confirmado,
// sin nada retenido; lo único que puede quedar son los archivos del store borrado que esperaban el fin de la copia
// (huérfanos que Repair borra).
func TestBackupCrash(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		t.Run(name, func(t *testing.T) {
			base := crashfs.New()
			proc := newProc(base)
			cfg := testConfig()
			e := openTest(t, proc, cfg)
			fillStores(t, e, []string{"a", "gone"}, 200)
			s := mustStore(t, e, "a")
			n := 0
			e.afterBackupStep = func(step string) {
				if !strings.HasPrefix(step, "data/") {
					return
				}
				if n++; n == 1 {
					mustOK(t, e.DropStore("gone"))
					mustOK(t, s.Put("during", []byte("backup")))
					mustOK(t, e.Checkpoint())
					mustOK(t, s.Put("late", []byte("value")))
					return
				}
				proc.kill(mode, int64(n))
			}
			if _, err := e.backup(crashfs.New(), testDir); err == nil {
				t.Fatal("backup survived the crash")
			}
			e.Close()

			r := openTest(t, newProc(base), cfg)
			defer r.Close()
			if slices.Contains(r.Stores(), "gone") {
				t.Fatal("the dropped store came back")
			}
			a := mustStore(t, r, "a")
			for id, v := range map[string]string{"during": "backup", "late": "value", "k0001": "a/1"} {
				if got, _ := readString(t, a.Read, id); got != v {
					t.Fatalf("a/%s = %q, want %q", id, got, v)
				}
			}
			checkReleased(t, r)
			report, err := r.Verify()
			mustOK(t, err)
			for _, f := range report.Findings {
				if f.Kind != FindOrphanFile {
					t.Fatalf("Verify after the crash:\n%s", report)
				}
			}
			if !report.Clean() {
				_, err := r.Repair()
				mustOK(t, err)
			}
			checkNoOrphans(t, r, base)
		})
	}
}

// TestBackupClose: Close en medio de un respaldo lo interrumpe (ErrClosed) sin colgarse; el motor reabre limpio.
func TestBackupClose(t *testing.T) {
	base := crashfs.New()
	cfg := testConfig()
	e := openTest(t, base, cfg)
	fillStores(t, e, []string{"a"}, 100)
	closed := make(chan error, 1)
	e.afterBackupStep = func(step string) {
		if step == "start" {
			go func() { closed <- e.Close() }()
			<-e.stop // Close ya espera al respaldo
		}
	}
	if _, err := e.backup(crashfs.New(), testDir); !errors.Is(err, ErrClosed) {
		t.Fatalf("backup during Close: %v", err)
	}
	mustOK(t, <-closed)
	r := openTest(t, base, cfg)
	defer r.Close()
	checkClean(t, r)
}

// TestBackupOS: Backup público sobre el disco real (vfs.OS) y la restauración con open.
func TestBackupOS(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "copia")
	cfg := testConfig()
	e, err := open(vfs.OS(), src, cfg)
	mustOK(t, err)
	defer e.Close()
	fillStores(t, e, []string{"a", "b"}, 200)
	info, err := e.Backup(dst)
	mustOK(t, err)
	want := dumpEngine(t, e)
	mustOK(t, mustStore(t, e, "a").Put("after", []byte("not in the copy")))

	r, err := open(vfs.OS(), dst, cfg)
	mustOK(t, err)
	defer r.Close()
	checkClean(t, r)
	checkContents(t, r, want)
	if _, err := os.Stat(filepath.Join(dst, backupLabelName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup_label after the restore: %v (info %+v)", err, info)
	}
}
