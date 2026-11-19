package btree

import (
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
)

// VerifyStats es lo que cuenta Verify.
type VerifyStats struct {
	Levels     int // niveles del árbol (la raíz está en Levels-1)
	Pages      int // páginas del árbol (sin la meta)
	Incomplete int // páginas con OpaqueIncompleteSplit
	Entries    int // entradas de hoja (todas las versiones)
}

// levelStats es lo que deja el recorrido de un nivel.
type levelStats struct {
	pages, incomplete, entries, downlinks int
	firstChild                            page.BlockNumber
}

/**
* Verify comprueba la estructura del árbol nivel por nivel, de izquierda a derecha: niveles y marcas, enlaces
* izquierda/derecha coherentes, claves estrictamente ordenadas en cada página y menores que su clave alta, la primera
* de cada página no menor que la clave alta de la anterior, y cada enlace de una interna apunta a una página del nivel
* de abajo cuya hermana izquierda tiene por clave alta la clave del enlace. Cada nivel tiene tantas páginas como
* enlaces el de arriba más sus particiones incompletas (la raíz cuenta como un enlace desde la meta). Es para pruebas
* y para Verify del motor (fase 3): supone que nadie escribe el store mientras tanto. Memoria constante.
* @return VerifyStats, error: ErrCorrupt con el detalle
**/
func (t *BTree) Verify() (VerifyStats, error) {
	m, err := t.readMeta()
	if err != nil {
		return VerifyStats{}, err
	}
	st := VerifyStats{Levels: int(m.level) + 1}
	blk, linksAbove := m.root, 1
	for level := int(m.level); level >= 0; level-- {
		ls, err := t.verifyLevel(blk, uint32(level), level == int(m.level))
		if err != nil {
			return st, err
		}
		if ls.pages != linksAbove+ls.incomplete {
			return st, corrupt("level %d has %d pages, %d links above and %d incomplete splits",
				level, ls.pages, linksAbove, ls.incomplete)
		}
		st.Pages += ls.pages
		st.Incomplete += ls.incomplete
		st.Entries += ls.entries
		blk, linksAbove = ls.firstChild, ls.downlinks
	}
	return st, nil
}

/**
* verifyLevel recorre un nivel desde su página de más a la izquierda.
* @param leftmost page.BlockNumber, level uint32, top bool: es el nivel de la raíz (su primera página es la raíz)
* @return levelStats, error
**/
func (t *BTree) verifyLevel(leftmost page.BlockNumber, level uint32, top bool) (levelStats, error) {
	var ls levelStats
	var prev page.BlockNumber
	var low *key // clave alta de la página anterior: la cota baja de esta
	for blk := leftmost; blk != 0; {
		h, n, err := t.lockNode(blk, buffer.Share)
		if err != nil {
			return ls, err
		}
		next, hk, err := t.verifyPage(n, level, prev, low, &ls, top && prev == 0)
		h.Release()
		if err != nil {
			return ls, err
		}
		prev, low, blk = n.blk, hk, next
	}
	return ls, nil
}

/**
* verifyPage comprueba una página de un nivel y suma sus cuentas.
* @param n node: bloqueada en compartido
* @param level uint32, prev page.BlockNumber, low *key: lo que se espera de su lugar en el nivel
* @param ls *levelStats, isRoot bool
* @return page.BlockNumber: la siguiente, *key: su clave alta (copia), error
**/
func (t *BTree) verifyPage(n node, level uint32, prev page.BlockNumber, low *key, ls *levelStats, isRoot bool) (page.BlockNumber, *key, error) {
	switch {
	case n.o.Level != level:
		return 0, nil, corruptAt(n.blk, "level %d, expected %d", n.o.Level, level)
	case n.o.Prev != prev:
		return 0, nil, corruptAt(n.blk, "left link %d, expected %d", n.o.Prev, prev)
	case (n.o.Flags&page.OpaqueRoot != 0) != isRoot:
		return 0, nil, corruptAt(n.blk, "root flag %v, expected %v", !isRoot, isRoot)
	case !n.leaf() && n.dataCount() == 0:
		return 0, nil, corruptAt(n.blk, "internal page without links")
	}
	var high *key
	if !n.rightmost() {
		hk, err := n.highKey()
		if err != nil {
			return 0, nil, err
		}
		k := cloneKey(hk.key())
		high = &k
	}
	if err := t.verifyKeys(n, low, high, ls); err != nil {
		return 0, nil, err
	}
	ls.pages++
	if n.incomplete() {
		ls.incomplete++
	}
	if n.leaf() {
		ls.entries += n.dataCount()
	}
	return n.o.Next, high, nil
}

