package xact

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Estados en los bordes de byte, página y archivo; sobreviven a Flush y a reabrir.
func TestStatusEdges(t *testing.T) {
	dir := t.TempDir()
	fl := &fakeFlusher{}
	s := openTest(t, vfs.OS(), dir, fl, 4)

	xids := []page.XID{1, 2, 3, 4, 5, XidsPerPage - 1, XidsPerPage, XidsPerPage + 1, XidsPerFile - 1, XidsPerFile, XidsPerFile + 3}
	want := map[page.XID]Status{}
	for i, xid := range xids {
		st := StatusCommitted
		if i%2 == 1 {
			st = StatusAborted
		}
		if err := s.SetStatus(xid, st, page.LSN(i+1)); err != nil {
			t.Fatalf("SetStatus(%d): %v", xid, err)
		}
		want[xid] = st
	}
	wantStatus(t, s, 6, StatusInProgress) // vecino sin anotar
	for xid, st := range want {
		wantStatus(t, s, xid, st)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if !fileExists(dir, 1) || !fileExists(dir, 2) {
		t.Fatal("want files 0001 and 0002")
	}
	info, _ := os.Stat(fileName(filepath.Join(dir, Dir), 1))
	if info.Size() != PagesPerFile*PageSize {
		t.Fatalf("file 0001 size = %d, want %d", info.Size(), PagesPerFile*PageSize)
	}

	s = openTest(t, vfs.OS(), dir, fl, 4)
	defer s.Close()
	for xid, st := range want {
		wantStatus(t, s, xid, st)
		if _, lsn, _ := s.Status(xid); lsn != 0 {
			t.Fatalf("group lsn after reopen = %d, want 0", lsn)
		}
	}
}

// Reglas de SetStatus: idempotente con el mismo estado, error al cambiar de estado final y con estados inválidos.
func TestSetStatusRules(t *testing.T) {
	s := openTest(t, vfs.OS(), t.TempDir(), &fakeFlusher{}, 2)
	defer s.Close()
	if err := s.SetStatus(7, StatusCommitted, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.SetStatus(7, StatusCommitted, 10); err != nil {
		t.Fatalf("repeat: %v", err)
	}
	if err := s.SetStatus(7, StatusAborted, 11); !errors.Is(err, ErrStatusConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if err := s.SetStatus(8, StatusInProgress, 1); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("in progress: %v", err)
	}
	if err := s.SetStatus(0, StatusCommitted, 1); !errors.Is(err, ErrInvalidXID) {
		t.Fatalf("xid 0: %v", err)
	}
	if _, _, err := s.Status(0); !errors.Is(err, ErrInvalidXID) {
		t.Fatalf("Status(0): %v", err)
	}
}

// El lsn de grupo lo comparten 32 xids y decide si se puede anotar la marca.
func TestGroupLSNAndHint(t *testing.T) {
	fl := &fakeFlusher{}
	s := openTest(t, vfs.OS(), t.TempDir(), fl, 2)
	defer s.Close()

	s.SetStatus(33, StatusCommitted, 100) // grupo 1 (xids 32..63)
	s.SetStatus(40, StatusCommitted, 50)
	if _, lsn, _ := s.Status(40); lsn != 100 {
		t.Fatalf("group lsn = %d, want 100", lsn)
	}
	if _, lsn, _ := s.Status(64); lsn != 0 {
		t.Fatalf("other group lsn = %d, want 0", lsn)
	}
	if ok, _ := s.CanHint(40); ok {
		t.Fatal("hint allowed before WAL durable")
	}
	fl.durable.Store(100)
	if ok, _ := s.CanHint(40); !ok {
		t.Fatal("hint not allowed with WAL durable")
	}
	if ok, _ := s.CanHint(41); ok {
		t.Fatal("hint allowed for an in-progress xid")
	}
}

// «WAL antes que la página»: ni el checkpoint ni la expulsión escriben una página antes de que el WAL sea durable
// hasta su lsn, y una página sin cambios no se escribe.
func TestWALBeforePage(t *testing.T) {
	fl := &fakeFlusher{}
	fs := newWatchFS(fl)
	s := openTest(t, fs, t.TempDir(), fl, 2)
	defer s.Close()

	// Cinco páginas con caché de dos: tres expulsiones de páginas sucias.
	for p := page.XID(0); p < 5; p++ {
		lsn := page.LSN(1000 + p)
		if err := s.SetStatus(p*XidsPerPage+1, StatusCommitted, lsn); err != nil {
			t.Fatal(err)
		}
	}
	fs.mu.Lock()
	evicted := len(fs.writes)
	for i, durable := range fs.writes {
		if durable < page.LSN(1000+i) {
			t.Fatalf("eviction %d written with WAL durable %d < %d", i, durable, 1000+i)
		}
	}
	fs.mu.Unlock()
	if evicted != 3 {
		t.Fatalf("evictions = %d, want 3", evicted)
	}

	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	fs.mu.Lock()
	if len(fs.writes) != 5 || fs.syncs == 0 {
		t.Fatalf("writes = %d, syncs = %d; want 5 and > 0", len(fs.writes), fs.syncs)
	}
	for _, durable := range fs.writes {
		if durable < 1000 {
			t.Fatalf("page written with WAL durable %d", durable)
		}
	}
	fs.mu.Unlock()

	// Leer páginas expulsadas no las ensucia: un segundo Flush no escribe nada.
	for p := page.XID(0); p < 5; p++ {
		wantStatus(t, s, p*XidsPerPage+1, StatusCommitted)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.writes) != 5 {
		t.Fatalf("clean pages rewritten: writes = %d", len(fs.writes))
	}
}

// Si el WAL no llega a durable, la página no se escribe y el archivo de estados queda inutilizable.
func TestWALFailure(t *testing.T) {
	boom := errors.New("wal fsync failed")
	fl := &fakeFlusher{err: boom}
	fs := newWatchFS(fl)
	s := openTest(t, fs, t.TempDir(), fl, 2)
	defer s.Close()

	s.SetStatus(1, StatusCommitted, 5)
	if err := s.Flush(); !errors.Is(err, boom) {
		t.Fatalf("Flush: %v", err)
	}
	if len(fs.writes) != 0 {
		t.Fatal("page written although the WAL failed")
	}
	if _, _, err := s.Status(1); !errors.Is(err, boom) {
		t.Fatalf("Status after failure: %v", err)
	}
}

// Muchos goroutines confirman, abortan y consultan sobre muchas páginas con una caché chica.
func TestConcurrentStatuses(t *testing.T) {
	dir := t.TempDir()
	fl := &fakeFlusher{}
	s := openTest(t, vfs.OS(), dir, fl, 3)

	const workers, perWorker = 12, 300
	statusOf := func(w, k int) Status {
		if (w+k)%3 == 0 {
			return StatusAborted
		}
		return StatusCommitted
	}
	xidOf := func(w, k int) page.XID { return page.XID(w)*XidsPerPage + page.XID(k*7+1) }

	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range perWorker {
				xid := xidOf(w, k)
				if err := s.SetStatus(xid, statusOf(w, k), page.LSN(w*perWorker+k+1)); err != nil {
					t.Error(err)
					return
				}
				// Consulta lo propio y lo de un vecino (otra página, quizá expulsada).
				if st, _, err := s.Status(xid); err != nil || st != statusOf(w, k) {
					t.Errorf("Status(%d) = %d, %v", xid, st, err)
					return
				}
				if _, _, err := s.Status(xidOf((w+1)%workers, k)); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() { // checkpoints a la vez
		defer wg.Done()
		for range 5 {
			if err := s.Flush(); err != nil {
				t.Error(err)
			}
		}
	}()
	wg.Wait()
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s = openTest(t, vfs.OS(), dir, fl, 3)
	defer s.Close()
	for w := range workers {
		for k := range perWorker {
			wantStatus(t, s, xidOf(w, k), statusOf(w, k))
		}
	}
}

// Truncate borra los archivos enteros por debajo del horizonte, también al reabrir.
func TestTruncate(t *testing.T) {
	dir := t.TempDir()
	fl := &fakeFlusher{}
	s := openTest(t, vfs.OS(), dir, fl, 2)
	s.SetStatus(5, StatusCommitted, 1)
	s.SetStatus(XidsPerFile+1, StatusCommitted, 2)
	s.SetStatus(2*XidsPerFile+1, StatusAborted, 3)
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}

	if err := s.Truncate(XidsPerFile - 1); err != nil { // no cubre el archivo 1 entero
		t.Fatal(err)
	}
	if !fileExists(dir, 1) {
		t.Fatal("file 0001 removed too early")
	}
	if err := s.Truncate(2*XidsPerFile + 10); err != nil {
		t.Fatal(err)
	}
	if fileExists(dir, 1) || fileExists(dir, 2) || !fileExists(dir, 3) {
		t.Fatal("want only file 0003")
	}
	if st, lsn, err := s.Status(5); err != nil || st != StatusCommitted || lsn != 0 {
		t.Fatalf("Status of truncated xid = %d, %d, %v; want committed", st, lsn, err)
	}
	if s.Oldest() != 2*XidsPerFile {
		t.Fatalf("Oldest = %d, want %d", s.Oldest(), 2*XidsPerFile)
	}
	if err := s.SetStatus(XidsPerFile+2, StatusCommitted, 4); !errors.Is(err, ErrTruncated) {
		t.Fatalf("SetStatus of truncated xid: %v", err)
	}
	wantStatus(t, s, 2*XidsPerFile+1, StatusAborted)
	s.Close()

	s = openTest(t, vfs.OS(), dir, fl, 2)
	defer s.Close()
	wantStatus(t, s, XidsPerFile+1, StatusCommitted) // truncado: se lee confirmado sin leer el disco
	wantStatus(t, s, 2*XidsPerFile+1, StatusAborted)
}

// Con archivos de prueba de una página: el xid va al archivo de su página, Truncate borra por debajo del límite sin
// volver a listar el directorio si ya lo hizo, y Truncate sobre un directorio sin archivos fija el límite igual (el
// motor lo llama al abrir con el frozenXID del control).
func TestTruncateSmallFiles(t *testing.T) {
	dir := t.TempDir()
	fl := &fakeFlusher{}
	open := func() *Statuses {
		s, err := Open(Config{FS: vfs.OS(), Dir: dir, Flusher: fl, CachePages: 2, FilePages: 1})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := open()
	for f := page.XID(0); f < 4; f++ {
		if err := s.SetStatus(f*XidsPerPage+7, StatusAborted, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	for n := uint64(1); n <= 4; n++ {
		if !fileExists(dir, n) {
			t.Fatalf("file %04d missing", n)
		}
	}
	if err := s.Truncate(3*XidsPerPage - 1); err != nil {
		t.Fatal(err)
	}
	if fileExists(dir, 1) || fileExists(dir, 2) || !fileExists(dir, 3) {
		t.Fatal("want files 0003 and 0004 only")
	}
	wantStatus(t, s, XidsPerPage+7, StatusCommitted) // estaba abortada, pero ya se truncó
	wantStatus(t, s, 2*XidsPerPage+7, StatusAborted)
	if err := s.Truncate(3*XidsPerPage - 1); err != nil { // nada nuevo que borrar
		t.Fatal(err)
	}
	s.Close()

	if err := os.Remove(fileName(filepath.Join(dir, Dir), 3)); err != nil {
		t.Fatal(err)
	}
	s = open()
	defer s.Close()
	if s.Oldest() != 3*XidsPerPage {
		t.Fatalf("Oldest after reopen = %d, want %d", s.Oldest(), 3*XidsPerPage)
	}
	if err := s.Truncate(3 * XidsPerPage); err != nil {
		t.Fatal(err)
	}
	if s.Oldest() != 3*XidsPerPage {
		t.Fatalf("Oldest = %d, want %d", s.Oldest(), 3*XidsPerPage)
	}
	wantStatus(t, s, 2*XidsPerPage+7, StatusCommitted)
	wantStatus(t, s, 3*XidsPerPage+7, StatusAborted)
}

// Recover: en curso por debajo del horizonte = abortada; desde el horizonte, todo vuelve a en curso.
func TestRecover(t *testing.T) {
	dir := t.TempDir()
	fl := &fakeFlusher{}
	s := openTest(t, vfs.OS(), dir, fl, 2)
	s.SetStatus(10, StatusCommitted, 1)
	s.SetStatus(100, StatusAborted, 0)             // abortada que la recuperación no conoce
	s.SetStatus(XidsPerPage+3, StatusAborted, 0)   // página siguiente
	s.SetStatus(XidsPerFile+3, StatusCommitted, 0) // archivo siguiente
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s = openTest(t, vfs.OS(), dir, fl, 2)
	if err := s.Recover(50); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, s, 10, StatusCommitted)
	wantStatus(t, s, 30, StatusAborted) // en curso al caerse
	wantStatus(t, s, 49, StatusAborted)
	wantStatus(t, s, 50, StatusInProgress)
	wantStatus(t, s, 100, StatusInProgress)
	wantStatus(t, s, XidsPerPage+3, StatusInProgress)
	if fileExists(dir, 2) {
		t.Fatal("file 0002 not removed")
	}
	if err := s.SetStatus(100, StatusCommitted, 7); err != nil { // el xid se reasigna
		t.Fatal(err)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s = openTest(t, vfs.OS(), dir, fl, 2)
	defer s.Close()
	s.Recover(200)
	wantStatus(t, s, 100, StatusCommitted)
	wantStatus(t, s, 50, StatusAborted)
	wantStatus(t, s, XidsPerPage+3, StatusInProgress)
}

// Lo que no llegó al Flush se pierde al cerrar; Close deja todo en ErrClosed.
func TestCloseWithoutFlush(t *testing.T) {
	dir := t.TempDir()
	fl := &fakeFlusher{}
	s := openTest(t, vfs.OS(), dir, fl, 2)
	s.SetStatus(3, StatusCommitted, 1)
	s.Close()
	if _, _, err := s.Status(3); !errors.Is(err, ErrClosed) {
		t.Fatalf("Status after Close: %v", err)
	}
	s = openTest(t, vfs.OS(), dir, fl, 2)
	defer s.Close()
	wantStatus(t, s, 3, StatusInProgress)
}

// TestDiskFull: con el disco lleno, escribir una página de estados falla sin dejar el archivo de estados inutilizable:
// la página sigue sucia en la caché, Status sigue (también de un archivo que no existe y no se puede crear), y al
// liberar espacio Flush la escribe y sobrevive a reabrir.
func TestDiskFull(t *testing.T) {
	fs := crashfs.New()
	fl := &fakeFlusher{}
	s, err := Open(Config{FS: fs, Dir: "/db", Flusher: fl, CachePages: 2, FilePages: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetStatus(XidsPerPage+1, StatusCommitted, 5); err != nil {
		t.Fatal(err)
	}
	fs.SetCapacity(fs.Used() + 1) // 0 sería sin límite
	if err := s.Flush(); !vfs.IsNoSpace(err) {
		t.Fatalf("Flush with the disk full: %v", err)
	}
	wantStatus(t, s, XidsPerPage+1, StatusCommitted)
	wantStatus(t, s, 3*XidsPerPage+1, StatusInProgress) // su archivo no existe

	fs.SetCapacity(0)
	if err := errors.Join(s.Flush(), s.Close()); err != nil {
		t.Fatal(err)
	}
	s, err = Open(Config{FS: fs, Dir: "/db", Flusher: fl, CachePages: 2, FilePages: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	wantStatus(t, s, XidsPerPage+1, StatusCommitted)
}
