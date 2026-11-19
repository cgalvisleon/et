package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/envar"
)

const lockName = "LOCK"

/**
* options: Lo configurable de un store. Open las lee de las variables de entorno; las pruebas las fijan a mano.
**/
type options struct {
	segmentSize int64 // RELSEG_SIZE (MB): al pasarlo, el segmento activo se sella y se crea otro
	sync        bool  // SYNC_ON_WRITE: Put espera a que su escritura esté en disco
}

/**
* envOptions: Las opciones desde las variables de entorno.
* @return options
**/
func envOptions() options {
	return options{
		segmentSize: envar.GetInt64("RELSEG_SIZE", 128) * 1024 * 1024,
		sync:        envar.GetBool("SYNC_ON_WRITE", true),
	}
}

/**
* Store: Un store abierto. Los campos de cada bloque los protege el lock de su comentario (orden en doc.go).
**/
type Store struct {
	dir     string
	opts    options
	lock    *os.File
	index   *index
	pending *pending // escrituras reservadas sin terminar, por id (ver pending.go)

	// writeMu: reservas (lsn y lugar en el log), rotación y cierre.
	writeMu sync.Mutex
	lastLSN uint64

	// segments, active y closed se cambian bajo writeMu y se leen sin él: Read no toma writeMu.
	segments atomic.Pointer[[]*segment] // copia al escribir: rotate publica una lista nueva
	active   atomic.Pointer[segment]    // el segmento donde se reserva
	closed   atomic.Bool
	mapsMu   *readLock // Read lo toma para leer de los mapas; close lo toma para escribir antes de deshacerlos

	// pubMu: qué escrituras terminaron.
	pubMu     sync.Mutex
	pubCond   *sync.Cond
	published uint64          // todos los lsn ≤ published están escritos
	finished  map[uint64]bool // lsn escritos por encima de published, esperando a los anteriores
	failure   error           // primer error de escritura: el store deja de aceptar escrituras

	// syncMu: qué está en disco.
	syncMu  sync.Mutex
	durable uint64 // todos los lsn ≤ durable tienen fsync
	fsyncs  atomic.Int64
}

/**
* open: Abre (o crea) el store de dir: toma el bloqueo, rehace el índice recorriendo los segmentos y deja listo el
* segmento activo.
* @param dir string, opts options
* @return *Store, error
**/
func open(dir string, opts options) (*Store, error) {
	if err := mkdirAll(dir); err != nil {
		return nil, err
	}
	lock, err := lockDir(filepath.Join(dir, lockName))
	if err != nil {
		return nil, err
	}

	s := &Store{dir: dir, opts: opts, lock: lock, index: newIndex(), pending: newPending(), mapsMu: newReadLock(),
		finished: map[uint64]bool{}}
	s.segments.Store(&[]*segment{})
	s.pubCond = sync.NewCond(&s.pubMu)
	if err := s.recover(); err != nil {
		s.closeFiles()
		return nil, err
	}
	s.published, s.durable = s.lastLSN, s.lastLSN
	return s, nil
}

/**
* recover: Abre los segmentos en orden y rehace el índice. El último puede terminar en un registro incompleto o en un
* hueco (escrituras nunca confirmadas): se corta ahí. En un segmento sellado, un registro inválido es corrupción.
* @return error
**/
func (s *Store) recover() error {
	nums, err := listSegments(s.dir)
	if err != nil {
		return err
	}
	if len(nums) == 0 {
		seg, err := createSegment(s.dir, 1, s.opts.segmentSize)
		if err != nil {
			return err
		}
		s.addSegment(seg)
		return nil
	}

	for i, num := range nums {
		seg, err := openSegment(s.dir, num, s.opts.segmentSize)
		if err != nil {
			return err
		}
		s.addSegment(seg)
		last := i == len(nums)-1
		if err := s.load(seg, last); err != nil {
			return err
		}
	}
	return nil
}

/**
* addSegment: Agrega seg al final de la lista y lo deja como activo. Con writeMu tomado (o al abrir): la lista se
* copia, así quien lee la anterior sin lock nunca la ve cambiar.
* @param seg *segment
**/
func (s *Store) addSegment(seg *segment) {
	list := append(append([]*segment{}, s.segmentList()...), seg)
	s.segments.Store(&list)
	s.active.Store(seg)
}

/**
* segmentList: Los segmentos abiertos, en orden. La lista no cambia: rotate publica otra.
* @return []*segment
**/
func (s *Store) segmentList() []*segment {
	return *s.segments.Load()
}

