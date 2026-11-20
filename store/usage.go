package store

import (
	"errors"
	iofs "io/fs"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// Uso de disco del motor (ver «Presión y uso de disco» en doc.go).

// DiskUsage es lo que ocupa en disco un motor, en bytes. Los tamaños salen del largo de los archivos (o de smgr para
// los stores), sin leer páginas. No es una foto atómica: Total y Stores se miden uno tras otro.
type DiskUsage struct {
	Total   int64            // todo el directorio: Data + WAL + Xact + Control
	Data    int64            // data/: los archivos de los stores (también los huérfanos que Repair borraría)
	WAL     int64            // wal/: los segmentos, con lo preasignado y los repuestos
	Xact    int64            // xact/: los archivos de estados de las transacciones
	Control int64            // la raíz: control, catalog, LOCK (y catalog.tmp si hay un guardado a medias)
	Stores  map[string]int64 // cada store: sus dos forks (árbol y mapa de visibilidad), según smgr
}

/**
* dirUsage mide el disco del motor: el largo de cada archivo de su directorio y, por store, los bloques de sus forks.
* Toma closeMu en lectura y ddlMu solo mientras pregunta a smgr por los stores (ningún store se crea, borra ni vacía
* a la vez).
* @return DiskUsage, error: ErrClosed con el motor cerrado
**/
func (e *Engine) dirUsage() (DiskUsage, error) {
	e.closeMu.RLock()
	defer e.closeMu.RUnlock()
	if e.closed {
		return DiskUsage{}, ErrClosed
	}
	stores, err := e.storesUsage()
	if err != nil {
		return DiskUsage{}, err
	}
	u := DiskUsage{Stores: stores}
	parts := []struct {
		dir string
		sum *int64
	}{
		{e.dir, &u.Control},
		{filepath.Join(e.dir, smgr.DataDir), &u.Data},
		{filepath.Join(e.dir, wal.Dir), &u.WAL},
		{filepath.Join(e.dir, xact.Dir), &u.Xact},
	}
	for _, p := range parts {
		if *p.sum, err = e.filesSize(p.dir); err != nil {
			return DiskUsage{}, err
		}
	}
	u.Total = u.Data + u.WAL + u.Xact + u.Control
	return u, nil
}

/**
* storesUsage mide cada store del catálogo con smgr, con ddlMu tomado.
* @return map[string]int64, error
**/
func (e *Engine) storesUsage() (map[string]int64, error) {
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	entries := e.cat.Catalog().Entries
	result := make(map[string]int64, len(entries))
	for _, entry := range entries {
		size, err := e.fileUsage(entry.File)
		if err != nil {
			return nil, err
		}
		result[entry.Name] = size
	}
	return result, nil
}

/**
* fileUsage son los bytes de los dos forks de un número de archivo; un fork que no existe mide 0.
* @param file smgr.FileNo
* @return int64, error
**/
func (e *Engine) fileUsage(file smgr.FileNo) (int64, error) {
	var total int64
	for _, fork := range []smgr.Fork{smgr.ForkMain, smgr.ForkVM} {
		n, err := e.sm.NBlocks(file, fork)
		switch {
		case errors.Is(err, smgr.ErrNoFork):
			continue
		case err != nil:
			return 0, err
		}
		total += int64(n) * page.Size
	}
	return total, nil
}

/**
* filesSize suma el largo de los archivos de dir (sin bajar a los subdirectorios del motor). Un directorio que falta
* mide 0 y un archivo que se borra mientras tanto no cuenta.
* @param dir string
* @return int64, error
**/
func (e *Engine) filesSize(dir string) (int64, error) {
	names, err := e.fs.ReadDir(dir)
	if errors.Is(err, iofs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var total int64
	for _, name := range names {
		if dir == e.dir && isEngineDir(name) {
			continue
		}
		size, err := e.fileSize(filepath.Join(dir, name))
		if err != nil {
			return 0, err
		}
		total += size
	}
	return total, nil
}

/**
* fileSize es el largo de un archivo, 0 si ya no existe.
* @param name string
* @return int64, error
**/
func (e *Engine) fileSize(name string) (int64, error) {
	f, err := e.fs.OpenFile(name, os.O_RDONLY)
	if errors.Is(err, iofs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	size, err := f.Size()
	return size, errors.Join(err, f.Close())
}

/**
* isEngineDir dice si name es uno de los subdirectorios del motor (data, wal, xact).
* @param name string
* @return bool
**/
func isEngineDir(name string) bool {
	return name == smgr.DataDir || name == wal.Dir || name == xact.Dir
}

/**
* dirUsage son los bytes en disco del store: sus dos forks según smgr, sin leer páginas.
* @return int64, error: ErrClosed o ErrStoreDropped
**/
func (s *Store) dirUsage() (int64, error) {
	if err := s.begin(); err != nil {
		return 0, err
	}
	defer s.end()
	return s.e.fileUsage(s.file)
}
