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
	dir   string
	opts  options
	lock  *os.File
	index *index

	// writeMu: reservas (lsn y lugar en el log), rotación y cierre.
	writeMu  sync.Mutex
	lastLSN  uint64
	segments []*segment
	closed   bool

	// active: el segmento donde se reserva; se cambia bajo writeMu y se lee sin él al confirmar.
	active atomic.Pointer[segment]

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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := syncDir(filepath.Dir(dir)); err != nil {
		return nil, err
	}
	lock, err := lockDir(filepath.Join(dir, lockName))
	if err != nil {
		return nil, err
	}

	s := &Store{dir: dir, opts: opts, lock: lock, index: newIndex(), finished: map[uint64]bool{}}
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
		seg, err := createSegment(s.dir, 1)
		if err != nil {
			return err
		}
		s.segments = []*segment{seg}
		s.active.Store(seg)
		return nil
	}

	for i, num := range nums {
		seg, err := openSegment(s.dir, num)
		if err != nil {
			return err
		}
		s.segments = append(s.segments, seg)
		last := i == len(nums)-1
		if err := s.load(seg, last); err != nil {
			return err
		}
	}
	s.active.Store(s.segments[len(s.segments)-1])
	return nil
}

/**
* load: Pasa al índice los registros de un segmento y fija su tamaño. Ver recover.
* @param seg *segment, last bool
* @return error
**/
func (s *Store) load(seg *segment, last bool) error {
	end, reached, err := seg.scan(func(rec scanned) {
		s.index.set(rec.id, rec.ref)
		s.lastLSN = max(s.lastLSN, rec.ref.lsn)
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
* reservation: El lugar de una escritura en el log.
**/
type reservation struct {
	lsn    uint64
	seg    *segment
	offset int64
}

/**
* reserve: Toma el siguiente lsn y el lugar para un registro de size bytes; si no cabe en el segmento activo, lo
* sella y crea el siguiente. Un registro más grande que un segmento va solo en uno nuevo.
* @param size int
* @return reservation, error
**/
func (s *Store) reserve(size int) (reservation, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.closed {
		return reservation{}, errors.New(MSG_CLOSED)
	}
	if err := s.failed(); err != nil {
		return reservation{}, err
	}
	active := s.active.Load()
	if active.size > 0 && active.size+int64(size) > s.opts.segmentSize {
		var err error
		if active, err = s.rotate(); err != nil {
			s.fail(err)
			return reservation{}, err
		}
	}

	s.lastLSN++
	r := reservation{lsn: s.lastLSN, seg: active, offset: active.size}
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
	seg, err := createSegment(s.dir, old.num+1)
	if err != nil {
		return nil, err
	}
	s.segments = append(s.segments, seg)
	s.active.Store(seg)
	return seg, nil
}

/**
* put: Guarda data para id sin mirar lo que había: reserva, escribe, publica y, con sync, espera el fsync.
* @param id string, data []byte
* @return error
**/
func (s *Store) put(id string, data []byte) error {
	if err := checkPut(id, data); err != nil {
		return err
	}
	buf := encodeRecord(kindPut, id, data)
	r, err := s.reserve(len(buf))
	if err != nil {
		return err
	}
	sealRecord(buf, r.lsn)

	err = r.seg.writeAt(buf, r.offset)
	if err == nil {
		s.index.set(id, ref{segment: r.seg.num, offset: r.offset, size: uint32(len(buf)), lsn: r.lsn})
	}
	r.seg.inflight.Done()
	if err != nil {
		s.fail(err)
		return err
	}

	s.publish(r.lsn)
	if s.opts.sync {
		return s.waitDurable(r.lsn)
	}
	return nil
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
	if s.closed {
		return nil
	}
	s.closed = true

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
* closeFiles: Cierra los segmentos y suelta el bloqueo.
* @return error
**/
func (s *Store) closeFiles() error {
	var errs []error
	for _, seg := range s.segments {
		errs = append(errs, seg.close())
	}
	if s.lock != nil {
		errs = append(errs, s.lock.Close())
	}
	return errors.Join(errs...)
}
