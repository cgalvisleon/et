package store

import (
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// DDL dentro de un lote (tarea 67, decisión 66): StoreIn crea un store que solo el lote ve hasta su Commit y
// DropStoreIn lo borra al confirmar. Reglas en doc.go, «DDL dentro de un lote».

/**
* storeIn abre el store name para el lote b, o lo crea dentro del lote: KindCreateStoreIn (durable) con el xid del
* lote, la entrada del catálogo con FlagPendingCreate y su árbol. Si otro lote lo está creando o borrando, espera a
* que termine (hasta LOCK_TIMEOUT sin pasar el tiempo del lote).
* @param b *Batch, name string
* @return *Store, error: ErrBadName, ErrSerialization (otro lote lo creó o borró y confirmó después de que b empezó),
* ErrConflict, ErrTimeout, ErrStoreDropped (b lo borra), ErrStorePending, ErrClosed, ErrFailed, ErrDiskFull
**/
func (e *Engine) storeIn(b *Batch, name string) (*Store, error) {
	if !validName(name) {
		return nil, fmt.Errorf(MSG_STORE_NAME, ErrBadName, name)
	}
	if err := b.acquire(e); err != nil {
		return nil, err
	}
	defer b.mu.Unlock()
	var waited page.XID
	waitedDrop := false
	for {
		s, wait, drop, err := e.storeInLocked(b, name, waitedDrop)
		if wait == 0 {
			if txError(err) {
				b.abortLocked(err)
			}
			return s, err
		}
		if err := b.waitDDL(name, wait, waited); err != nil {
			return nil, err
		}
		waited, waitedDrop = wait, waitedDrop || drop
	}
}

/**
* storeInLocked es una vuelta de storeIn con el lote tomado.
* @param b *Batch, name string, waitedDrop bool: ya esperó a un lote que lo borraba
* @return *Store, page.XID: el lote al que esperar (0 = ninguno), bool: si ese lote lo borra, error
**/
func (e *Engine) storeInLocked(b *Batch, name string, waitedDrop bool) (*Store, page.XID, bool, error) {
	if err := e.enterRead(); err != nil {
		return nil, 0, false, err
	}
	defer e.leave()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	s, err := e.openLocked(name)
	if err != nil {
		return nil, 0, false, err
	}
	if s == nil {
		if waitedDrop {
			return nil, 0, false, ErrSerialization // lo borró un lote que confirmó después de que b empezó
		}
		s, err = e.createIn(b, name)
		return s, 0, false, err
	}
	if wait, drop := s.ddlOwner(b.xid); wait != 0 {
		return nil, wait, drop, nil
	}
	if err := b.sees(s); err != nil {
		return nil, 0, false, err
	}
	return s, 0, false, nil
}

/**
* createIn crea el store name dentro del lote b. Un disco lleno limpio al crear su árbol deshace el alta (KindDropStore).
* Con ddlMu tomado.
* @param b *Batch, name string
* @return *Store, error
**/
func (e *Engine) createIn(b *Batch, name string) (*Store, error) {
	if err := e.writable(); err != nil {
		return nil, err
	}
	entry, err := e.createStore(name, b.xid)
	if err != nil {
		return nil, e.fail(err)
	}
	tree, err := e.openTree(entry.File)
	if err != nil {
		err = e.treeError(err)
		if !errors.Is(err, ErrFailed) {
			if derr := e.dropFiles(entry); derr != nil {
				return nil, e.fail(derr)
			}
		}
		return nil, err
	}
	s := &Store{e: e, name: name, file: entry.File, tree: tree, creator: b.xid}
	e.handles[name] = s
	b.creates = append(b.creates, s)
	b.wrote = true
	return s, nil
}

/**
* dropStoreIn borra el store name al confirmar el lote b: hasta entonces sigue entero para todos (las escrituras de
* otros lotes en él dan ErrStorePending; las sueltas siguen). Un store creado por el mismo lote se borra enseguida.
* Si otro lote abierto escribió en él devuelve ErrBusy (sin abortar b); si otro lote lo está creando o borrando,
* espera a que termine.
* @param b *Batch, name string
* @return error: ErrStoreNotFound, ErrBusy, ErrSerialization, ErrConflict, ErrTimeout, ErrStoreDropped (b ya lo
* borra), ErrStorePending, ErrClosed, ErrFailed
**/
func (e *Engine) dropStoreIn(b *Batch, name string) error {
	if err := b.acquire(e); err != nil {
		return err
	}
	defer b.mu.Unlock()
	var waited page.XID
	waitedDrop := false
	for {
		wait, drop, err := e.dropStoreInLocked(b, name, waitedDrop)
		if wait == 0 {
			if txError(err) {
				b.abortLocked(err)
			}
			return err
		}
		if err := b.waitDDL(name, wait, waited); err != nil {
			return err
		}
		waited, waitedDrop = wait, waitedDrop || drop
	}
}

/**
* dropStoreInLocked es una vuelta de dropStoreIn con el lote tomado.
* @param b *Batch, name string, waitedDrop bool: ya esperó a un lote que lo borraba
* @return page.XID: el lote al que esperar (0 = ninguno), bool: si ese lote lo borra, error
**/
func (e *Engine) dropStoreInLocked(b *Batch, name string, waitedDrop bool) (page.XID, bool, error) {
	if err := e.enter(); err != nil {
		return 0, false, err
	}
	defer e.leave()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	s, err := e.openLocked(name)
	switch {
	case err != nil:
		return 0, false, err
	case s == nil && waitedDrop:
		return 0, false, ErrSerialization
	case s == nil:
		return 0, false, fmt.Errorf(MSG_STORE_NAME, ErrStoreNotFound, name)
	}
	if wait, drop := s.ddlOwner(b.xid); wait != 0 {
		return wait, drop, nil
	}
	if s.creator == b.xid {
		return 0, false, b.dropOwnCreate(s)
	}
	if err := b.sees(s); err != nil {
		return 0, false, err
	}
	return 0, false, b.markDrop(s)
}

/**
* markDrop anota que b borra s al confirmar, si ningún otro lote abierto escribió en él. Con ddlMu tomado; toma
* s.mu en escritura (espera a las operaciones en curso del store y ninguna escritura de un lote queda a medias).
* @param s *Store
* @return error: ErrBusy
**/
func (b *Batch) markDrop(s *Store) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	others := s.batchWriters.Load()
	if _, ok := b.written[s]; ok {
		others--
	}
	if others > 0 {
		return fmt.Errorf(MSG_BUSY_WRITERS, ErrBusy, others, s.name)
	}
	s.dropper = b.xid
	b.drops = append(b.drops, s)
	b.wrote = true
	return nil
}

