package store

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* store abre el store name, o lo crea si no existe. Pedirlo dos veces devuelve el mismo *Store. Si lo está creando
* un lote abierto, espera hasta LOCK_TIMEOUT a que termine: confirmado, lo devuelve; deshecho, lo crea.
* @param name string: UTF-8 válido sin nulos, de 1 a 255 bytes
* @return *Store, error: ErrBadName, ErrStorePending, ErrClosed, ErrFailed
**/
func (e *Engine) store(name string) (*Store, error) {
	if !validName(name) {
		return nil, fmt.Errorf(MSG_STORE_NAME, ErrBadName, name)
	}
	var waited page.XID
	for {
		s, err := e.handle(name, true)
		if err != nil {
			return nil, err
		}
		creator := s.pendingCreator()
		if creator == 0 {
			return s, nil
		}
		if creator == waited || e.txs.Wait(creator, e.cfg.lockTimeout) != nil {
			return nil, fmt.Errorf(MSG_STORE_PENDING, ErrStorePending, name) // el lote no terminó, o terminó sin resolverlo
		}
		waited = creator
	}
}

/**
* existing abre el store name solo si está en el catálogo (la limpieza no debe crear uno que se acaba de borrar).
* @param name string
* @return *Store: nil si no existe, error: ErrClosed, ErrFailed
**/
func (e *Engine) existing(name string) (*Store, error) {
	return e.handle(name, false)
}

/**
* handle devuelve el *Store abierto de name, o abre su árbol; si no está en el catálogo lo crea (create) o devuelve
* nil. Abrir uno que existe también anda con el motor de solo lectura; crearlo necesita el motor usable y lugar.
* @param name string, create bool
* @return *Store, error
**/
func (e *Engine) handle(name string, create bool) (*Store, error) {
	if err := e.enterRead(); err != nil {
		return nil, err
	}
	defer e.leave()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	s, err := e.openLocked(name)
	if err != nil || s != nil || !create {
		return s, err
	}
	if err := e.writable(); err != nil {
		return nil, err
	}
	entry, err := e.createStore(name, 0)
	if err != nil {
		return nil, e.fail(err)
	}
	return e.openEntry(entry)
}

/**
* openLocked devuelve el *Store abierto de name, o abre el árbol del que está en el catálogo. Con ddlMu tomado.
* @param name string
* @return *Store: nil si no está en el catálogo, error
**/
func (e *Engine) openLocked(name string) (*Store, error) {
	if s := e.handles[name]; s != nil {
		return s, nil
	}
	entry, ok := e.cat.Get(name)
	if !ok {
		return nil, nil
	}
	return e.openEntry(entry)
}

/**
* openEntry abre (o crea, si no tiene página meta) el árbol de un store del catálogo y guarda su *Store. Con ddlMu
* tomado.
* @param entry control.Entry
* @return *Store, error
**/
func (e *Engine) openEntry(entry control.Entry) (*Store, error) {
	tree, err := e.openTree(entry.File)
	if err != nil {
		return nil, e.treeError(err)
	}
	s := &Store{e: e, name: entry.Name, file: entry.File, tree: tree}
	e.handles[entry.Name] = s
	return s, nil
}

/**
* treeError trata el error de abrir o crear el árbol de un store del catálogo: un disco lleno limpio al crearlo
* vuelve como ErrDiskFull sin dejar el motor inutilizable (el store queda en el catálogo sin páginas válidas, y el
* próximo Store lo crea de nuevo, como tras un corte antes de su KindInitStore); otro fallo de escritura sí lo deja.
* Los errores de lectura ya vienen tratados por openTree.
* @param err error
* @return error
**/
func (e *Engine) treeError(err error) error {
	if errors.Is(err, ErrFailed) || errors.Is(err, ErrDiskFull) {
		return err
	}
	return e.writeError(err)
}

/**
* storeCatalog guarda el catálogo tras un cambio de stores ya durable en el WAL (crear, borrar, vaciar). Con el disco
* lleno al escribir catalog.tmp (limpio) no falla: el catálogo anterior sigue entero, la recuperación rehace el cambio
* desde el WAL (como tras un corte antes de guardarlo) y el próximo checkpoint guarda el catálogo antes de reciclar ese
* WAL.
* @return error
**/
func (e *Engine) storeCatalog() error {
	err := e.catFile.Save(e.cat.Catalog())
	switch {
	case err == nil:
		e.catalogBehind.Store(false)
	case e.cleanNoSpace(err):
		e.catalogBehind.Store(true)
		return nil
	}
	return err
}

/**
* validName dice si name sirve de nombre de store (pregunta 57).
* @param name string
* @return bool
**/
func validName(name string) bool {
	return len(name) >= 1 && len(name) <= MaxNameLen && utf8.ValidString(name) && !strings.ContainsRune(name, 0)
}

/**
* createStore da de alta un store: KindCreateStore en el WAL (durable), después el catálogo. Las páginas las crea
* openTree. Dentro de un lote (xid ≠ 0) el registro es KindCreateStoreIn con el xid del lote y la entrada queda con
* FlagPendingCreate hasta el commit. Con ddlMu tomado.
* @param name string, xid page.XID: el lote que lo crea, 0 = suelto
* @return control.Entry, error
**/
func (e *Engine) createStore(name string, xid page.XID) (control.Entry, error) {
	file, err := e.cat.AllocFileNo()
	if err != nil {
		return control.Entry{}, err
	}
	kind, flags := wal.KindCreateStore, uint32(0)
	if xid != 0 {
		kind, flags = wal.KindCreateStoreIn, control.FlagPendingCreate
	}
	if _, err := e.logTxRecord(xid, kind, encodeStoreMain(file, name)); err != nil {
		return control.Entry{}, err
	}
	if _, err := e.cat.Create(name, file); err != nil {
		return control.Entry{}, err
	}
	e.cat.SetFlags(file, flags)
	e.counts.set(file, 0)
	if err := e.storeCatalog(); err != nil {
		return control.Entry{}, err
	}
	return control.Entry{Name: name, File: file}, nil
}

