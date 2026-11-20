package store

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// Store es un store abierto de un motor: un árbol con sus archivos. Es seguro usarlo desde varias goroutines.
type Store struct {
	e    *Engine
	name string
	file smgr.FileNo
	tree *btree.BTree

	// mu: las operaciones lo toman en lectura; DropStore y Empty en escritura (esperan a las que están en curso).
	// file y tree solo cambian con mu en escritura (Empty los cambia por los del archivo nuevo).
	mu      sync.RWMutex
	dropped bool

	// vacuumMu: una limpieza del store a la vez (btree.Vacuum no admite dos sobre el mismo árbol).
	vacuumMu sync.Mutex

	// DDL en lote (ddlbatch.go). Se escriben con Engine.ddlMu y mu en escritura; se leen con cualquiera de los dos.
	// creator: el xid del lote que lo creó y todavía no confirmó (0 = ninguno); dropper: el del lote que lo borra al
	// confirmar; createdBy: el del lote que lo creó y confirmó (un lote cuyo snapshot no lo ve recibe
	// ErrSerialization).
	creator, dropper, createdBy page.XID

	// batchWriters: lotes abiertos que escribieron en el store (DropStoreIn no lo borra si escribió otro).
	batchWriters atomic.Int64
}

// change es una escritura suelta: la versión nueva y su condición.
type change struct {
	tombstone bool
	data      []byte
	check     func(latest btree.Latest) btree.Action // nil = siempre (Put, Remove)
	deferred  bool                                   // confirma sin esperar el fsync del WAL (Deferred)
}

/**
* begin deja pasar una operación que solo lee el store (también con el motor de solo lectura); quien entra llama a
* end al terminar.
* @return error: ErrClosed, ErrFailed o ErrStoreDropped
**/
func (s *Store) begin() error {
	return s.enterWith(s.e.enterRead)
}

/**
* beginWrite es begin para una operación que escribe: el motor tiene que estar usable.
* @return error: ErrClosed, ErrFailed o ErrStoreDropped
**/
func (s *Store) beginWrite() error {
	return s.enterWith(s.e.enter)
}

/**
* enterWith entra al motor con enter (o enterRead) y toma el store en lectura.
* @param enter func() error
* @return error
**/
func (s *Store) enterWith(enter func() error) error {
	if err := enter(); err != nil {
		return err
	}
	s.e.replica.readLock()
	s.mu.RLock()
	var err error
	switch {
	case s.dropped:
		err = ErrStoreDropped
	case s.tree == nil: // una réplica que rehízo el vaciado y todavía no el KindInitStore del archivo nuevo
		err = fmt.Errorf(MSG_STORE_PENDING, ErrStorePending, s.name)
	}
	if err != nil {
		s.end()
	}
	return err
}

/**
* end termina una operación que pasó por begin.
**/
func (s *Store) end() {
	s.mu.RUnlock()
	s.e.replica.readUnlock()
	s.e.leave()
}

/**
* pendingFor dice si la transacción own puede escribir en el store: no si lo está creando otro lote (ni una
* operación suelta: own = 0), ni si otro lote lo borra (una suelta sí: confirma antes que ese borrado). Con mu o
* Engine.ddlMu tomado.
* @param own page.XID: el xid del lote, 0 para una escritura suelta
* @return error: ErrStorePending
**/
func (s *Store) pendingFor(own page.XID) error {
	if s.creator != 0 && s.creator != own || own != 0 && s.dropper != 0 && s.dropper != own {
		return fmt.Errorf(MSG_STORE_PENDING, ErrStorePending, s.name)
	}
	return nil
}

/**
* ddlOwner es el xid del otro lote que crea o borra el store, al que hay que esperar (0 si no hay). Con
* Engine.ddlMu tomado.
* @param own page.XID: el lote que pregunta
* @return page.XID, bool: si ese lote lo borra
**/
func (s *Store) ddlOwner(own page.XID) (page.XID, bool) {
	switch {
	case s.creator != 0 && s.creator != own:
		return s.creator, false
	case s.dropper != 0 && s.dropper != own:
		return s.dropper, true
	}
	return 0, false
}