/**
* segmentAt: El segmento número num, o nil si no está. Los números son contiguos desde el primero de la lista.
* @param num int
* @return *segment
**/
func (s *Store) segmentAt(num int) *segment {
	list := s.segmentList()
	if len(list) == 0 {
		return nil
	}
	i := num - list[0].num
	if i < 0 || i >= len(list) {
		return nil
	}
	return list[i]
}

/**
* load: Pasa al índice los registros de un segmento y fija su tamaño. Cada registro debe tener el lsn siguiente al
* último aplicado (lsn contiguos y crecientes en todo el log, ver doc.go, «Recuperación»): uno fuera de orden, aunque
* su crc sea válido (por ejemplo, la copia de un registro viejo), cuenta como inválido. Ver recover.
* @param seg *segment, last bool
* @return error
**/
func (s *Store) load(seg *segment, last bool) error {
	end, reached, err := seg.scan(func(rec scanned) bool {
		if rec.ref.lsn != s.lastLSN+1 {
			return false
		}
		s.apply(rec)
		s.lastLSN = rec.ref.lsn
		return true
	})
	if err != nil {
		return err
	}
	if !reached {
		if !last {
			return fmt.Errorf(MSG_CORRUPTED, seg.path, end)
		}
		if err := seg.truncate(end); err != nil {
			return err
		}
	}
	seg.size = end
	return nil
}

/**
* apply: Pasa al índice un registro leído al recuperar. Los segmentos se recorren en orden de lsn (ver doc.go), así
* que un borrado simplemente quita el id: no hace falta guardar la lápida.
* @param rec scanned
**/
func (s *Store) apply(rec scanned) {
	if rec.ref.deleted {
		s.index.remove(rec.id)
		return
	}
	s.index.set(rec.id, rec.ref)
}

/**
* reservation: El lugar de una escritura en el log.
**/
type reservation struct {
	lsn    uint64
	seg    *segment
	offset int64
	size   int
}

/**
* condition: Decide, con el estado lógico del id, si una escritura condicional se hace. nil = escritura ciega.
**/
type condition func(cur current) bool

/**
* reserve: Bajo writeMu, mira la condición contra el estado lógico de id y, si se cumple, toma el siguiente lsn y el
* lugar para un registro de size bytes y anota la escritura como pendiente de id. Comprobar y reservar en el mismo
* lock es lo que hace atómicas a Insert, Update y Delete frente a cualquier otro escritor del mismo id.
* @param id string, deleted bool (es un borrado), size int, cond condition (nil: siempre)
* @return reservation, bool (si la condición se cumplió), error
**/
func (s *Store) reserve(id string, deleted bool, size int, cond condition) (reservation, bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.closed.Load() {
		return reservation{}, false, errors.New(MSG_CLOSED)
	}
	if err := s.failed(); err != nil {
		return reservation{}, false, err
	}
	if cond != nil {
		cur, _ := s.pending.state(id, s.index)
		if !cond(cur) {
			return reservation{}, false, nil
		}
	}
	r, err := s.allocate(size)
	if err != nil {
		return reservation{}, false, err
	}
	s.pending.add(id, r.lsn, deleted)
	return r, true, nil
}

/**
* allocate: Toma el siguiente lsn y size bytes del segmento activo; si no caben, lo sella y crea el siguiente. Un
* registro más grande que un segmento va solo en uno nuevo. Con writeMu tomado.
* @param size int
* @return reservation, error
**/
func (s *Store) allocate(size int) (reservation, error) {
	active := s.active.Load()
	if active.size > 0 && active.size+int64(size) > s.opts.segmentSize {
		var err error
		if active, err = s.rotate(); err != nil {
			s.fail(err)
			return reservation{}, err
		}
	}

	s.lastLSN++
	r := reservation{lsn: s.lastLSN, seg: active, offset: active.size, size: size}
	active.size += int64(size)
	active.inflight.Add(1)
	return r, nil
}

/**
* rotate: Sella el segmento activo y crea el siguiente. Con writeMu tomado: nadie reserva mientras tanto, y las
* escrituras en curso terminan sin necesitar el lock.
* @return *segment (el nuevo activo), error
**/
func (s *Store) rotate() (*segment, error) {
	old := s.active.Load()
	if err := old.seal(); err != nil {
		return nil, err
	}
	s.fsyncs.Add(1)
	seg, err := createSegment(s.dir, old.num+1, s.opts.segmentSize)
	if err != nil {
		return nil, err
	}
	s.addSegment(seg)
	return seg, nil
}

