package btree

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Borrado de páginas del árbol (tarea 70), como _bt_pagedel de nbtree: la limpieza marca medio muerta una hoja vacía
// y quita su enlace del padre del subárbol (KindMarkHalfDead), la desengancha de sus hermanas de arriba hacia abajo
// con su cadena de padres de un solo hijo (KindUnlinkPage, que además la marca borrada con su safexid y la apila en la
// meta) y, en una pasada posterior, cuando ningún lector puede tener todavía un enlace a ella, la devuelve a la lista
// de libres (KindRecyclePage). Formatos y reglas en doc.go, «Borrado de páginas».

const (
	// deletedInfoSize es el único ítem de una página borrada: safeXID(8) nextDeleted(4) reservado(4).
	deletedInfoSize = 16

	// unlinkTargetSize es el delta de la página que se borra: prev(4) next(4) level(4) safeXID(8) nextDeleted(4)
	// flags(2).
	unlinkTargetSize = 26

	// halfDeadParentSize es el delta del padre del subárbol: off(2) child(4) right(4).
	halfDeadParentSize = 2 + 4 + 4

	// recycleMetaSize es el delta de la meta al reciclar: deleted(4) freeHead(4).
	recycleMetaSize = 4 + 4

	// fastDeltaSize es la raíz rápida que agrega un registro a la meta: fastRoot(4) fastLevel(4).
	fastDeltaSize = 4 + 4

	// guardTimeout es lo que el reciclado espera a las operaciones en vuelo; si no terminan, recicla en otra pasada.
	guardTimeout = 2 * time.Second
)

// Marcas del delta de la página borrada (KindUnlinkPage): qué bloques opcionales trae el registro.
const (
	unlinkHasLeft  uint16 = 1 << 0 // bloque 3: la hermana izquierda (la página no era la de más a la izquierda)
	unlinkHasLeaf  uint16 = 1 << 1 // bloque 4 (o 3 sin izquierda): la hoja medio muerta (la borrada es un padre)
	unlinkSetsFast uint16 = 1 << 2 // la meta trae también la raíz rápida nueva
	unlinkAllFlags        = unlinkHasLeft | unlinkHasLeaf | unlinkSetsFast
)

// Papeles de los bloques de un KindUnlinkPage, después de quitar los opcionales que no vienen (unlinkRole).
const (
	roleTarget = 0
	roleRight  = 1
	roleMeta   = 2
	roleLeft   = 3
	roleLeaf   = 4
)

// deletedInfo es lo que guarda una página borrada.
type deletedInfo struct {
	safe page.XID         // se recicla cuando el horizonte de la limpieza lo supera
	next page.BlockNumber // la siguiente de la pila de borradas (0 = fin)
}

// unlinkTarget es el delta de la página que se borra, ya leído.
type unlinkTarget struct {
	prev, next page.BlockNumber
	level      uint32
	info       deletedInfo
	flags      uint16
}

// chainTop es la página más alta del subárbol que se borra con la hoja (la hoja o un padre de un solo hijo), con lo
// que hace falta para buscar su enlace en el padre.
type chainTop struct {
	blk   page.BlockNumber
	level uint32
	next  page.BlockNumber
	high  key // copia de su clave alta
}

// halfDeadPlan es el padre del subárbol (bloqueado en exclusivo) y el enlace que cambia el KindMarkHalfDead.
type halfDeadPlan struct {
	ph    buffer.Handle
	off   page.OffsetNumber // el enlace a la cima del subárbol
	top   page.BlockNumber  // la cima, si no es la hoja (0 = la hoja)
	child page.BlockNumber  // la cima (el enlace de off)
	right page.BlockNumber  // su hermana derecha (el enlace de off+1, que pasa a off)
}

/**
* deleteLeaf intenta borrar una hoja que la limpieza dejó vacía: la marca medio muerta y la desengancha con su cadena
* de padres de un solo hijo. Si no se puede borrar (es la de más a la derecha de su padre, tiene una partición
* incompleta cerca, su hermana derecha está medio muerta…) no hace nada: la próxima pasada lo intenta otra vez.
* @param blk page.BlockNumber, opts VacuumOptions, st *VacuumStats
* @return error
**/
func (t *BTree) deleteLeaf(blk page.BlockNumber, opts VacuumOptions, st *VacuumStats) error {
	marked, err := t.markHalfDead(blk)
	if err != nil || !marked {
		return err
	}
	st.HalfDead++
	return t.unlinkHalfDead(blk, opts, st)
}

/**
* FinishDeletes termina los borrados de páginas que un corte dejó a medias (las hojas medio muertas), sin limpiar nada
* más: lo usa Repair del motor. Recorre las hojas por sus enlaces; la memoria no crece con el árbol.
* @param nextXID func() page.XID: el próximo xid (la marca de cada página borrada)
* @return int: páginas desenganchadas, error
**/
func (t *BTree) FinishDeletes(nextXID func() page.XID) (int, error) {
	opts := VacuumOptions{NextXID: nextXID}
	var st VacuumStats
	if err := t.finishLeftmost(opts, &st); err != nil {
		return st.Deleted, err
	}
	h, n, _, err := t.descend(minusInf, 0, buffer.Share, false)
	for err == nil {
		blk, next, half := n.blk, n.o.Next, n.halfDead()
		h.Release()
		if half {
			if err = t.unlinkHalfDead(blk, opts, &st); err != nil {
				break
			}
		}
		if next == 0 {
			break
		}
		h, n, err = t.lockNode(next, buffer.Share)
	}
	return st.Deleted, err
}

