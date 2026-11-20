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

// VerifyOptions dice qué más hace VerifyWith mientras recorre (Verify del motor, fase 3).
type VerifyOptions struct {
	// Strategy es la de la caché para todas las páginas que lee: buffer.Ring para no vaciarla.
	Strategy buffer.Strategy

	// Page, si no es nil, recibe cada página del árbol (una vez cada una, ya soltada); un error corta el recorrido.
	Page func(blk page.BlockNumber) error

	// Leaf, si no es nil, recibe cada hoja ya soltada con sus entradas en orden de clave (Version sin el valor en
	// línea; la referencia a desborde sí). El slice es del recorrido: vale hasta la próxima llamada.
	Leaf func(leaf LeafInfo, entries []Version) error
}

// LeafInfo es lo que VerifyWith cuenta de una hoja.
type LeafInfo struct {
	Block      page.BlockNumber
	AllVisible bool // la página lleva page.FlagAllVisible
}

// verifier es un recorrido de VerifyWith: el árbol, las opciones y las entradas de la hoja que se está leyendo.
type verifier struct {
	t       *BTree
	opts    VerifyOptions
	entries []Version
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
* enlaces el de arriba más sus particiones incompletas (la raíz cuenta como un enlace desde la meta). Supone que
* nadie escribe el store mientras tanto. Memoria constante.
* @return VerifyStats, error: ErrCorrupt con el detalle
**/
func (t *BTree) Verify() (VerifyStats, error) {
	return t.VerifyWith(VerifyOptions{})
}

/**
* VerifyWith es Verify con opciones: la estrategia de la caché y quién recibe cada página y cada hoja (el Verify del
* motor arma con eso su inventario de páginas, los desbordes y el conteo). No pone marcas de visibilidad ni cambia
* nada. Lo que guarda en RAM es una hoja de entradas.
* @param opts VerifyOptions
* @return VerifyStats, error: ErrCorrupt, el de la caché o el de un callback
**/
func (t *BTree) VerifyWith(opts VerifyOptions) (VerifyStats, error) {
	v := &verifier{t: t, opts: opts}
	m, err := t.readMeta()
	if err != nil {
		return VerifyStats{}, err
	}
	st := VerifyStats{Levels: int(m.level) + 1}
	blk, linksAbove := m.root, 1
	for level := int(m.level); level >= 0; level-- {
		ls, err := v.level(blk, uint32(level), level == int(m.level))
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
* lock fija y bloquea en compartido una página del árbol con la estrategia del recorrido.
* @param blk page.BlockNumber
* @return buffer.Handle, node, error
**/
func (v *verifier) lock(blk page.BlockNumber) (buffer.Handle, node, error) {
	return v.t.lockNodeWith(blk, buffer.Share, v.opts.Strategy)
}

/**
* level recorre un nivel desde su página de más a la izquierda y entrega cada página (y cada hoja) a las opciones.
* @param leftmost page.BlockNumber, level uint32, top bool: es el nivel de la raíz (su primera página es la raíz)
* @return levelStats, error
**/
func (v *verifier) level(leftmost page.BlockNumber, level uint32, top bool) (levelStats, error) {
	var ls levelStats
	var prev page.BlockNumber
	var low *key // clave alta de la página anterior: la cota baja de esta
	for blk := leftmost; blk != 0; {
		h, n, err := v.lock(blk)
		if err != nil {
			return ls, err
		}
		next, hk, err := v.page(n, level, prev, low, &ls, top && prev == 0)
		info := LeafInfo{Block: n.blk, AllVisible: n.p.Flags()&page.FlagAllVisible != 0}
		h.Release()
		if err == nil {
			err = v.deliver(n.leaf(), info)
		}
		if err != nil {
			return ls, err
		}
		prev, low, blk = n.blk, hk, next
	}
	return ls, nil
}

/**
* deliver entrega una página ya soltada a las opciones: Page siempre, Leaf si es una hoja.
* @param leaf bool, info LeafInfo
* @return error
**/
func (v *verifier) deliver(leaf bool, info LeafInfo) error {
	if v.opts.Page != nil {
		if err := v.opts.Page(info.Block); err != nil {
			return err
		}
	}
	if leaf && v.opts.Leaf != nil {
		return v.opts.Leaf(info, v.entries)
	}
	return nil
}

/**
* page comprueba una página de un nivel y suma sus cuentas; en una hoja, copia sus entradas si hay quien las reciba.
* @param n node: bloqueada en compartido
* @param level uint32, prev page.BlockNumber, low *key: lo que se espera de su lugar en el nivel
* @param ls *levelStats, isRoot bool
* @return page.BlockNumber: la siguiente, *key: su clave alta (copia), error
**/
func (v *verifier) page(n node, level uint32, prev page.BlockNumber, low *key, ls *levelStats, isRoot bool) (page.BlockNumber, *key, error) {
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
	if err := v.keys(n, low, high, ls); err != nil {
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
* keys comprueba el orden de las claves de una página dentro de [low, high) y, en una interna, cada enlace; en una
* hoja copia las entradas para Leaf.
* @param n node, low, high *key: nil = sin cota, ls *levelStats
* @return error
**/
func (v *verifier) keys(n node, low, high *key, ls *levelStats) error {
	v.entries = v.entries[:0]
	var last *key
	for off := n.firstData(); off <= n.maxOff(); off++ {
		k, err := n.keyAt(off)
		if err != nil {
			return err
		}
		first := off == n.firstData()
		if !n.leaf() && first {
			if err := v.firstLink(n, low, ls); err != nil {
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
		if n.leaf() {
			v.collect(n, off)
		} else {
			p, _ := n.pivotAt(off)
			if err := v.link(p.downlink(), n.o.Level-1, &k); err != nil {
				return err
			}
			ls.downlinks++
		}
		last = &k
	}
	return nil
}

/**
* collect copia la entrada off de una hoja (sin el valor en línea) si hay quien reciba las hojas. keyAt ya la validó.
* @param n node, off page.OffsetNumber
**/
func (v *verifier) collect(n node, off page.OffsetNumber) {
	if v.opts.Leaf == nil {
		return
	}
	e, _ := n.entryAt(off)
	v.entries = append(v.entries, e.header())
}

/**
* firstLink comprueba el primer enlace de una interna: clave menos infinito; su cota real es la de la página.
* @param n node, low *key: la cota baja de la página, ls *levelStats
* @return error
**/
func (v *verifier) firstLink(n node, low *key, ls *levelStats) error {
	p, err := n.pivotAt(n.firstData())
	if err != nil {
		return err
	}
	if !p.minusInf() {
		return corruptAt(n.blk, "first link of an internal page is not minus infinity")
	}
	if ls.downlinks == 0 && ls.pages == 0 {
		ls.firstChild = p.downlink()
	}
	ls.downlinks++
	return v.link(p.downlink(), n.o.Level-1, low)
}

/**
* link comprueba un enlace: la hija está en el nivel de abajo y, si tiene hermana izquierda, la clave alta de esa
* hermana es la clave del enlace (si no tiene, el enlace es menos infinito).
* @param child page.BlockNumber, level uint32, k *key: la clave del enlace; nil = menos infinito
* @return error
**/
func (v *verifier) link(child page.BlockNumber, level uint32, k *key) error {
	h, n, err := v.lock(child)
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
	h, n, err = v.lock(prev)
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