/**
* pendingCreator es el xid del lote que está creando el store (0 si ninguno).
* @return page.XID
**/
func (s *Store) pendingCreator() page.XID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.creator
}

/**
* markDropped espera a las operaciones en curso y deja el store inservible.
**/
func (s *Store) markDropped() {
	s.mu.Lock()
	s.dropped = true
	s.mu.Unlock()
}

/**
* checkID valida el largo de un id.
* @param id string
* @return error: ErrBadID
**/
func checkID(id string) error {
	if len(id) == 0 || len(id) > MaxIDLen {
		return ErrBadID
	}
	return nil
}

/**
* checkWrite valida el id y el valor de una escritura.
* @param id string, data []byte
* @return error: ErrBadID o ErrValueTooLarge
**/
func checkWrite(id string, data []byte) error {
	if err := checkID(id); err != nil {
		return err
	}
	if len(data) > MaxValueLen {
		return ErrValueTooLarge
	}
	return nil
}

/**
* inline dice si la versión cabe en la hoja o va a desborde.
* @param id string, data []byte
* @return bool
**/
func inline(id string, data []byte) bool {
	return btree.EntryHeaderSize+len(id)+len(data) <= btree.MaxEntrySize
}

/**
* write hace una escritura suelta (el orden de un cambio en doc.go): el freno del WAL y la entrada de lugar, su
* propia transacción, el valor grande en desborde antes que la hoja, la condición con la hoja bloqueada y la
* confirmación al volver. Si otra transacción en curso tiene el id, la espera (hasta LOCK_TIMEOUT) fuera del store y
* del motor, sin Store.mu, closeMu ni páginas (como XactLockTableWait de PostgreSQL), y repite desde la entrada: así
* esa espera no cierra un ciclo con DropStoreIn, DropStore, Empty ni Close. Un disco lleno antes de escribir la hoja
* devuelve ErrDiskFull sin tocar nada (el desborde ya escrito se libera); otro fallo de escritura deja el motor
* inutilizable.
* @param id string, c change
* @return bool: si escribió, error
**/
func (s *Store) write(id string, c change) (bool, error) {
	if err := checkWrite(id, c.data); err != nil {
		return false, err
	}
	if err := s.e.throttle(); err != nil {
		return false, err
	}
	var xid page.XID
	defer func() {
		if xid != 0 {
			s.e.heads.forget(xid) // después de la hoja: ChangeHead ya ve su confirmación
			s.e.txs.End(xid)
		}
	}()
	for {
		applied, wait, err := s.writeRound(&xid, id, c)
		if wait == 0 {
			return applied, err
		}
		if err := s.e.txs.Wait(wait, s.e.cfg.lockTimeout); err != nil {
			return false, ErrConflict
		}
		if err := s.e.failed(); err != nil {
			return false, err // la otra pudo terminar sin estado porque el motor falló: no repetir sin fin
		}
	}
}

/**
* writeRound es una vuelta de write con el store tomado: entra, prepara la versión (con su desborde), la escribe y
* confirma. Si hay que esperar a otra transacción libera el desborde antes de salir (el store puede vaciarse o
* borrarse durante la espera) y devuelve a quién esperar. La transacción propia se abre en la primera vuelta y sigue
* la misma en las siguientes (no escribió ninguna hoja).
* @param xid *page.XID: la transacción de la escritura (0 = abrirla), id string, c change
* @return bool: si escribió, page.XID: la transacción a esperar (0 = ninguna), error
**/
func (s *Store) writeRound(xid *page.XID, id string, c change) (bool, page.XID, error) {
	if err := s.beginWrite(); err != nil {
		return false, 0, err
	}
	defer s.end()
	if err := s.pendingFor(0); err != nil {
		return false, 0, err
	}
	if err := s.e.admit(); err != nil {
		return false, 0, err
	}
	if *xid == 0 {
		*xid = s.e.txs.Begin()
	}
	w, err := s.prepare(*xid, id, c)
	if err != nil {
		return false, 0, s.e.writeError(err)
	}
	applied, wait, err := s.writeLeaf(id, w, c.deferred)
	unused := wait != 0 || err == nil || isCleanDiskFull(err)
	if !applied && unused && w.Overflow != nil {
		if ferr := s.e.chains.Free(s.file, *w.Overflow, *xid); ferr != nil {
			return false, 0, s.e.fail(ferr)
		}
	}
	return applied, wait, err
}

