package wal

import (
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Las pruebas usan vfs.OS() como disco real y crashfs para las fallas, los cortes y el fuzzer (en memoria, sin fsync
// real). Lo único que queda de un adaptador propio es syncCountFS: crashfs no cuenta los fsync.

// syncCountFS envuelve un vfs.FS y cuenta los Sync de archivos (para el group commit y para ver que un fsync fallido
// no se reintenta), los Truncate y los bytes escritos (lo que ahorra Options.CleanEnd, tarea 109).
type syncCountFS struct {
	vfs.FS
	syncs     atomic.Int64
	truncates atomic.Int64
	written   atomic.Int64
}

type syncCountFile struct {
	vfs.File
	fs *syncCountFS
}

func (c *syncCountFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := c.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return syncCountFile{f, c}, nil
}

func (f syncCountFile) Sync() error {
	f.fs.syncs.Add(1)
	return f.File.Sync()
}

func (f syncCountFile) Truncate(size int64) error {
	f.fs.truncates.Add(1)
	return f.File.Truncate(size)
}

func (f syncCountFile) WriteAt(p []byte, off int64) (int, error) {
	n, err := f.File.WriteAt(p, off)
	f.fs.written.Add(int64(n))
	return n, err
}

// reset pone los contadores en cero.
func (c *syncCountFS) reset() {
	c.syncs.Store(0)
	c.truncates.Store(0)
	c.written.Store(0)
}