/**
* deletableLeaf dice si una hoja se puede marcar medio muerta: vacía, ni la de más a la derecha ni la raíz, sin
* partición incompleta y sin estar ya borrándose (como las comprobaciones de _bt_pagedel).
* @param n node
* @return bool
**/
func deletableLeaf(n node) bool {
	return n.leaf() && !n.ignore() && n.dataCount() == 0 && !n.rightmost() && !n.root() && !n.incomplete()
}

/**
* markHalfDead es el primer paso (como _bt_mark_page_halfdead): con la hoja bloqueada en exclusivo, busca el padre del
* subárbol (subiendo por los padres de un solo hijo) y, en un KindMarkHalfDead, marca la hoja medio muerta (con la cima
* del subárbol en su clave alta) y en el padre cambia el enlace a la cima por el de su hermana derecha, quitando el de
* la hermana: el espacio de claves del subárbol pasa a la hermana derecha. Antes mira, sin la hoja bloqueada, que la
* hoja no sea la mitad derecha de una partición incompleta (su hermana izquierda), y con ella bloqueada, que su hermana
* derecha no esté medio muerta (no tendría enlace en el padre).
* @param blk page.BlockNumber
* @return bool: si la marcó, error
**/
func (t *BTree) markHalfDead(blk page.BlockNumber) (bool, error) {
	h, n, err := t.lockNode(blk, buffer.Share)
	if err != nil {
		return false, err
	}
	ok, prev := deletableLeaf(n), n.o.Prev
	h.Release()
	if !ok {
		return false, nil
	}
	if pending, err := t.leftSplitPending(prev, blk); err != nil || pending {
		return false, err
	}
	if h, n, err = t.lockNode(blk, buffer.Exclusive); err != nil {
		return false, err
	}
	plan, ok, err := t.halfDeadPlanFor(n)
	if err != nil || !ok {
		h.Release()
		return false, err
	}
	refs := []wal.BlockRef{
		{Delta: blockDelta(plan.top)},
		{Delta: halfDeadParentDelta(plan)},
	}
	hs := []buffer.Handle{h, plan.ph}
	res, err := t.logChange(wal.KindMarkHalfDead, 0, 0, refs, hs, nil)
	releaseAll(hs...)
	if err != nil {
		return false, err
	}
	return true, res.Write()
}

/**
* halfDeadPlanFor comprueba otra vez la hoja ya bloqueada en exclusivo, borra su bit del mapa de visibilidad (si
* lo tiene), mira su hermana derecha y busca el padre del subárbol.
* @param n node: la hoja, bloqueada en exclusivo
* @return halfDeadPlan: con el padre bloqueado si bool, bool: si se puede marcar, error
**/
func (t *BTree) halfDeadPlanFor(n node) (halfDeadPlan, bool, error) {
	if !deletableLeaf(n) {
		return halfDeadPlan{}, false, nil
	}
	if err := t.clearVisible(n); err != nil {
		return halfDeadPlan{}, false, err
	}
	if half, err := t.rightHalfDead(n); err != nil || half {
		return halfDeadPlan{}, false, err
	}
	return t.subtreeParent(n)
}

/**
* leftSplitPending dice si blk es la mitad derecha de una partición incompleta de su hermana izquierda: entonces no
* tiene enlace en el padre (como _bt_leftsib_splitflag). Una vez que lo tiene, no lo pierde: la izquierda no se vuelve
* a partir sin terminar antes esa partición.
* @param prev, blk page.BlockNumber
* @return bool, error
**/
func (t *BTree) leftSplitPending(prev, blk page.BlockNumber) (bool, error) {
	if prev == 0 {
		return false, nil
	}
	h, n, err := t.lockNode(prev, buffer.Share)
	if err != nil {
		return false, err
	}
	defer h.Release()
	return n.o.Next == blk && n.incomplete(), nil
}

/**
* rightHalfDead dice si la hermana derecha de la hoja está medio muerta (como _bt_rightsib_halfdeadflag); la bloquea en
* compartido con la hoja bloqueada (de izquierda a derecha).
* @param n node: la hoja, no la de más a la derecha
* @return bool, error
**/
func (t *BTree) rightHalfDead(n node) (bool, error) {
	h, rn, err := t.lockNode(n.o.Next, buffer.Share)
	if err != nil {
		return false, err
	}
	defer h.Release()
	return rn.halfDead(), nil
}

/**
* subtreeParent busca, con la hoja bloqueada, el padre del subárbol que se borra (como _bt_lock_subtree_parent):
* sube mientras la página sea el único enlace de un padre que se puede borrar (ni el de más a la derecha de su nivel,
* ni la raíz, ni con partición incompleta); con la hoja bloqueada nadie puede agregarle hijos a esa cadena. Si la cima
* es el último enlace de un padre con más hijos, no se puede borrar (su hermana derecha es de otro padre).
* @param n node: la hoja, bloqueada en exclusivo
* @return halfDeadPlan, bool: si se encontró, error
**/
func (t *BTree) subtreeParent(n node) (halfDeadPlan, bool, error) {
	hk, err := n.highKey()
	if err != nil {
		return halfDeadPlan{}, false, err
	}
	top := chainTop{blk: n.blk, next: n.o.Next, high: cloneKey(hk.key())}
	for {
		search := top.high
		search.before = true // baja hasta la página cuya clave alta es esta, no hasta la hermana
		ph, pn, off, err := t.findParent(top.blk, search, top.level+1)
		var fe *finishError
		if errors.As(err, &fe) {
			return halfDeadPlan{}, false, nil // una partición sin terminar en el camino: otra pasada
		}
		if err != nil {
			return halfDeadPlan{}, false, err
		}
		if off < pn.maxOff() {
			plan, err := planAt(ph, pn, off, top, n.blk)
			return plan, err == nil, err
		}
		next, ok, err := chainParent(pn, off)
		ph.Release()
		if err != nil || !ok {
			return halfDeadPlan{}, false, err
		}
		top = next
	}
}

