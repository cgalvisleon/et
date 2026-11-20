package crashfs

import (
	"errors"
	"io"
	"io/fs"
)

// errAccess es el error de leer un archivo abierto solo para escribir, o al revés.
var errAccess = errors.New("bad file descriptor")

// file es un archivo abierto de memFS.
type file struct {
	fs       *memFS
	node     *inode
	name     string
	epoch    int
	closed   bool
	canRead  bool
	canWrite bool
}

// check dice si el archivo se puede usar; se llama con el mutex de fs tomado.
func (f *file) check(op string) error {
	switch {
	case f.closed:
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	case f.epoch != f.fs.epoch:
		return &fs.PathError{Op: op, Path: f.name, Err: ErrCrashed}
	}
	return nil
}

// ReadAt lee desde off; como os.File, devuelve io.EOF si no llena p.
func (f *file) ReadAt(p []byte, off int64) (int, error) {
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if err := f.check("read"); err != nil {
		return 0, err
	}
	if !f.canRead || off < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: errAccess}
	}
	if off >= int64(len(f.node.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.node.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// WriteAt escribe p en off (extendiendo con ceros si off pasa del final); no es durable hasta Sync.
func (f *file) WriteAt(p []byte, off int64) (int, error) {
	delay(&f.fs.writeDelay)
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if err := f.writable("write", off); err != nil {
		return 0, err
	}
	if err := f.fs.checkSpace(off + int64(len(p)) - int64(len(f.node.data))); err != nil {
		return 0, &fs.PathError{Op: "write", Path: f.name, Err: err}
	}
	f.node.change(fileOp{off: off, data: append([]byte(nil), p...)})
	return len(p), nil
}

// Truncate fija el largo del archivo; no es durable hasta Sync.
func (f *file) Truncate(size int64) error {
	delay(&f.fs.writeDelay)
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if err := f.writable("truncate", size); err != nil {
		return err
	}
	if err := f.fs.checkSpace(size - int64(len(f.node.data))); err != nil {
		return &fs.PathError{Op: "truncate", Path: f.name, Err: err}
	}
	f.node.change(fileOp{off: size, truncate: true})
	return nil
}

// writable revisa que se pueda escribir en el archivo en off y cuenta la escritura para las fallas inyectadas.
func (f *file) writable(op string, off int64) error {
	if err := f.check(op); err != nil {
		return err
	}
	if !f.canWrite || off < 0 {
		return &fs.PathError{Op: op, Path: f.name, Err: errAccess}
	}
	if err := f.fs.inject(false); err != nil {
		return &fs.PathError{Op: op, Path: f.name, Err: err}
	}
	return nil
}

// Sync vuelve durable lo escrito. Si falla, olvida lo pendiente sin volverlo durable (ver doc.go); con el disco por
// encima de su capacidad, un Sync con escrituras pendientes falla con ErrNoSpace.
func (f *file) Sync() error {
	delay(&f.fs.syncDelay)
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if err := f.check("sync"); err != nil {
		return err
	}
	err := f.fs.inject(true)
	if err == nil && len(f.node.pending) > 0 && f.fs.overCapacity() {
		err = ErrNoSpace
	}
	if err != nil {
		f.node.dropPending()
		return &fs.PathError{Op: "sync", Path: f.name, Err: err}
	}
	f.node.sync()
	return nil
}

// Size devuelve el largo actual del archivo.
func (f *file) Size() (int64, error) {
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if err := f.check("stat"); err != nil {
		return 0, err
	}
	return int64(len(f.node.data)), nil
}

// Close cierra el archivo; cerrar uno abierto antes de un corte no falla.
func (f *file) Close() error {
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if f.closed {
		return &fs.PathError{Op: "close", Path: f.name, Err: fs.ErrClosed}
	}
	f.closed = true
	return nil
}
