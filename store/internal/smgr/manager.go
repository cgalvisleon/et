package smgr

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"math"
	"path/filepath"
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// DefaultMaxFiles es el tope de descriptores abiertos de un gestor cuando Open recibe 0 (a confirmar: PostgreSQL usa
// max_files_per_process = 1000 por proceso; aquí es por motor).
const DefaultMaxFiles = 256

// forkKey identifica un fork de un store.
type forkKey struct {
	file FileNo
	fork Fork
}

// forkState es lo que el gestor sabe de un fork: si existe y cuántos bloques tiene. Se lee de los archivos la primera
// vez que se usa el fork y después lo mantiene el gestor (un motor por directorio: nadie más escribe sus archivos).
//
// mu protege el fork: Read, Write, Exists y NBlocks lo toman compartido, y Sync solo para tomar los descriptores (el
// fsync va con el fork suelto); Create, Extend y Unlink, exclusivo.
type forkState struct {
	mu      sync.RWMutex
	loaded  bool
	exists  bool
	dropped bool // Unlink lo sacó del mapa: quien lo tenga debe buscar el estado de nuevo
	nblocks page.BlockNumber
}

// manager implementa Manager sobre un vfs.FS.
//
// Orden de bloqueos: forkState.mu → manager.mu, forkState.mu → fileCache.mu. manager.mu y fileCache.mu son hojas.
type manager struct {
	fs            vfs.FS
	dataDir       string
	blocksPerFile page.BlockNumber // BlocksPerFile; las pruebas usan menos para cambiar de archivo sin escribir 1 GB
	files         *fileCache

	mu    sync.Mutex
	forks map[forkKey]*forkState
	dirty map[segKey]struct{} // archivos escritos desde su último fsync
	// dirDirty: Create o Unlink cambiaron data/ desde su último fsync; lo hace SyncAll (el checkpoint), no cada
	// alta o baja: hasta el checkpoint el WAL rehace el archivo que falte o el borrado (tarea 98 de F-23)
	dirDirty bool
	failed   error // primer fsync fallido: desde ahí todo devuelve este error
	closed   bool
}

/**
* Open crea el gestor de los archivos de datos del motor en dir (crea dir/data si falta).
* @param fs vfs.FS, dir string (directorio del motor), maxFiles int (tope de descriptores abiertos; 0 = DefaultMaxFiles)
* @return Manager, error
**/
func Open(fs vfs.FS, dir string, maxFiles int) (Manager, error) {
	return open(fs, dir, maxFiles, BlocksPerFile)
}

// open es Open con el tamaño de los archivos en bloques como parámetro (las pruebas lo achican).
func open(fs vfs.FS, dir string, maxFiles int, blocksPerFile page.BlockNumber) (*manager, error) {
	if maxFiles <= 0 {
		maxFiles = DefaultMaxFiles
	}
	dataDir := filepath.Join(dir, DataDir)
	if err := fs.MkdirAll(dataDir); err != nil {
		return nil, fmt.Errorf("smgr: crear %s: %w", dataDir, err)
	}

	m := &manager{
		fs:            fs,
		dataDir:       dataDir,
		blocksPerFile: blocksPerFile,
		forks:         map[forkKey]*forkState{},
		dirty:         map[segKey]struct{}{},
	}
	m.files = newFileCache(fs, maxFiles, m.evict)
	return m, nil
}

/**
* Create crea el primer archivo vacío del fork (idempotente); el fsync de data/ lo hace SyncAll.
* @param file FileNo, fork Fork
* @return error
**/
func (m *manager) Create(file FileNo, fork Fork) error {
	st, err := m.lockFork(file, fork, true)
	if err != nil {
		return err
	}
	defer st.mu.Unlock()

	if st.exists {
		return nil
	}
	key := segKey{file: file, fork: fork, seg: 1}
	h, err := m.files.acquire(key, m.segName(key), true)
	if err != nil {
		return fmt.Errorf("smgr: crear %s: %w", m.segName(key), err)
	}
	m.files.release(h)
	m.markDirDirty()
	st.exists = true
	st.nblocks = 0
	return nil
}

