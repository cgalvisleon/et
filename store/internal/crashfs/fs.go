package crashfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// errIsDir es el error de usar un directorio como archivo.
var errIsDir = errors.New("is a directory")

// New crea un disco vacío, sin directorios: el primer MkdirAll crea la ruta desde la raíz.
func New() FS {
	return &memFS{dirs: map[string]*directory{}, locks: map[string]bool{}, failAfter: -1, failSyncAfter: -1}
}

// memFS es el disco simulado. Un solo mutex protege todo: es para pruebas.
type memFS struct {
	mu            sync.Mutex
	dirs          map[string]*directory // por ruta limpia
	locks         map[string]bool       // archivos con Lock tomado
	epoch         int                   // sube en cada corte: invalida archivos y bloqueos anteriores
	failAfter     int                   // escrituras o Sync que faltan para fallar; < 0 sin falla
	failSyncAfter int                   // Sync que faltan para fallar; < 0 sin falla
	capacity      int64                 // bytes del disco; <= 0 sin límite

	writeDelay, syncDelay atomic.Int64 // demoras de SetDelay, en nanosegundos (se leen sin el mutex)
}

// FailAfter programa la falla de E/S (ver FS.FailAfter).
func (m *memFS) FailAfter(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failAfter = n
}

// FailSyncAfter programa la falla de E/S de Sync (ver FS.FailSyncAfter).
func (m *memFS) FailSyncAfter(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failSyncAfter = n
}

// SetCapacity fija el tamaño del disco (ver FS.SetCapacity).
func (m *memFS) SetCapacity(n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capacity = n
}

// Used devuelve lo que ocupa el disco (ver FS.Used).
func (m *memFS) Used() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.used()
}

// SetDelay fija las demoras de escritura y de Sync (ver FS.SetDelay).
func (m *memFS) SetDelay(write, sync time.Duration) {
	m.writeDelay.Store(int64(write))
	m.syncDelay.Store(int64(sync))
}

// delay duerme la demora d (en nanosegundos) fuera del mutex: un disco lento no frena a los demás archivos.
func delay(d *atomic.Int64) {
	if n := d.Load(); n > 0 {
		time.Sleep(time.Duration(n))
	}
}

// overCapacity dice si el disco usa más que su capacidad (se la bajó por debajo de lo usado): los Sync con
// escrituras pendientes fallan con ErrNoSpace.
func (m *memFS) overCapacity() bool {
	return m.capacity > 0 && m.used() > m.capacity
}

// inject cuenta una escritura (o un Sync si isSync) y dice si debe fallar con ErrIO.
func (m *memFS) inject(isSync bool) error {
	failed := countdown(&m.failAfter)
	if isSync && countdown(&m.failSyncAfter) {
		failed = true
	}
	if failed {
		return ErrIO
	}
	return nil
}

// countdown descuenta una operación de *left y dice si ya no quedan (< 0 es sin falla; en 0 se queda).
func countdown(left *int) bool {
	switch {
	case *left < 0:
		return false
	case *left == 0:
		return true
	default:
		*left--
		return false
	}
}

// checkSpace dice si crecer grow bytes cabe en el disco.
func (m *memFS) checkSpace(grow int64) error {
	if m.capacity <= 0 || grow <= 0 {
		return nil
	}
	if m.used()+grow > m.capacity {
		return ErrNoSpace
	}
	return nil
}

// used suma el lugar asignado de los archivos con nombre (un nodo con dos nombres cuenta una vez): lo escrito, sin
// los huecos.
func (m *memFS) used() int64 {
	seen := map[*inode]bool{}
	var total int64
	for _, d := range m.dirs {
		for _, node := range d.live {
			if !seen[node] {
				seen[node] = true
				total += node.alloc.size()
			}
		}
	}
	return total
}

// lookup devuelve el directorio padre de name y el nombre base; falla si el padre no existe.
func (m *memFS) lookup(op, name string) (*directory, string, error) {
	d := m.dirs[filepath.Dir(name)]
	if d == nil {
		return nil, "", &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
	}
	return d, filepath.Base(name), nil
}

// OpenFile abre o crea name según las marcas de os.OpenFile (O_RDONLY, O_WRONLY, O_RDWR, O_CREATE, O_EXCL, O_TRUNC;
// las demás se ignoran). Crear el archivo es un cambio del directorio, durable tras SyncDir.
func (m *memFS) OpenFile(name string, flag int) (vfs.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.openFile(filepath.Clean(name), flag)
}

// openFile es OpenFile con el mutex tomado y name limpio.
func (m *memFS) openFile(name string, flag int) (*file, error) {
	if m.dirs[name] != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errIsDir}
	}
	d, base, err := m.lookup("open", name)
	if err != nil {
		return nil, err
	}
	node := d.live[base]
	switch {
	case node == nil && flag&os.O_CREATE == 0:
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	case node == nil:
		if m.capacity > 0 && m.used() >= m.capacity {
			return nil, &fs.PathError{Op: "open", Path: name, Err: ErrNoSpace}
		}
		node = &inode{}
		d.change(entryChange{name: base, node: node})
	case flag&os.O_CREATE != 0 && flag&os.O_EXCL != 0:
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrExist}
	}

	access := flag & (os.O_RDONLY | os.O_WRONLY | os.O_RDWR)
	f := &file{fs: m, node: node, name: name, epoch: m.epoch,
		canRead: access != os.O_WRONLY, canWrite: access != os.O_RDONLY}
	if flag&os.O_TRUNC != 0 && f.canWrite && len(node.data) > 0 {
		node.change(fileOp{truncate: true})
	}
	return f, nil
}

