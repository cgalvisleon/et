package btree

import (
	"bytes"
	"cmp"
	"math"

	"github.com/cgalvisleon/et/store/internal/page"
)

// MaxLSN hace de «infinito» en la segunda parte de la clave: como el lsn ordena descendente, (id, MaxLSN) va antes
// que toda versión de id. Un pivote sin PivotHasLSN vale (prefijo, MaxLSN).
const MaxLSN page.LSN = math.MaxUint64

// key es una clave del árbol: (id ascendente por bytes, lsn descendente). Es la misma para entradas, pivotes y
// búsquedas: un pivote (k, l) manda a la derecha toda clave ≥ (k, l). top es mayor que toda clave (recorrer hacia
// atrás desde el final). before es una clave de búsqueda apenas menor que (id, lsn): baja hasta la página cuya clave
// alta es (id, lsn), no hasta su hermana derecha (como pivotsearch de nbtree; lo usa el borrado de páginas para hallar
// el padre). id puede ser una vista dentro de una página: vale mientras la página siga bloqueada.
type key struct {
	id     []byte
	lsn    page.LSN
	top    bool
	before bool
}

// minusInf es la clave del primer enlace de una página interna: menor que toda clave real (los ids miden 1 o más).
var minusInf = key{lsn: MaxLSN}

/**
* searchKey es la clave que va antes que todas las versiones de id: la de una escritura nueva (cuyo lsn todavía no se
* conoce y será el mayor) y la de una lectura de la versión más nueva.
* @param id []byte
* @return key
**/
func searchKey(id []byte) key {
	return key{id: id, lsn: MaxLSN}
}

/**
* compareKeys ordena dos claves: id ascendente, después lsn descendente (la versión más nueva primero); con todo
* igual, la que tiene before va primero.
* @param a, b key
* @return int: -1, 0 o 1
**/
func compareKeys(a, b key) int {
	switch {
	case a.top && b.top:
		return 0
	case a.top:
		return 1
	case b.top:
		return -1
	}
	if c := bytes.Compare(a.id, b.id); c != 0 {
		return c
	}
	if c := cmp.Compare(b.lsn, a.lsn); c != 0 {
		return c
	}
	switch {
	case a.before == b.before:
		return 0
	case a.before:
		return -1
	}
	return 1
}

/**
* separator es el pivote más corto s con last < s ≤ first (truncado de sufijos de nbtree): si las dos claves son del
* mismo id, el id con el lsn de first; si no, el prefijo más corto de first.id que ya es mayor que last.id, sin lsn.
* @param last key: la última clave que queda a la izquierda
* @param first key: la primera que pasa a la derecha (last < first)
* @return key: copia, no una vista de la página
**/
func separator(last, first key) key {
	if bytes.Equal(last.id, first.id) {
		return key{id: bytes.Clone(first.id), lsn: first.lsn}
	}
	n := commonPrefix(last.id, first.id) + 1
	return key{id: bytes.Clone(first.id[:n]), lsn: MaxLSN}
}

/**
* commonPrefix es el largo del prefijo común de a y b.
* @param a, b []byte
* @return int
**/
func commonPrefix(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

/**
* cloneKey copia la clave para que sobreviva a soltar la página de donde salió.
* @param k key
* @return key
**/
func cloneKey(k key) key {
	return key{id: bytes.Clone(k.id), lsn: k.lsn, top: k.top, before: k.before}
}