/**
* verifyKeys comprueba el orden de las claves de una página dentro de [low, high) y, en una interna, cada enlace.
* @param n node, low, high *key: nil = sin cota, ls *levelStats
* @return error
**/
func (t *BTree) verifyKeys(n node, low, high *key, ls *levelStats) error {
	var last *key
	for off := n.firstData(); off <= n.maxOff(); off++ {
		k, err := n.keyAt(off)
		if err != nil {
			return err
		}
		first := off == n.firstData()
		if !n.leaf() && first {
			if err := t.verifyFirstLink(n, low, ls); err != nil {
				return err
			}
			continue
		}
		switch {
		case last != nil && compareKeys(*last, k) >= 0:
			return corruptAt(n.blk, "item %d is not after the previous one", off)
		case low != nil && compareKeys(k, *low) < 0:
			return corruptAt(n.blk, "item %d is before the page's low bound", off)
		case high != nil && compareKeys(k, *high) >= 0:
			return corruptAt(n.blk, "item %d is not before the high key", off)
		}
		if !n.leaf() {
			v, _ := n.pivotAt(off)
			if err := t.verifyLink(v.downlink(), n.o.Level-1, &k); err != nil {
				return err
			}
			ls.downlinks++
		}
		last = &k
	}
	return nil
}

/**
* verifyFirstLink comprueba el primer enlace de una interna: clave menos infinito; su cota real es la de la página.
* @param n node, low *key: la cota baja de la página, ls *levelStats
* @return error
**/
func (t *BTree) verifyFirstLink(n node, low *key, ls *levelStats) error {
	v, err := n.pivotAt(n.firstData())
	if err != nil {
		return err
	}
	if !v.minusInf() {
		return corruptAt(n.blk, "first link of an internal page is not minus infinity")
	}
	if ls.downlinks == 0 && ls.pages == 0 {
		ls.firstChild = v.downlink()
	}
	ls.downlinks++
	return t.verifyLink(v.downlink(), n.o.Level-1, low)
}

/**
* verifyLink comprueba un enlace: la hija está en el nivel de abajo y, si tiene hermana izquierda, la clave alta de
* esa hermana es la clave del enlace (si no tiene, el enlace es menos infinito).
* @param child page.BlockNumber, level uint32, k *key: la clave del enlace; nil = menos infinito
* @return error
**/
func (t *BTree) verifyLink(child page.BlockNumber, level uint32, k *key) error {
	h, n, err := t.lockNode(child, buffer.Share)
	if err != nil {
		return err
	}
	prev, childLevel := n.o.Prev, n.o.Level
	h.Release()
	switch {
	case childLevel != level:
		return corruptAt(child, "child at level %d, expected %d", childLevel, level)
	case (prev == 0) != (k == nil):
		return corruptAt(child, "child left link %d does not match its link key", prev)
	case prev == 0:
		return nil
	}
	h, n, err = t.lockNode(prev, buffer.Share)
	if err != nil {
		return err
	}
	defer h.Release()
	if n.rightmost() {
		return corruptAt(prev, "left sibling of %d has no high key", child)
	}
	hk, err := n.highKey()
	if err != nil {
		return err
	}
	if compareKeys(hk.key(), *k) != 0 {
		return corruptAt(child, "link key does not match the left sibling's high key")
	}
	return nil
}