// Remove borra el archivo name (un cambio del directorio, durable tras SyncDir).
func (m *memFS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	name = filepath.Clean(name)
	if m.dirs[name] != nil {
		return &fs.PathError{Op: "remove", Path: name, Err: errIsDir}
	}
	d, base, err := m.lookup("remove", name)
	if err != nil {
		return err
	}
	if d.live[base] == nil {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}
	d.change(entryChange{name: base})
	return nil
}

// Rename cambia oldName por newName, reemplazando el destino. En un mismo directorio es un cambio atómico; entre dos,
// un cambio en cada uno.
func (m *memFS) Rename(oldName, newName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldName, newName = filepath.Clean(oldName), filepath.Clean(newName)
	if m.dirs[oldName] != nil || m.dirs[newName] != nil {
		return &os.LinkError{Op: "rename", Old: oldName, New: newName, Err: errIsDir}
	}
	from, oldBase, err := m.lookup("rename", oldName)
	if err != nil {
		return err
	}
	to, newBase, err := m.lookup("rename", newName)
	if err != nil {
		return err
	}
	node := from.live[oldBase]
	if node == nil {
		return &os.LinkError{Op: "rename", Old: oldName, New: newName, Err: fs.ErrNotExist}
	}
	switch {
	case oldName == newName:
	case from == to:
		from.change(entryChange{name: newBase, node: node}, entryChange{name: oldBase})
	default:
		to.change(entryChange{name: newBase, node: node})
		from.change(entryChange{name: oldBase})
	}
	return nil
}

// MkdirAll crea dir y los padres que falten; cada uno es durable enseguida, con el fsync de su padre.
func (m *memFS) MkdirAll(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var missing []string
	for path := filepath.Clean(dir); m.dirs[path] == nil; path = filepath.Dir(path) {
		if parent := m.dirs[filepath.Dir(path)]; parent != nil && parent.live[filepath.Base(path)] != nil {
			return &fs.PathError{Op: "mkdir", Path: path, Err: fs.ErrExist}
		}
		missing = append([]string{path}, missing...)
		if filepath.Dir(path) == path {
			break
		}
	}
	for _, path := range missing {
		m.dirs[path] = newDirectory()
		if parent := m.dirs[filepath.Dir(path)]; parent != nil && filepath.Dir(path) != path {
			parent.sync()
		}
	}
	return nil
}

// SyncDir vuelve durables los cambios de las entradas de dir.
func (m *memFS) SyncDir(dir string) error {
	delay(&m.syncDelay)
	m.mu.Lock()
	defer m.mu.Unlock()
	dir = filepath.Clean(dir)
	d := m.dirs[dir]
	if d == nil {
		return &fs.PathError{Op: "fsync", Path: dir, Err: fs.ErrNotExist}
	}
	if err := m.inject(true); err != nil {
		return &fs.PathError{Op: "fsync", Path: dir, Err: err}
	}
	if len(d.pending) > 0 && m.overCapacity() {
		return &fs.PathError{Op: "fsync", Path: dir, Err: ErrNoSpace}
	}
	d.sync()
	return nil
}

// ReadDir lista los archivos y subdirectorios de dir en orden alfabético.
func (m *memFS) ReadDir(dir string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dir = filepath.Clean(dir)
	d := m.dirs[dir]
	if d == nil {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: fs.ErrNotExist}
	}
	names := slices.Collect(maps.Keys(d.live))
	for path := range m.dirs {
		if path != dir && filepath.Dir(path) == dir {
			names = append(names, filepath.Base(path))
		}
	}
	slices.Sort(names)
	return names, nil
}

// Lock toma el bloqueo exclusivo de name, creándolo si falta; un corte lo suelta.
func (m *memFS) Lock(name string) (io.Closer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name = filepath.Clean(name)
	if m.locks[name] {
		return nil, fmt.Errorf("%w: %s", vfs.ErrLocked, name)
	}
	f, err := m.openFile(name, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	m.locks[name] = true
	return &lock{fs: m, name: name, epoch: f.epoch}, nil
}

// lock es un bloqueo tomado con Lock.
type lock struct {
	fs       *memFS
	name     string
	epoch    int
	released bool
}

// Close suelta el bloqueo; si hubo un corte desde que se tomó, ya estaba suelto.
func (l *lock) Close() error {
	l.fs.mu.Lock()
	defer l.fs.mu.Unlock()
	if !l.released && l.epoch == l.fs.epoch {
		delete(l.fs.locks, l.name)
	}
	l.released = true
	return nil
}
