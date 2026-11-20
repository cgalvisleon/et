package xact

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// DefaultCachePages es la cantidad de páginas de la caché del archivo de estados (1 MB) si Config no dice otra.
const DefaultCachePages = 128

// groupsPerPage es la cantidad de grupos de lsn de confirmación de una página.
const groupsPerPage = XidsPerPage / GroupSize

var (
	// ErrInvalidXID: el xid 0 no es una transacción.
	ErrInvalidXID = errors.New("xact: invalid xid")

	// ErrInvalidStatus: SetStatus solo acepta confirmada o abortada.
	ErrInvalidStatus = errors.New("xact: invalid status")

	// ErrStatusConflict: el xid ya tiene el otro estado final (confirmada contra abortada).
	ErrStatusConflict = errors.New("xact: xid already has another final status")

	// ErrTruncated: SetStatus de un xid cuyo estado ya se borró con Truncate (Status lo lee confirmado).
	ErrTruncated = errors.New("xact: status of xid was truncated")

	// ErrCorrupt: los bits del xid tienen el valor reservado.
	ErrCorrupt = errors.New("xact: corrupt status")

	// ErrClosed: el archivo de estados está cerrado.
	ErrClosed = errors.New("xact: status file closed")
)

// Config es lo que necesita Open.
type Config struct {
	FS         vfs.FS      // sistema de archivos
	Dir        string      // directorio del motor; los archivos van en Dir/xact
	Flusher    wal.Flusher // el WAL, para no escribir una página antes de que sea durable hasta su lsn
	CachePages int         // páginas de la caché; 0 = DefaultCachePages
	FilePages  int         // páginas por archivo; 0 = PagesPerFile (otro valor solo en pruebas: cambia el formato)
}

// slotState dice qué tiene una ranura de la caché.
type slotState uint8

const (
	slotEmpty   slotState = iota // libre
	slotReading                  // cargándose del disco: nadie la toca hasta que pase a slotValid
	slotValid                    // con la página pageNo
)

// slot es una ranura de la caché: una página del archivo de estados y el lsn de cada uno de sus grupos.
type slot struct {
	pageNo   uint64     // número de página global: xid / XidsPerPage
	state    slotState  // libre, cargándose o válida
	writing  bool       // se está escribiendo una copia al disco; no se puede reemplazar
	dirty    bool       // cambió desde la última escritura
	maxLSN   page.LSN   // el mayor lsn de sus grupos: el WAL tiene que ser durable hasta acá antes de escribirla
	lastUsed uint64     // para elegir la menos usada últimamente
	data     []byte     // PageSize bytes, 2 bits por xid
	groupLSN []page.LSN // lsn de confirmación por grupo de GroupSize xids (solo en memoria)
}

// Statuses es el archivo de estados (xact/%04d) con su caché de tamaño fijo, como pg_xact con su SLRU.
//
// Concurrencia: mu guarda las ranuras, oldest, horizon y failed, y es una hoja: con mu tomado no se toma otro lock
// ni se hace E/S (se suelta para leer, escribir o esperar al WAL; la ranura queda marcada slotReading o writing).
// filesMu guarda solo el mapa de archivos abiertos y tampoco se toma con otro lock.
type Statuses struct {
	fs           vfs.FS
	dir          string
	flusher      wal.Flusher
	pagesPerFile uint64 // PagesPerFile salvo en pruebas

	mu      sync.Mutex
	cond    *sync.Cond // avisa cuando una ranura termina de cargarse o de escribirse
	slots   []slot
	tick    uint64   // reloj de uso de las ranuras
	oldest  page.XID // el primer xid que todavía tiene estado (los anteriores se truncaron: se leen confirmados)
	horizon page.XID // horizonte de recuperación: en curso por debajo = abortada
	failed  error    // si no es nil, todo devuelve este error

	filesMu  sync.Mutex
	files    map[uint64]vfs.File // archivos abiertos por número (acotados por los archivos que hay)
	unsynced map[uint64]bool     // archivos escritos desde el último fsync
	dirDirty bool                // hubo archivos nuevos o cortados desde el último fsync del directorio
}

var _ StatusFile = (*Statuses)(nil)