/**
* planAt arma el plan con el padre del subárbol bloqueado: el enlace siguiente al de la cima tiene que ser su hermana
* derecha (si no, el árbol está dañado). Con error suelta el padre.
* @param ph buffer.Handle, pn node: el padre, off page.OffsetNumber: el enlace a la cima, top chainTop
* @param leaf page.BlockNumber: la hoja
* @return halfDeadPlan, error
**/
func planAt(ph buffer.Handle, pn node, off page.OffsetNumber, top chainTop, leaf page.BlockNumber) (halfDeadPlan, error) {
	v, err := pn.pivotAt(off + 1)
	if err == nil && v.downlink() != top.next {
		err = corruptAt(pn.blk, "link after %d is %d, not its right sibling %d", top.blk, v.downlink(), top.next)
	}
	if err != nil {
		ph.Release()
		return halfDeadPlan{}, err
	}
	plan := halfDeadPlan{ph: ph, off: off, child: top.blk, right: top.next}
	if top.blk != leaf {
		plan.top = top.blk
	}
	return plan, nil
}

/**
* chainParent decide si el padre, del que la cima es el último enlace, entra en la cadena que se borra: solo si es su
* único enlace y el padre se puede borrar.
* @param pn node: el padre bloqueado, off page.OffsetNumber: el enlace a la cima
* @return chainTop: el padre como cima nueva, bool, error
**/
func chainParent(pn node, off page.OffsetNumber) (chainTop, bool, error) {
	if off != pn.firstData() || pn.rightmost() || pn.root() || pn.incomplete() {
		return chainTop{}, false, nil
	}
	hk, err := pn.highKey()
	if err != nil {
		return chainTop{}, false, err
	}
	return chainTop{blk: pn.blk, level: pn.o.Level, next: pn.o.Next, high: cloneKey(hk.key())}, true, nil
}

/**
* unlinkHalfDead termina el borrado de una hoja medio muerta (como el bucle de _bt_unlink_halfdead_page): desengancha
* la cima de su subárbol, después cada padre de un solo hijo hacia abajo, y por último la hoja. También termina un
* borrado que un corte dejó a medias.
* @param leaf page.BlockNumber, opts VacuumOptions, st *VacuumStats
* @return error
**/
func (t *BTree) unlinkHalfDead(leaf page.BlockNumber, opts VacuumOptions, st *VacuumStats) error {
	for {
		unlinked, done, err := t.unlinkStep(leaf, opts)
		if unlinked {
			st.Deleted++
		}
		if err != nil || done {
			return err
		}
	}
}

// unlinkLocks son las páginas que bloquea un KindUnlinkPage, en el orden en que se bloquean.
type unlinkLocks struct {
	leaf, left, target, right, meta buffer.Handle
}

// release suelta todo lo que esté bloqueado.
func (l *unlinkLocks) release() {
	releaseAll(l.leaf, l.left, l.target, l.right, l.meta)
}

/**
* unlinkStep desengancha la página de más arriba que queda del subárbol de la hoja medio muerta leaf. Bloquea como
* nbtree: la hoja (si no es ella la que se borra), la hermana izquierda actual de la que se borra (siguiendo a la
* derecha si se partió), la que se borra, su hermana derecha y la meta; y en un KindUnlinkPage enlaza las hermanas
* entre sí, deja la página borrada (con sus enlaces, para los lectores que todavía llegan a ella) con su safexid,
* la apila en la meta y, si la borrada era un padre, deja en la hoja la cima nueva.
* @param leaf page.BlockNumber, opts VacuumOptions
* @return bool: si desenganchó una página, bool: si la hoja ya quedó borrada, error
**/
func (t *BTree) unlinkStep(leaf page.BlockNumber, opts VacuumOptions) (bool, bool, error) {
	target, left, err := t.unlinkTargetOf(leaf)
	if err != nil || target == 0 {
		return false, target == 0, err
	}
	var l unlinkLocks
	refs, err := t.lockForUnlink(&l, leaf, target, left, opts)
	if err != nil {
		l.release()
		return false, false, err
	}
	if t.beforeUnlink != nil {
		if err := t.beforeUnlink(target); err != nil {
			l.release()
			return false, false, err
		}
	}
	hs := []buffer.Handle{l.target, l.right, l.meta}
	for _, h := range []buffer.Handle{l.left, l.leaf} {
		if h != nil {
			hs = append(hs, h)
		}
	}
	res, err := t.logChange(wal.KindUnlinkPage, 0, 0, refs, hs, nil)
	l.release()
	if err != nil {
		return false, false, err
	}
	return true, target == leaf, res.Write()
}