/**
* Exists dice si el fork tiene al menos un archivo.
* @param file FileNo, fork Fork
* @return bool, error
**/
func (m *manager) Exists(file FileNo, fork Fork) (bool, error) {
	st, err := m.lockFork(file, fork, false)
	if err != nil {
		return false, err
	}
	defer st.mu.RUnlock()
	return st.exists, nil
}

/**
* NBlocks es la cantidad de bloques del fork.
* @param file FileNo, fork Fork
* @return page.BlockNumber, error (ErrNoFork si no existe)
**/
func (m *manager) NBlocks(file FileNo, fork Fork) (page.BlockNumber, error) {
	st, err := m.lockFork(file, fork, false)
	if err != nil {
		return 0, err
	}
	defer st.mu.RUnlock()
	if !st.exists {
		return 0, m.forkErr(ErrNoFork, file, fork)
	}
	return st.nblocks, nil
}

/**
* Extend escribe buf como el bloque block (≥ NBlocks; los intermedios quedan en ceros). Los archivos anteriores al del
* bloque se completan a su largo entero y, si nace un archivo, hace fsync de data/.
* @param file FileNo, fork Fork, block page.BlockNumber, buf page.Page
* @return error
**/
func (m *manager) Extend(file FileNo, fork Fork, block page.BlockNumber, buf page.Page) error {
	if err := m.checkBlock(block, buf); err != nil {
		return err
	}
	st, err := m.lockFork(file, fork, true)
	if err != nil {
		return err
	}
	defer st.mu.Unlock()

	if !st.exists {
		return m.forkErr(ErrNoFork, file, fork)
	}
	if block < st.nblocks {
		return m.blockErr(ErrBlockExists, file, fork, block)
	}

	// Los archivos entre el último y el del bloque nuevo quedan llenos (en ceros): todos menos el último miden 1 GB.
	lastSeg := m.lastSeg(st.nblocks)
	target := m.segOf(block)
	fullSize := int64(m.blocksPerFile) * page.Size
	for seg := m.segOf(st.nblocks); seg < target; seg++ {
		key := segKey{file: file, fork: fork, seg: seg}
		if err := m.withFile(key, true, func(f vfs.File) error { return f.Truncate(fullSize) }); err != nil {
			return err
		}
		m.markDirty(key)
	}

	// Antes de que nazca un archivo nuevo, los anteriores tienen que estar llenos en disco: si el nombre del nuevo
	// sobrevive a un corte y la extensión del anterior no, el fork queda con un archivo corto en el medio y no abre.
	// La marca de sucio queda: el próximo Sync del fork los vuelve a sincronizar sin costo.
	if target > lastSeg {
		var prev []segKey
		for seg := lastSeg; seg < target; seg++ {
			prev = append(prev, segKey{file: file, fork: fork, seg: seg})
		}
		if err := m.syncSegs(prev); err != nil {
			return err
		}
	}

	key := segKey{file: file, fork: fork, seg: target}
	if err := m.withFile(key, true, func(f vfs.File) error { return m.writeBlock(f, block, buf) }); err != nil {
		return err
	}
	m.markDirty(key)
	if target > lastSeg {
		if err := m.syncDataDir(); err != nil {
			return err
		}
	}
	st.nblocks = block + 1
	return nil
}

