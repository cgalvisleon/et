package smgr

import (
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/page"
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

// writtenFS envuelve un vfs.FS y anota qué páginas de cada archivo se escribieron con WriteAt. Un Truncate que
// agranda no anota nada: deja un hueco, que ext4 y XFS no reservan en el disco.
type writtenFS struct {
	vfs.FS
	mu      sync.Mutex
	written map[string]map[int64]bool // archivo → páginas escritas
}

func newWrittenFS() *writtenFS { return &writtenFS{FS: vfs.OS(), written: map[string]map[int64]bool{}} }

type writtenFile struct {
	vfs.File
	fs   *writtenFS
	name string
}

func (f *writtenFile) WriteAt(b []byte, off int64) (int, error) {
	n, err := f.File.WriteAt(b, off)
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	pages := f.fs.written[f.name]
	if pages == nil {
		pages = map[int64]bool{}
		f.fs.written[f.name] = pages
	}
	for p := off / page.Size; p*page.Size < off+int64(n); p++ {
		pages[p] = true
	}
	return n, err
}

func (fs *writtenFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := fs.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return &writtenFile{File: f, fs: fs, name: name}, nil
}

// holes devuelve las páginas de name, dentro de su largo, que nunca se escribieron.
func (fs *writtenFS) holes(name string, size int64) []int64 {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	var out []int64
	for p := int64(0); p*page.Size < size; p++ {
		if !fs.written[name][p] {
			out = append(out, p)
		}
	}
	return out
}