/**
* openTree abre el árbol de un store del catálogo. Si su página meta no existe o está en ceros (un store nuevo, o
* uno cuyo KindInitStore no llegó a ser durable antes de un corte), borra lo que haya de sus archivos y lo crea:
* ningún registro durable apunta a sus páginas. Un store nuevo nace con frozenXID = el xid vivo más viejo: ninguna
* transacción que pueda escribir en él tiene un xid menor. Con ddlMu tomado.
* @param file smgr.FileNo
* @return *btree.BTree, error
**/
func (e *Engine) openTree(file smgr.FileNo) (*btree.BTree, error) {
	cfg := e.treeConfig(file)
	ready, err := e.initialized(file)
	if err != nil {
		return nil, e.readError(err)
	}
	if ready {
		return e.openExisting(cfg)
	}
	if err := e.failed(); err != nil {
		return nil, err // de solo lectura: no se crea nada
	}
	e.pool.DropFile(file)
	if err := e.sm.Unlink(file); err != nil {
		return nil, e.fail(err)
	}
	tree, err := btree.Create(cfg)
	if err != nil {
		return nil, err // quien llama decide: handle admite el disco lleno (ver treeError), Empty no
	}
	e.frozen.set(file, cfg.FrozenXID)
	return tree, nil
}

/**
* treeConfig arma la configuración del árbol de un store: las piezas del motor y, para un store nuevo, frozenXID = el
* xid vivo más viejo.
* @param file smgr.FileNo
* @return btree.Config
**/
func (e *Engine) treeConfig(file smgr.FileNo) btree.Config {
	return btree.Config{
		File: file, Pool: e.pool, Smgr: e.sm, Free: e.free, Log: e.log, Status: e.status, HintGate: &e.hintGate,
		VM: e.free, FrozenXID: e.txs.OldestActive(),
	}
}

/**
* openExisting abre el árbol de un store con su página meta escrita y anota su frozenXID.
* @param cfg btree.Config
* @return *btree.BTree, error
**/
func (e *Engine) openExisting(cfg btree.Config) (*btree.BTree, error) {
	tree, err := btree.Open(cfg)
	if err != nil {
		return nil, err
	}
	frozen, err := tree.FrozenXID()
	if err != nil {
		return nil, e.readError(err)
	}
	e.frozen.set(cfg.File, frozen)
	return tree, nil
}

/**
* initialized dice si el store tiene su página meta escrita.
* @param file smgr.FileNo
* @return bool, error
**/
func (e *Engine) initialized(file smgr.FileNo) (bool, error) {
	n, err := e.sm.NBlocks(file, smgr.ForkMain)
	if errors.Is(err, smgr.ErrNoFork) || err == nil && n == 0 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	h, err := e.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: btree.MetaBlock}, buffer.Normal)
	if err != nil {
		return false, err
	}
	defer h.Release()
	h.Lock(buffer.Share)
	return !h.Page().IsNew(), nil
}

/**
* dropStore borra el store name: espera a sus operaciones en curso, escribe KindDropStore (durable), lo saca del
* catálogo y borra sus páginas de la caché y sus archivos. Su *Store queda inservible (ErrStoreDropped).
* @param name string
* @return error: ErrStoreNotFound, ErrClosed, ErrFailed
**/
func (e *Engine) dropStore(name string) error {
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	entry, ok := e.cat.Get(name)
	if !ok {
		return fmt.Errorf(MSG_STORE_NAME, ErrStoreNotFound, name)
	}
	if s := e.handles[name]; s != nil {
		if s.creator != 0 || s.dropper != 0 {
			return fmt.Errorf(MSG_STORE_PENDING, ErrStorePending, name)
		}
		s.markDropped()
		delete(e.handles, name)
	}
	if err := e.dropFiles(entry); err != nil {
		return e.fail(err)
	}
	return nil
}

/**
* dropFiles hace la baja en el orden del WAL: registro durable, catálogo, caché y archivos. Con ddlMu tomado.
* @param entry control.Entry
* @return error
**/
func (e *Engine) dropFiles(entry control.Entry) error {
	if _, err := e.logRecord(wal.KindDropStore, encodeStoreMain(entry.File, entry.Name)); err != nil {
		return err
	}
	return e.removeFiles(entry)
}

/**
* removeFiles saca el store del catálogo (y lo guarda) y borra sus páginas de la caché y sus archivos, con el
* registro que lo justifica ya durable (KindDropStore, o el KindCommit de un lote que lo borra). Con ddlMu tomado.
* @param entry control.Entry
* @return error
**/
func (e *Engine) removeFiles(entry control.Entry) error {
	e.cat.Drop(entry.Name, entry.File)
	e.counts.remove(entry.File)
	e.frozen.forget(entry.File)
	if err := e.storeCatalog(); err != nil {
		return err
	}
	e.pool.DropFile(entry.File)
	return e.unlinkData(entry.File)
}

/**
* storeNames devuelve los nombres de los stores, en orden, sin los que está creando un lote abierto.
* @return []string
**/
func (e *Engine) storeNames() []string {
	entries := e.cat.Catalog().Entries
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Flags&control.FlagPendingCreate == 0 {
			names = append(names, entry.Name)
		}
	}
	return names
}
