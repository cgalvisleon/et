package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// group son las versiones de un id dentro de una hoja: el id y la primera visible (la más nueva que ve la lectura).
type group struct {
	id    []byte
	found bool
	v     Version
}

/**
* Scan recorre en orden de id ascendente, desde from (incluido; "" = desde el principio), la versión que ve snap de
* cada id (nil = la más nueva confirmada), sin las lápidas. Baja una vez y después avanza por las hojas: lee cada hoja
* entera con bloqueo compartido, la suelta y recién entonces llama a fn (fn puede leer el store; no puede escribir).
* Una hoja que se parte después de leída solo movió a la derecha lo ya leído: se sigue al enlace anotado al leerla.
* La memoria es una hoja de versiones, no crece con el store.
* @param from string, snap *xact.Snapshot
* @param fn func(v Version) (bool, error): false para parar
* @return error: el de fn o el del árbol
**/
func (t *BTree) Scan(from string, snap *xact.Snapshot, fn func(v Version) (bool, error)) error {
	k := searchKey([]byte(from))
	h, n, _, err := t.descend(k, 0, buffer.Share, false)
	if err != nil {
		return err
	}
	off, err := n.lowerBound(k)
	var cur []byte // id de las versiones que se están mirando; puede seguir en la hoja siguiente
	decided := false
	var batch []Version
	var hs hints
	for err == nil {
		batch = batch[:0]
		hs.at(h)
		for ; off <= n.maxOff() && err == nil; off++ {
			var e entry
			if e, err = n.entryAt(off); err != nil {
				break
			}
			if !bytes.Equal(e.id(), cur) || cur == nil {
				cur, decided = append(cur[:0], e.id()...), false
			}
			if decided {
				continue
			}
			var ok bool
			if ok, err = t.visible(e, snap, &hs); ok {
				decided = true
				if !e.tombstone() {
					batch = append(batch, e.version())
				}
			}
		}
		if err != nil {
			break
		}
		last, next := n.rightmost(), n
		h.Release()
		h = nil
		hs.at(nil)
		t.flushHints(&hs)
		if stop, ferr := emit(batch, fn); stop || ferr != nil {
			return ferr
		}
		if last {
			return nil
		}
		h, n, err = t.stepRightFrom(next)
		off = n.firstData()
	}
	if h != nil {
		h.Release()
	}
	return err
}

/**
* stepRightFrom bloquea en compartido la hermana derecha de una hoja ya soltada.
* @param n node: la hoja leída (sus campos, no su página)
* @return buffer.Handle, node, error
**/
func (t *BTree) stepRightFrom(n node) (buffer.Handle, node, error) {
	h, nn, err := t.lockNode(n.o.Next, buffer.Share)
	if err == nil && nn.o.Level != n.o.Level {
		h.Release()
		return nil, node{}, corruptAt(nn.blk, "right sibling at level %d, expected %d", nn.o.Level, n.o.Level)
	}
	return h, nn, err
}

/**
* emit llama a fn con cada versión.
* @param batch []Version, fn func(v Version) (bool, error)
* @return bool: si fn pidió parar, error
**/
func emit(batch []Version, fn func(v Version) (bool, error)) (bool, error) {
	for _, v := range batch {
		cont, err := fn(v)
		if err != nil || !cont {
			return true, err
		}
	}
	return false, nil
}

/**
* ScanBackward recorre en orden de id descendente, desde to (incluido; "" = desde el final), la versión que ve snap
* de cada id, sin las lápidas. Como las versiones de un id van de la más nueva a la más vieja, la que decide puede
* estar en la hoja de la izquierda: el primer id de cada hoja queda pendiente hasta leer la anterior. Para ir a la
* izquierda (como _bt_walk_left) bloquea la hermana anotada y, si se partió entre medio, sigue a la derecha hasta la
* página cuyo enlace derecho es la que se acaba de leer.
* @param to string, snap *xact.Snapshot, fn func(v Version) (bool, error): false para parar
* @return error
**/
func (t *BTree) ScanBackward(to string, snap *xact.Snapshot, fn func(v Version) (bool, error)) error {
	k := key{top: true}
	if to != "" {
		k = key{id: []byte(to), lsn: 0} // después de todas las versiones de to
	}
	h, n, _, err := t.descend(k, 0, buffer.Share, false)
	if err != nil {
		return err
	}
	end, err := n.lowerBound(k)
	var pending *group
	var batch []Version
	var hs hints
	for err == nil {
		var groups []group
		hs.at(h)
		if groups, err = t.pageGroups(n, end-1, snap, &hs); err != nil {
			break
		}
		batch, pending = backwardBatch(batch[:0], groups, pending, n.o.Prev == 0)
		cur := n
		h.Release()
		h = nil
		hs.at(nil)
		t.flushHints(&hs)
		if stop, ferr := emit(batch, fn); stop || ferr != nil {
			return ferr
		}
		if cur.o.Prev == 0 {
			return nil
		}
		if h, n, err = t.stepLeft(cur); err == nil {
			end = n.maxOff() + 1
		}
	}
	if h != nil {
		h.Release()
	}
	return err
}