/**
* unlinkTargetOf lee qué página se desengancha ahora: la cima que anota la hoja medio muerta, o la hoja si no anota
* ninguna; y su hermana izquierda (una pista: se confirma con los bloqueos).
* @param leaf page.BlockNumber
* @return page.BlockNumber: 0 si la hoja ya no está medio muerta, page.BlockNumber: la izquierda, error
**/
func (t *BTree) unlinkTargetOf(leaf page.BlockNumber) (page.BlockNumber, page.BlockNumber, error) {
	h, n, err := t.lockNode(leaf, buffer.Share)
	if err != nil {
		return 0, 0, err
	}
	if !n.halfDead() {
		h.Release()
		return 0, 0, nil
	}
	top, left := leafTop(n), n.o.Prev
	h.Release()
	if top == 0 {
		return leaf, left, nil
	}
	th, tn, err := t.lockNode(top, buffer.Share)
	if err != nil {
		return 0, 0, err
	}
	defer th.Release()
	if tn.leaf() || tn.ignore() {
		return 0, 0, corruptAt(top, "top of a half-dead leaf is not an internal page")
	}
	return top, tn.o.Prev, nil
}

/**
* lockForUnlink bloquea las páginas en el orden de nbtree (la hoja, la izquierda, la que se borra, la derecha, la meta)
* y arma los bloques del registro. Con error, lo bloqueado queda en l para que quien llama lo suelte.
* @param l *unlinkLocks, leaf, target, left page.BlockNumber, opts VacuumOptions
* @return []wal.BlockRef, error
**/
func (t *BTree) lockForUnlink(l *unlinkLocks, leaf, target, left page.BlockNumber, opts VacuumOptions) ([]wal.BlockRef, error) {
	var leafRef *wal.BlockRef
	if target != leaf {
		h, n, err := t.lockNode(leaf, buffer.Exclusive)
		if l.leaf = h; err != nil {
			return nil, err
		}
		if !n.halfDead() || leafTop(n) != target {
			return nil, corruptAt(leaf, "half-dead leaf changed while unlinking %d", target)
		}
	}
	left, err := t.lockLeftOf(l, left, target)
	if err != nil {
		return nil, err
	}
	tn, next, err := t.lockTarget(l, target, left, leaf)
	if err != nil {
		return nil, err
	}
	rn, err := t.lockRight(l, tn)
	if err != nil {
		return nil, err
	}
	if target != leaf {
		leafRef = &wal.BlockRef{Delta: blockDelta(next)}
	}
	return t.unlinkRefs(l, tn, rn, left, leafRef, opts)
}

/**
* lockLeftOf bloquea en exclusivo la hermana izquierda actual de target: desde la pista, siguiendo a la derecha hasta
* la página no borrada cuyo enlace derecho es target (la pista pudo partirse). Un enlace que no avanza es un daño.
* @param l *unlinkLocks, left, target page.BlockNumber
* @return page.BlockNumber: la izquierda (0 si target es la de más a la izquierda), error
**/
func (t *BTree) lockLeftOf(l *unlinkLocks, left, target page.BlockNumber) (page.BlockNumber, error) {
	for left != 0 {
		h, n, err := t.lockNode(left, buffer.Exclusive)
		if err != nil {
			return 0, err
		}
		if !n.deleted() && n.o.Next == target {
			l.left = h
			return left, nil
		}
		next := n.o.Next
		h.Release()
		if next == 0 || next == left || n.deleted() {
			return 0, corruptAt(left, "left sibling chain of %d is broken", target)
		}
		left = next
	}
	return 0, nil
}

/**
* lockTarget bloquea en exclusivo la página que se borra y comprueba su estado: no es la de más a la derecha ni la
* raíz ni está borrada, su izquierda es la bloqueada; si es la hoja, medio muerta y vacía; si es un padre, con un solo
* enlace (la cima nueva que queda en la hoja, 0 si ese enlace es la hoja).
* @param l *unlinkLocks, target, left, leaf page.BlockNumber
* @return node, page.BlockNumber: la cima siguiente, error
**/
func (t *BTree) lockTarget(l *unlinkLocks, target, left, leaf page.BlockNumber) (node, page.BlockNumber, error) {
	h, n, err := t.lockNode(target, buffer.Exclusive)
	if l.target = h; err != nil {
		return node{}, 0, err
	}
	switch {
	case n.rightmost() || n.root() || n.deleted():
		return node{}, 0, corruptAt(target, "page to unlink changed status")
	case n.o.Prev != left:
		return node{}, 0, corruptAt(target, "left link %d, expected %d", n.o.Prev, left)
	case target == leaf:
		if !n.halfDead() || n.dataCount() != 0 {
			return node{}, 0, corruptAt(target, "leaf to unlink is not half-dead and empty")
		}
		return n, 0, nil
	case n.leaf() || n.ignore() || n.dataCount() != 1:
		return node{}, 0, corruptAt(target, "parent to unlink does not have exactly one link")
	}
	v, err := n.pivotAt(n.firstData())
	if err != nil {
		return node{}, 0, err
	}
	next := v.downlink()
	if next == leaf {
		next = 0
	}
	return n, next, nil
}

/**
* lockRight bloquea en exclusivo la hermana derecha de la página que se borra y comprueba que la tenga a la izquierda.
* @param l *unlinkLocks, tn node: la que se borra
* @return node, error
**/
func (t *BTree) lockRight(l *unlinkLocks, tn node) (node, error) {
	h, rn, err := t.lockNode(tn.o.Next, buffer.Exclusive)
	if l.right = h; err != nil {
		return node{}, err
	}
	if rn.o.Prev != tn.blk || rn.o.Level != tn.o.Level || rn.deleted() {
		return node{}, corruptAt(rn.blk, "right sibling does not link back to %d", tn.blk)
	}
	return rn, nil
}

