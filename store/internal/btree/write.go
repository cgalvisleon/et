package btree

import (
	"bytes"
	"errors"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

/**
* Write escribe una versión nueva de id, con los pasos 2 a 9 del orden de un cambio (store/doc.go): bajar hasta la
* hoja y bloquearla en exclusivo (terminando antes las particiones incompletas del camino), mirar la última versión
* que cuenta, comprobar la condición, partir la hoja si no hay espacio (y volver a bajar), reservar en el WAL con la
* hoja bloqueada, aplicar con la clave (id, lsn), soltar y escribir el registro. La versión nueva va antes que todas las
* del id: su lsn es el mayor.
* @param id string, w Write
* @return Result, error: *WaitError si otra transacción en curso tiene la versión más nueva (no se escribió nada),
* ErrSerialization, ErrIDLength, ErrEntryTooLarge o el error de la caché o del WAL
**/
func (t *BTree) Write(id string, w Write) (Result, error) {
	if err := checkWrite(id, w); err != nil {
		return Result{}, err
	}
	item := encodeEntry([]byte(id), w)
	k := searchKey([]byte(id))
	defer t.guard.leave(t.guard.enter())
	var hs hints
	defer t.flushHints(&hs)
	prep := beforeSplit{prune: t.horizon != nil, grow: t.grower != nil}
	for {
		h, n, rootLevel, err := t.leafForWrite(k, len(item))
		var fe *finishError
		if errors.As(err, &fe) {
			if err := t.finishSplit(fe.block); err != nil {
				return Result{}, err
			}
			continue
		}
		if err != nil {
			return Result{}, err
		}
		res, done, err := t.writeLeaf(h, n, k, item, w, rootLevel, &hs, &prep)
		if err != nil || done {
			return res, err
		}
	}
}

/**
* leafForWrite devuelve la hoja donde va k, bloqueada en exclusivo y sin partición incompleta: la de la pista de
* inserción rápida si sirve, si no bajando por el árbol.
* @param k key, size int: largo de la entrada a insertar
* @return buffer.Handle, node, uint32: nivel de la raíz (al menos 1 con la pista), error: *finishError
**/
func (t *BTree) leafForWrite(k key, size int) (buffer.Handle, node, uint32, error) {
	if h, n, ok := t.tryFastLeaf(k, size); ok {
		return h, n, 1, nil
	}
	return t.descend(k, 0, buffer.Exclusive, true)
}

/**
* tryFastLeaf es la inserción rápida al final (como el fastpath de _bt_search_insert): si la pista es la hoja de más
* a la derecha, sin partición incompleta, con lugar, y k va después de su primera clave, k es suya sin bajar: una hoja
* sin clave alta recibe todo lo que pasa su cota baja. Si no sirve, borra la pista.
* @param k key, size int
* @return buffer.Handle, node: bloqueada en exclusivo, bool: si sirvió
**/
func (t *BTree) tryFastLeaf(k key, size int) (buffer.Handle, node, bool) {
	blk := page.BlockNumber(t.fastLeaf.Load())
	if blk == 0 {
		return nil, node{}, false
	}
	h, n, err := t.lockNode(blk, buffer.Exclusive)
	if err == nil {
		if n.leaf() && n.rightmost() && !n.incomplete() && n.dataCount() > 0 && fits(n.p, size) {
			if first, err := n.keyAt(n.firstData()); err == nil && compareKeys(k, first) > 0 {
				return h, n, true
			}
		}
		h.Release()
	}
	t.fastLeaf.CompareAndSwap(uint32(blk), 0)
	return nil, node{}, false
}

/**
* writeLeaf hace los pasos 3 a 9 sobre la hoja ya bloqueada, y la suelta.
* @param h buffer.Handle, n node: la hoja, bloqueada en exclusivo
* @param k key: searchKey(id), item []byte: la entrada sin lsn, w Write, rootLevel uint32
* @param hs *hints: las marcas de visibilidad que se ponen al mirar las versiones
* @param prep *beforeSplit: lo que todavía se puede intentar antes de partir (una vez cada cosa por Write)
* @return Result, bool: false si la hoja se partió (o se soltó para extender) y hay que volver a bajar, error
**/
func (t *BTree) writeLeaf(h buffer.Handle, n node, k key, item []byte, w Write, rootLevel uint32, hs *hints, prep *beforeSplit) (Result, bool, error) {
	pos, err := n.lowerBound(k)
	if err != nil {
		h.Release()
		return Result{}, true, err
	}
	hs.at(h)
	latest, err := t.latest(n, pos, k.id, w, hs)
	hs.at(nil)
	if err != nil {
		h.Release()
		return Result{}, true, err
	}
	if !t.decide(latest, w) {
		h.Release()
		return Result{}, true, nil
	}
	if !fits(n.p, len(item)) {
		if prep.prune {
			prep.prune = false
			removed, err := t.pruneLeaf(h, n)
			if err != nil {
				h.Release()
				return Result{}, true, err
			}
			if removed > 0 { // la hoja cambió: se vuelve a buscar la posición y la última versión
				return t.writeLeaf(h, n, k, item, w, rootLevel, hs, prep)
			}
		}
		if prep.grow {
			// La página nueva de la partición se pide antes, con la hoja suelta: extender el archivo es lento y con
			// la hoja de más a la derecha bloqueada frenaba a todos los escritores (tarea 100 de F-23)
			prep.grow = false
			h.Release()
			return Result{}, false, t.grower.Grow(t.file, w.XID)
		}
		return Result{}, false, t.splitAndFinish(h, n, pos, w.XID)
	}

	if err := t.clearVisible(n); err != nil {
		h.Release()
		return Result{}, true, err
	}
	flags, delta := countChange(latest.Live, !w.Tombstone)
	if w.Commit {
		flags |= wal.FlagCommit
	}
	refs := []wal.BlockRef{{Delta: itemDelta(pos, item)}}
	res, err := t.logChange(wal.KindLeafWrite, w.XID, flags, refs, []buffer.Handle{h}, underLockOf(w, delta))
	h.Release()
	if err != nil {
		return Result{}, true, err
	}
	lsn := res.LSN()
	if err := res.Write(); err != nil {
		return Result{}, true, err
	}
	if n.rightmost() && rootLevel >= 1 {
		t.fastLeaf.Store(uint32(n.blk))
	}
	return Result{Applied: true, LSN: lsn, CountDelta: delta}, true, nil
}

// beforeSplit es lo que una escritura que no cabe intenta antes de partir la hoja, una vez cada cosa: podar sus
// versiones muertas (pruneLeaf) y extender el archivo con la hoja suelta (fsm.Grower).
type beforeSplit struct {
	prune, grow bool
}

/**
* pruneLeaf quita de la hoja, bloqueada en exclusivo, las versiones en línea que ningún snapshot puede ver, con el
* mismo criterio que la limpieza (planLeaf con el horizonte de Config.Horizon, sin congelar, sin marcar
* «liberándose» ni liberar desbordes, sin bits del mapa) y su mismo registro: un KindLeafVacuum con solo la lista de
* quitar y xid 0. Es la limpieza simple / bottom-up deletion de nbtree: una hoja llena de versiones muertas se vacía
* en lugar de partirse. Como el registro lleva la lista de números, una réplica que lo rehace queda igual byte a
* byte. Una hoja con page.FlagAllVisible no tiene nada muerto (y quitar no le cambia la marca): no se mira. Las
* versiones de un id que vienen de la hoja izquierda no heredan su versión de referencia (no se mira esa hoja): poda
* menos que la limpieza, nunca más; tampoco se mira la hermana derecha (la limpieza puede estar desenganchando hojas
* ahí): si el último id de la hoja puede seguir a la derecha, su lápida se queda.
* @param h buffer.Handle, n node: la hoja, bloqueada en exclusivo; sigue bloqueada al volver
* @return int: cuántas versiones quitó, error
**/
func (t *BTree) pruneLeaf(h buffer.Handle, n node) (int, error) {
	if n.p.Flags()&page.FlagAllVisible != 0 {
		return 0, nil
	}
	plan, err := t.planLeaf(n, VacuumOptions{Horizon: t.horizon()}, &vacuumGroup{}, false)
	if err != nil || len(plan.remove) == 0 {
		return 0, err
	}
	if err := t.logVacuum(h, pageVacuum{remove: plan.remove}, 0); err != nil {
		return 0, err
	}
	t.pruned.Add(uint64(len(plan.remove)))
	return len(plan.remove), nil
}

/**
* clearVisible borra los bits de la hoja en el mapa de visibilidad antes de cambiarla (KindVMClear, con la hoja
* bloqueada en exclusivo): si la hoja tiene page.FlagAllVisible su bit puede estar puesto, y el registro del mapa va
* antes que el de la hoja, así ningún prefijo del WAL deja una hoja con una versión nueva y su bit puesto (la limpieza
* la saltaría). El registro de la hoja apaga la marca de la página. Sin mapa no hace nada.
* @param n node: la hoja, bloqueada en exclusivo
* @return error
**/
func (t *BTree) clearVisible(n node) error {
	if t.vm == nil || n.p.Flags()&page.FlagAllVisible == 0 {
		return nil
	}
	return t.vm.Clear(t.file, n.blk)
}

/**
* underLockOf adapta Write.UnderLock a la firma de wal.Log.Reserve, con el delta de Count ya decidido.
* @param w Write, delta int
* @return func(lsn page.LSN): nil si w no tiene UnderLock
**/
func underLockOf(w Write, delta int) func(lsn page.LSN) {
	if w.UnderLock == nil {
		return nil
	}
	return func(lsn page.LSN) { w.UnderLock(lsn, delta) }
}

/**
* decide aplica la condición de la escritura a la última versión que cuenta. Remove (lápida) de un id sin versión
* viva no escribe nada.
* @param latest Latest, w Write
* @return bool: si se escribe
**/
func (t *BTree) decide(latest Latest, w Write) bool {
	if w.Tombstone && !latest.Live {
		return false
	}
	return w.Check == nil || w.Check(latest) == Apply
}

/**
* countChange dice cómo cambia la cantidad de ids vivos: marcas del registro y delta de Result.
* @param wasLive, isLive bool
* @return wal.Flags, int
**/
func countChange(wasLive, isLive bool) (wal.Flags, int) {
	switch {
	case isLive && !wasLive:
		return wal.FlagCountAdd, 1
	case !isLive && wasLive:
		return wal.FlagCountSub, -1
	}
	return 0, 0
}

/**
* latest busca, desde pos, la versión del id que cuenta para la escritura (paso 3 y 4 del orden de un cambio): la
* propia de su transacción, o la última confirmada; salta las abortadas. Si la más nueva no abortada es de otra
* transacción en curso devuelve *WaitError. Sigue a la derecha si las versiones del id continúan en la hermana: la
* bloquea en compartido con la hoja todavía bloqueada (de izquierda a derecha, como _bt_check_unique), y la suelta
* antes de volver.
* @param n node: la hoja, bloqueada en exclusivo; pos page.OffsetNumber, id []byte, w Write
* @param hs *hints: apunta a la hoja; al seguir a la derecha, a la hermana
* @return Latest, error: *WaitError o ErrSerialization
**/
func (t *BTree) latest(n node, pos page.OffsetNumber, id []byte, w Write, hs *hints) (Latest, error) {
	var right buffer.Handle
	defer func() { releaseAll(right) }()
	cur, off := n, pos
	for {
		if off > cur.maxOff() {
			if cur.rightmost() {
				return Latest{}, nil
			}
			next, level := cur.o.Next, cur.o.Level
			releaseAll(right)
			h, nn, err := t.lockNode(next, buffer.Share)
			if right = h; err != nil {
				return Latest{}, err
			}
			hs.at(h)
			if nn.o.Level != level {
				return Latest{}, corruptAt(next, "right sibling at level %d, expected %d", nn.o.Level, level)
			}
			cur, off = nn, nn.firstData()
			continue
		}
		e, err := cur.entryAt(off)
		if err != nil || !bytes.Equal(e.id(), id) {
			return Latest{}, err
		}
		if l, done, err := t.counts(e, w, hs); done || err != nil {
			return l, err
		}
		off++
	}
}

/**
* counts decide si una versión es la que cuenta para la escritura w. Una «liberándose» nunca cuenta.
* @param e entry, w Write, hs *hints
* @return Latest, bool: si la búsqueda termina acá, error: *WaitError o ErrSerialization
**/
func (t *BTree) counts(e entry, w Write, hs *hints) (Latest, bool, error) {
	if e.freeing() {
		return Latest{}, false, nil
	}
	own := w.XID
	if w.Snapshot != nil && w.Snapshot.Own != 0 {
		own = w.Snapshot.Own
	}
	if own != 0 && e.xid() == own {
		return Latest{Version: e.version(), Live: !e.tombstone()}, true, nil
	}
	st, err := t.statusOf(e, hs)
	switch {
	case err != nil:
		return Latest{}, true, err
	case st == xact.StatusAborted:
		return Latest{}, false, nil
	case st != xact.StatusCommitted:
		return Latest{}, true, &WaitError{XID: e.xid()}
	case w.Snapshot != nil && e.flags()&EntryFrozen == 0 && !w.Snapshot.Visible(e.xid(), st):
		return Latest{}, true, ErrSerialization
	}
	return Latest{Version: e.version(), Live: !e.tombstone()}, true, nil
}