/**
* dropOwnCreate borra enseguida un store que creó el mismo lote (KindDropStore durable, como el aborto): nadie más
* lo vio. Con ddlMu tomado.
* @param s *Store
* @return error
**/
func (b *Batch) dropOwnCreate(s *Store) error {
	e := b.e
	s.markDropped()
	delete(e.handles, s.name)
	b.creates = slices.DeleteFunc(b.creates, func(c *Store) bool { return c == s })
	delete(b.deltas, s.file)
	if err := e.dropFiles(control.Entry{Name: s.name, File: s.file}); err != nil {
		return e.fail(err)
	}
	return nil
}

/**
* sees dice si el snapshot del lote ve el store: uno creado por otro lote que confirmó después de que b empezó no
* existía para b. Con ddlMu tomado.
* @param s *Store
* @return error: ErrStoreDropped (b lo borra) o ErrSerialization
**/
func (b *Batch) sees(s *Store) error {
	if s.dropper == b.xid {
		return fmt.Errorf(MSG_STORE_NAME, ErrStoreDropped, s.name)
	}
	if s.createdBy != 0 && !b.snap.Visible(s.createdBy, xact.StatusCommitted) {
		return ErrSerialization
	}
	return nil
}

/**
* waitDDL espera al lote wait que crea o borra name. Si ya lo esperó (waited) y sigue pendiente, ese lote terminó
* sin resolverlo (el motor quedó cerrado o inutilizable). Un tiempo vencido aborta b. Con b tomado.
* @param name string, wait, waited page.XID
* @return error: ErrStorePending, ErrConflict, ErrTimeout o el fallo del motor
**/
func (b *Batch) waitDDL(name string, wait, waited page.XID) error {
	if wait == waited {
		return fmt.Errorf(MSG_STORE_PENDING, ErrStorePending, name)
	}
	if werr := b.e.txs.Wait(wait, b.waitLimit()); werr != nil {
		err := ErrConflict
		if b.expired() {
			err = ErrTimeout
		}
		b.abortLocked(err)
		return err
	}
	return b.e.failed()
}

