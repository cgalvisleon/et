package smgr

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// stallFS traba el fsync de los archivos mientras stall está puesto: avisa en started y espera release.
type stallFS struct {
	vfs.FS
	stall   atomic.Bool
	started chan struct{}
	release chan struct{}
}

type stallFile struct {
	vfs.File
	fs *stallFS
}

func (f *stallFile) Sync() error {
	if f.fs.stall.CompareAndSwap(true, false) {
		close(f.fs.started)
		<-f.fs.release
	}
	return f.File.Sync()
}

func (fs *stallFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := fs.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return &stallFile{File: f, fs: fs}, nil
}

/**
* within corre fn y falla la prueba si no termina antes de limit.
* @param t *testing.T, what string, limit time.Duration, fn func() error
**/
func within(t *testing.T, what string, limit time.Duration, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	case <-time.After(limit):
		t.Fatalf("%s waited for the fsync of Sync", what)
	}
}

// TestSyncOutsideForkLock (tarea 93): mientras el fsync de Sync de un fork está trabado, Extend, Read y Write del
// mismo fork no lo esperan (Extend llega desde una partición del árbol con páginas bloqueadas). Un Unlink en medio no
// cierra el descriptor que el fsync usa.
func TestSyncOutsideForkLock(t *testing.T) {
	fs := &stallFS{FS: vfs.OS(), started: make(chan struct{}), release: make(chan struct{})}
	m, err := open(fs, t.TempDir(), 8, testBlocksPerFile)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := m.Extend(1, ForkMain, 0, pageOf(1)); err != nil {
		t.Fatal(err)
	}
	fs.stall.Store(true)
	synced := make(chan error, 1)
	go func() { synced <- m.Sync(1, ForkMain) }()
	<-fs.started

	within(t, "Extend", time.Second, func() error { return m.Extend(1, ForkMain, 1, pageOf(2)) })
	within(t, "Read", time.Second, func() error { return m.Read(1, ForkMain, 0, make([]byte, len(pageOf(0)))) })
	within(t, "Write", time.Second, func() error { return m.Write(1, ForkMain, 0, pageOf(3)) })
	within(t, "Unlink", time.Second, func() error { return m.Unlink(1) })
	close(fs.release)
	if err := <-synced; err != nil {
		t.Fatalf("Sync with the fork unlinked during its fsync: %v", err)
	}
	if err := m.check(); err != nil {
		t.Fatalf("the manager failed: %v", err)
	}
}
