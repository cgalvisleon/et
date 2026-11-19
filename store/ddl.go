package store

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* store abre el store name, o lo crea si no existe. Pedirlo dos veces devuelve el mismo *Store.
* @param name string: UTF-8 válido sin nulos, de 1 a 255 bytes
* @return *Store, error: ErrBadName, ErrClosed, ErrFailed
**/
func (e *Engine) store(name string) (*Store, error) {
	if !validName(name) {
		return nil, fmt.Errorf(MSG_STORE_NAME, ErrBadName, name)
	}
	return e.handle(name, true)
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
* nil.
* @param name string, create bool
* @return *Store, error
**/
func (e *Engine) handle(name string, create bool) (*Store, error) {
	if err := e.enter(); err != nil {
		return nil, err
	}
	defer e.leave()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	if s := e.handles[name]; s != nil {
		return s, nil
	}
	entry, ok := e.cat.Get(name)
	if !ok && !create {
		return nil, nil
	}
	if !ok {
		var err error
		if entry, err = e.createStore(name); err != nil {
			return nil, e.fail(err)
		}
	}
	tree, err := e.openTree(entry.File)
	if err != nil {
		return nil, err
	}
	s := &Store{e: e, name: name, file: entry.File, tree: tree}
	e.handles[name] = s
	return s, nil
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
* openTree. Con ddlMu tomado.
* @param name string
* @return control.Entry, error
**/
func (e *Engine) createStore(name string) (control.Entry, error) {
	file, err := e.cat.AllocFileNo()
	if err != nil {
		return control.Entry{}, err
	}
	if _, err := e.logRecord(wal.KindCreateStore, encodeStoreMain(file, name)); err != nil {
		return control.Entry{}, err
	}
	if _, err := e.cat.Create(name, file); err != nil {
		return control.Entry{}, err
	}
	e.counts.set(file, 0)
	if err := e.catFile.Save(e.cat.Catalog()); err != nil {
		return control.Entry{}, err
	}
	return control.Entry{Name: name, File: file}, nil
}

/**
* openTree abre el árbol de un store del catálogo. Si su página meta no existe o está en ceros (un store nuevo, o
* uno cuyo KindInitStore no llegó a ser durable antes de un corte), borra lo que haya de sus archivos y lo crea:
* ningún registro durable apunta a sus páginas. Con ddlMu tomado.
* @param file smgr.FileNo
* @return *btree.BTree, error
**/
func (e *Engine) openTree(file smgr.FileNo) (*btree.BTree, error) {
	cfg := btree.Config{
		File: file, Pool: e.pool, Smgr: e.sm, Free: e.free, Log: e.log, Status: e.status, HintGate: &e.hintGate,
	}
	ready, err := e.initialized(file)
	if err != nil {
		return nil, e.readError(err)
	}
	if ready {
		return btree.Open(cfg)
	}
	e.pool.DropFile(file)
	if err := e.sm.Unlink(file); err != nil {
		return nil, e.fail(err)
	}
	tree, err := btree.Create(cfg)
	if err != nil {
		return nil, e.fail(err)
	}
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
	e.cat.Drop(entry.Name, entry.File)
	e.counts.remove(entry.File)
	if err := e.catFile.Save(e.cat.Catalog()); err != nil {
		return err
	}
	e.pool.DropFile(entry.File)
	return e.sm.Unlink(entry.File)
}

/**
* storeNames devuelve los nombres de los stores, en orden.
* @return []string
**/
func (e *Engine) storeNames() []string {
	return e.cat.Names()
}