/**
* prepare arma la versión a escribir; si el valor no cabe en la hoja, escribe antes su cadena de desborde.
* @param xid page.XID, id string, c change
* @return btree.Write, error
**/
func (s *Store) prepare(xid page.XID, id string, c change) (btree.Write, error) {
	w := btree.Write{
		XID:       xid,
		Tombstone: c.tombstone,
		Commit:    true,
		Check:     c.check,
		UnderLock: s.countUnderLock,
	}
	if c.tombstone {
		return w, nil
	}
	if inline(id, c.data) {
		w.Value = c.data
		return w, nil
	}
	s.e.heads.note(xid, s.e.log.LastLSN) // la cadena va antes que la hoja: ChangeHead no debe partir en medio
	ref, err := s.e.chains.Write(s.file, xid, c.data)
	if err != nil {
		return w, err
	}
	w.Overflow = &ref
	return w, nil
}

/**
* countUnderLock suma la escritura al contador del store, bajo el writeMu del WAL (en orden de lsn).
* @param lsn page.LSN, delta int
**/
func (s *Store) countUnderLock(_ page.LSN, delta int) {
	if delta != 0 {
		s.e.counts.add(s.file, int64(delta))
	}
}

/**
* writeLeaf intenta una vez escribir la versión y confirmarla. commitMu va en lectura desde antes de reservar hasta
* anotar el estado, y se suelta antes de devolver la transacción a esperar.
* @param id string, w btree.Write, deferred bool: confirmar sin esperar el fsync del WAL
* @return bool: si escribió, page.XID: la transacción en curso que tiene el id (0 = ninguna), error: ErrDiskFull, o
* ErrFailed con la causa
**/
func (s *Store) writeLeaf(id string, w btree.Write, deferred bool) (bool, page.XID, error) {
	s.e.commitMu.RLock()
	res, err := s.tree.Write(id, w)
	var wait *btree.WaitError
	if errors.As(err, &wait) {
		s.e.commitMu.RUnlock()
		return false, wait.XID, nil
	}
	if err != nil {
		s.e.commitMu.RUnlock()
		return false, 0, s.e.writeError(err) // la hoja no se escribió: un disco lleno limpio no deja nada
	}
	if res.Applied {
		s.e.writes.Add(1)
		err = s.e.commit(w.XID, res.LSN, deferred)
	}
	s.e.commitMu.RUnlock()
	if err != nil {
		return false, 0, s.e.fail(err)
	}
	return res.Applied, 0, nil
}

/**
* commit confirma una escritura suelta (paso 10): salvo diferida, con SYNC_ON_WRITE espera el WAL durable hasta su
* lsn (group commit); anota el lsn para WaitDurable y confirmada en el archivo de estados: desde ahí la ven las
* lecturas, sea o no durable todavía.
* @param xid page.XID, lsn page.LSN, deferred bool: no esperar el fsync
* @return error
**/
func (e *Engine) commit(xid page.XID, lsn page.LSN, deferred bool) error {
	start := time.Now()
	if err := e.syncCommit(lsn, modeOf(deferred)); err != nil {
		return err
	}
	if err := e.status.SetStatus(xid, xact.StatusCommitted, lsn); err != nil {
		return err
	}
	e.observeCommit(start)
	return nil
}

/**
* put guarda id con data, lo cree o lo reemplace.
* @param id string, data []byte, deferred bool: confirmar sin esperar el fsync
* @return error
**/
func (s *Store) put(id string, data []byte, deferred bool) error {
	_, err := s.write(id, change{data: data, deferred: deferred})
	return err
}

