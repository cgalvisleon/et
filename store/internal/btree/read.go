package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/xact"
)

/**
* statusOf devuelve el estado de la transacción que escribió una versión: primero sus marcas, después el archivo de
* estados. Una versión «liberándose» cuenta como abortada (ya nadie la ve). Sin archivo de estados (Config.Status
* nil) o con xid 0, cuenta como confirmada. Si el archivo de estados da un estado definitivo —abortada, o confirmada con
* el WAL durable hasta el lsn de confirmación de su grupo (pregunta 49)—, lo anota en la versión (hint).
* @param e entry, hs *hints: dónde marcar (nil = no marcar)
* @return xact.Status, error
**/
func (t *BTree) statusOf(e entry, hs *hints) (xact.Status, error) {
	flags := e.flags()
	switch {
	case flags&EntryFreeing != 0:
		return xact.StatusAborted, nil
	case flags&(EntryXidCommitted|EntryFrozen) != 0:
		return xact.StatusCommitted, nil
	case flags&EntryXidAborted != 0:
		return xact.StatusAborted, nil
	case t.status == nil || e.xid() == 0:
		return xact.StatusCommitted, nil
	}
	st, commitLSN, err := t.status.Status(e.xid())
	if err != nil {
		return st, err
	}
	switch {
	case st == xact.StatusAborted:
		t.hint(hs, e, EntryXidAborted)
	case xact.HintAllowed(st, commitLSN, t.log.Durable()):
		t.hint(hs, e, EntryXidCommitted)
	}
	return st, nil
}

/**
* visible dice si una lectura ve la versión: nunca si está «liberándose»; sin snapshot (Read suelto, pregunta 41), si
* está confirmada; con snapshot, la regla de xact.Snapshot.Visible.
* @param e entry, snap *xact.Snapshot, hs *hints: dónde marcar (nil = no marcar)
* @return bool, error
**/
func (t *BTree) visible(e entry, snap *xact.Snapshot, hs *hints) (bool, error) {
	if e.freeing() {
		return false, nil
	}
	st, err := t.statusOf(e, hs)
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
	return t.lookup(id, snap, true)
}

/**
* Exists dice si id tiene una versión viva visible para snap (nil = la más nueva confirmada), como Lookup pero sin
* copiar el valor en línea ni leer el de desborde.
* @param id string, snap *xact.Snapshot
* @return bool, error
**/
func (t *BTree) Exists(id string, snap *xact.Snapshot) (bool, error) {
	_, found, err := t.lookup(id, snap, false)
	return found, err
}

/**
* lookup es Lookup; withValue dice si copia el valor en línea de la versión encontrada.
* @param id string, snap *xact.Snapshot, withValue bool
* @return Version, bool, error
**/
func (t *BTree) lookup(id string, snap *xact.Snapshot, withValue bool) (Version, bool, error) {
	if err := checkID(id); err != nil {
		return Version{}, false, err
	}
	defer t.guard.leave(t.guard.enter())
	var hs hints
	defer t.flushHints(&hs)
	k := searchKey([]byte(id))
	h, n, _, err := t.descend(k, 0, buffer.Share, false)
	if err != nil {
		return Version{}, false, err
	}
	hs.at(h)
	off, err := n.lowerBound(k)
	for err == nil {
		if off > n.maxOff() {
			if n.rightmost() {
				break
			}
			if h, n, err = t.stepRight(h, n); err != nil {
				return Version{}, false, err
			}
			hs.at(h)
			off = n.firstData()
			continue
		}
		var e entry
		if e, err = n.entryAt(off); err != nil || !bytes.Equal(e.id(), k.id) {
			break
		}
		var ok bool
		if ok, err = t.visible(e, snap, &hs); err == nil && ok {
			v := e.header() // copia: e apunta a la página, que no se toca después de soltarla
			if withValue {
				v = e.version()
			}
			h.Release()
			return v, v.Flags&EntryTombstone == 0, nil
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