// Open abre (o crea) el archivo de estados en cfg.Dir/xact con la caché vacía.
// @param cfg: sistema de archivos, directorio del motor, WAL y tamaño de la caché.
// @return el archivo de estados, o el error de crear o leer el directorio.
func Open(cfg Config) (*Statuses, error) {
	n := cfg.CachePages
	if n <= 0 {
		n = DefaultCachePages
	}
	dir := filepath.Join(cfg.Dir, Dir)
	if err := cfg.FS.MkdirAll(dir); err != nil {
		return nil, err
	}
	numbers, err := fileNumbers(cfg.FS, dir)
	if err != nil {
		return nil, err
	}

	s := &Statuses{
		fs:           cfg.FS,
		dir:          dir,
		flusher:      cfg.Flusher,
		pagesPerFile: PagesPerFile,
		slots:        make([]slot, n),
		files:        map[uint64]vfs.File{},
		unsynced:     map[uint64]bool{},
	}
	if cfg.FilePages > 0 {
		s.pagesPerFile = uint64(cfg.FilePages)
	}
	s.cond = sync.NewCond(&s.mu)
	for i := range s.slots {
		s.slots[i].data = make([]byte, PageSize)
		s.slots[i].groupLSN = make([]page.LSN, groupsPerPage)
	}
	if len(numbers) > 0 {
		s.oldest = page.XID((numbers[0] - 1) * s.xidsPerFile())
	}
	return s, nil
}

