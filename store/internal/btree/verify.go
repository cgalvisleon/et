package btree

import (
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// VerifyStats es lo que cuenta Verify.
type VerifyStats struct {
	Levels     int // niveles del árbol (la raíz está en Levels-1)
	Pages      int // páginas del árbol enganchadas en sus niveles (sin la meta ni las borradas)
	Incomplete int // páginas con OpaqueIncompleteSplit
	Entries    int // entradas de hoja (todas las versiones)
	HalfDead   int // hojas medio muertas: borrados que un corte dejó a medias (la próxima limpieza los termina)
	Deleted    int // páginas borradas en la pila de la meta, esperando reciclarse

	// LeavesForm1 son las hojas de la forma 1 (de un directorio del formato 1) que la migración perezosa todavía no
	// reescribió en la forma 2; LeavesForm2 dice si la meta ya da a todas por reescritas.
	LeavesForm1 int
	LeavesForm2 bool
}

// VerifyOptions dice qué más hace VerifyWith mientras recorre (Verify del motor, fase 3).
type VerifyOptions struct {
	// Strategy es la de la caché para todas las páginas que lee: buffer.Ring para no vaciarla.
	Strategy buffer.Strategy

	// Page, si no es nil, recibe cada página del árbol (una vez cada una, ya soltada), también las borradas de la
	// pila de la meta; un error corta el recorrido.
	Page func(blk page.BlockNumber) error

	// Leaf, si no es nil, recibe cada hoja ya soltada con sus entradas en orden de clave (Version sin el valor en
	// línea; la referencia a desborde sí). El slice es del recorrido: vale hasta la próxima llamada.
	Leaf func(leaf LeafInfo, entries []Version) error
}

// LeafInfo es lo que VerifyWith cuenta de una hoja.
type LeafInfo struct {
	Block      page.BlockNumber
	AllVisible bool // la página lleva page.FlagAllVisible
	HalfDead   bool // la hoja está medio muerta: un borrado que un corte dejó a medias (FinishDeletes lo termina)
	Form1      bool // la hoja es de la forma 1 (formato 1): la limpieza la reescribe en la forma 2
}

// verifier es un recorrido de VerifyWith: el árbol, las opciones y las entradas de la hoja que se está leyendo.
type verifier struct {
	t       *BTree
	opts    VerifyOptions
	entries []Version
}

// levelStats es lo que deja el recorrido de un nivel: tops son las páginas sin enlace en el padre por un borrado a
// medias (la cima de su subárbol).
type levelStats struct {
	pages, incomplete, entries, downlinks, tops, halfDead, form1 int
	firstChild                                                   page.BlockNumber
}

// pendingInfo es lo que hace falta de una página (ya soltada) para saber si es parte de un borrado a medias.
type pendingInfo struct {
	blk       page.BlockNumber
	level     uint32
	leaf      bool
	halfDead  bool
	top       page.BlockNumber // la cima que anota una hoja medio muerta
	links     int              // enlaces de una interna
	firstDown page.BlockNumber // su primer enlace
}

/**
* Verify comprueba la estructura del árbol nivel por nivel, de izquierda a derecha: niveles y marcas, enlaces
* izquierda/derecha coherentes, claves estrictamente ordenadas en cada página y menores que su clave alta, la primera
* de cada página no menor que la clave alta de la anterior, y cada enlace de una interna apunta a una página del nivel
* de abajo cuya hermana izquierda tiene por clave alta la clave del enlace. Cada nivel tiene tantas páginas como
* enlaces el de arriba más sus particiones incompletas y las cimas de borrados a medias (la raíz cuenta como un enlace
* desde la meta). Un borrado a medias (una hoja medio muerta y la cadena de padres de un solo hijo hasta la cima que
* anota) no cuenta para las cotas de claves ni para la clave de los enlaces: su espacio de claves ya es de la hermana
* derecha. Comprueba además la raíz rápida (la de más a la izquierda de su nivel) y la pila de páginas borradas.
* Supone que nadie escribe el store mientras tanto. Memoria constante.
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
	defer t.guard.leave(t.guard.enter())
	v := &verifier{t: t, opts: opts}
	m, err := t.readMeta()
	if err != nil {
		return VerifyStats{}, err
	}
	if m.fastLevel > m.level {
		return VerifyStats{}, corruptAt(MetaBlock, "fast root level %d above the root level %d", m.fastLevel, m.level)
	}
	st := VerifyStats{Levels: int(m.level) + 1}
	blk, linksAbove := m.root, 1
	for level := int(m.level); level >= 0; level-- {
		leftmost, err := v.leftmost(blk)
		if err == nil && uint32(level) == m.fastLevel && leftmost != m.fastRoot {
			err = corruptAt(MetaBlock, "fast root %d is not the leftmost page %d of level %d", m.fastRoot, leftmost, level)
		}
		if err != nil {
			return st, err
		}
		ls, err := v.level(leftmost, uint32(level), level == int(m.level))
		if err != nil {
			return st, err
		}
		if ls.pages != linksAbove+ls.incomplete+ls.tops {
			return st, corrupt("level %d has %d pages, %d links above, %d incomplete splits and %d pending deletions",
				level, ls.pages, linksAbove, ls.incomplete, ls.tops)
		}
		st.Pages += ls.pages
		st.Incomplete += ls.incomplete
		st.Entries += ls.entries
		st.HalfDead += ls.halfDead
		st.LeavesForm1 += ls.form1
		blk, linksAbove = ls.firstChild, ls.downlinks
	}
	st.LeavesForm2 = m.flags&MetaLeavesForm2 != 0
	if st.LeavesForm2 && st.LeavesForm1 > 0 {
		return st, corruptAt(MetaBlock, "meta says every leaf is of form 2 but %d are of form 1", st.LeavesForm1)
	}
	st.Deleted, err = v.deletedStack(m.deleted)
	return st, err
}

/**
* leftmost va desde la página del primer enlace de un nivel hasta la de más a la izquierda: las que quedan a su
* izquierda son cimas de borrados a medias (sin enlace en el padre), que el recorrido del nivel comprueba.
* @param blk page.BlockNumber
* @return page.BlockNumber, error
**/
func (v *verifier) leftmost(blk page.BlockNumber) (page.BlockNumber, error) {
	for steps := 0; ; steps++ {
		h, n, err := v.lock(blk)
		if err != nil {
			return 0, err
		}
		prev := n.o.Prev
		h.Release()
		if prev == 0 {
			return blk, nil
		}
		if steps > 1<<20 {
			return 0, corruptAt(blk, "left links do not end")
		}
		blk = prev
	}
}

/**
* deletedStack recorre la pila de páginas borradas desde la meta: cada una tiene que estar borrada, y la pila no puede
* tener más páginas que el archivo (un ciclo). Entrega cada una a Page.
* @param head page.BlockNumber
* @return int: cuántas, error
**/
func (v *verifier) deletedStack(head page.BlockNumber) (int, error) {
	nblocks, err := v.t.sm.NBlocks(v.t.file, smgr.ForkMain)
	if err != nil {
		return 0, err
	}
	count := 0
	for blk := head; blk != 0; count++ {
		if count >= int(nblocks) {
			return count, corruptAt(blk, "deleted stack has a cycle")
		}
		h, n, err := v.lock(blk)
		if err != nil {
			return count, err
		}
		deleted, next := n.deleted(), deletedInfoOf(n).next
		h.Release()
		if !deleted {
			return count, corruptAt(blk, "page in the deleted stack is not deleted")
		}
		if v.opts.Page != nil {
			if err := v.opts.Page(blk); err != nil {
				return count, err
			}
		}
		blk = next
	}
	return count, nil
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
	var low *key // clave alta de la página anterior que no se está borrando: la cota baja de esta
	for blk := leftmost; blk != 0; {
		h, n, err := v.lock(blk)
		if err != nil {
			return ls, err
		}
		if n.deleted() {
			h.Release()
			return ls, corruptAt(blk, "deleted page still linked in level %d", level)
		}
		next, hk, err := v.page(n, level, prev, low, &ls, top && prev == 0)
		info := LeafInfo{Block: n.blk, AllVisible: n.p.Flags()&page.FlagAllVisible != 0, HalfDead: n.halfDead(),
			Form1: n.leaf() && !n.v2}
		pi := pendingOf(n)
		h.Release()
		if err == nil {
			err = v.deliver(n.leaf(), info)
		}
		pending, isTop := false, false
		if err == nil {
			pending, isTop, err = v.pending(pi)
		}
		if err != nil {
			return ls, err
		}
		if isTop {
			ls.tops++
		}
		if !pending {
			low = hk
		}
		prev, blk = n.blk, next
	}
	return ls, nil
}

/**
* pendingOf copia de una página bloqueada lo que pending necesita.
* @param n node
* @return pendingInfo
**/
func pendingOf(n node) pendingInfo {
	pi := pendingInfo{blk: n.blk, level: n.o.Level, leaf: n.leaf(), halfDead: n.halfDead(), links: n.dataCount()}
	if pi.halfDead {
		pi.top = leafTop(n)
	}
	if !pi.leaf && pi.links > 0 {
		if v, err := n.pivotAt(n.firstData()); err == nil {
			pi.firstDown = v.downlink()
		}
	}
	return pi
}

/**
* pending dice si una página es parte de un borrado a medias y si es su cima: una hoja medio muerta (cima si no anota
* otra), o una interna con un solo enlace cuya cadena de enlaces únicos llega a una hoja medio muerta que anota como
* cima a esta página o a una de más arriba.
* @param pi pendingInfo
* @return bool: parte de un borrado, bool: su cima (sin enlace en el padre), error
**/
func (v *verifier) pending(pi pendingInfo) (bool, bool, error) {
	if pi.leaf {
		if pi.halfDead {
			return true, pi.top == 0, nil
		}
		return false, false, nil
	}
	if pi.links != 1 {
		return false, false, nil
	}
	top, err := v.chainTop(pi.firstDown)
	if err != nil || top == 0 {
		return false, false, err
	}
	if top == pi.blk {
		return true, true, nil
	}
	h, n, err := v.lock(top)
	if err != nil {
		return false, false, err
	}
	above := n.o.Level > pi.level
	h.Release()
	return above, false, nil
}

/**
* chainTop baja por enlaces únicos desde blk hasta una hoja y, si está medio muerta, da la cima que anota.
* @param blk page.BlockNumber
* @return page.BlockNumber: 0 si la cadena no llega a una hoja medio muerta con cima, error
**/
func (v *verifier) chainTop(blk page.BlockNumber) (page.BlockNumber, error) {
	for {
		h, n, err := v.lock(blk)
		if err != nil {
			return 0, err
		}
		pi := pendingOf(n)
		h.Release()
		switch {
		case pi.leaf && pi.halfDead:
			return pi.top, nil
		case pi.leaf || pi.links != 1:
			return 0, nil
		}
		blk = pi.firstDown
	}
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
	if n.halfDead() {
		ls.halfDead++
	}
	if n.leaf() {
		ls.entries += n.dataCount()
		if !n.v2 {
			ls.form1++
		}
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
* collect copia la entrada off de una hoja si hay quien reciba las hojas: sin el valor en línea, salvo el de una
* versión comprimida (el motor comprueba que se descomprime). keyAt ya la validó.
* @param n node, off page.OffsetNumber
**/
func (v *verifier) collect(n node, off page.OffsetNumber) {
	if v.opts.Leaf == nil {
		return
	}
	e, _ := n.entryAt(off)
	if e.flags()&(EntryCompressed|EntryOverflow) == EntryCompressed {
		v.entries = append(v.entries, e.version())
		return
	}
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
	prev, childLevel, deleted := n.o.Prev, n.o.Level, n.deleted()
	h.Release()
	switch {
	case childLevel != level:
		return corruptAt(child, "child at level %d, expected %d", childLevel, level)
	case deleted:
		return corruptAt(child, "link to a deleted page")
	}
	for prev != 0 {
		next, match, err := v.leftMatches(prev, k)
		if err != nil || match {
			return err
		}
		prev = next
	}
	if k != nil {
		return corruptAt(child, "link key does not match the left sibling's high key")
	}
	return nil
}

/**
* leftMatches mira una hermana izquierda de la hija de un enlace: coincide si su clave alta es la del enlace; si no,
* tiene que ser parte de un borrado a medias (su espacio de claves pasó a la derecha) y se sigue con la de su izquierda.
* @param blk page.BlockNumber, k *key: la clave del enlace (nil = menos infinito)
* @return page.BlockNumber: la siguiente a la izquierda, bool: si coincide, error
**/
func (v *verifier) leftMatches(blk page.BlockNumber, k *key) (page.BlockNumber, bool, error) {
	h, n, err := v.lock(blk)
	if err != nil {
		return 0, false, err
	}
	pi, prev := pendingOf(n), n.o.Prev
	match := false
	if k != nil && !n.rightmost() {
		hk, err := n.highKey()
		if err != nil {
			h.Release()
			return 0, false, err
		}
		match = compareKeys(hk.key(), *k) == 0
	}
	h.Release()
	if match {
		return 0, true, nil
	}
	pending, _, err := v.pending(pi)
	if err == nil && !pending {
		err = corruptAt(blk, "left sibling's high key does not match the link key")
	}
	return prev, false, err
}
