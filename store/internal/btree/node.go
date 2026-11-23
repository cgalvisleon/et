package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/page"
)

// notTreePage son las marcas de páginas que no son del árbol.
const notTreePage = page.OpaqueMeta | page.OpaqueFree | page.OpaqueOverflow

// ignoredPage son las marcas de las páginas que se están borrando: un recorrido las salta siguiendo a la derecha (como
// P_IGNORE de nbtree).
const ignoredPage = page.OpaqueDeleted | page.OpaqueHalfDead

// capacity es el espacio de una página del árbol para ítems y sus punteros.
const capacity = page.Size - page.HeaderSize - page.OpaqueSize

// node es una página del árbol ya validada, con su bloque. Vale mientras la página siga bloqueada.
//
// Como en nbtree, en una página que no es la de más a la derecha de su nivel el ítem 1 es su clave alta (un pivote
// sin enlace) y los datos empiezan en el 2; en la de más a la derecha empiezan en el 1. En una hoja los datos son
// entradas; en una interna, pivotes con enlace, y el primero tiene la clave menos infinito. Una hoja de la forma 2
// (LayoutVersion 2, formato 2) lleva después de la clave alta (o primero, en la de más a la derecha) su ítem de
// prefijo, y los datos empiezan uno después; sus entradas guardan el id sin el prefijo (leaf2.go).
type node struct {
	blk page.BlockNumber
	p   page.Page
	o   page.Opaque
	pre []byte // hoja de la forma 2: el prefijo (vista de la página)
	v2  bool   // hoja de la forma 2
}

/**
* readNode valida una página del árbol: iniciada, con espacio especial de árbol, nivel coherente con la marca de
* hoja, punteros a ítems sanos (page.CheckItems: la caché solo verifica la suma) y clave alta si no es la de más a la
* derecha. Una página borrada se valida con checkDeleted (no tiene clave alta: solo sus datos de borrada); una medio
* muerta es una hoja vacía con su clave alta.
* @param blk page.BlockNumber, p page.Page
* @return node, error: ErrCorrupt con el bloque
**/
func readNode(blk page.BlockNumber, p page.Page) (node, error) {
	return readNodeWith(blk, p, true)
}

/**
* readNodeWith es readNode; sin checkItems no repite page.CheckItems (la caché recuerda que la página ya lo pasó
* desde que la cargó: buffer.ItemsChecker).
* @param blk page.BlockNumber, p page.Page, checkItems bool
* @return node, error
**/
func readNodeWith(blk page.BlockNumber, p page.Page, checkItems bool) (node, error) {
	if p.IsNew() {
		return node{}, corruptAt(blk, "page is new (all zeros)")
	}
	o, err := p.Opaque()
	if err != nil {
		return node{}, corruptAt(blk, "%v", err)
	}
	if o.Flags&notTreePage != 0 || (o.Flags&page.OpaqueLeaf != 0) != (o.Level == 0) {
		return node{}, corruptAt(blk, "not a tree page: flags %#x level %d", o.Flags, o.Level)
	}
	if checkItems {
		if err := p.CheckItems(); err != nil {
			return node{}, corruptAt(blk, "%v", err)
		}
	}
	var n node
	if err := n.init(p, o); err != nil {
		return node{}, corruptAt(blk, "%v", err)
	}
	n.blk = blk
	if n.deleted() {
		if n.v2 {
			return node{}, corruptAt(blk, "deleted page with layout version 2")
		}
		return n, checkDeleted(n)
	}
	if n.firstData() > n.maxOff()+1 {
		return node{}, corruptAt(blk, "page without its high key")
	}
	if n.halfDead() && (!n.leaf() || n.dataCount() != 0 || n.rightmost()) {
		return node{}, corruptAt(blk, "half-dead page that is not an empty leaf with a high key")
	}
	return n, nil
}

// leaf dice si la página es una hoja.
func (n *node) leaf() bool { return n.o.Flags&page.OpaqueLeaf != 0 }

// rightmost dice si es la de más a la derecha de su nivel (sin clave alta).
func (n *node) rightmost() bool { return n.o.Next == 0 }

// incomplete dice si su hermana derecha todavía no tiene enlace en el padre.
func (n *node) incomplete() bool { return n.o.Flags&page.OpaqueIncompleteSplit != 0 }

// root dice si lleva la marca de raíz.
func (n *node) root() bool { return n.o.Flags&page.OpaqueRoot != 0 }

// deleted dice si la página está borrada (desenganchada de sus hermanas, esperando que se recicle).
func (n *node) deleted() bool { return n.o.Flags&page.OpaqueDeleted != 0 }

// halfDead dice si es una hoja medio muerta: vacía, sin enlace en el padre y todavía enganchada a sus hermanas.
func (n *node) halfDead() bool { return n.o.Flags&page.OpaqueHalfDead != 0 }