// Status devuelve el estado de xid y el lsn de confirmación de su grupo (0 si la página no está en la caché: ya es
// durable). Un xid en curso por debajo del horizonte de recuperación se lee como abortado. Un xid truncado se lee
// confirmado (y congelado) sin tocar el disco: el motor solo trunca por debajo de su frozenXID, y por debajo no
// queda ninguna versión abortada ni sin congelar (store/doc.go, «Congelado»).
func (s *Statuses) Status(xid page.XID) (Status, page.LSN, error) {
	if xid == 0 {
		return 0, 0, ErrInvalidXID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if xid < s.oldest {
		return StatusCommitted, 0, nil
	}
	i, err := s.getSlot(pageOf(xid))
	if err != nil {
		return 0, 0, err
	}
	sl := &s.slots[i]
	st := getBits(sl.data, xid)
	if st == StatusReserved {
		return 0, 0, ErrCorrupt
	}
	if st == StatusInProgress && xid < s.horizon {
		st = StatusAborted
	}
	return st, sl.groupLSN[groupOf(xid)], nil
}

// SetStatus anota el estado final de xid con el lsn de su registro de confirmación o aborto. Repetir el mismo
// estado no es error (la recuperación reproduce el WAL); cambiar de un estado final al otro sí.
func (s *Statuses) SetStatus(xid page.XID, status Status, lsn page.LSN) error {
	if xid == 0 {
		return ErrInvalidXID
	}
	if status != StatusCommitted && status != StatusAborted {
		return ErrInvalidStatus
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if xid < s.oldest {
		return ErrTruncated
	}
	i, err := s.getSlot(pageOf(xid))
	if err != nil {
		return err
	}
	sl := &s.slots[i]
	current := getBits(sl.data, xid)
	if current != StatusInProgress && current != status {
		return ErrStatusConflict
	}
	setBits(sl.data, xid, status)
	g := groupOf(xid)
	sl.groupLSN[g] = max(sl.groupLSN[g], lsn)
	sl.maxLSN = max(sl.maxLSN, lsn)
	sl.dirty = true
	return nil
}

// CanHint dice si ya se puede anotar la marca de confirmada de xid en una versión: está confirmada y el WAL es
// durable hasta el lsn de confirmación de su grupo (pregunta 49).
func (s *Statuses) CanHint(xid page.XID) (bool, error) {
	st, lsn, err := s.Status(xid)
	if err != nil {
		return false, err
	}
	return HintAllowed(st, lsn, s.flusher.Durable()), nil
}

// Flush escribe al disco las páginas cambiadas (cada una tras el WAL durable hasta su lsn) y hace fsync de los
// archivos escritos y del directorio. Lo llama el checkpoint.
func (s *Statuses) Flush() error {
	s.mu.Lock()
	for i := range s.slots {
		for s.slots[i].writing && s.failed == nil {
			s.cond.Wait()
		}
		if s.failed != nil {
			err := s.failed
			s.mu.Unlock()
			return err
		}
		if s.slots[i].state == slotValid && s.slots[i].dirty {
			if err := s.writeSlot(i); err != nil {
				s.mu.Unlock()
				return err
			}
		}
	}
	s.mu.Unlock()
	return s.syncFiles()
}

// Truncate borra los archivos con todos sus xids menores que before (los estados anteriores al horizonte de
// congelado). Después Status lee esos xids confirmados sin tocar el disco y SetStatus devuelve ErrTruncated. Borra
// de menor a mayor: un corte a mitad deja un prefijo borrado, y al abrir el primer archivo que queda marca el límite.
// Si ya no hay nada que borrar no lista el directorio.
func (s *Statuses) Truncate(before page.XID) error {
	last := uint64(before) / s.xidsPerFile() // se borran los archivos 1..last
	if last == 0 {
		return nil
	}
	limit := page.XID(last * s.xidsPerFile())
	s.mu.Lock()
	if s.failed != nil {
		err := s.failed
		s.mu.Unlock()
		return err
	}
	if s.oldest >= limit {
		s.mu.Unlock()
		return nil
	}
	s.oldest = limit
	s.discardPages(func(pageNo uint64) bool { return s.fileOf(pageNo) <= last })
	s.mu.Unlock()

	return s.removeFiles(func(n uint64) bool { return n <= last })
}

// Recover fija el horizonte de recuperación, después de reproducir el WAL: los xid menores que next en curso se
// leen como abortados, y los estados de next en adelante se borran (en su página y en las siguientes), por si una
// página llegó al disco con estados de xids que la recuperación vuelve a asignar (como TrimCLOG de PostgreSQL).
// @param next: el nextXID final de la recuperación.
func (s *Statuses) Recover(next page.XID) error {
	if next == 0 {
		return ErrInvalidXID
	}
	p := pageOf(next)
	s.mu.Lock()
	s.horizon = next
	s.discardPages(func(pageNo uint64) bool { return pageNo > p })
	if next < s.oldest {
		s.mu.Unlock()
		return nil
	}
	i, err := s.getSlot(p)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	clearFrom(s.slots[i].data, uint64(next)%XidsPerPage)
	s.slots[i].dirty = true
	s.mu.Unlock()

	if err := s.removeFiles(func(n uint64) bool { return n > s.fileOf(p) }); err != nil {
		return err
	}
	return s.cutFile(s.fileOf(p), s.offsetOf(p)+PageSize)
}

// Close cierra los archivos sin escribir nada: el motor hace antes su checkpoint final (Flush).
func (s *Statuses) Close() error {
	s.mu.Lock()
	if s.failed == nil {
		s.failed = ErrClosed
	}
	s.cond.Broadcast()
	s.mu.Unlock()

	s.filesMu.Lock()
	defer s.filesMu.Unlock()
	var first error
	for n, f := range s.files {
		if err := f.Close(); err != nil && first == nil {
			first = err
		}
		delete(s.files, n)
	}
	return first
}

// getSlot devuelve la ranura válida de la página pageNo, cargándola si no está. Se llama con mu tomado y vuelve con
// mu tomado, pero lo suelta mientras lee o escribe.
func (s *Statuses) getSlot(pageNo uint64) (int, error) {
	for {
		if s.failed != nil {
			return -1, s.failed
		}
		if i := s.find(pageNo); i >= 0 {
			if s.slots[i].state == slotReading {
				s.cond.Wait()
				continue
			}
			s.touch(i)
			return i, nil
		}
		v := s.victim()
		if v < 0 {
			s.cond.Wait() // todas cargándose o escribiéndose
			continue
		}
		if s.slots[v].dirty {
			if err := s.writeSlot(v); err != nil {
				return -1, err // disco lleno: la ranura sigue sucia en la caché
			}
			continue // se vuelve a mirar todo
		}
		if err := s.loadSlot(v, pageNo); err != nil {
			return -1, err
		}
		s.touch(v)
		return v, nil
	}
}

// find devuelve la ranura que tiene (o está cargando) pageNo, o -1.
func (s *Statuses) find(pageNo uint64) int {
	for i := range s.slots {
		if s.slots[i].state != slotEmpty && s.slots[i].pageNo == pageNo {
			return i
		}
	}
	return -1
}

// victim elige la ranura a reemplazar: una libre, si no la limpia usada hace más tiempo, si no la sucia usada hace más
// tiempo (con el disco lleno, así las lecturas siguen sin tener que escribir); nunca una ocupada en E/S.
func (s *Statuses) victim() int {
	clean, dirty := -1, -1
	for i := range s.slots {
		sl := &s.slots[i]
		if sl.writing || sl.state == slotReading {
			continue
		}
		if sl.state == slotEmpty {
			return i
		}
		best := &clean
		if sl.dirty {
			best = &dirty
		}
		if *best < 0 || sl.lastUsed < s.slots[*best].lastUsed {
			*best = i
		}
	}
	if clean >= 0 {
		return clean
	}
	return dirty
}

// touch marca la ranura i como recién usada.
func (s *Statuses) touch(i int) {
	s.tick++
	s.slots[i].lastUsed = s.tick
}

// loadSlot lee la página pageNo en la ranura libre o limpia i. Suelta mu durante la lectura.
func (s *Statuses) loadSlot(i int, pageNo uint64) error {
	sl := &s.slots[i]
	sl.pageNo = pageNo
	sl.state = slotReading
	sl.dirty = false
	sl.maxLSN = 0
	clear(sl.groupLSN)

	s.mu.Unlock()
	err := s.readPage(pageNo, sl.data)
	s.mu.Lock()

	if err != nil {
		sl.state = slotEmpty
	} else {
		sl.state = slotValid
	}
	s.cond.Broadcast()
	return err
}

// writeSlot escribe una copia de la ranura sucia i al disco, después de esperar al WAL. Suelta mu durante la
// espera y la escritura; los cambios que lleguen mientras tanto la dejan sucia otra vez. Un fallo deja el archivo
// de estados inutilizable, salvo el de disco lleno: la página sigue sucia en la caché (no se perdió nada) y se
// vuelve a intentar en el próximo Flush o reemplazo.
func (s *Statuses) writeSlot(i int) error {
	sl := &s.slots[i]
	sl.writing = true
	sl.dirty = false
	buf := make([]byte, PageSize)
	copy(buf, sl.data)
	pageNo, lsn := sl.pageNo, sl.maxLSN

	s.mu.Unlock()
	err := s.writePage(pageNo, buf, lsn)
	s.mu.Lock()

	sl.writing = false
	if err != nil {
		sl.dirty = true
		if s.failed == nil && !vfs.IsNoSpace(err) {
			s.failed = err
		}
	}
	s.cond.Broadcast()
	return err
}

// discardPages vacía, sin escribirlas, las ranuras cuyas páginas cumplen match, esperando las que están en E/S.
// Se llama con mu tomado.
func (s *Statuses) discardPages(match func(pageNo uint64) bool) {
	for i := range s.slots {
		for {
			sl := &s.slots[i]
			if sl.state == slotEmpty || !match(sl.pageNo) {
				break
			}
			if sl.state == slotReading || sl.writing {
				s.cond.Wait()
				continue
			}
			sl.state = slotEmpty
			sl.dirty = false
			break
		}
	}
}

// readPage lee la página pageNo en buf; lo que falta en el archivo (o el archivo entero) se lee como ceros. Un
// archivo que no existe y no se puede crear por disco lleno también se lee como ceros.
func (s *Statuses) readPage(pageNo uint64, buf []byte) error {
	f, err := s.file(s.fileOf(pageNo))
	if vfs.IsNoSpace(err) {
		clear(buf)
		return nil
	}
	if err != nil {
		return err
	}
	n, err := f.ReadAt(buf, s.offsetOf(pageNo))
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	clear(buf[n:])
	return nil
}

// writePage espera a que el WAL sea durable hasta lsn («WAL antes que la página») y escribe la página pageNo.
func (s *Statuses) writePage(pageNo uint64, buf []byte, lsn page.LSN) error {
	if lsn > s.flusher.Durable() {
		if err := s.flusher.WaitDurable(lsn); err != nil {
			return err
		}
	}
	n := s.fileOf(pageNo)
	f, err := s.file(n)
	if err != nil {
		return err
	}
	if _, err := f.WriteAt(buf, s.offsetOf(pageNo)); err != nil {
		return err
	}
	s.filesMu.Lock()
	s.unsynced[n] = true
	s.dirDirty = true // el archivo pudo ser nuevo: un fsync de más del directorio por checkpoint no cuesta
	s.filesMu.Unlock()
	return nil
}

// syncFiles hace fsync de los archivos escritos desde el último y del directorio.
func (s *Statuses) syncFiles() error {
	s.filesMu.Lock()
	var pending []vfs.File
	for n := range s.unsynced {
		pending = append(pending, s.files[n])
		delete(s.unsynced, n)
	}
	syncDir := s.dirDirty
	s.dirDirty = false
	s.filesMu.Unlock()

	for _, f := range pending {
		if err := f.Sync(); err != nil {
			return s.fail(err)
		}
	}
	if syncDir {
		if err := s.fs.SyncDir(s.dir); err != nil {
			return s.fail(err)
		}
	}
	return nil
}

// removeFiles cierra y borra los archivos cuyo número cumple match, y hace fsync del directorio.
func (s *Statuses) removeFiles(match func(n uint64) bool) error {
	numbers, err := fileNumbers(s.fs, s.dir)
	if err != nil {
		return err
	}
	removed := false
	for _, n := range numbers {
		if !match(n) {
			continue
		}
		s.filesMu.Lock()
		f := s.files[n]
		delete(s.files, n)
		delete(s.unsynced, n)
		s.filesMu.Unlock()
		if f != nil {
			f.Close()
		}
		if err := s.fs.Remove(fileName(s.dir, n)); err != nil {
			return s.fail(err)
		}
		removed = true
	}
	if removed {
		if err := s.fs.SyncDir(s.dir); err != nil {
			return s.fail(err)
		}
	}
	return nil
}

// cutFile corta el archivo n a size bytes si es más largo; el fsync queda para el próximo Flush.
func (s *Statuses) cutFile(n uint64, size int64) error {
	f, err := s.file(n)
	if err != nil {
		return err
	}
	current, err := f.Size()
	if err != nil {
		return err
	}
	if current <= size {
		return nil
	}
	if err := f.Truncate(size); err != nil {
		return s.fail(err)
	}
	s.filesMu.Lock()
	s.unsynced[n] = true
	s.filesMu.Unlock()
	return nil
}

// file devuelve el archivo n abierto, abriéndolo (o creándolo) si hace falta.
func (s *Statuses) file(n uint64) (vfs.File, error) {
	s.filesMu.Lock()
	defer s.filesMu.Unlock()
	if f := s.files[n]; f != nil {
		return f, nil
	}
	f, err := s.fs.OpenFile(fileName(s.dir, n), os.O_RDWR|os.O_CREATE)
	if err != nil {
		return nil, err
	}
	s.files[n] = f
	return f, nil
}

// fail deja el archivo de estados inutilizable con err (ver store/doc.go, «Fallos de escritura»).
func (s *Statuses) fail(err error) error {
	s.mu.Lock()
	if s.failed == nil {
		s.failed = err
	}
	s.cond.Broadcast()
	s.mu.Unlock()
	return err
}

// fileNumbers lista, ordenados, los números de los archivos de estados de dir (ignora otros nombres).
func fileNumbers(fs vfs.FS, dir string) ([]uint64, error) {
	names, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var numbers []uint64
	for _, name := range names {
		n, err := strconv.ParseUint(name, 10, 64)
		if err == nil && n > 0 {
			numbers = append(numbers, n)
		}
	}
	// ReadDir viene en orden alfabético; con anchos distintos (%04d pasa a 5 cifras) hay que ordenar por número.
	for i := 1; i < len(numbers); i++ {
		for j := i; j > 0 && numbers[j] < numbers[j-1]; j-- {
			numbers[j], numbers[j-1] = numbers[j-1], numbers[j]
		}
	}
	return numbers, nil
}

// fileName es la ruta del archivo de estados n.
func fileName(dir string, n uint64) string {
	return filepath.Join(dir, fmt.Sprintf("%04d", n))
}

// pageOf es la página global del xid.
func pageOf(xid page.XID) uint64 { return uint64(xid) / XidsPerPage }

// xidsPerFile es la cantidad de xids de cada archivo (XidsPerFile salvo en pruebas).
func (s *Statuses) xidsPerFile() uint64 { return s.pagesPerFile * XidsPerPage }

// fileOf es el número de archivo (desde 1) de una página global.
func (s *Statuses) fileOf(pageNo uint64) uint64 { return pageNo/s.pagesPerFile + 1 }

// offsetOf es el byte donde empieza una página global dentro de su archivo.
func (s *Statuses) offsetOf(pageNo uint64) int64 { return int64(pageNo%s.pagesPerFile) * PageSize }

// Oldest es el primer xid cuyo estado sigue en el archivo: los anteriores se truncaron y se leen confirmados.
func (s *Statuses) Oldest() page.XID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.oldest
}

// groupOf es el grupo de lsn del xid dentro de su página.
func groupOf(xid page.XID) uint64 { return (uint64(xid) % XidsPerPage) / GroupSize }

// getBits lee los 2 bits del xid en su página.
func getBits(data []byte, xid page.XID) Status {
	i := uint64(xid) % XidsPerPage
	return Status(data[i/4]>>(2*(i%4))) & 3
}

// setBits escribe los 2 bits del xid en su página.
func setBits(data []byte, xid page.XID, st Status) {
	i := uint64(xid) % XidsPerPage
	shift := 2 * (i % 4)
	data[i/4] = data[i/4]&^(3<<shift) | byte(st)<<shift
}

// clearFrom pone en cero (en curso) los bits de la página desde el índice from hasta el final.
func clearFrom(data []byte, from uint64) {
	for ; from%4 != 0; from++ {
		data[from/4] &^= 3 << (2 * (from % 4))
	}
	clear(data[from/4:])
}
