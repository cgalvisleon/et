package smgr

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// errEvictSync es el fsync fallido del descriptor que se expulsa.
var errEvictSync = errors.New("test: fsync of the evicted descriptor failed")

// evictFailFS hace que el primer fsync del archivo match (el de la expulsión de su descriptor sucio) espere release
// y falle; los fsync de los descriptores que se abran después funcionan, como en Linux (errseq), donde un descriptor
// nuevo puede no ver un error de escritura que ya se informó.
type evictFailFS struct {
	vfs.FS
	match   string
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

type evictFailFile struct {
	vfs.File
	fs    *evictFailFS
	first bool
}

func (f *evictFailFile) Sync() error {
	if f.first {
		failed := false
		f.fs.once.Do(func() {
			close(f.fs.started)
			<-f.fs.release
			failed = true
		})
		if failed {
			return errEvictSync
		}
	}
	return f.File.Sync()
}

func (fs *evictFailFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := fs.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	first := false
	if strings.HasSuffix(name, fs.match) && fs.match != "" {
		first, fs.match = true, "" // solo el primer descriptor del archivo
	}
	return &evictFailFile{File: f, fs: fs, first: first}, nil
}

// TestSyncAllSeesEvictFailure (S2 de la revisión del 2026-11-20): la expulsión de un descriptor sucio hace fsync en
// la goroutine que lo expulsa; si ese fsync está en curso mientras corre SyncAll y falla, SyncAll no devuelve nil
// (el checkpoint avanzaría el punto de rehacer con páginas perdidas): espera a las expulsiones en curso y termina
// mirando el fallo del gestor.
func TestSyncAllSeesEvictFailure(t *testing.T) {
	fs := &evictFailFS{FS: vfs.OS(), started: make(chan struct{}), release: make(chan struct{})}
	m, err := open(fs, t.TempDir(), 1, testBlocksPerFile)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for f := FileNo(1); f <= 2; f++ {
		if err := m.Create(f, ForkMain); err != nil {
			t.Fatal(err)
		}
	}
	fs.match = "0000000001.0001"                                // el próximo descriptor del archivo 1 (Create ya cerró el suyo al abrir el 2)
	if err := m.Extend(1, ForkMain, 0, pageOf(1)); err != nil { // archivo 1 sucio, en la caché
		t.Fatal(err)
	}
	evicting := make(chan error, 1)
	go func() { evicting <- m.Extend(2, ForkMain, 0, pageOf(2)) }() // expulsa al 1: su fsync se traba
	select {
	case <-fs.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the eviction of the dirty file 1 did not fsync")
	}

	synced := make(chan error, 1)
	go func() { synced <- m.SyncAll() }()
	select {
	case err := <-synced:
		close(fs.release)
		t.Fatalf("SyncAll returned %v while the eviction fsync of a dirty file was still running", err)
	case <-time.After(200 * time.Millisecond):
	}
	close(fs.release)
	if err := <-synced; !errors.Is(err, errEvictSync) {
		t.Fatalf("SyncAll after the eviction fsync failed: %v, want the failure", err)
	}
	<-evicting
}
