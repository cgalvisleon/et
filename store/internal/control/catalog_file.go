package control

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// catalogFile implementa CatalogFile sobre un vfs.FS.
type catalogFile struct {
	fs      vfs.FS
	dir     string
	path    string
	tmpPath string

	mu     sync.Mutex // serializa Load y Save: los dos usan catalog.tmp
	failed error      // primera escritura o fsync fallido: desde entonces Save falla
}

// NewCatalogFile devuelve el catálogo del motor en dir.
// fs: el sistema de archivos; dir: el directorio del motor. Devuelve el CatalogFile (no toca el disco).
func NewCatalogFile(fs vfs.FS, dir string) CatalogFile {
	return &catalogFile{
		fs:      fs,
		dir:     dir,
		path:    filepath.Join(dir, CatalogFileName),
		tmpPath: filepath.Join(dir, CatalogTempName),
	}
}

// Load lee y valida el catálogo. Un catalog.tmp sobrante (un Save cortado antes del rename) se borra sin leerlo.
func (f *catalogFile) Load() (Catalog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.removeTemp(); err != nil {
		return Catalog{}, err
	}
	b, err := f.readAll()
	if err != nil {
		return Catalog{}, err
	}
	c, err := decodeCatalog(b)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: %w", f.path, err)
	}
	return c, nil
}

// Read lee y valida el catálogo sin borrar un catalog.tmp sobrante (Verify del motor no cambia nada).
func (f *catalogFile) Read() (Catalog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, err := f.readAll()
	if err != nil {
		return Catalog{}, err
	}
	c, err := decodeCatalog(b)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: %w", f.path, err)
	}
	return c, nil
}

// removeTemp borra catalog.tmp si existe. Se busca en el directorio para no depender del error de cada vfs.
func (f *catalogFile) removeTemp() error {
	names, err := f.fs.ReadDir(f.dir)
	if err != nil {
		return fmt.Errorf("control: read dir %s: %w", f.dir, err)
	}
	for _, name := range names {
		if name == CatalogTempName {
			if err := f.fs.Remove(f.tmpPath); err != nil {
				return fmt.Errorf("control: remove %s: %w", f.tmpPath, err)
			}
		}
	}
	return nil
}

// readAll lee el catálogo entero.
func (f *catalogFile) readAll() ([]byte, error) {
	file, err := f.fs.OpenFile(f.path, os.O_RDONLY)
	if err != nil {
		return nil, fmt.Errorf("control: open %s: %w", f.path, err)
	}
	defer file.Close()
	size, err := file.Size()
	if err != nil {
		return nil, fmt.Errorf("control: size of %s: %w", f.path, err)
	}
	if size < 0 || size > maxCatalogSize {
		return nil, fmt.Errorf("%w: %s has %d bytes", ErrBadCatalog, f.path, size)
	}
	b := make([]byte, size)
	if n, err := file.ReadAt(b, 0); n != len(b) {
		if err == nil || errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("control: read %s: %w", f.path, err)
	}
	return b, nil
}

// maxCatalogSize acota lo que Load reserva: el catálogo más grande posible (2^32 − 1 entradas con nombres de
// MaxNameLen) no llega a 1,2 TB, pero uno real con millones de stores ronda los cientos de MB. A confirmar.
const maxCatalogSize = 1 << 30

// Save escribe c en catalog.tmp, hace Sync, lo renombra a catalog y hace SyncDir: un corte deja el catálogo anterior
// o el nuevo, nunca una mezcla. Tras un fallo de escritura o de Sync no reintenta: devuelve ErrFailed hasta reabrir.
// El disco lleno al crear o escribir catalog.tmp es la excepción: el catálogo anterior sigue entero, así que devuelve
// el error y se puede volver a guardar.
func (f *catalogFile) Save(c Catalog) error {
	b, err := encodeCatalog(c)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return fmt.Errorf("%w: %v", ErrFailed, f.failed)
	}
	if err := f.writeTemp(b); err != nil {
		if errors.Is(err, errTempNoSpace) {
			return err
		}
		return f.fail(err)
	}
	if err := f.fs.Rename(f.tmpPath, f.path); err != nil {
		return f.fail(fmt.Errorf("control: rename %s: %w", f.tmpPath, err))
	}
	if err := f.fs.SyncDir(f.dir); err != nil {
		return f.fail(fmt.Errorf("control: sync dir %s: %w", f.dir, err))
	}
	return nil
}

// errTempNoSpace marca el disco lleno al crear o escribir catalog.tmp (antes de su Sync): Save no queda fallado.
var errTempNoSpace = errors.New("control: no space for the catalog")

// writeTemp crea catalog.tmp con b (vacía lo que hubiera), hace Sync y lo cierra.
func (f *catalogFile) writeTemp(b []byte) error {
	file, err := f.fs.OpenFile(f.tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return tempError(fmt.Errorf("control: create %s: %w", f.tmpPath, err))
	}
	if err := file.Truncate(int64(len(b))); err != nil {
		file.Close()
		return tempError(fmt.Errorf("control: truncate %s: %w", f.tmpPath, err))
	}
	if _, err := file.WriteAt(b, 0); err != nil {
		file.Close()
		return tempError(fmt.Errorf("control: write %s: %w", f.tmpPath, err))
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("control: sync %s: %w", f.tmpPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("control: close %s: %w", f.tmpPath, err)
	}
	return nil
}

// tempError marca con errTempNoSpace un error de disco lleno al escribir catalog.tmp.
func tempError(err error) error {
	if vfs.IsNoSpace(err) {
		return fmt.Errorf("%w: %w", errTempNoSpace, err)
	}
	return err
}

// fail anota err como el primer fallo y lo devuelve.
func (f *catalogFile) fail(err error) error {
	if f.failed == nil {
		f.failed = err
	}
	return err
}