/**
* unlinkRefs bloquea la meta y arma los bloques del KindUnlinkPage: la borrada (iniciada entera con sus enlaces, su
* safexid y la cabeza anterior de la pila), la derecha (prev), la meta (la pila y, si cambia, la raíz rápida), la
* izquierda (next) y la hoja (la cima nueva). La raíz rápida pasa a la derecha si queda sola en su nivel; si la
* borrada era la raíz rápida y la derecha no queda sola, vuelve a la raíz.
* @param l *unlinkLocks, tn, rn node, left page.BlockNumber, leafRef *wal.BlockRef: nil si se borra la hoja
* @param opts VacuumOptions
* @return []wal.BlockRef, error
**/
func (t *BTree) unlinkRefs(l *unlinkLocks, tn, rn node, left page.BlockNumber, leafRef *wal.BlockRef, opts VacuumOptions) ([]wal.BlockRef, error) {
	h, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return nil, err
	}
	l.meta = h
	h.Lock(buffer.Exclusive)
	m, err := readMetaPage(h.Page())
	if err != nil {
		return nil, err
	}
	d := unlinkTarget{prev: left, next: rn.blk, level: tn.o.Level, info: deletedInfo{safe: opts.safeXID(), next: m.deleted}}
	metaDelta := blockDelta(tn.blk)
	switch {
	case left == 0 && rn.rightmost():
		d.flags |= unlinkSetsFast
		metaDelta = append(metaDelta, metaFastDelta(rn.blk, rn.o.Level)...)
	case m.fastRoot == tn.blk:
		d.flags |= unlinkSetsFast
		metaDelta = append(metaDelta, metaFastDelta(m.root, m.level)...)
	}
	refs := []wal.BlockRef{
		{Flags: wal.BlockWillInit, Delta: unlinkTargetDelta(d)},
		{Delta: blockDelta(left)},
		{Delta: metaDelta},
	}
	if left != 0 {
		refs = append(refs, wal.BlockRef{Delta: blockDelta(rn.blk)})
		d.flags |= unlinkHasLeft
	}
	if leafRef != nil {
		refs = append(refs, *leafRef)
		d.flags |= unlinkHasLeaf
	}
	refs[0].Delta = unlinkTargetDelta(d)
	return refs, nil
}

/**
* safeXID es la marca de una página que se borra: el próximo xid (o el horizonte de la pasada, sin NextXID).
* @return page.XID
**/
func (o VacuumOptions) safeXID() page.XID {
	if o.NextXID != nil {
		return o.NextXID()
	}
	return o.Horizon
}

/**
* recycle devuelve a la lista de libres las páginas borradas que ya nadie puede ver (como _bt_pendingfsm_finalize):
* la pila de la meta tiene arriba la más nueva, así que se recicla mientras la de arriba tenga safexid < Horizon (todo
* snapshot de antes de su borrado ya cerró) y, antes de la primera, se espera a las operaciones sueltas en vuelo
* (guard): ninguna puede tener ya un enlace a una página desenganchada antes. Cada una es un KindRecyclePage.
* @param opts VacuumOptions, st *VacuumStats
* @return error
**/
func (t *BTree) recycle(opts VacuumOptions, st *VacuumStats) error {
	if ok, err := t.recyclable(opts); err != nil || !ok {
		return err
	}
	if !t.guard.drain(guardTimeout) {
		return nil
	}
	for {
		done, err := t.recycleOne(opts)
		if err != nil || done {
			return err
		}
		st.Recycled++
	}
}

/**
* recyclable dice si la página de arriba de la pila de borradas se puede reciclar ya.
* @param opts VacuumOptions
* @return bool, error
**/
func (t *BTree) recyclable(opts VacuumOptions) (bool, error) {
	m, err := t.readMeta()
	if err != nil || m.deleted == 0 {
		return false, err
	}
	h, n, err := t.lockNode(m.deleted, buffer.Share)
	if err != nil {
		return false, err
	}
	defer h.Release()
	if !n.deleted() {
		return false, corruptAt(n.blk, "page in the deleted stack is not deleted")
	}
	return deletedInfoOf(n).safe < opts.Horizon, nil
}

/**
* recycleOne recicla la página de arriba de la pila si todavía se puede: con la meta y la página bloqueadas en
* exclusivo (meta antes que la página, como fsm), un KindRecyclePage la saca de la pila y la deja libre en la cabeza de
* la lista de libres.
* @param opts VacuumOptions
* @return bool: true si no quedó nada que reciclar, error
**/
func (t *BTree) recycleOne(opts VacuumOptions) (bool, error) {
	metaH, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return false, err
	}
	metaH.Lock(buffer.Exclusive)
	m, err := readMetaPage(metaH.Page())
	if err != nil || m.deleted == 0 {
		metaH.Release()
		return true, err
	}
	h, n, err := t.lockNode(m.deleted, buffer.Exclusive)
	if err != nil {
		metaH.Release()
		return false, err
	}
	info := deletedInfoOf(n)
	if !n.deleted() || info.safe >= opts.Horizon {
		releaseAll(h, metaH)
		if !n.deleted() {
			return false, corruptAt(n.blk, "page in the deleted stack is not deleted")
		}
		return true, nil
	}
	refs := []wal.BlockRef{
		{Delta: le.AppendUint32(blockDelta(info.next), uint32(n.blk))},
		{Flags: wal.BlockWillInit, Delta: blockDelta(m.freeHead)},
	}
	hs := []buffer.Handle{metaH, h}
	res, err := t.logChange(wal.KindRecyclePage, 0, 0, refs, hs, nil)
	releaseAll(hs...)
	if err != nil {
		return false, err
	}
	return false, res.Write()
}

