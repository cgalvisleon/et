package store

import (
	"bytes"
	"errors"
	"sync"

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

	// mu: las operaciones lo toman en lectura; DropStore en escritura (espera a las que están en curso).
	mu      sync.RWMutex
	dropped bool

	// vacuumMu: una limpieza del store a la vez (btree.Vacuum no admite dos sobre el mismo árbol).
	vacuumMu sync.Mutex
}

// change es una escritura suelta: la versión nueva y su condición.
type change struct {
	tombstone bool
	data      []byte
	check     func(latest btree.Latest) btree.Action // nil = siempre (Put, Remove)
}

/**
* begin deja pasar una operación sobre el store; quien entra llama a end al terminar.
* @return error: ErrClosed, ErrFailed o ErrStoreDropped
**/
func (s *Store) begin() error {
	if err := s.e.enter(); err != nil {
		return err
	}
	s.mu.RLock()
	if s.dropped {
		s.mu.RUnlock()
		s.e.leave()
		return ErrStoreDropped
	}
	return nil
}

/**
* end termina una operación que pasó por begin.
**/
func (s *Store) end() {
	s.mu.RUnlock()
	s.e.leave()
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
* write hace una escritura suelta (el orden de un cambio en doc.go): su propia transacción, el valor grande en
* desborde antes que la hoja, la condición con la hoja bloqueada, la espera a otra transacción sin páginas
* bloqueadas y la confirmación al volver. Un fallo de escritura deja el motor inutilizable.
* @param id string, c change
* @return bool: si escribió, error
**/
func (s *Store) write(id string, c change) (bool, error) {
	if err := checkWrite(id, c.data); err != nil {
		return false, err
	}
	if err := s.begin(); err != nil {
		return false, err
	}
	defer s.end()

	xid := s.e.txs.Begin()
	defer s.e.txs.End(xid)
	w, err := s.prepare(xid, id, c)
	if err != nil {
		return false, s.e.fail(err)
	}
	applied, err := s.writeLoop(id, w)
	unused := err == nil || errors.Is(err, ErrConflict)
	if !applied && unused && w.Overflow != nil {
		if ferr := s.e.chains.Free(s.file, *w.Overflow, xid); ferr != nil {
			return false, s.e.fail(ferr)
		}
	}
	return applied, err
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
* writeLoop escribe la versión y confirma; si otra transacción en curso tiene el id, la espera (hasta LOCK_TIMEOUT)
* sin páginas bloqueadas y repite. commitMu va en lectura desde antes de reservar hasta anotar el estado.
* @param id string, w btree.Write
* @return bool: si escribió, error: ErrConflict, o ErrFailed con la causa
**/
func (s *Store) writeLoop(id string, w btree.Write) (bool, error) {
	for {
		s.e.commitMu.RLock()
		res, err := s.tree.Write(id, w)
		var wait *btree.WaitError
		if errors.As(err, &wait) {
			s.e.commitMu.RUnlock()
			if err := s.e.txs.Wait(wait.XID, s.e.cfg.lockTimeout); err != nil {
				return false, ErrConflict
			}
			if err := s.e.failed(); err != nil {
				return false, err // la otra pudo terminar sin estado porque el motor falló: no repetir sin fin
			}
			continue
		}
		if err == nil && res.Applied {
			s.e.writes.Add(1)
			err = s.e.commit(w.XID, res.LSN)
		}
		s.e.commitMu.RUnlock()
		if err != nil {
			return false, s.e.fail(err)
		}
		return res.Applied, nil
	}
}

/**
* commit confirma una escritura suelta (paso 10): con SYNC_ON_WRITE espera el WAL durable hasta su lsn (group
* commit) y anota confirmada en el archivo de estados; desde ahí la ven las lecturas.
* @param xid page.XID, lsn page.LSN
* @return error
**/
func (e *Engine) commit(xid page.XID, lsn page.LSN) error {
	if e.cfg.syncOnWrite {
		if err := e.log.WaitDurable(lsn); err != nil {
			return err
		}
	}
	return e.status.SetStatus(xid, xact.StatusCommitted, lsn)
}

/**
* put guarda id con data, lo cree o lo reemplace.
* @param id string, data []byte
* @return error
**/
func (s *Store) put(id string, data []byte) error {
	_, err := s.write(id, change{data: data})
	return err
}

/**
* remove borra id sin preguntar; si no tiene versión viva no escribe nada.
* @param id string
* @return error
**/
func (s *Store) remove(id string) error {
	_, err := s.write(id, change{tombstone: true})
	return err
}

/**
* insert guarda id solo si no existe. Con un valor grande mira antes si existe, para no escribir el desborde en vano.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) insert(id string, data []byte) (bool, error) {
	if checkWrite(id, data) == nil && !inline(id, data) {
		if _, found, err := s.read(id); err != nil || found {
			return false, err
		}
	}
	return s.write(id, change{data: data, check: func(latest btree.Latest) btree.Action {
		if latest.Live {
			return btree.Skip
		}
		return btree.Apply
	}})
}

/**
* delete borra id solo si existe.
* @param id string
* @return bool: si borró, error
**/
func (s *Store) delete(id string) (bool, error) {
	return s.write(id, change{tombstone: true})
}

/**
* update reemplaza id solo si existe y el valor cambió. Compara antes de bloquear la hoja (leer de desborde con la
* hoja bloqueada frenaría a todos) y, con la hoja bloqueada, comprueba que la versión comparada siga siendo la
* última (por su lsn); si cambió entre medio, repite.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) update(id string, data []byte) (bool, error) {
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
		applied, err := s.write(id, change{data: data, check: func(latest btree.Latest) btree.Action {
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
	return s.e.counts.get(s.file)
}