/**
* remove borra id sin preguntar; si no tiene versión viva no escribe nada.
* @param id string, deferred bool
* @return error
**/
func (s *Store) remove(id string, deferred bool) error {
	_, err := s.write(id, change{tombstone: true, deferred: deferred})
	return err
}

/**
* insert guarda id solo si no existe. Con un valor grande mira antes si existe, para no escribir el desborde en vano.
* @param id string, data []byte, deferred bool
* @return bool: si escribió, error
**/
func (s *Store) insert(id string, data []byte, deferred bool) (bool, error) {
	if checkWrite(id, data) == nil && !inline(id, data) {
		if _, found, err := s.read(id); err != nil || found {
			return false, err
		}
	}
	return s.write(id, change{data: data, deferred: deferred, check: func(latest btree.Latest) btree.Action {
		if latest.Live {
			return btree.Skip
		}
		return btree.Apply
	}})
}

/**
* delete borra id solo si existe.
* @param id string, deferred bool
* @return bool: si borró, error
**/
func (s *Store) delete(id string, deferred bool) (bool, error) {
	return s.write(id, change{tombstone: true, deferred: deferred})
}

/**
* update reemplaza id solo si existe y el valor cambió. Compara antes de bloquear la hoja (leer de desborde con la
* hoja bloqueada frenaría a todos) y, con la hoja bloqueada, comprueba que la versión comparada siga siendo la
* última (por su lsn); si cambió entre medio, repite.
* @param id string, data []byte, deferred bool
* @return bool: si escribió, error
**/
func (s *Store) update(id string, data []byte, deferred bool) (bool, error) {
	if err := checkWrite(id, data); err != nil {
		return false, err
	}
	for {
		cur, found, err := s.readVersion(id, nil)
		if err != nil || !found {
			return false, err
		}
		if bytes.Equal(cur.value, data) {
			return false, nil
		}
		moved := false
		applied, err := s.write(id, change{data: data, deferred: deferred, check: func(latest btree.Latest) btree.Action {
			moved = !latest.Live || latest.Version.LSN != cur.lsn
			if moved {
				return btree.Skip
			}
			return btree.Apply
		}})
		if err != nil || applied || !moved {
			return applied, err
		}
	}
}

// current es la última versión confirmada de un id, con su valor.
type current struct {
	lsn   page.LSN
	value []byte
}

/**
* read devuelve la versión más nueva confirmada de id (Read suelto, sin snapshot).
* @param id string
* @return []byte: copia del valor, bool: si existe, error
**/
func (s *Store) read(id string) ([]byte, bool, error) {
	cur, found, err := s.readVersion(id, nil)
	return cur.value, found, err
}

/**
* readVersion devuelve la versión de id que ve snap (nil = la más nueva confirmada), con su lsn (lo usa Update).
* @param id string, snap *xact.Snapshot
* @return current, bool, error
**/
func (s *Store) readVersion(id string, snap *xact.Snapshot) (current, bool, error) {
	if err := checkID(id); err != nil {
		return current{}, false, err
	}
	if err := s.begin(); err != nil {
		return current{}, false, err
	}
	defer s.end()
	defer s.e.observeRead(time.Now())
	v, found, err := s.tree.Lookup(id, snap)
	if err != nil || !found {
		return current{}, false, s.e.readError(err)
	}
	value, err := s.valueOf(v)
	if err != nil {
		return current{}, false, s.e.readError(err)
	}
	return current{lsn: v.LSN, value: value}, true, nil
}

/**
* valueOf devuelve el valor de una versión: el de la hoja (ya es una copia) o el leído de su cadena de desborde.
* @param v btree.Version
* @return []byte, error
**/
func (s *Store) valueOf(v btree.Version) ([]byte, error) {
	if v.Flags&btree.EntryOverflow != 0 {
		return s.e.chains.Read(s.file, v.Overflow)
	}
	if v.Value == nil {
		return []byte{}, nil
	}
	return v.Value, nil
}

/**
* count devuelve los ids vivos del store según lo último escrito (0 si se borró).
* @return int64
**/
func (s *Store) count() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.e.counts.get(s.file)
}
