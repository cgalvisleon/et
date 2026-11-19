package wal

import (
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Las pruebas usan vfs.OS() como disco real y crashfs para las fallas, los cortes y el fuzzer (en memoria, sin fsync
// real). Lo único que queda de un adaptador propio es syncCountFS: crashfs no cuenta los fsync.

// syncCountFS envuelve un vfs.FS y cuenta los Sync de archivos (para el group commit y para ver que un fsync fallido
// no se reintenta).
type syncCountFS struct {
	vfs.FS
	syncs atomic.Int64
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