// ignore dice si un recorrido la salta siguiendo a la derecha (borrada o medio muerta).
func (n *node) ignore() bool { return n.o.Flags&ignoredPage != 0 }

/**
* pageNode arma el node de una página del árbol ya leída (sin bloque): en una hoja de la forma 2, lee su ítem de
* prefijo. Solo una hoja puede ser de la forma 2. No valida los punteros (readNode llama antes a CheckItems; los
* rehaceres validan cada ítem que tocan).
* @param p page.Page, o page.Opaque
* @return node, error: ErrCorrupt
**/
func pageNode(p page.Page, o page.Opaque) (node, error) {
	var n node
	if err := n.init(p, o); err != nil {
		return node{}, err
	}
	return n, nil
}

/**
* init es pageNode sobre un node ya reservado (sin copiar el node en cada vuelta: va en el camino de cada lectura).
* @param p page.Page, o page.Opaque
* @return error: ErrCorrupt; con error n queda a medio armar
**/
func (n *node) init(p page.Page, o page.Opaque) error {
	n.p, n.o = p, o
	if p.LayoutVersion() != page.LayoutVersion2 {
		return nil
	}
	if !n.leaf() || n.deleted() {
		return corrupt("page with layout version 2 is not a leaf")
	}
	n.v2 = true
	pre, err := readPrefixItem(p, n.prefixOff())
	if err != nil {
		return err
	}
	n.pre = pre
	return nil
}

// prefixOff es el número del ítem de prefijo de una hoja de la forma 2: después de la clave alta, si tiene.
func (n *node) prefixOff() page.OffsetNumber {
	if n.rightmost() {
		return page.FirstOffset
	}
	return page.FirstOffset + 1
}

// firstData es el número del primer ítem de datos.
func (n *node) firstData() page.OffsetNumber {
	off := page.FirstOffset
	if !n.rightmost() {
		off++
	}
	if n.v2 {
		off++
	}
	return off
}

// maxOff es el número del último ítem.
func (n *node) maxOff() page.OffsetNumber { return n.p.MaxOffset() }

// dataCount es la cantidad de ítems de datos.
func (n *node) dataCount() int { return int(n.maxOff()) - int(n.firstData()) + 1 }

/**
* item devuelve el ítem off (vista).
* @param off page.OffsetNumber
* @return []byte, error: ErrCorrupt
**/
func (n *node) item(off page.OffsetNumber) ([]byte, error) {
	b, err := n.p.Item(off)
	if err != nil {
		return nil, corruptAt(n.blk, "item %d: %v", off, err)
	}
	return b, nil
}

/**
* entryAt lee la entrada off de una hoja.
* @param off page.OffsetNumber
* @return entry, error
**/
func (n *node) entryAt(off page.OffsetNumber) (entry, error) {
	b, err := n.item(off)
	if err != nil {
		return entry{}, err
	}
	var e entry
	if n.v2 {
		e, err = parseEntry2(b, n.pre)
	} else {
		e, err = parseEntry(b)
	}
	if err != nil {
		return entry{}, corruptAt(n.blk, "item %d: %v", off, err)
	}
	return e, nil
}

/**
* pivotAt lee el pivote off (clave alta, o un enlace de una página interna).
* @param off page.OffsetNumber
* @return pivot, error
**/
func (n *node) pivotAt(off page.OffsetNumber) (pivot, error) {
	b, err := n.item(off)
	if err != nil {
		return nil, err
	}
	v, err := parsePivot(b)
	if err != nil {
		return nil, corruptAt(n.blk, "item %d: %v", off, err)
	}
	return v, nil
}

/**
* keyAt es la clave del ítem de datos off: la de la entrada en una hoja, la del pivote en una interna.
* @param off page.OffsetNumber
* @return key, error
**/
func (n *node) keyAt(off page.OffsetNumber) (key, error) {
	if n.leaf() {
		e, err := n.entryAt(off)
		if err != nil {
			return key{}, err
		}
		return e.key(), nil
	}
	v, err := n.pivotAt(off)
	if err != nil {
		return key{}, err
	}
	return v.key(), nil
}

/**
* highKey lee la clave alta. Solo en una página que no es la de más a la derecha.
* @return pivot, error
**/
func (n *node) highKey() (pivot, error) {
	return n.pivotAt(page.FirstOffset)
}

/**
* pastHighKey es la prueba de Lehman-Yao: si k es mayor o igual que la clave alta, k ya no es de esta página (se partió
* después de que alguien leyera el enlace que llevó hasta acá) y hay que seguir a la derecha.
* @param k key
* @return bool, error
**/
func (n *node) pastHighKey(k key) (bool, error) {
	if n.rightmost() {
		return false, nil
	}
	hk, err := n.highKey()
	if err != nil {
		return false, err
	}
	return compareKeys(k, hk.key()) >= 0, nil
}