/**
* pageGroups agrupa por id las entradas de datos de la hoja hasta last (incluido), en orden, con la primera versión
* visible de cada grupo.
* @param n node, last page.OffsetNumber, snap *xact.Snapshot, hs *hints: dónde marcar
* @return []group, error
**/
func (t *BTree) pageGroups(n node, last page.OffsetNumber, snap *xact.Snapshot, hs *hints) ([]group, error) {
	var groups []group
	for off := n.firstData(); off <= last; off++ {
		e, err := n.entryAt(off)
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 || !bytes.Equal(groups[len(groups)-1].id, e.id()) {
			groups = append(groups, group{id: bytes.Clone(e.id())})
		}
		g := &groups[len(groups)-1]
		if g.found {
			continue
		}
		ok, err := t.visible(e, snap, hs)
		if err != nil {
			return nil, err
		}
		if ok {
			g.found, g.v = true, e.version()
		}
	}
	return groups, nil
}

/**
* backwardBatch arma, de mayor a menor, los ids ya decididos de una hoja leída hacia atrás. El pendiente (el primer id
* de la hoja leída antes, a la derecha) se junta con el último grupo de esta si es el mismo id: gana la versión
* visible de esta hoja, que es más nueva. El primer grupo de esta hoja queda pendiente, salvo en la de más a la
* izquierda.
* @param batch []Version: se agrega acá, groups []group: en orden, pending *group, leftmost bool
* @return []Version, *group: el pendiente nuevo
**/
func backwardBatch(batch []Version, groups []group, pending *group, leftmost bool) ([]Version, *group) {
	if pending != nil && len(groups) > 0 && bytes.Equal(groups[len(groups)-1].id, pending.id) {
		g := &groups[len(groups)-1]
		if !g.found {
			g.found, g.v = pending.found, pending.v
		}
		pending = nil
	}
	if pending != nil && (len(groups) > 0 || leftmost) {
		batch = appendLive(batch, *pending)
		pending = nil
	}
	for i := len(groups) - 1; i >= 0; i-- {
		if i == 0 && !leftmost {
			pending = &groups[0]
			break
		}
		batch = appendLive(batch, groups[i])
	}
	return batch, pending
}

// appendLive agrega la versión del grupo si hay una visible y no es una lápida.
func appendLive(batch []Version, g group) []Version {
	if g.found && g.v.Flags&EntryTombstone == 0 {
		batch = append(batch, g.v)
	}
	return batch
}

/**
* stepLeft bloquea en compartido la hoja que hoy está a la izquierda de cur (ya soltada): la hermana anotada, o
* siguiendo a la derecha desde ella si se partió después de leer cur. Sin borrado de páginas (fase 1) siempre se llega.
* @param cur node: la hoja leída (sus campos)
* @return buffer.Handle, node, error
**/
func (t *BTree) stepLeft(cur node) (buffer.Handle, node, error) {
	h, n, err := t.lockNode(cur.o.Prev, buffer.Share)
	for err == nil && n.o.Next != cur.blk {
		if n.rightmost() || n.o.Level != cur.o.Level {
			h.Release()
			return nil, node{}, corruptAt(cur.blk, "left sibling chain does not reach the page")
		}
		next := n.o.Next
		h.Release()
		h, n, err = t.lockNode(next, buffer.Share)
	}
	return h, n, err
}