/**
* leafTop lee la cima del subárbol que anota una hoja medio muerta en el enlace de su clave alta (0 = la hoja misma).
* @param n node: medio muerta (tiene clave alta)
* @return page.BlockNumber
**/
func leafTop(n node) page.BlockNumber {
	hk, err := n.highKey()
	if err != nil {
		return 0
	}
	return hk.downlink()
}

/**
* checkDeleted valida una página borrada: un solo ítem con sus datos, no es la de más a la derecha (los lectores que
* llegan siguen a la derecha) y no está medio muerta.
* @param n node
* @return error: ErrCorrupt
**/
func checkDeleted(n node) error {
	if n.maxOff() != page.FirstOffset || n.rightmost() || n.halfDead() {
		return corruptAt(n.blk, "deleted page without its data or its right link")
	}
	if b, err := n.p.Item(page.FirstOffset); err != nil || len(b) != deletedInfoSize {
		return corruptAt(n.blk, "deleted page data of a wrong size")
	}
	return nil
}

/**
* deletedInfoOf lee los datos de una página borrada ya validada (cero si no es una borrada).
* @param n node
* @return deletedInfo
**/
func deletedInfoOf(n node) deletedInfo {
	if !n.deleted() {
		return deletedInfo{}
	}
	b, _ := n.p.Item(page.FirstOffset)
	return deletedInfo{safe: page.XID(le.Uint64(b)), next: page.BlockNumber(le.Uint32(b[8:]))}
}

/**
* encodeDeletedInfo arma el ítem de una página borrada.
* @param info deletedInfo
* @return []byte
**/
func encodeDeletedInfo(info deletedInfo) []byte {
	b := le.AppendUint64(make([]byte, 0, deletedInfoSize), uint64(info.safe))
	b = le.AppendUint32(b, uint32(info.next))
	return le.AppendUint32(b, 0)
}

// halfDeadParentDelta arma el delta del padre del subárbol: off(2) child(4) right(4).
func halfDeadParentDelta(plan halfDeadPlan) []byte {
	b := le.AppendUint16(nil, uint16(plan.off))
	b = le.AppendUint32(b, uint32(plan.child))
	return le.AppendUint32(b, uint32(plan.right))
}

// metaFastDelta arma la raíz rápida de un delta de la meta: fastRoot(4) fastLevel(4).
func metaFastDelta(fast page.BlockNumber, level uint32) []byte {
	return le.AppendUint32(blockDelta(fast), level)
}

// unlinkTargetDelta arma el delta de la página que se borra.
func unlinkTargetDelta(d unlinkTarget) []byte {
	b := le.AppendUint32(nil, uint32(d.prev))
	b = le.AppendUint32(b, uint32(d.next))
	b = le.AppendUint32(b, d.level)
	b = le.AppendUint64(b, uint64(d.info.safe))
	b = le.AppendUint32(b, uint32(d.info.next))
	return le.AppendUint16(b, d.flags)
}

/**
* parseUnlinkTarget lee el delta de la página que se borra: largo exacto, enlaces que no apuntan a la meta (la derecha
* tiene que existir) y marcas conocidas.
* @param delta []byte
* @return unlinkTarget, error: ErrBadRecord
**/
func parseUnlinkTarget(delta []byte) (unlinkTarget, error) {
	if len(delta) != unlinkTargetSize {
		return unlinkTarget{}, badRecord(wal.KindUnlinkPage, "bad unlinked page delta length")
	}
	d := unlinkTarget{
		prev:  page.BlockNumber(le.Uint32(delta)),
		next:  page.BlockNumber(le.Uint32(delta[4:])),
		level: le.Uint32(delta[8:]),
		info:  deletedInfo{safe: page.XID(le.Uint64(delta[12:])), next: page.BlockNumber(le.Uint32(delta[20:]))},
		flags: le.Uint16(delta[24:]),
	}
	if d.next == MetaBlock || d.flags&^unlinkAllFlags != 0 || (d.prev != 0) != (d.flags&unlinkHasLeft != 0) {
		return unlinkTarget{}, badRecord(wal.KindUnlinkPage, "bad unlinked page links or flags")
	}
	return d, nil
}

/**
* unlinkRole da el papel del bloque i de un KindUnlinkPage según las marcas de su primer bloque: los opcionales
* (izquierda, hoja) que no vienen no ocupan lugar.
* @param blocks []wal.BlockRef, i int
* @return int: roleTarget… roleLeaf, o -1 si no hay tal bloque
**/
func unlinkRole(blocks []wal.BlockRef, i int) int {
	if i < roleLeft {
		return i
	}
	d, err := parseUnlinkTarget(blocks[0].Delta)
	if err != nil {
		return -1
	}
	roles := []int{roleTarget, roleRight, roleMeta}
	if d.flags&unlinkHasLeft != 0 {
		roles = append(roles, roleLeft)
	}
	if d.flags&unlinkHasLeaf != 0 {
		roles = append(roles, roleLeaf)
	}
	if i >= len(roles) {
		return -1
	}
	return roles[i]
}

