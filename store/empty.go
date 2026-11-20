package store

import (
	"fmt"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* empty vacía el store en O(1), como el TRUNCATE de PostgreSQL: un archivo de datos nuevo (relfilenode nuevo), el
* catálogo apuntando a él y el archivo viejo borrado. No es MVCC: con lotes o snapshots abiertos en el motor
* devuelve ErrBusy (cualquiera podría leer el store después, y un lote podría tener versiones sin confirmar en él);
* mientras vacía no empieza ninguno (txGate) y espera a las operaciones en curso del store (Store.mu).
* @return error: ErrBusy, ErrStoreDropped, ErrClosed, ErrFailed
**/
func (s *Store) empty() error {
	e := s.e
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	e.txGate.Lock()
	defer e.txGate.Unlock()
	if n := e.openTx.Load(); n > 0 {
		return fmt.Errorf(MSG_BUSY, ErrBusy, n)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped {
		return ErrStoreDropped
	}
	file, tree, err := e.emptyFiles(s.name, s.file)
	if err != nil {
		return err
	}
	s.file, s.tree = file, tree
	return nil
}

/**
* emptyFiles hace el vaciado en el orden del WAL: KindEmptyStore durable, catálogo con el archivo nuevo (y su
* contador en 0), caché y archivos del viejo, y el árbol nuevo (KindInitStore). Con ddlMu tomado. Un fallo después
* del registro deja el motor inutilizable: la recuperación lo termina.
* @param name string, old smgr.FileNo
* @return smgr.FileNo: el archivo nuevo, *btree.BTree: su árbol, error
**/
func (e *Engine) emptyFiles(name string, old smgr.FileNo) (smgr.FileNo, *btree.BTree, error) {
	file, err := e.cat.AllocFileNo()
	if err != nil {
		return 0, nil, err
	}
	if _, err := e.logRecord(wal.KindEmptyStore, encodeEmptyMain(old, file, name)); err != nil {
		return 0, nil, e.fail(err)
	}
	if _, err := e.cat.Empty(name, old, file); err != nil {
		return 0, nil, e.fail(err)
	}
	e.counts.remove(old)
	e.counts.set(file, 0)
	e.frozen.forget(old)
	if err := e.storeCatalog(); err != nil {
		return 0, nil, e.fail(err)
	}
	e.pool.DropFile(old)
	if err := e.sm.Unlink(old); err != nil {
		return 0, nil, e.fail(err)
	}
	tree, err := e.openTree(file)
	if err != nil {
		return 0, nil, e.fail(err)
	}
	return file, tree, nil
}