/**
* Read lee el bloque en buf. Un bloque en ceros se devuelve en ceros (página nueva).
* @param file FileNo, fork Fork, block page.BlockNumber, buf page.Page
* @return error (ErrBlockOutOfRange más allá del final; ErrShortFile si el archivo no tiene el bloque)
**/
func (m *manager) Read(file FileNo, fork Fork, block page.BlockNumber, buf page.Page) error {
	if err := m.checkBlock(block, buf); err != nil {
		return err
	}
	st, err := m.lockExisting(file, fork, block)
	if err != nil {
		return err
	}
	defer st.mu.RUnlock()

	key := segKey{file: file, fork: fork, seg: m.segOf(block)}
	return m.withFile(key, false, func(f vfs.File) error {
		n, err := f.ReadAt(buf, m.offsetOf(block))
		if n < page.Size {
			return fmt.Errorf("%w: %s bloque %d: se leyeron %d de %d bytes", ErrShortFile, m.segName(key), block, n, page.Size)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		return nil
	})
}

/**
* Write escribe buf en un bloque existente. No es durable hasta Sync.
* @param file FileNo, fork Fork, block page.BlockNumber, buf page.Page
* @return error (ErrBlockOutOfRange si el bloque no existe)
**/
func (m *manager) Write(file FileNo, fork Fork, block page.BlockNumber, buf page.Page) error {
	if err := m.checkBlock(block, buf); err != nil {
		return err
	}
	st, err := m.lockExisting(file, fork, block)
	if err != nil {
		return err
	}
	defer st.mu.RUnlock()

	key := segKey{file: file, fork: fork, seg: m.segOf(block)}
	if err := m.withFile(key, false, func(f vfs.File) error { return m.writeBlock(f, block, buf) }); err != nil {
		return err
	}
	m.markDirty(key)
	return nil
}

/**
* Sync hace fsync de los archivos del fork escritos desde su último fsync. Toma los descriptores con el fork en
* compartido y hace los fsync con el fork suelto: Extend (que lo pide en exclusivo, a veces con páginas del árbol
* bloqueadas) y las lecturas y escrituras que esperarían detrás no esperan un fsync. Un Unlink en medio no cierra
* esos descriptores hasta que se sueltan (fileCache.drop), y el fsync de un archivo ya borrado no hace daño.
* @param file FileNo, fork Fork
* @return error
**/
func (m *manager) Sync(file FileNo, fork Fork) error {
	st, err := m.lockFork(file, fork, false)
	if err != nil {
		return err
	}
	keys := m.takeDirty(func(k segKey) bool { return k.file == file && k.fork == fork })
	handles, err := m.acquireSegs(keys)
	st.mu.RUnlock()
	if err != nil {
		return err
	}
	return m.syncHandles(handles)
}

/**
* SyncAll hace fsync de todos los archivos escritos desde su último fsync (al terminar un checkpoint).
* @return error
**/
func (m *manager) SyncAll() error {
	if err := m.check(); err != nil {
		return err
	}
	m.mu.Lock()
	forks := map[forkKey]bool{}
	for k := range m.dirty {
		forks[forkKey{file: k.file, fork: k.fork}] = true
	}
	m.mu.Unlock()

	for fk := range forks {
		if err := m.Sync(fk.file, fk.fork); err != nil {
			return err
		}
	}
	return m.SyncDir()
}

/**
* SyncDir hace fsync de data/ si Create o Unlink lo cambiaron desde el último. Un fsync fallido deja el gestor
* inutilizable (no se reintenta).
* @return error
**/
func (m *manager) SyncDir() error {
	if err := m.check(); err != nil {
		return err
	}
	m.mu.Lock()
	dirDirty := m.dirDirty
	m.dirDirty = false
	m.mu.Unlock()
	if dirDirty {
		return m.syncDataDir()
	}
	return nil
}

// markDirDirty anota que data/ cambió (un archivo nació o se borró) y espera el fsync de SyncAll.
func (m *manager) markDirDirty() {
	m.mu.Lock()
	m.dirDirty = true
	m.mu.Unlock()
}

/**
* Unlink borra todos los archivos de todos los forks del store; el fsync de data/ lo hace SyncAll. Idempotente.
* @param file FileNo
* @return error
**/
func (m *manager) Unlink(file FileNo) error {
	removed := false
	for _, fork := range []Fork{ForkMain, ForkVM} {
		did, err := m.unlinkFork(file, fork)
		if err != nil {
			return err
		}
		removed = removed || did
	}
	if removed {
		m.markDirDirty()
	}
	return nil
}

/**
* Close cierra los descriptores abiertos, sin fsync (el checkpoint final ya hizo SyncAll). Nadie debe estar usando
* el gestor al cerrarlo.
* @return error
**/
func (m *manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.mu.Unlock()
	return m.files.closeAll()
}

// unlinkFork borra los archivos de un fork, del último al primero (un corte a mitad deja un prefijo que se puede
// volver a borrar), y olvida su estado. Dice si borró algo.
func (m *manager) unlinkFork(file FileNo, fork Fork) (bool, error) {
	if err := m.check(); err != nil {
		return false, err
	}
	if err := checkFork(fork); err != nil {
		return false, err
	}
	key := forkKey{file: file, fork: fork}
	st := m.state(key)
	st.mu.Lock()
	defer st.mu.Unlock()

	count, err := m.countSegs(file, fork)
	if err != nil {
		return false, err
	}
	m.files.drop(file, fork)
	for seg := count; seg >= 1; seg-- {
		name := m.segName(segKey{file: file, fork: fork, seg: seg})
		if err := m.fs.Remove(name); err != nil && !errors.Is(err, iofs.ErrNotExist) {
			return false, fmt.Errorf("smgr: borrar %s: %w", name, err)
		}
	}

	m.mu.Lock()
	for k := range m.dirty {
		if k.file == file && k.fork == fork {
			delete(m.dirty, k)
		}
	}
	if m.forks[key] == st {
		delete(m.forks, key)
	}
	m.mu.Unlock()
	st.dropped = true
	return count > 0, nil
}

// countSegs cuenta los archivos seguidos del fork desde el 1. Con el fork bloqueado en exclusivo.
func (m *manager) countSegs(file FileNo, fork Fork) (uint32, error) {
	var seg uint32
	for {
		key := segKey{file: file, fork: fork, seg: seg + 1}
		_, found, err := m.segSize(key)
		if err != nil {
			return 0, err
		}
		if !found {
			return seg, nil
		}
		seg++
	}
}

// lockFork devuelve el estado del fork, leído de los archivos si hace falta y bloqueado (exclusivo o compartido).
// Quien llama lo suelta.
func (m *manager) lockFork(file FileNo, fork Fork, exclusive bool) (*forkState, error) {
	if err := checkFork(fork); err != nil {
		return nil, err
	}
	key := forkKey{file: file, fork: fork}
	for {
		if err := m.check(); err != nil {
			return nil, err
		}
		st := m.state(key)
		if !exclusive {
			st.mu.RLock()
			if st.loaded && !st.dropped {
				return st, nil
			}
			st.mu.RUnlock()
		}

		st.mu.Lock()
		if st.dropped {
			st.mu.Unlock()
			continue
		}
		if !st.loaded {
			if err := m.load(file, fork, st); err != nil {
				st.mu.Unlock()
				return nil, err
			}
		}
		if exclusive {
			return st, nil
		}
		st.mu.Unlock()
	}
}

// lockExisting bloquea en compartido un fork que existe y tiene el bloque.
func (m *manager) lockExisting(file FileNo, fork Fork, block page.BlockNumber) (*forkState, error) {
	st, err := m.lockFork(file, fork, false)
	if err != nil {
		return nil, err
	}
	if !st.exists {
		st.mu.RUnlock()
		return nil, m.forkErr(ErrNoFork, file, fork)
	}
	if block >= st.nblocks {
		st.mu.RUnlock()
		return nil, m.blockErr(ErrBlockOutOfRange, file, fork, block)
	}
	return st, nil
}

// state devuelve el estado del fork, creándolo vacío (sin leer) si no está.
func (m *manager) state(key forkKey) *forkState {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.forks[key]
	if st == nil {
		st = &forkState{}
		m.forks[key] = st
	}
	return st
}

// load lee de los archivos si el fork existe y cuántos bloques tiene. Todos los archivos menos el último deben medir
// exactamente blocksPerFile bloques; un archivo corto seguido de otro es ErrShortFile. Un bloque escrito a medias al
// final del último no cuenta (a confirmar: PostgreSQL hace lo mismo).
func (m *manager) load(file FileNo, fork Fork, st *forkState) error {
	fullSize := int64(m.blocksPerFile) * page.Size
	var seg uint32 = 1
	for {
		key := segKey{file: file, fork: fork, seg: seg}
		size, found, err := m.segSize(key)
		if err != nil {
			return err
		}
		if !found {
			break
		}
		st.exists = true
		if size > fullSize {
			return fmt.Errorf("%w: %s mide %d bytes", ErrLongFile, m.segName(key), size)
		}
		if size < fullSize {
			// Es el último: el siguiente no puede existir.
			next := segKey{file: file, fork: fork, seg: seg + 1}
			if _, nextFound, err := m.segSize(next); err != nil {
				return err
			} else if nextFound {
				return fmt.Errorf("%w: %s mide %d bytes y existe %s", ErrShortFile, m.segName(key), size, m.segName(next))
			}
			st.nblocks = page.BlockNumber(seg-1)*m.blocksPerFile + page.BlockNumber(size/page.Size)
			st.loaded = true
			return nil
		}
		seg++
	}
	st.nblocks = page.BlockNumber(seg-1) * m.blocksPerFile
	st.loaded = true
	return nil
}

// segSize devuelve el largo de un archivo y si existe.
func (m *manager) segSize(key segKey) (int64, bool, error) {
	h, err := m.files.acquire(key, m.segName(key), false)
	if errors.Is(err, iofs.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("smgr: abrir %s: %w", m.segName(key), err)
	}
	defer m.files.release(h)
	size, err := h.f.Size()
	if err != nil {
		return 0, false, fmt.Errorf("smgr: largo de %s: %w", m.segName(key), err)
	}
	return size, true, nil
}

// withFile toma el descriptor del archivo (creándolo si create), corre fn con él y lo suelta.
func (m *manager) withFile(key segKey, create bool, fn func(f vfs.File) error) error {
	h, err := m.files.acquire(key, m.segName(key), create)
	if errors.Is(err, iofs.ErrNotExist) {
		return fmt.Errorf("%w: falta %s", ErrShortFile, m.segName(key))
	}
	if err != nil {
		return fmt.Errorf("smgr: abrir %s: %w", m.segName(key), err)
	}
	defer m.files.release(h)
	if err := fn(h.f); err != nil {
		return fmt.Errorf("smgr: %s: %w", m.segName(key), err)
	}
	return nil
}

// writeBlock escribe una página en su lugar del archivo. Quien llama marca el archivo sucio después de escribir:
// así un Sync que quite la marca antes ve la escritura con su fsync o la vuelve a encontrar marcada.
func (m *manager) writeBlock(f vfs.File, block page.BlockNumber, buf page.Page) error {
	n, err := f.WriteAt(buf, m.offsetOf(block))
	if err != nil {
		return err
	}
	if n < page.Size {
		return io.ErrShortWrite
	}
	return nil
}

// markDirty anota que el archivo tiene escrituras sin fsync.
func (m *manager) markDirty(key segKey) {
	m.mu.Lock()
	m.dirty[key] = struct{}{}
	m.mu.Unlock()
}

// takeDirty quita y devuelve los archivos sucios que cumplen match. Se quitan antes del fsync: una escritura que
// llegue durante el fsync los vuelve a marcar.
func (m *manager) takeDirty(match func(segKey) bool) []segKey {
	m.mu.Lock()
	defer m.mu.Unlock()
	var keys []segKey
	for k := range m.dirty {
		if match(k) {
			keys = append(keys, k)
			delete(m.dirty, k)
		}
	}
	return keys
}

// syncSegs hace fsync de cada archivo. Un fsync fallido deja el gestor inutilizable (doc.go, «Fallos de escritura»):
// la marca de sucio ya se quitó y reintentar daría un éxito falso.
func (m *manager) syncSegs(keys []segKey) error {
	handles, err := m.acquireSegs(keys)
	if err != nil {
		return err
	}
	return m.syncHandles(handles)
}

// acquireSegs toma los descriptores de los archivos (quien llama los suelta con syncHandles). Si uno no abre, suelta
// los ya tomados y deja el gestor inutilizable: sus marcas de sucio ya se quitaron.
func (m *manager) acquireSegs(keys []segKey) ([]*handle, error) {
	handles := make([]*handle, 0, len(keys))
	for _, key := range keys {
		h, err := m.files.acquire(key, m.segName(key), false)
		if err != nil {
			for _, taken := range handles {
				m.files.release(taken)
			}
			return nil, m.fail(fmt.Errorf("smgr: abrir %s para fsync: %w", m.segName(key), err))
		}
		handles = append(handles, h)
	}
	return handles, nil
}

// syncHandles hace fsync de cada descriptor tomado y los suelta todos. Un fsync fallido deja el gestor inutilizable.
func (m *manager) syncHandles(handles []*handle) error {
	var err error
	for _, h := range handles {
		if err == nil {
			if serr := h.f.Sync(); serr != nil {
				err = m.fail(fmt.Errorf("smgr: fsync de %s: %w", m.segName(h.key), serr))
			}
		}
		m.files.release(h)
	}
	return err
}

// evict cierra un descriptor que la caché expulsó. Si el archivo tiene escrituras sin fsync, antes las lleva al
// dispositivo por ese mismo descriptor (tras cerrarlo, un error de escritura del sistema podría perderse) con
// vfs.FlushFile: el fsync simple basta para eso, y F_FULLFSYNC en macOS costaba 4 ms por cada store nuevo pasados
// los DefaultMaxFiles (tarea 98 de F-23). No quita la marca de sucio: el próximo Sync lo sincroniza de verdad, así
// un Sync que corre a la vez no vuelve antes de tiempo.
func (m *manager) evict(h *handle) {
	m.mu.Lock()
	_, dirty := m.dirty[h.key]
	m.mu.Unlock()
	if dirty {
		if err := vfs.FlushFile(h.f); err != nil {
			m.fail(fmt.Errorf("smgr: fsync de %s al cerrarlo: %w", m.segName(h.key), err))
		}
	}
	h.f.Close()
}

// syncDataDir hace fsync de data/ (vuelve durables las altas y bajas de archivos).
func (m *manager) syncDataDir() error {
	if err := m.fs.SyncDir(m.dataDir); err != nil {
		return m.fail(fmt.Errorf("smgr: fsync de %s: %w", m.dataDir, err))
	}
	return nil
}

// fail deja el gestor inutilizable con err (se queda el primero) y lo devuelve.
func (m *manager) fail(err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failed == nil {
		m.failed = err
	}
	return m.failed
}

// check devuelve ErrClosed o el fsync fallido, si los hay.
func (m *manager) check() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	return m.failed
}