/**
* checkUnlinkShape valida un KindUnlinkPage antes de tocar páginas: tantos bloques como dicen las marcas, la borrada
* con BlockWillInit, la meta en su lugar, los bloques de la derecha y de la izquierda iguales a los enlaces del delta,
* y los deltas de cada uno.
* @param rec *wal.Record
* @return error: ErrBadRecord
**/
func checkUnlinkShape(rec *wal.Record) error {
	if len(rec.Blocks) < 3 {
		return badRecord(rec.Kind, "needs at least three blocks")
	}
	d, err := parseUnlinkTarget(rec.Blocks[0].Delta)
	if err != nil {
		return err
	}
	want := 3
	for _, f := range []uint16{unlinkHasLeft, unlinkHasLeaf} {
		if d.flags&f != 0 {
			want++
		}
	}
	if len(rec.Blocks) != want || rec.Blocks[0].Flags&wal.BlockWillInit == 0 || rec.Blocks[roleRight].Block != d.next {
		return badRecord(rec.Kind, "blocks do not match the unlinked page delta")
	}
	for i := range rec.Blocks {
		role, b := unlinkRole(rec.Blocks, i), rec.Blocks[i]
		if (b.Block == MetaBlock) != (role == roleMeta) || (role == roleLeft && b.Block != d.prev) {
			return badRecord(rec.Kind, "block in the wrong place")
		}
		if err := checkUnlinkDelta(role, b.Delta, d); err != nil {
			return err
		}
	}
	return nil
}

/**
* checkUnlinkDelta valida el delta de un bloque de KindUnlinkPage por su papel.
* @param role int, delta []byte, d unlinkTarget: el delta de la borrada
* @return error: ErrBadRecord
**/
func checkUnlinkDelta(role int, delta []byte, d unlinkTarget) error {
	want := blockDeltaSize
	switch role {
	case roleTarget:
		return nil
	case roleMeta:
		if d.flags&unlinkSetsFast != 0 {
			want += fastDeltaSize
		}
	}
	if len(delta) != want {
		return badRecord(wal.KindUnlinkPage, "bad delta length")
	}
	return nil
}

/**
* applyUnlink aplica un bloque de KindUnlinkPage según su papel.
* @param role int, p page.Page, delta []byte
* @return error
**/
func applyUnlink(role int, p page.Page, delta []byte) error {
	switch role {
	case roleTarget:
		d, err := parseUnlinkTarget(delta)
		if err != nil {
			return err
		}
		return initDeletedPage(p, d)
	case roleRight:
		return applySetPrev(p, page.BlockNumber(le.Uint32(delta)))
	case roleMeta:
		return applyMetaUnlink(p, delta)
	case roleLeft:
		return applySetNext(p, page.BlockNumber(le.Uint32(delta)))
	case roleLeaf:
		return applyLeafTop(wal.KindUnlinkPage, p, page.BlockNumber(le.Uint32(delta)))
	}
	return badRecord(wal.KindUnlinkPage, "block without a role")
}

/**
* initDeletedPage deja p como una página borrada: vacía salvo sus datos de borrada, con sus enlaces y su nivel (una
* hoja conserva page.OpaqueLeaf) y la marca page.OpaqueDeleted.
* @param p page.Page, d unlinkTarget
* @return error
**/
func initDeletedPage(p page.Page, d unlinkTarget) error {
	flags := page.OpaqueDeleted
	if d.level == 0 {
		flags |= page.OpaqueLeaf
	}
	if err := initTreePage(p, page.Opaque{Prev: d.prev, Next: d.next, Level: d.level, Flags: flags}); err != nil {
		return err
	}
	_, err := p.AddItem(encodeDeletedInfo(d.info))
	return err
}

/**
* applyMetaUnlink anota en la meta la página borrada como cabeza de la pila y, si viene, la raíz rápida.
* @param p page.Page, delta []byte: deleted(4) [fastRoot(4) fastLevel(4)]
* @return error
**/
func applyMetaUnlink(p page.Page, delta []byte) error {
	if _, err := readMetaPage(p); err != nil {
		return err
	}
	le.PutUint32(p[offMetaDeleted:], le.Uint32(delta))
	if len(delta) == blockDeltaSize+fastDeltaSize {
		setMetaFast(p, page.BlockNumber(le.Uint32(delta[4:])), le.Uint32(delta[8:]))
	}
	return nil
}

/**
* applySetNext cambia el enlace a la derecha de la hermana izquierda de una página borrada.
* @param p page.Page, next page.BlockNumber
* @return error
**/
func applySetNext(p page.Page, next page.BlockNumber) error {
	o, err := p.Opaque()
	if err != nil {
		return badRecord(wal.KindUnlinkPage, err.Error())
	}
	o.Next = next
	return p.SetOpaque(o)
}

/**
* applyHalfDeadLeaf marca la hoja medio muerta (KindMarkHalfDead, bloque 0): tiene que ser una hoja vacía, no la de
* más a la derecha, sin partición incompleta; apaga page.FlagAllVisible (su bit del mapa se borró antes) y anota la
* cima del subárbol.
* @param p page.Page, delta []byte: top(4)
* @return error
**/
func applyHalfDeadLeaf(p page.Page, delta []byte) error {
	o, err := p.Opaque()
	if err != nil {
		return badRecord(wal.KindMarkHalfDead, err.Error())
	}
	n, err := pageNode(p, o)
	if err != nil {
		return badRecord(wal.KindMarkHalfDead, err.Error())
	}
	if !n.leaf() || n.ignore() || n.rightmost() || n.incomplete() || n.root() || n.dataCount() != 0 {
		return badRecord(wal.KindMarkHalfDead, "page is not an empty leaf that can be deleted")
	}
	o.Flags |= page.OpaqueHalfDead
	if err := p.SetOpaque(o); err != nil {
		return err
	}
	p.SetFlags(p.Flags() &^ page.FlagAllVisible)
	return applyLeafTop(wal.KindMarkHalfDead, p, page.BlockNumber(le.Uint32(delta)))
}