/**
* write: El camino de toda escritura: reserva (si cond se cumple), arma el registro, lo escribe en su lugar, pasa la
* ref al índice, publica y, con sync, espera el fsync. El registro se arma después de reservar: una condicional que
* no se cumple no asigna memoria.
* @param id string, k kind, data []byte, cond condition (nil: escritura ciega)
* @return bool (si escribió), error
**/
func (s *Store) write(id string, k kind, data []byte, cond condition) (bool, error) {
	deleted := k == kindDelete
	r, ok, err := s.reserve(id, deleted, recordSize(id, data), cond)
	if err != nil || !ok {
		return false, err
	}
	buf := encodeRecord(k, id, data)
	sealRecord(buf, r.lsn)

	err = r.seg.writeAt(buf, r.offset)
	at := ref{segment: r.seg.num, offset: r.offset, size: uint32(r.size), deleted: deleted, lsn: r.lsn}
	s.pending.finish(id, at, err == nil, s.index)
	r.seg.inflight.Done()
	if err != nil {
		s.fail(err)
		return false, err
	}

	s.publish(r.lsn)
	if s.opts.sync {
		return true, s.waitDurable(r.lsn)
	}
	return true, nil
}

/**
* publish: Avisa que lsn ya está escrito y avanza published sobre todos los contiguos que hayan terminado.
* @param lsn uint64
**/
func (s *Store) publish(lsn uint64) {
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	s.finished[lsn] = true
	for s.finished[s.published+1] {
		delete(s.finished, s.published+1)
		s.published++
	}
	s.pubCond.Broadcast()
}

/**
* waitPublished: Espera a que todas las escrituras hasta lsn estén escritas.
* @param lsn uint64
* @return error (si una escritura falló antes de llegar a lsn)
**/
func (s *Store) waitPublished(lsn uint64) error {
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	for s.published < lsn && s.failure == nil {
		s.pubCond.Wait()
	}
	if s.published < lsn {
		return fmt.Errorf(MSG_FAILED, s.failure)
	}
	return nil
}

/**
* waitDurable: Espera a que lsn esté en disco. Quien llega sin que su lsn esté confirmado hace un fsync que confirma
* todo lo publicado hasta ese momento, y los que esperaban detrás de él ya no necesitan otro (group commit).
* @param lsn uint64
* @return error
**/
func (s *Store) waitDurable(lsn uint64) error {
	if err := s.waitPublished(lsn); err != nil {
		return err
	}

	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.durable >= lsn {
		return nil
	}
	// target se lee ya con syncMu tomado: el fsync confirma todo lo publicado mientras se esperaba el turno, no solo
	// hasta lsn. El activo se lee después de target: los anteriores ya se sellaron con su fsync (ver rotate).
	target := s.publishedNow()
	if err := s.active.Load().file.Sync(); err != nil {
		s.fail(err)
		return err
	}
	s.fsyncs.Add(1)
	s.durable = max(s.durable, target)
	return nil
}

/**
* publishedNow: El valor actual de published.
* @return uint64
**/
func (s *Store) publishedNow() uint64 {
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	return s.published
}

/**
* fail: Deja el store en error: ninguna escritura nueva se acepta y quien espera publicación se entera.
* @param err error
**/
func (s *Store) fail(err error) {
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	if s.failure == nil {
		s.failure = err
	}
	s.pubCond.Broadcast()
}

/**
* failed: El error que dejó al store sin aceptar escrituras, o nil.
* @return error
**/
func (s *Store) failed() error {
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	if s.failure != nil {
		return fmt.Errorf(MSG_FAILED, s.failure)
	}
	return nil
}

/**
* close: Deja de aceptar escrituras, espera las que están en curso, hace fsync del activo y cierra los archivos y el
* bloqueo. Cerrar dos veces no hace nada.
* @return error
**/
func (s *Store) close() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.closed.Load() {
		return nil
	}
	s.closed.Store(true)

	s.waitReaders()
	err := s.active.Load().seal()
	if err == nil {
		// Lo publicado ya está en disco: quien espera su fsync no debe tocar un archivo cerrado.
		s.syncMu.Lock()
		s.durable = max(s.durable, s.publishedNow())
		s.syncMu.Unlock()
	}
	return errors.Join(err, s.closeFiles())
}

/**
* waitReaders: Espera a que terminen las lecturas que están usando los mapas. Se llama con closed ya en true, así que
* ninguna lectura nueva entra (ver readRef): tomar y soltar mapsMu para escribir basta.
**/
func (s *Store) waitReaders() {
	s.mapsMu.lock()
	s.mapsMu.unlock()
}

/**
* closeFiles: Cierra los segmentos y suelta el bloqueo.
* @return error
**/
func (s *Store) closeFiles() error {
	var errs []error
	for _, seg := range s.segmentList() {
		errs = append(errs, seg.close())
	}
	if s.lock != nil {
		errs = append(errs, s.lock.Close())
	}
	return errors.Join(errs...)
}