/**
* lowerBound busca el primer ítem de datos con clave ≥ k (búsqueda binaria).
* @param k key
* @return page.OffsetNumber: de firstData a maxOff + 1, error
**/
func (n *node) lowerBound(k key) (page.OffsetNumber, error) {
	return n.search(k, false)
}

/**
* upperBound busca el primer ítem de datos con clave > k.
* @param k key
* @return page.OffsetNumber, error
**/
func (n *node) upperBound(k key) (page.OffsetNumber, error) {
	return n.search(k, true)
}

/**
* search es la búsqueda binaria de lowerBound y upperBound: el primer ítem de datos cuya clave es ≥ k (> k con
* strict). Los ítems están ordenados, así que la condición es monótona.
* @param k key, strict bool
* @return page.OffsetNumber, error
**/
func (n *node) search(k key, strict bool) (page.OffsetNumber, error) {
	lo, hi := int(n.firstData()), int(n.maxOff())+1
	if len(n.pre) > 0 {
		var inside bool
		if k, inside = n.trim(k); !inside {
			if stopAt(compareKeys(key{pre: n.pre, lsn: MaxLSN}, k), strict) {
				return page.OffsetNumber(lo), nil // k va antes de toda la hoja
			}
			return page.OffsetNumber(hi), nil // después
		}
	}
	leaf := n.leaf()
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		var mk key
		var err error
		if leaf {
			mk, err = n.entryKey(page.OffsetNumber(mid))
		} else {
			mk, err = n.pivotKey(page.OffsetNumber(mid))
		}
		if err != nil {
			return 0, err
		}
		if stopAt(cmpKeys(&mk, &k), strict) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return page.OffsetNumber(lo), nil
}

// stopAt dice si la búsqueda para en un ítem cuya comparación con la clave buscada es c (≥ 0, o > 0 con strict).
func stopAt(c int, strict bool) bool { return c > 0 || c == 0 && !strict }

/**
* entryKey es la clave de la entrada off de una hoja para comparar con una clave ya recortada por trim: sin el
* prefijo de la hoja (en una de la forma 2 se comparan los sufijos).
* @param off page.OffsetNumber
* @return key, error
**/
func (n *node) entryKey(off page.OffsetNumber) (key, error) {
	e, err := n.entryAt(off)
	if err != nil {
		return key{}, err
	}
	return key{id: e.suffix(), lsn: e.lsn()}, nil
}

/**
* pivotKey es la clave del pivote off de una página interna.
* @param off page.OffsetNumber
* @return key, error
**/
func (n *node) pivotKey(off page.OffsetNumber) (key, error) {
	v, err := n.pivotAt(off)
	if err != nil {
		return key{}, err
	}
	return v.key(), nil
}

/**
* trim recorta una vez por hoja la clave de búsqueda al prefijo de una hoja de la forma 2: si el id de k empieza con
* el prefijo, k sin él (se compara con los sufijos de las entradas); si no, k cae antes o después de toda la hoja.
* @param k key: sin pre (una clave de búsqueda o top)
* @return key, bool: si k tiene el prefijo
**/
func (n *node) trim(k key) (key, bool) {
	if k.top || len(k.pre) > 0 || !bytes.HasPrefix(k.id, n.pre) {
		return k, false
	}
	k.id = k.id[len(n.pre):]
	return k, true
}

/**
* childFor elige el enlace a seguir en una página interna: el último pivote con clave ≤ k (el primero es menos
* infinito, así que siempre hay uno).
* @param k key
* @return page.BlockNumber, error
**/
func (n *node) childFor(k key) (page.BlockNumber, error) {
	off, err := n.upperBound(k)
	if err != nil {
		return 0, err
	}
	if off <= n.firstData() {
		return 0, corruptAt(n.blk, "internal page without a link for the key")
	}
	v, err := n.pivotAt(off - 1)
	if err != nil {
		return 0, err
	}
	return v.downlink(), nil
}

/**
* findDownlink busca en una página interna el enlace a child.
* @param child page.BlockNumber
* @return page.OffsetNumber: 0 si no está, error
**/
func (n *node) findDownlink(child page.BlockNumber) (page.OffsetNumber, error) {
	for off := n.firstData(); off <= n.maxOff(); off++ {
		v, err := n.pivotAt(off)
		if err != nil {
			return 0, err
		}
		if v.downlink() == child {
			return off, nil
		}
	}
	return 0, nil
}

/**
* fits dice si un ítem de size bytes cabe en la página.
* @param p page.Page, size int
* @return bool
**/
func fits(p page.Page, size int) bool {
	return p.FreeSpace() >= page.Align(size)
}
