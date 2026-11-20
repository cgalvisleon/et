package btree

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* finishSplit hace el segundo paso de la partición de blk (como _bt_finish_split): pone en el padre el enlace a su
* hermana derecha (KindInsertPivot) o, si blk es la raíz, crea la raíz nueva (KindNewRoot); los dos apagan
* OpaqueIncompleteSplit en blk. Si el padre necesita trabajo antes (partirse, o terminar su propia partición), lo
* hace soltando todo y repite. Si otro ya la terminó, no hace nada.
* @param blk page.BlockNumber: la izquierda de la partición
* @return error
**/
func (t *BTree) finishSplit(blk page.BlockNumber) error {
	for {
		done, err := t.finishSplitOnce(blk)
		if err != nil || done {
			return err
		}
	}
}

/**
* finishSplitOnce intenta terminar la partición de blk una vez.
* @param blk page.BlockNumber
* @return bool: false si hay que repetir (el padre cambió o se arregló), error
**/
func (t *BTree) finishSplitOnce(blk page.BlockNumber) (bool, error) {
	h, n, err := t.lockNode(blk, buffer.Exclusive)
	if err != nil {
		return false, err
	}
	if !n.incomplete() {
		h.Release()
		return true, nil
	}
	hk, err := n.highKey()
	if err != nil {
		h.Release()
		return false, err
	}
	link := withDownlink(hk, n.o.Next) // copia: sobrevive a soltar la página
	// La raíz no cambia mientras blk siga bloqueada: solo la cambia quien termina la partición de la raíz, que la
	// tiene bloqueada.
	m, err := t.readMeta()
	if err != nil {
		h.Release()
		return false, err
	}
	if m.root == blk {
		return true, t.newRoot(h, n, link)
	}
	return t.insertPivot(h, n, link)
}

/**
* insertPivot pone en el padre de la hija el enlace a su hermana derecha, con la hija bloqueada (de hija a padre),
* y le apaga OpaqueIncompleteSplit en el mismo registro (con la meta, si la raíz rápida sube al padre). Si el padre no tiene lugar, lo parte y pide repetir; si el
* camino tiene una partición incompleta, la termina y pide repetir. Consume h.
* @param h buffer.Handle, n node: la hija, bloqueada en exclusivo e incompleta
* @param link []byte: el pivote (clave alta de la hija, enlace a su hermana derecha)
* @return bool: si terminó, error
**/
func (t *BTree) insertPivot(h buffer.Handle, n node, link []byte) (bool, error) {
	sep := pivot(link).key()
	ph, pn, off, err := t.findParent(n.blk, sep, n.o.Level+1)
	var fe *finishError
	if errors.As(err, &fe) {
		h.Release()
		return false, t.finishSplit(fe.block)
	}
	if err != nil {
		h.Release()
		return false, err
	}
	pos := off + 1
	if !fits(pn.p, len(link)) {
		h.Release()
		atEnd := pn.rightmost() && pos > pn.maxOff()
		return false, t.split(ph, pn, atEnd, 0)
	}
	refs := []wal.BlockRef{{Delta: itemDelta(pos, link)}, {}}
	hs := []buffer.Handle{ph, h}
	metaH, err := t.fastRootUp(n, pn)
	if err != nil {
		releaseAll(hs...)
		return false, err
	}
	if metaH != nil {
		refs = append(refs, wal.BlockRef{Delta: metaFastDelta(pn.blk, pn.o.Level)})
		hs = append(hs, metaH)
	}
	res, err := t.logChange(wal.KindInsertPivot, 0, 0, refs, hs, nil)
	releaseAll(hs...)
	if err != nil {
		return false, err
	}
	return true, res.Write()
}

