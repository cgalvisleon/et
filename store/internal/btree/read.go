package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/xact"
)

/**
* statusOf devuelve el estado de la transacción que escribió una versión: primero sus marcas de visibilidad, después
* el archivo de estados. Sin archivo de estados (Config.Status nil) o con xid 0, cuenta como confirmada.
* @param e entry
* @return xact.Status, error
**/
func (t *BTree) statusOf(e entry) (xact.Status, error) {
	flags := e.flags()
	switch {
	case flags&(EntryXidCommitted|EntryFrozen) != 0:
		return xact.StatusCommitted, nil
	case flags&EntryXidAborted != 0:
		return xact.StatusAborted, nil
	case t.status == nil || e.xid() == 0:
		return xact.StatusCommitted, nil
	}
	st, _, err := t.status.Status(e.xid())
	return st, err
}

/**
* visible dice si una lectura ve la versión: sin snapshot (Read suelto, pregunta 41), si está confirmada; con
* snapshot, la regla de xact.Snapshot.Visible.
* @param e entry, snap *xact.Snapshot
* @return bool, error
**/
func (t *BTree) visible(e entry, snap *xact.Snapshot) (bool, error) {
	st, err := t.statusOf(e)
	if err != nil {
		return false, err
	}
	if snap == nil {
		return st == xact.StatusCommitted, nil
	}
	if e.flags()&EntryFrozen != 0 {
		return true, nil
	}
	return snap.Visible(e.xid(), st), nil
}

/**
* Lookup devuelve la versión de id que ve snap (nil = la más nueva confirmada): baja con Lehman-Yao hasta la hoja y
* mira las versiones del id de la más nueva a la más vieja, siguiendo a la derecha si siguen en la vecina. Una lápida
* visible es «no está». El valor de desborde no se lee (queda en Version.Overflow).
* @param id string, snap *xact.Snapshot
* @return Version, bool: si hay una versión viva visible, error
**/
func (t *BTree) Lookup(id string, snap *xact.Snapshot) (Version, bool, error) {
	if err := checkID(id); err != nil {
		return Version{}, false, err
	}
	k := searchKey([]byte(id))
	h, n, _, err := t.descend(k, 0, buffer.Share, false)
	if err != nil {
		return Version{}, false, err
	}
	off, err := n.lowerBound(k)
	for err == nil {
		if off > n.maxOff() {
			if n.rightmost() {
				break
			}
			if h, n, err = t.stepRight(h, n); err != nil {
				return Version{}, false, err
			}
			off = n.firstData()
			continue
		}
		var e entry
		if e, err = n.entryAt(off); err != nil || !bytes.Equal(e.id(), k.id) {
			break
		}
		var ok bool
		if ok, err = t.visible(e, snap); err == nil && ok {
			v := e.version()
			h.Release()
			return v, !e.tombstone(), nil
		}
		off++
	}
	h.Release()
	return Version{}, false, err
}

/**
* stepRight suelta la página y bloquea su hermana derecha en compartido (recorrido hacia adelante). Nunca tiene las
* dos bloqueadas: lo que se movió a una hermana nueva entre medio ya estaba en la página leída.
* @param h buffer.Handle, n node: la página actual, no la de más a la derecha
* @return buffer.Handle, node, error
**/
func (t *BTree) stepRight(h buffer.Handle, n node) (buffer.Handle, node, error) {
	next, level := n.o.Next, n.o.Level
	h.Release()
	h, n, err := t.lockNode(next, buffer.Share)
	if err == nil && n.o.Level != level {
		h.Release()
		return nil, node{}, corruptAt(next, "right sibling at level %d, expected %d", n.o.Level, level)
	}
	return h, n, err
}
