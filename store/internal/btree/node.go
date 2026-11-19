package btree

import "github.com/cgalvisleon/et/store/internal/page"

// notTreePage son las marcas de páginas que no son del árbol (o que el árbol de la fase 1 no usa).
const notTreePage = page.OpaqueMeta | page.OpaqueFree | page.OpaqueOverflow | page.OpaqueDeleted | page.OpaqueHalfDead

// capacity es el espacio de una página del árbol para ítems y sus punteros.
const capacity = page.Size - page.HeaderSize - page.OpaqueSize

// node es una página del árbol ya validada, con su bloque. Vale mientras la página siga bloqueada.
//
// Como en nbtree, en una página que no es la de más a la derecha de su nivel el ítem 1 es su clave alta (un pivote
// sin enlace) y los datos empiezan en el 2; en la de más a la derecha empiezan en el 1. En una hoja los datos son
// entradas; en una interna, pivotes con enlace, y el primero tiene la clave menos infinito.
type node struct {
	blk page.BlockNumber
	p   page.Page
	o   page.Opaque
}

/**
* readNode valida una página del árbol: iniciada, con espacio especial de árbol, nivel coherente con la marca de
* hoja, punteros a ítems sanos (page.CheckItems: la caché solo verifica la suma) y clave alta si no es la de más a la
* derecha.
* @param blk page.BlockNumber, p page.Page
* @return node, error: ErrCorrupt con el bloque
**/
func readNode(blk page.BlockNumber, p page.Page) (node, error) {
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
	if err := p.CheckItems(); err != nil {
		return node{}, corruptAt(blk, "%v", err)
	}
	n := node{blk: blk, p: p, o: o}
	if n.firstData() > n.maxOff()+1 {
		return node{}, corruptAt(blk, "page without its high key")
	}
	return n, nil
}

// leaf dice si la página es una hoja.
func (n node) leaf() bool { return n.o.Flags&page.OpaqueLeaf != 0 }

// rightmost dice si es la de más a la derecha de su nivel (sin clave alta).
func (n node) rightmost() bool { return n.o.Next == 0 }

// incomplete dice si su hermana derecha todavía no tiene enlace en el padre.
func (n node) incomplete() bool { return n.o.Flags&page.OpaqueIncompleteSplit != 0 }

// firstData es el número del primer ítem de datos.
func (n node) firstData() page.OffsetNumber {
	if n.rightmost() {
		return page.FirstOffset
	}
	return page.FirstOffset + 1
}

// maxOff es el número del último ítem.
func (n node) maxOff() page.OffsetNumber { return n.p.MaxOffset() }

// dataCount es la cantidad de ítems de datos.
func (n node) dataCount() int { return int(n.maxOff()) - int(n.firstData()) + 1 }

/**
* item devuelve el ítem off (vista).
* @param off page.OffsetNumber
* @return []byte, error: ErrCorrupt
**/
func (n node) item(off page.OffsetNumber) ([]byte, error) {
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
func (n node) entryAt(off page.OffsetNumber) (entry, error) {
	b, err := n.item(off)
	if err != nil {
		return nil, err
	}
	e, err := parseEntry(b)
	if err != nil {
		return nil, corruptAt(n.blk, "item %d: %v", off, err)
	}
	return e, nil
}

/**
* pivotAt lee el pivote off (clave alta, o un enlace de una página interna).
* @param off page.OffsetNumber
* @return pivot, error
**/
func (n node) pivotAt(off page.OffsetNumber) (pivot, error) {
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
func (n node) keyAt(off page.OffsetNumber) (key, error) {
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
func (n node) highKey() (pivot, error) {
	return n.pivotAt(page.FirstOffset)
}

/**
* pastHighKey es la prueba de Lehman-Yao: si k es mayor o igual que la clave alta, k ya no es de esta página (se partió
* después de que alguien leyera el enlace que llevó hasta acá) y hay que seguir a la derecha.
* @param k key
* @return bool, error
**/
func (n node) pastHighKey(k key) (bool, error) {
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
func (n node) lowerBound(k key) (page.OffsetNumber, error) {
	return n.search(func(c int) bool { return c >= 0 }, k)
}

/**
* upperBound busca el primer ítem de datos con clave > k.
* @param k key
* @return page.OffsetNumber, error
**/
func (n node) upperBound(k key) (page.OffsetNumber, error) {
	return n.search(func(c int) bool { return c > 0 }, k)
}

/**
* search es la búsqueda binaria de lowerBound y upperBound: el primer ítem de datos cuya comparación con k cumple
* stop (stop es monótona porque los ítems están ordenados).
* @param stop func(c int) bool: recibe compareKeys(clave del ítem, k)
* @param k key
* @return page.OffsetNumber, error
**/
func (n node) search(stop func(c int) bool, k key) (page.OffsetNumber, error) {
	lo, hi := int(n.firstData()), int(n.maxOff())+1
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		mk, err := n.keyAt(page.OffsetNumber(mid))
		if err != nil {
			return 0, err
		}
		if stop(compareKeys(mk, k)) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return page.OffsetNumber(lo), nil
}

/**
* childFor elige el enlace a seguir en una página interna: el último pivote con clave ≤ k (el primero es menos
* infinito, así que siempre hay uno).
* @param k key
* @return page.BlockNumber, error
**/
func (n node) childFor(k key) (page.BlockNumber, error) {
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
func (n node) findDownlink(child page.BlockNumber) (page.OffsetNumber, error) {
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