/**
* applyLeafTop anota la cima del subárbol en el enlace de la clave alta de una hoja medio muerta.
* @param kind wal.Kind, p page.Page, top page.BlockNumber: 0 = la hoja misma
* @return error
**/
func applyLeafTop(kind wal.Kind, p page.Page, top page.BlockNumber) error {
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueHalfDead == 0 || o.Next == 0 {
		return badRecord(kind, "not a half-dead leaf")
	}
	hk, err := p.Item(page.FirstOffset)
	if err != nil || len(hk) < PivotHeaderSize {
		return badRecord(kind, "half-dead leaf without its high key")
	}
	le.PutUint32(hk[offPivotDownlink:], uint32(top))
	return nil
}

/**
* applyHalfDeadParent cambia el enlace a la cima del subárbol por el de su hermana derecha y quita el de la hermana
* (KindMarkHalfDead, bloque 1): el espacio de claves del subárbol pasa a la hermana. Valida los dos enlaces.
* @param p page.Page, delta []byte: off(2) child(4) right(4)
* @return error
**/
func applyHalfDeadParent(p page.Page, delta []byte) error {
	off := page.OffsetNumber(le.Uint16(delta))
	child, right := page.BlockNumber(le.Uint32(delta[2:])), page.BlockNumber(le.Uint32(delta[6:]))
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf != 0 {
		return badRecord(wal.KindMarkHalfDead, "parent is not an internal page")
	}
	n, err := pageNode(p, o)
	if err != nil {
		return badRecord(wal.KindMarkHalfDead, err.Error())
	}
	if off < n.firstData() || off >= n.maxOff() {
		return badRecord(wal.KindMarkHalfDead, "parent link outside the page")
	}
	at, err := p.Item(off)
	if err != nil {
		return badRecord(wal.KindMarkHalfDead, err.Error())
	}
	nextItem, err := p.Item(off + 1)
	if err != nil || len(at) < PivotHeaderSize || len(nextItem) < PivotHeaderSize ||
		pivot(at).downlink() != child || pivot(nextItem).downlink() != right {
		return badRecord(wal.KindMarkHalfDead, "parent links do not match")
	}
	le.PutUint32(at[offPivotDownlink:], uint32(right))
	if err := p.DeleteItem(off + 1); err != nil {
		return badRecord(wal.KindMarkHalfDead, err.Error())
	}
	return nil
}

/**
* checkHalfDeadDelta valida los deltas de KindMarkHalfDead: la cima (0 = la hoja misma) y el padre (número de enlace
* y dos bloques que no son la meta).
* @param i int, delta []byte
* @return error: ErrBadRecord
**/
func checkHalfDeadDelta(i int, delta []byte) error {
	if i == 0 {
		if len(delta) != blockDeltaSize {
			return badRecord(wal.KindMarkHalfDead, "bad half-dead leaf delta")
		}
		return nil
	}
	if len(delta) != halfDeadParentSize || le.Uint16(delta) == 0 ||
		page.BlockNumber(le.Uint32(delta[2:])) <= MetaBlock || page.BlockNumber(le.Uint32(delta[6:])) <= MetaBlock {
		return badRecord(wal.KindMarkHalfDead, "bad half-dead parent delta")
	}
	return nil
}

/**
* checkRecycleShape valida un KindRecyclePage: la meta con la pila y la lista nuevas (la página reciclada es la cabeza
* nueva de la lista), y la página con BlockWillInit y su siguiente libre.
* @param rec *wal.Record
* @return error: ErrBadRecord
**/
func checkRecycleShape(rec *wal.Record) error {
	if len(rec.Blocks) != 2 || len(rec.Blocks[0].Delta) != recycleMetaSize || len(rec.Blocks[1].Delta) != blockDeltaSize {
		return badRecord(rec.Kind, "bad recycle blocks or deltas")
	}
	meta, pg := rec.Blocks[0], rec.Blocks[1]
	if meta.Block != MetaBlock || pg.Block == MetaBlock || pg.Flags&wal.BlockWillInit == 0 ||
		page.BlockNumber(le.Uint32(meta.Delta[4:])) != pg.Block {
		return badRecord(rec.Kind, "recycle blocks do not match")
	}
	return nil
}

/**
* applyRecycle aplica un bloque de KindRecyclePage: en la meta, la pila de borradas y la lista de libres nuevas; la
* página queda libre con el formato de fsm (page.OpaqueFree, next = la cabeza anterior de la lista).
* @param i int, p page.Page, delta []byte
* @return error
**/
func applyRecycle(i int, p page.Page, delta []byte) error {
	if i == 1 {
		return initTreePage(p, page.Opaque{Next: page.BlockNumber(le.Uint32(delta)), Flags: page.OpaqueFree})
	}
	if _, err := readMetaPage(p); err != nil {
		return err
	}
	le.PutUint32(p[offMetaDeleted:], le.Uint32(delta))
	le.PutUint32(p[offMetaFreeHead:], le.Uint32(delta[4:]))
	return nil
}
