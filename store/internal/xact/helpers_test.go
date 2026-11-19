package xact

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// fakeFlusher es un WAL doble: Durable avanza con WaitDurable (o falla si err no es nil).
type fakeFlusher struct {
	durable atomic.Uint64
	waits   atomic.Int64
	err     error
}

func (f *fakeFlusher) Durable() page.LSN { return page.LSN(f.durable.Load()) }

func (f *fakeFlusher) WaitDurable(lsn page.LSN) error {
	f.waits.Add(1)
	if f.err != nil {
		return f.err
	}
	for {
		cur := f.durable.Load()
		if cur >= uint64(lsn) || f.durable.CompareAndSwap(cur, uint64(lsn)) {
			return nil
		}
	}
}

// watchFS envuelve el disco real y anota, por cada WriteAt, cuán durable era el WAL en ese momento, y cuenta los
// Sync. Es lo único que queda de un adaptador propio: crashfs no puede mirar el WAL en cada escritura ni cuenta Sync.
type watchFS struct {
	vfs.FS
	flusher *fakeFlusher

	mu     sync.Mutex
	writes []page.LSN // Durable() al momento de cada escritura
	syncs  int
}

// newWatchFS envuelve vfs.OS() mirando el WAL fl.
func newWatchFS(fl *fakeFlusher) *watchFS { return &watchFS{FS: vfs.OS(), flusher: fl} }

func (w *watchFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := w.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return &watchFile{File: f, fs: w}, nil
}

type watchFile struct {
	vfs.File
	fs *watchFS
}

func (f *watchFile) WriteAt(p []byte, off int64) (int, error) {
	f.fs.mu.Lock()
	f.fs.writes = append(f.fs.writes, f.fs.flusher.Durable())
	f.fs.mu.Unlock()
	return f.File.WriteAt(p, off)
}

func (f *watchFile) Sync() error {
	f.fs.mu.Lock()
	f.fs.syncs++
	f.fs.mu.Unlock()
	return f.File.Sync()
}

// openTest abre un archivo de estados en dir con cachePages páginas.
func openTest(t *testing.T, fs vfs.FS, dir string, fl *fakeFlusher, cachePages int) *Statuses {
	t.Helper()
	s, err := Open(Config{FS: fs, Dir: dir, Flusher: fl, CachePages: cachePages})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

// wantStatus comprueba el estado de xid.
func wantStatus(t *testing.T, s *Statuses, xid page.XID, want Status) {
	t.Helper()
	got, _, err := s.Status(xid)
	if err != nil {
		t.Fatalf("Status(%d): %v", xid, err)
	}
	if got != want {
		t.Fatalf("Status(%d) = %d, want %d", xid, got, want)
	}
}

// fileExists dice si existe el archivo de estados n en el motor dir.
func fileExists(dir string, n uint64) bool {
	_, err := os.Stat(fileName(filepath.Join(dir, Dir), n))
	return err == nil
}
