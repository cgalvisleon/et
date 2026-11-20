package store

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"strconv"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// Verificación y reparación (fase 3, tarea 45 de F-23): ver «Verificación y reparación» en doc.go. Este archivo
// tiene el recorrido del motor (control, catálogo, archivos de datos y cada store); storecheck.go, el de un store.

/**
* check es Verify (repair false) y Repair (repair true): revisa el control, el catálogo y los archivos de datos, y
* después cada store del catálogo, uno a la vez. Con repair arregla lo reparable en el momento en que lo encuentra.
* @param repair bool
* @return Report, error: ErrClosed, ErrFailed (un fallo de escritura o de la caché); el daño va en el informe
**/
func (e *Engine) check(repair bool) (Report, error) {
	var r Report
	if err := e.checkFiles(&r, repair); err != nil {
		return r, err
	}
	if err := e.enter(); err != nil {
		return r, err
	}
	names := e.cat.Names()
	e.leave()
	for _, name := range names {
		if err := e.checkStore(&r, name, repair); err != nil {
			return r, err
		}
	}
	return r, nil
}

/**
* checkFiles revisa el control, el catálogo guardado y los archivos de data/, con ckptMu (ningún checkpoint escribe
* el control ni el catálogo mientras tanto) y ddlMu (ningún store se crea, vacía ni borra).
* @param r *Report, repair bool: borra los archivos huérfanos
* @return error
**/
func (e *Engine) checkFiles(r *Report, repair bool) error {
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	e.ckptMu.Lock()
	defer e.ckptMu.Unlock()
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	e.checkControl(r)
	e.checkCatalog(r)
	return e.checkDataFiles(r, repair)
}

/**
* checkControl lee el control del disco y lo compara con el motor: el punto de rehacer es el del último checkpoint,
* el checkpoint no es anterior a él y el frozenXID no pasa al siguiente xid. Un motor que todavía no hizo su primer
* checkpoint no tiene control.
* @param r *Report
**/
func (e *Engine) checkControl(r *Report) {
	if !e.hasControl {
		return
	}
	add := func(format string, args ...any) {
		r.add(Finding{Kind: FindControl, File: control.FileName, Block: -1, Desc: fmt.Sprintf(format, args...)})
	}
	ctl, err := e.ctl.Read()
	if err != nil {
		add(MSG_CHECK_CONTROL_READ, err)
		return
	}
	if redo := e.log.RedoPoint().LSN; ctl.Redo.LSN != redo {
		add(MSG_CHECK_CONTROL_REDO, ctl.Redo.LSN, redo)
	}
	if ctl.CheckpointLSN < ctl.Redo.LSN {
		add(MSG_CHECK_CONTROL_CKPT, ctl.CheckpointLSN, ctl.Redo.LSN)
	}
	if next := e.txs.NextXID(); ctl.FrozenXID > next {
		add(MSG_CHECK_CONTROL_XID, ctl.FrozenXID, next)
	}
}

/**
* checkCatalog lee el catálogo del disco sin tocarlo y lo compara con el de memoria: los mismos stores con los mismos
* números de archivo (los contadores pueden diferir: el guardado es el del último checkpoint). Si un cambio de stores
* no pudo guardarlo por disco lleno (catalogBehind), el guardado está atrás a propósito y solo se comprueba que se lea.
* @param r *Report
**/
func (e *Engine) checkCatalog(r *Report) {
	add := func(format string, args ...any) {
		r.add(Finding{Kind: FindCatalog, File: control.CatalogFileName, Block: -1, Desc: fmt.Sprintf(format, args...)})
	}
	saved, err := e.catFile.Read()
	if err != nil {
		add(MSG_CHECK_CATALOG_READ, err)
		return
	}
	if e.catalogBehind.Load() {
		return // el WAL tiene los cambios que le faltan: la recuperación los rehace
	}
	mem := e.cat.Catalog()
	if len(saved.Entries) != len(mem.Entries) {
		add(MSG_CHECK_CATALOG_LEN, len(saved.Entries), len(mem.Entries))
	}
	for _, s := range saved.Entries {
		if m, ok := e.cat.Get(s.Name); !ok || m.File != s.File {
			add(MSG_CHECK_CATALOG_STORE, s.Name, s.File, m.File)
		}
	}
	if saved.NextFileNo > mem.NextFileNo {
		add(MSG_CHECK_CATALOG_NEXT, saved.NextFileNo, mem.NextFileNo)
	}
}

/**
* checkDataFiles busca en data/ los archivos de números de archivo que ningún store del catálogo tiene (huérfanos de
* un DropStore o un Empty cortados entre el catálogo y el borrado) y los nombres que no son de archivos de datos. Con
* repair borra los huérfanos (los números no se reutilizan: nadie los va a pedir). Con ddlMu tomado.
* @param r *Report, repair bool
* @return error: ErrFailed si un borrado falla
**/
func (e *Engine) checkDataFiles(r *Report, repair bool) error {
	dir := filepath.Join(e.dir, smgr.DataDir)
	names, err := e.fs.ReadDir(dir)
	if err != nil {
		r.add(Finding{Kind: FindUnknownFile, File: smgr.DataDir, Block: -1, Desc: fmt.Sprintf(MSG_CHECK_DATA_DIR, err)})
		return nil
	}
	var run []string
	var runFile smgr.FileNo
	for _, name := range names {
		file, ok := parseDataName(name)
		if !ok {
			r.add(Finding{Kind: FindUnknownFile, File: filepath.Join(smgr.DataDir, name), Block: -1, Desc: MSG_CHECK_UNKNOWN_FILE})
			continue
		}
		if file != runFile {
			if err := e.checkFileRun(r, runFile, run, repair); err != nil {
				return err
			}
			runFile, run = file, run[:0]
		}
		run = append(run, name)
	}
	return e.checkFileRun(r, runFile, run, repair)
}