/**
* fastRootUp decide si la raíz rápida sube al padre (como split_only_page en _bt_insertonpg): la hija que se partió es
* la de más a la izquierda de su nivel y la raíz rápida está en su nivel o más abajo, así que ese nivel dejó de tener
* una sola página. Bloquea la meta en exclusivo después de las páginas del árbol.
* @param n node: la hija, pn node: el padre, los dos bloqueados en exclusivo
* @return buffer.Handle: la meta bloqueada si sube (nil si no), error
**/
func (t *BTree) fastRootUp(n, pn node) (buffer.Handle, error) {
	if n.o.Prev != 0 {
		return nil, nil
	}
	h, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return nil, err
	}
	h.Lock(buffer.Exclusive)
	m, err := readMetaPage(h.Page())
	if err != nil || m.fastLevel > n.o.Level || m.fastRoot == pn.blk {
		h.Release()
		return nil, err
	}
	return h, nil
}

/**
* findParent busca, con la hija bloqueada, la página del nivel level que tiene el enlace a child: baja hasta donde va
* sep (la clave alta de la hija; su enlace está en esa página, porque ningún separador del padre cae entre la cota baja
* de la hija y sep) y sigue a la derecha hasta encontrarlo, por si el padre se partió (como _bt_getstackbuf). Sin pila
* de la bajada: una bajada más por partición, a cambio de no guardar nada.
* @param child page.BlockNumber, sep key, level uint32
* @return buffer.Handle, node: el padre bloqueado en exclusivo, page.OffsetNumber: el enlace, error: *finishError
**/
func (t *BTree) findParent(child page.BlockNumber, sep key, level uint32) (buffer.Handle, node, page.OffsetNumber, error) {
	h, n, _, err := t.descend(sep, level, buffer.Exclusive, true)
	for err == nil {
		var off page.OffsetNumber
		if off, err = n.findDownlink(child); err == nil && off != 0 {
			return h, n, off, nil
		}
		if err == nil && n.rightmost() {
			err = corruptAt(child, "no link in its parent level %d", level)
		}
		if err != nil {
			break
		}
		next := n.o.Next
		h.Release()
		if h, n, err = t.lockNode(next, buffer.Exclusive); err == nil && n.incomplete() {
			h.Release()
			return nil, node{}, 0, &finishError{block: next}
		}
		if err != nil {
			return nil, node{}, 0, err
		}
	}
	if h != nil {
		h.Release()
	}
	return nil, node{}, 0, err
}

/**
* newRoot termina la partición de la raíz (como _bt_newroot): pide una página, la inicia como raíz un nivel arriba con
* dos enlaces (menos infinito → la izquierda, la clave alta → la derecha), la anota en la meta y apaga OpaqueRoot y
* OpaqueIncompleteSplit en la izquierda, en un KindNewRoot (toca dos niveles, como el XLOG_BTREE_NEWROOT de nbtree).
* La meta se bloquea después de las páginas del árbol. Consume h.
* @param h buffer.Handle, n node: la raíz vieja, bloqueada en exclusivo e incompleta
* @param link []byte: el pivote de la derecha
* @return error
**/
func (t *BTree) newRoot(h buffer.Handle, n node, link []byte) error {
	rootH, err := t.free.Allocate(t.file, 0)
	if err != nil {
		h.Release()
		return err
	}
	metaH, err := t.pool.Pin(buffer.Tag{File: t.file, Fork: smgr.ForkMain, Block: MetaBlock}, buffer.Normal)
	if err != nil {
		releaseAll(h, rootH)
		return err
	}
	metaH.Lock(buffer.Exclusive)
	if m, err := readMetaPage(metaH.Page()); err != nil || m.root != n.blk {
		releaseAll(h, rootH, metaH)
		return errors.Join(err, corruptAt(n.blk, "root changed while it was locked"))
	}
	level := n.o.Level + 1
	refs := []wal.BlockRef{
		{Flags: wal.BlockWillInit, Delta: newRootDelta(level, n.blk, link)},
		{Delta: metaRootDelta(rootH.Tag().Block, level)},
		{},
	}
	hs := []buffer.Handle{rootH, metaH, h}
	res, err := t.logChange(wal.KindNewRoot, 0, 0, refs, hs, nil)
	releaseAll(hs...)
	if err != nil {
		return err
	}
	return res.Write()
}
