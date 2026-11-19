package smgr

import (
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// countFS envuelve un vfs.FS (vfs.OS() en estas pruebas) y cuenta descriptores abiertos, fsync por archivo y fsync
// del directorio. Es lo único que queda del adaptador propio: crashfs no lleva esas cuentas. Las fallas de fsync y
// los cortes los pone crashfs.
type countFS struct {
	vfs.FS
	open      atomic.Int64 // descriptores abiertos ahora
	maxOpen   atomic.Int64 // el mayor número de abiertos a la vez
	dirSyncs  atomic.Int64 // fsync de directorios
	mu        sync.Mutex
	syncNames map[string]int // fsync por archivo
}

// newCountFS cuenta sobre el disco real.
func newCountFS() *countFS { return &countFS{FS: vfs.OS(), syncNames: map[string]int{}} }

type countFile struct {
	vfs.File
	fs     *countFS
	name   string
	closed atomic.Bool
}

func (f *countFile) Sync() error {
	f.fs.mu.Lock()
	f.fs.syncNames[f.name]++
	f.fs.mu.Unlock()
	return f.File.Sync()
}

func (f *countFile) Close() error {
	if f.closed.CompareAndSwap(false, true) {
		f.fs.open.Add(-1)
	}
	return f.File.Close()
}

func (fs *countFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := fs.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	n := fs.open.Add(1)
	for {
		max := fs.maxOpen.Load()
		if n <= max || fs.maxOpen.CompareAndSwap(max, n) {
			break
		}
	}
	return &countFile{File: f, fs: fs, name: name}, nil
}

func (fs *countFS) SyncDir(dir string) error {
	fs.dirSyncs.Add(1)
	return fs.FS.SyncDir(dir)
}

// syncCount dice cuántos fsync recibió un archivo.
func (fs *countFS) syncCount(name string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.syncNames[name]
}