/**
* noteWrite anota que b escribe en s (DropStoreIn de otro lote espera a que b termine). Con s.mu en lectura.
* @param s *Store
**/
func (b *Batch) noteWrite(s *Store) {
	if _, ok := b.written[s]; !ok {
		b.written[s] = struct{}{}
		s.batchWriters.Add(1)
	}
}

/**
* forgetWrites descuenta a b de los stores en los que escribió (al terminar el lote).
**/
func (b *Batch) forgetWrites() {
	for s := range b.written {
		s.batchWriters.Add(-1)
	}
	clear(b.written)
}

/**
* hasDDL dice si el lote creó o borra algún store.
* @return bool
**/
func (b *Batch) hasDDL() bool {
	return len(b.creates) > 0 || len(b.drops) > 0
}

/**
* commitInfo arma los datos del KindCommit del lote: lo que suma a Count (sin los stores que borra) y su DDL.
* @return commitInfo
**/
func (b *Batch) commitInfo() commitInfo {
	c := commitInfo{deltas: b.deltas}
	for _, s := range b.creates {
		c.creates = append(c.creates, s.file)
	}
	for _, s := range b.drops {
		c.drops = append(c.drops, s.file)
	}
	return c
}

/**
* finishDDL termina un lote con DDL, con ddlMu tomado de punta a punta (el paso 3 del checkpoint guarda el catálogo
* antes o después, nunca en medio): el registro de confirmación siempre con fsync (los archivos de un store borrado
* se borran solo con el commit durable) o el de aborto, y después el catálogo.
* @param commit bool
* @return error
**/
func (b *Batch) finishDDL(commit bool) error {
	e := b.e
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	for _, s := range b.drops {
		delete(b.deltas, s.file)
	}
	if err := b.logEnd(commit, false); err != nil {
		return e.fail(err)
	}
	var err error
	if commit {
		err = b.applyDDL()
	} else {
		err = b.undoDDL()
	}
	if err != nil {
		return e.fail(err)
	}
	return nil
}

/**
* applyDDL aplica el DDL de un lote ya confirmado y durable: los creados dejan de estar pendientes y los borrados
* salen del catálogo y se borran sus archivos (un corte en medio lo termina la recuperación al rehacer el
* KindCommit). Con ddlMu tomado.
* @return error
**/
func (b *Batch) applyDDL() error {
	e := b.e
	for _, s := range b.creates {
		e.cat.ClearFlags(s.file, control.FlagPendingCreate)
		s.mu.Lock()
		s.creator, s.createdBy = 0, b.xid
		s.mu.Unlock()
	}
	for _, s := range b.drops {
		s.markDropped()
		delete(e.handles, s.name)
		if err := e.removeFiles(control.Entry{Name: s.name, File: s.file}); err != nil {
			return err
		}
	}
	if len(b.drops) == 0 {
		return e.storeCatalog()
	}
	return nil
}

/**
* undoDDL deshace el DDL de un lote abortado: los borrados siguen como estaban y los creados se borran con un
* KindDropStore durable (si no, al rehacer, su KindCreateStoreIn taparía un store suelto del mismo nombre creado
* después). Con ddlMu tomado.
* @return error
**/
func (b *Batch) undoDDL() error {
	e := b.e
	for _, s := range b.drops {
		s.mu.Lock()
		s.dropper = 0
		s.mu.Unlock()
	}
	for _, s := range b.creates {
		s.markDropped()
		delete(e.handles, s.name)
		if err := e.dropFiles(control.Entry{Name: s.name, File: s.file}); err != nil {
			return err
		}
	}
	return nil
}

/**
* pendingFiles son los números de archivo de las entradas del catálogo con FlagPendingCreate.
* @param entries []control.Entry
* @return []smgr.FileNo
**/
func pendingFiles(entries []control.Entry) []smgr.FileNo {
	var files []smgr.FileNo
	for _, entry := range entries {
		if entry.Flags&control.FlagPendingCreate != 0 {
			files = append(files, entry.File)
		}
	}
	return files
}