/**
* checkFileRun revisa los archivos de un mismo número (ReadDir los da juntos, en orden): si ningún store lo tiene,
* son huérfanos.
* @param r *Report, file smgr.FileNo, names []string, repair bool
* @return error
**/
func (e *Engine) checkFileRun(r *Report, file smgr.FileNo, names []string, repair bool) error {
	if len(names) == 0 {
		return nil
	}
	if _, ok := e.cat.ByFile(file); ok {
		return nil
	}
	r.OrphanFiles++
	i := r.add(Finding{Kind: FindOrphanFile, File: dataFileName(file), Block: -1, Repairable: true,
		Desc: fmt.Sprintf(MSG_CHECK_ORPHAN_FILE, len(names), file)})
	if !repair {
		return nil
	}
	if err := e.removeOrphan(file, names); err != nil {
		return e.fail(err)
	}
	r.repaired(i)
	return nil
}

/**
* removeOrphan borra los archivos de un número sin store: primero por smgr (que olvida su estado) y después los que
* queden con hueco, por nombre; al final fsync de data/.
* @param file smgr.FileNo, names []string: los nombres que tiene en data/
* @return error
**/
func (e *Engine) removeOrphan(file smgr.FileNo, names []string) error {
	e.pool.DropFile(file)
	if err := e.sm.Unlink(file); err != nil {
		return err
	}
	dir := filepath.Join(e.dir, smgr.DataDir)
	for _, name := range names {
		if err := e.fs.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, iofs.ErrNotExist) {
			return err
		}
	}
	return e.fs.SyncDir(dir)
}

/**
* parseDataName lee el número de archivo de un nombre de data/: <fileNo:%010d>.<n:%04d> o <fileNo:%010d>_vm.<n:%04d>.
* @param name string
* @return smgr.FileNo, bool: si es un nombre de archivo de datos
**/
func parseDataName(name string) (smgr.FileNo, bool) {
	var seg string
	switch {
	case len(name) == 15 && name[10] == '.':
		seg = name[11:]
	case len(name) == 18 && name[10:14] == "_vm.":
		seg = name[14:]
	default:
		return 0, false
	}
	file, err := strconv.ParseUint(name[:10], 10, 32)
	if err != nil || file == 0 || !digits(name[:10]) || !digits(seg) {
		return 0, false
	}
	if n, err := strconv.ParseUint(seg, 10, 32); err != nil || n == 0 {
		return 0, false
	}
	return smgr.FileNo(file), true
}

/**
* digits dice si s son solo dígitos decimales.
* @param s string
* @return bool
**/
func digits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

/**
* dataFileName es el nombre de un número de archivo en los hallazgos: data/<fileNo:%010d>.
* @param file smgr.FileNo
* @return string
**/
func dataFileName(file smgr.FileNo) string {
	return filepath.Join(smgr.DataDir, fmt.Sprintf("%010d", file))
}

/**
* checkStore revisa (y con repair, arregla) el store name con vacuumMu y el store tomado en escritura: espera a sus
* operaciones en curso y frena las nuevas de ese store mientras dura; los demás stores siguen. Un store que se borró
* entre la lista y su turno se salta.
* @param r *Report, name string, repair bool
* @return error
**/
func (e *Engine) checkStore(r *Report, name string, repair bool) error {
	s, err := e.checkHandle(r, name)
	if err != nil || s == nil {
		return err
	}
	s.vacuumMu.Lock()
	defer s.vacuumMu.Unlock()
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped {
		return nil
	}
	return newStoreCheck(s, r, repair).run()
}

/**
* checkHandle da el *Store de name para revisarlo sin cambiar nada: el abierto, o abre su árbol si su página meta
* existe. Un store sin meta (Store lo crearía vacío) o con la meta ilegible queda en el informe y no se abre.
* @param r *Report, name string
* @return *Store: nil si no hay que revisarlo, error: ErrClosed o ErrFailed
**/
func (e *Engine) checkHandle(r *Report, name string) (*Store, error) {
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
	if !ok {
		return nil, nil
	}
	ready, err := e.initialized(entry.File)
	if err == nil && !ready {
		r.add(Finding{Kind: FindNoMeta, Store: name, File: dataFileName(entry.File), Block: -1, Desc: MSG_CHECK_NO_META})
		return nil, nil
	}
	var tree *btree.BTree
	if err == nil {
		tree, err = e.openExisting(e.treeConfig(entry.File))
	}
	if err != nil {
		if err := e.fatal(err); err != nil {
			return nil, err
		}
		r.add(Finding{Kind: FindTree, Store: name, File: dataFileName(entry.File), Block: int64(btree.MetaBlock), Blocks: 1,
			Desc: fmt.Sprintf(MSG_CHECK_META, err)})
		return nil, nil
	}
	s := &Store{e: e, name: name, file: entry.File, tree: tree}
	e.handles[name] = s
	return s, nil
}

/**
* fatal dice si el error de una lectura dejó el motor inutilizable (la caché o el WAL fallaron); una página dañada no.
* @param err error
* @return error: ErrFailed con la causa, o nil si es un daño que va al informe
**/
func (e *Engine) fatal(err error) error {
	if err = e.readError(err); errors.Is(err, ErrFailed) {
		return err
	}
	return nil
}