// checkBlock valida el número de bloque y el largo del buffer.
func (m *manager) checkBlock(block page.BlockNumber, buf page.Page) error {
	if len(buf) != page.Size {
		return fmt.Errorf("%w: mide %d", ErrBadPage, len(buf))
	}
	if block == math.MaxUint32 {
		return fmt.Errorf("%w: %d es el número inválido", ErrBlockOutOfRange, block)
	}
	return nil
}

// checkFork valida el número de fork.
func checkFork(fork Fork) error {
	if fork != ForkMain && fork != ForkVM {
		return fmt.Errorf("%w: %d", ErrBadFork, fork)
	}
	return nil
}

// segOf es el archivo (desde 1) que guarda el bloque.
func (m *manager) segOf(block page.BlockNumber) uint32 {
	return uint32(block/m.blocksPerFile) + 1
}

// lastSeg es el último archivo de un fork con nblocks bloques (el 1 si está vacío).
func (m *manager) lastSeg(nblocks page.BlockNumber) uint32 {
	if nblocks == 0 {
		return 1
	}
	return m.segOf(nblocks - 1)
}

// offsetOf es la posición del bloque dentro de su archivo.
func (m *manager) offsetOf(block page.BlockNumber) int64 {
	return int64(block%m.blocksPerFile) * page.Size
}

// segName es la ruta del archivo: data/<fileNo:%010d>.<n:%04d>, con _vm para el mapa de visibilidad.
func (m *manager) segName(key segKey) string {
	suffix := ""
	if key.fork == ForkVM {
		suffix = "_vm"
	}
	return filepath.Join(m.dataDir, fmt.Sprintf("%010d%s.%04d", key.file, suffix, key.seg))
}

// forkErr envuelve err con el fork.
func (m *manager) forkErr(err error, file FileNo, fork Fork) error {
	return fmt.Errorf("%w: store %d fork %d", err, file, fork)
}

// blockErr envuelve err con el fork y el bloque.
func (m *manager) blockErr(err error, file FileNo, fork Fork, block page.BlockNumber) error {
	return fmt.Errorf("%w: store %d fork %d bloque %d", err, file, fork, block)
}

// manager cumple Manager.
var _ Manager = (*manager)(nil)
