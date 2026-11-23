package store

import (
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/smgr"
)

// Migración perezosa de las hojas al formato 2 (tarea 134 de F-23; doc.go «Formato 2: migración desde el formato 1
// (e)»): un store del formato 1 tiene hojas de la forma 1 hasta que una escritura o la limpieza las reescribe; la
// limpieza de un store que no está migrado recorre todas sus hojas (sin saltar por el mapa) y al terminar lo anota en
// su meta (btree.MetaLeavesForm2).

// leafMigration son los stores abiertos cuyas hojas todavía no son todas de la forma 2, y si el directorio era del
// formato 1 al abrir (la limpieza de fondo corre entonces aunque no haya escrituras, hasta una pasada entera).
type leafMigration struct {
	mu      sync.Mutex
	pending map[smgr.FileNo]struct{}
	fromV1  atomic.Bool
}

/**
* note anota si las hojas del store file son todas de la forma 2.
* @param file smgr.FileNo, form2 bool
**/
func (m *leafMigration) note(file smgr.FileNo, form2 bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if form2 {
		delete(m.pending, file)
		return
	}
	if m.pending == nil {
		m.pending = map[smgr.FileNo]struct{}{}
	}
	m.pending[file] = struct{}{}
}

/**
* forget olvida un store borrado o vaciado (su archivo viejo).
* @param file smgr.FileNo
**/
func (m *leafMigration) forget(file smgr.FileNo) {
	m.note(file, true)
}

/**
* count es cuántos stores abiertos tienen hojas de la forma 1.
* @return int
**/
func (m *leafMigration) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.pending)
}

/**
* due dice si la limpieza de fondo tiene que correr para terminar la migración: el directorio era del formato 1 al
* abrir (todavía no hubo una pasada entera) o un store abierto tiene hojas de la forma 1.
* @return bool
**/
func (m *leafMigration) due() bool {
	return m.fromV1.Load() || m.count() > 0
}
