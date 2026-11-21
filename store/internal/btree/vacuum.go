package btree

import (
	"bytes"
	"slices"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

const (
	// vacuumHeaderSize es el encabezado del delta de KindLeafVacuum: nMark(2) nRemove(2).
	vacuumHeaderSize = 2 + 2

	// freezeHeaderSize es el encabezado del congelado, al final del delta: nFreeze(2) pageFlags(2).
	freezeHeaderSize = 2 + 2

	// vacuumSetVisible, en pageFlags del delta: la hoja queda con page.FlagAllVisible.
	vacuumSetVisible uint16 = 1 << 0
)

// leafPlan es lo que la limpieza hace en una hoja: marcar «liberándose» las versiones muertas con desborde, quitar
// las demás muertas, congelar las visibles para todos, las versiones cuyas cadenas hay que liberar (las recién
// marcadas y las que ya lo estaban) y, si la hoja queda toda visible o toda congelada, sus bits del mapa.
type leafPlan struct {
	mark   []page.OffsetNumber
	remove []page.OffsetNumber
	freeze []page.OffsetNumber
	chains []Version
	kept   []keptEntry // versiones que no se quitan, en orden de número

	busy       bool // queda una versión que no es visible para todos (o «liberándose»)
	shared     bool // la hoja comparte un id con una vecina: no lleva bits (la limpieza no puede saltarla)
	vmBits     fsm.VMBits
	oldest     page.XID // el menor xid de lo que queda sin congelar
	setVisible bool     // el registro pone page.FlagAllVisible
}

// keptEntry es una versión que la limpieza deja en la hoja, con lo que hace falta para congelarla.
type keptEntry struct {
	off    page.OffsetNumber
	xid    page.XID
	status xact.Status
	frozen bool
}

// pageVacuum es el cambio de un KindLeafVacuum en una hoja.
type pageVacuum struct {
	mark, remove, freeze []page.OffsetNumber
	setVisible           bool
}

// empty dice si el cambio no hace nada.
func (pv pageVacuum) empty() bool {
	return len(pv.mark)+len(pv.remove)+len(pv.freeze) == 0 && !pv.setVisible
}

// vacuumGroup son las versiones de un id que la limpieza está mirando, de la más nueva a la más vieja.
type vacuumGroup struct {
	id        []byte            // copia
	anchored  bool              // ya apareció la versión de referencia: la más nueva confirmada visible para todos
	anchorLSN page.LSN          // su lsn: toda versión del id más vieja está muerta
	tomb      page.OffsetNumber // la versión de referencia, si es una lápida de esta hoja (0 = no)
	behind    bool              // detrás de la versión de referencia queda una versión que no se quita
}

// vacuumRun es una pasada de la limpieza sobre el árbol.
type vacuumRun struct {
	t     *BTree
	opts  VacuumOptions
	st    VacuumStats
	carry vacuumGroup // el último id de la hoja recorrida antes (sus versiones pueden seguir en la siguiente)
}

// parentLinks son los enlaces de una página del nivel 1, copiados con ella bloqueada, y su hermana derecha.
type parentLinks struct {
	downs []page.BlockNumber
	next  page.BlockNumber
}

/**
* Vacuum limpia el store (como VACUUM): quita las versiones que ningún snapshot puede ver (las de transacciones
* abortadas, las que reemplazó una versión confirmada visible para todos —xid < Horizon— y la lápida que queda sin
* versiones detrás), congela las confirmadas con xid < FreezeLimit y anota en el mapa de visibilidad las hojas que
* quedan todo-visibles o todo-congeladas. Con mapa y un árbol de dos niveles o más recorre las hojas por los enlaces
* del nivel 1 y salta, sin leerlas, las que el mapa da por todo-congeladas (y sin Aggressive, también las
* todo-visibles); si no, recorre todas por sus enlaces. Las versiones con desborde se marcan «liberándose» en el
* mismo registro (KindLeafVacuum), se libera su cadena con opts.Free y recién entonces se quitan (pregunta 58). Antes
* de recorrer recicla las páginas borradas que ya nadie puede ver, y cada hoja que queda vacía se borra del árbol
* (deleteLeaf; «Borrado de páginas» en doc.go). Puede correr con lecturas y escrituras; quien llama no corre dos
* limpiezas del mismo store a la vez.
* @param opts VacuumOptions
* @return VacuumStats, error
**/
func (t *BTree) Vacuum(opts VacuumOptions) (VacuumStats, error) {
	opts.FreezeLimit = min(opts.FreezeLimit, opts.Horizon)
	r := &vacuumRun{t: t, opts: opts}
	r.st.FrozenXID = opts.Horizon
	err := t.recycle(opts, &r.st)
	if err == nil {
		err = t.finishLeftmost(opts, &r.st)
	}
	var m meta
	if err == nil {
		m, err = t.readMeta()
	}
	if err == nil {
		if m.level == 0 || t.vm == nil {
			err = r.leftmostChain()
		} else {
			err = r.byParents()
		}
	}
	r.st.Complete = err == nil && r.st.SkippedVisible == 0
	return r.st, err
}

/**
* finishLeftmost termina los borrados a medias que quedaron a la izquierda de la primera hoja con enlace: esas hojas
* medio muertas no se alcanzan bajando (su espacio de claves ya es de su hermana derecha) ni siguiendo enlaces a la
* derecha desde ahí, así que se buscan por los enlaces izquierdos.
* @param opts VacuumOptions, st *VacuumStats
* @return error
**/
func (t *BTree) finishLeftmost(opts VacuumOptions, st *VacuumStats) error {
	for {
		h, n, _, err := t.descend(minusInf, 0, buffer.Share, false)
		if err != nil {
			return err
		}
		blk, prev := n.blk, n.o.Prev
		h.Release()
		if prev == 0 {
			return nil
		}
		for prev != 0 {
			if h, n, err = t.lockNode(prev, buffer.Share); err != nil {
				return err
			}
			blk, prev = n.blk, n.o.Prev
			h.Release()
		}
		if !n.halfDead() {
			return corruptAt(blk, "leaf left of the first linked leaf is not half-dead")
		}
		if err := t.unlinkHalfDead(blk, opts, st); err != nil {
			return err
		}
	}
}

/**
* leftmostChain recorre todas las hojas desde la de más a la izquierda por sus enlaces, sin saltar ninguna.
* @return error
**/
func (r *vacuumRun) leftmostChain() error {
	h, n, _, err := r.t.descend(minusInf, 0, buffer.Share, false)
	if err != nil {
		return err
	}
	first := n.blk
	h.Release()
	return r.chain(first, 0)
}

/**
* byParents recorre las hojas por los enlaces del nivel 1, de izquierda a derecha. Cada página del nivel 1 se lee con
* bloqueo compartido y se copian sus enlaces (también los de la siguiente, para saber dónde termina la última cadena);
* cada enlace es una hoja que puede saltarse por el mapa, y desde una hoja leída se sigue a la derecha hasta el enlace
* siguiente: así se ven las hojas nuevas de particiones posteriores a la copia (o sin enlace en el padre todavía).
* @return error
**/
func (r *vacuumRun) byParents() error {
	h, n, _, err := r.t.descend(minusInf, 1, buffer.Share, false)
	if err != nil {
		return err
	}
	cur, err := readParentLinks(h, n)
	for err == nil {
		var nxt parentLinks
		if cur.next != 0 {
			if nxt, err = r.t.parentLinksAt(cur.next); err != nil {
				return err
			}
		}
		for i, d := range cur.downs {
			stop := page.BlockNumber(0)
			if i+1 < len(cur.downs) {
				stop = cur.downs[i+1]
			} else if len(nxt.downs) > 0 {
				stop = nxt.downs[0]
			}
			if err = r.downlink(d, stop); err != nil {
				return err
			}
		}
		if cur.next == 0 {
			return nil
		}
		cur = nxt
	}
	return err
}

/**
* parentLinksAt bloquea en compartido la página blk del nivel 1 y copia sus enlaces.
* @param blk page.BlockNumber
* @return parentLinks, error
**/
func (t *BTree) parentLinksAt(blk page.BlockNumber) (parentLinks, error) {
	h, n, err := t.lockNode(blk, buffer.Share)
	if err != nil {
		return parentLinks{}, err
	}
	if n.o.Level != 1 {
		h.Release()
		return parentLinks{}, corruptAt(blk, "right sibling at level %d, expected 1", n.o.Level)
	}
	return readParentLinks(h, n)
}

/**
* readParentLinks copia los enlaces de una página del nivel 1 bloqueada y la suelta.
* @param h buffer.Handle, n node
* @return parentLinks, error
**/
func readParentLinks(h buffer.Handle, n node) (parentLinks, error) {
	defer h.Release()
	pl := parentLinks{next: n.o.Next}
	for off := n.firstData(); off <= n.maxOff(); off++ {
		v, err := n.pivotAt(off)
		if err != nil {
			return parentLinks{}, err
		}
		pl.downs = append(pl.downs, v.downlink())
	}
	return pl, nil
}

/**
* downlink limpia la hoja de un enlace del padre (o la salta por el mapa) y las que la siguen por la derecha hasta
* stop. Saltar es seguro: el bit solo lo pone la limpieza con la hoja sin partición incompleta, y toda escritura o
* partición de la hoja lo borra antes de cambiarla, así que una hoja con el bit puesto no tiene a su derecha páginas
* fuera de la copia de los enlaces (y lo que una partición posterior se lleve está congelado o es de un xid ≥ Horizon).
* @param d, stop page.BlockNumber
* @return error
**/
func (r *vacuumRun) downlink(d, stop page.BlockNumber) error {
	bits, err := r.t.vm.Get(r.t.file, d)
	if err != nil {
		return err
	}
	frozen, visible := bits&fsm.VMAllFrozen != 0, bits&fsm.VMAllVisible != 0
	if frozen || (visible && !r.opts.Aggressive) {
		r.st.Skipped++
		if !frozen {
			r.st.SkippedVisible++
		}
		r.carry = vacuumGroup{} // la hoja saltada no comparte ids con la siguiente (no tendría bits)
		return nil
	}
	return r.chain(d, stop)
}

/**
* chain limpia la hoja blk y las que la siguen por la derecha hasta stop (sin incluirla; 0 = hasta el final).
* @param blk, stop page.BlockNumber
* @return error
**/
func (r *vacuumRun) chain(blk, stop page.BlockNumber) error {
	for blk != 0 && blk != stop {
		if r.opts.Stop != nil && r.opts.Stop() {
			return ErrVacuumStopped
		}
		next, err := r.leaf(blk)
		if err != nil {
			return err
		}
		blk = next
	}
	return nil
}

/**
* leaf limpia una hoja bloqueada en exclusivo: planea, registra el KindLeafVacuum, pone los bits del mapa, la suelta y
* libera las cadenas de lo que quedó «liberándose».
* @param blk page.BlockNumber
* @return page.BlockNumber: la hermana derecha, error
**/
func (r *vacuumRun) leaf(blk page.BlockNumber) (page.BlockNumber, error) {
	h, n, err := r.t.lockNode(blk, buffer.Exclusive)
	if err != nil {
		return 0, err
	}
	if !n.leaf() {
		h.Release()
		return 0, corruptAt(blk, "vacuum reached a page of level %d", n.o.Level)
	}
	next := n.o.Next
	if n.ignore() { // medio muerta (un corte dejó su borrado a medias) o borrada: se termina, no se limpia
		half := n.halfDead()
		h.Release()
		if half {
			return next, r.t.unlinkHalfDead(blk, r.opts, &r.st)
		}
		return next, nil
	}
	plan, err := r.t.planLeaf(n, r.opts, &r.carry, true)
	if err == nil {
		err = r.t.vacuumLeaf(h, n, plan, r.opts.XID)
	}
	empty := n.dataCount() == 0
	h.Release()
	if err != nil {
		return 0, err
	}
	r.st.Leaves++
	r.st.Removed += len(plan.remove)
	r.st.Frozen += len(plan.freeze)
	r.st.FrozenXID = min(r.st.FrozenXID, plan.oldest)
	if err := r.t.freeChains(blk, plan.chains, r.opts, &r.st); err != nil {
		return 0, err
	}
	if empty || len(plan.chains) > 0 {
		err = r.t.deleteLeaf(blk, r.opts, &r.st)
	}
	return next, err
}

/**
* vacuumLeaf registra y aplica el KindLeafVacuum de la hoja (si hay algo que hacer) y después pone sus bits del mapa,
* todo con la hoja bloqueada en exclusivo: así ninguna escritura se cruza entre la marca de la página y el bit. El
* registro de la hoja va antes que el del mapa (y lo escribe antes de pedir la página del mapa: nunca se espera entre
* Reserve y Write).
* @param h buffer.Handle, n node: la hoja, plan leafPlan, xid page.XID
* @return error
**/
func (t *BTree) vacuumLeaf(h buffer.Handle, n node, plan leafPlan, xid page.XID) error {
	pv := pageVacuum{mark: plan.mark, remove: plan.remove, freeze: plan.freeze, setVisible: plan.setVisible}
	if err := t.logVacuum(h, pv, xid); err != nil {
		return err
	}
	if plan.vmBits == 0 || t.vm == nil {
		return nil
	}
	return t.vm.Set(t.file, n.blk, plan.vmBits)
}

/**
* planLeaf decide, con la hoja bloqueada en exclusivo, qué versiones quitar, cuáles marcar y cuáles congelar, y qué
* bits del mapa le tocan. carry es el último id de la hoja anterior (sus versiones pueden seguir en esta) y al volver
* es el último de esta. Con lookRight, si el último id puede seguir en la hermana derecha se mira ahí
* (rightStartsWith); sin él (la poda de una escritura, que no recorre hermanas) se da por compartido y su lápida se
* queda.
* @param n node, opts VacuumOptions, carry *vacuumGroup, lookRight bool
* @return leafPlan, error
**/
func (t *BTree) planLeaf(n node, opts VacuumOptions, carry *vacuumGroup, lookRight bool) (leafPlan, error) {
	plan := leafPlan{oldest: opts.Horizon}
	var g *vacuumGroup
	for off := n.firstData(); off <= n.maxOff(); off++ {
		e, err := n.entryAt(off)
		if err != nil {
			return leafPlan{}, err
		}
		if g == nil || !bytes.Equal(g.id, e.id()) {
			if g != nil {
				plan.endGroup(g)
			} else if bytes.Equal(carry.id, e.id()) {
				plan.shared = true // el primer id viene de la hoja anterior
			}
			g = startGroup(e.id(), carry)
		}
		if err := t.planEntry(&plan, g, off, e, opts); err != nil {
			return leafPlan{}, err
		}
	}
	if g != nil { // una hoja vacía deja pasar el grupo de la anterior
		shared := continuesRight(n, g.id)
		if shared && lookRight {
			var err error
			if shared, err = t.rightStartsWith(n, g.id); err != nil {
				return leafPlan{}, err
			}
		}
		if shared {
			plan.shared = true
		} else {
			plan.endGroup(g)
		}
		*carry = *g
	}
	slices.Sort(plan.remove)
	plan.finish(n, opts)
	return plan, nil
}

/**
* finish decide, con todo lo que queda en la hoja, qué se congela (confirmada y xid < FreezeLimit), el menor xid sin
* congelar y los bits del mapa: todo-visible si todo lo que queda es visible para todos, todo-congelada si además todo
* queda congelado. Una hoja con partición incompleta, que comparte un id con una vecina o que queda vacía no lleva bits.
* @param n node, opts VacuumOptions
**/
func (plan *leafPlan) finish(n node, opts VacuumOptions) {
	allFrozen, remaining := true, 0
	for _, k := range plan.kept {
		if _, removed := slices.BinarySearch(plan.remove, k.off); removed {
			continue
		}
		remaining++
		committed := k.status == xact.StatusCommitted
		if !k.frozen && !(committed && k.xid < opts.Horizon) {
			plan.busy = true
		}
		switch {
		case k.frozen:
		case committed && k.xid < opts.FreezeLimit:
			plan.freeze = append(plan.freeze, k.off)
		default:
			allFrozen = false
			plan.oldest = min(plan.oldest, k.xid)
		}
	}
	if plan.busy || plan.shared || n.incomplete() || remaining == 0 {
		return // una hoja que queda vacía no lleva bits: se va a borrar (y si no se puede, la próxima pasada la mira)
	}
	plan.vmBits = fsm.VMAllVisible
	if allFrozen {
		plan.vmBits = fsm.VMAllFrozen
	}
	plan.setVisible = n.p.Flags()&page.FlagAllVisible == 0
}

/**
* startGroup empieza el grupo de un id: si es el último de la hoja anterior, hereda su versión de referencia.
* @param id []byte, carry *vacuumGroup
* @return *vacuumGroup
**/
func startGroup(id []byte, carry *vacuumGroup) *vacuumGroup {
	g := &vacuumGroup{id: bytes.Clone(id)}
	if carry.anchored && bytes.Equal(carry.id, id) {
		g.anchored, g.anchorLSN = true, carry.anchorLSN
	}
	return g
}

/**
* planEntry decide una versión: ya «liberándose» (seguir liberando), muerta (detrás de la de referencia, o abortada),
* la de referencia (confirmada y visible para todos) o viva (se queda). Las que quedan van a plan.kept.
* @param plan *leafPlan, g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions
* @return error: el del archivo de estados
**/
func (t *BTree) planEntry(plan *leafPlan, g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions) error {
	if e.freeing() {
		plan.busy = true
		if opts.Free != nil {
			plan.chains = append(plan.chains, e.version())
		}
		return nil
	}
	if g.anchored && e.lsn() < g.anchorLSN {
		plan.dead(g, off, e, opts)
		return nil
	}
	st, err := t.statusOf(e, nil)
	if err != nil {
		return err
	}
	if st == xact.StatusAborted {
		plan.dead(g, off, e, opts)
		return nil
	}
	frozen := e.flags()&EntryFrozen != 0
	if st == xact.StatusCommitted && (frozen || e.xid() < opts.Horizon) {
		g.anchored, g.anchorLSN = true, e.lsn()
		if e.tombstone() {
			g.tomb = off
		}
	}
	plan.kept = append(plan.kept, keptEntry{off: off, xid: e.xid(), status: st, frozen: frozen})
	return nil
}

/**
* dead anota una versión muerta: se quita, o se marca «liberándose» si tiene desborde (y hay con qué liberarlo; si no,
* se queda y la hoja no puede quedar toda visible).
* @param g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions
**/
func (plan *leafPlan) dead(g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions) {
	switch {
	case e.flags()&EntryOverflow == 0:
		plan.remove = append(plan.remove, off)
	case opts.Free != nil:
		plan.busy = true // sigue en la hoja hasta que se libere su cadena
		plan.mark = append(plan.mark, off)
		v := e.version()
		v.Flags |= EntryFreeing
		plan.chains = append(plan.chains, v)
	default:
		plan.busy = true
		plan.oldest = min(plan.oldest, e.xid())
		if g.anchored {
			g.behind = true
		}
	}
}

/**
* endGroup cierra el grupo de un id: su lápida de referencia se quita si no queda nada detrás (sin versiones, una
* lápida es lo mismo que nada).
* @param g *vacuumGroup
**/
func (plan *leafPlan) endGroup(g *vacuumGroup) {
	if g.tomb != 0 && !g.behind {
		plan.remove = append(plan.remove, g.tomb)
	}
}

/**
* continuesRight dice si las versiones de id pueden seguir en la hermana derecha: solo si la clave alta es de ese
* mismo id (el corte de la partición cayó entre sus versiones). Las versiones nuevas de un id van antes que las
* viejas, así que nunca llegan a la derecha después.
* @param n node, id []byte
* @return bool
**/
func continuesRight(n node, id []byte) bool {
	if n.rightmost() {
		return false
	}
	hk, err := n.highKey()
	return err != nil || bytes.Equal(hk.key().id, id)
}

/**
* rightStartsWith dice si a la derecha de la hoja todavía hay versiones de id: si la primera hoja con entradas a su
* derecha empieza con ese id (las versiones de un id van juntas, así que una hoja vacía o que se está borrando en el
* medio no corta la cadena). Bloquea cada hermana en compartido con la hoja bloqueada, de izquierda a derecha y de a
* una (como latest). Cuando la limpieza ya quitó esas versiones, la clave alta sigue nombrando el id pero la hoja ya
* no comparte nada: puede llevar bits y su lápida de referencia se quita. Una versión nueva del id va siempre a la
* izquierda de la clave alta (a esta hoja, que está bloqueada), así que la derecha no vuelve a tener ese id.
* @param n node: la hoja, no la de más a la derecha; id []byte
* @return bool, error
**/
func (t *BTree) rightStartsWith(n node, id []byte) (bool, error) {
	blk := n.o.Next
	for {
		h, rn, err := t.lockNode(blk, buffer.Share)
		if err != nil {
			return false, err
		}
		if rn.o.Level != n.o.Level {
			h.Release()
			return false, corruptAt(rn.blk, "right sibling at level %d, expected %d", rn.o.Level, n.o.Level)
		}
		if rn.dataCount() > 0 {
			e, err := rn.entryAt(rn.firstData())
			h.Release()
			return err == nil && bytes.Equal(e.id(), id), err
		}
		next := rn.o.Next
		h.Release()
		if next == 0 {
			return false, nil
		}
		blk = next
	}
}

/**
* logVacuum registra, aplica y escribe un KindLeafVacuum sobre la hoja bloqueada en exclusivo (si hay algo que
* hacer). La hoja sigue bloqueada al volver.
* @param h buffer.Handle, pv pageVacuum: listas en orden ascendente, xid page.XID
* @return error
**/
func (t *BTree) logVacuum(h buffer.Handle, pv pageVacuum, xid page.XID) error {
	if pv.empty() {
		return nil
	}
	refs := []wal.BlockRef{{Delta: vacuumDelta(pv)}}
	res, err := t.logChange(wal.KindLeafVacuum, xid, 0, refs, []buffer.Handle{h}, nil)
	if err != nil {
		return err
	}
	return res.Write()
}

/**
* freeChains libera las cadenas de las versiones «liberándose» de una hoja ya soltada y después las quita de la hoja.
* @param blk page.BlockNumber: la hoja donde estaban, chains []Version: en orden de clave
* @param opts VacuumOptions, st *VacuumStats
* @return error: el de opts.Free (las versiones quedan marcadas: la próxima pasada sigue) o el del árbol
**/
func (t *BTree) freeChains(blk page.BlockNumber, chains []Version, opts VacuumOptions, st *VacuumStats) error {
	if len(chains) == 0 {
		return nil
	}
	for _, v := range chains {
		if err := opts.Free(v); err != nil {
			return err
		}
	}
	st.Freed += len(chains)
	removed, err := t.removeFreeing(blk, chains, opts.XID)
	st.Removed += removed
	return err
}

/**
* removeFreeing quita de las hojas las versiones «liberándose» ya liberadas, buscándolas por su clave desde blk y
* siguiendo a la derecha si una partición las movió (una hoja bloqueada a la vez). Una que ya no está se salta.
* @param blk page.BlockNumber, vs []Version: en orden de clave, xid page.XID
* @return int: cuántas quitó, error
**/
func (t *BTree) removeFreeing(blk page.BlockNumber, vs []Version, xid page.XID) (int, error) {
	h, n, err := t.lockNode(blk, buffer.Exclusive)
	if err != nil {
		return 0, err
	}
	removed := 0
	var offs []page.OffsetNumber
	for _, v := range vs {
		k := key{id: []byte(v.ID), lsn: v.LSN}
		for {
			past := n.ignore() // una que se está borrando no tiene entradas: se sigue a la derecha
			if !past {
				if past, err = n.pastHighKey(k); err != nil {
					h.Release()
					return removed, err
				}
			}
			if !past {
				break
			}
			next := n.o.Next
			err = t.logVacuum(h, pageVacuum{remove: offs}, xid)
			h.Release()
			if err != nil {
				return removed, err
			}
			removed, offs = removed+len(offs), nil
			if h, n, err = t.lockNode(next, buffer.Exclusive); err != nil {
				return removed, err
			}
		}
		if off, ok := findEntry(n, k); ok {
			if e, err := n.entryAt(off); err == nil && e.freeing() {
				offs = append(offs, off)
			}
		}
	}
	err = t.logVacuum(h, pageVacuum{remove: offs}, xid)
	h.Release()
	return removed + len(offs), err
}

/**
* vacuumDelta arma el delta de KindLeafVacuum: nMark(2) nRemove(2), los números a marcar «liberándose» y los números
* a quitar y, si congela o marca la página, nFreeze(2) pageFlags(2) y los números a congelar; cada lista en orden
* ascendente. Sin congelado el delta es el de la fase 1B.
* @param pv pageVacuum
* @return []byte
**/
func vacuumDelta(pv pageVacuum) []byte {
	b := le.AppendUint16(nil, uint16(len(pv.mark)))
	b = le.AppendUint16(b, uint16(len(pv.remove)))
	b = appendOffsets(b, pv.mark)
	b = appendOffsets(b, pv.remove)
	if len(pv.freeze) == 0 && !pv.setVisible {
		return b
	}
	var flags uint16
	if pv.setVisible {
		flags = vacuumSetVisible
	}
	b = le.AppendUint16(b, uint16(len(pv.freeze)))
	b = le.AppendUint16(b, flags)
	return appendOffsets(b, pv.freeze)
}

/**
* appendOffsets agrega números de ítem de 2 bytes.
* @param b []byte, offs []page.OffsetNumber
* @return []byte
**/
func appendOffsets(b []byte, offs []page.OffsetNumber) []byte {
	for _, off := range offs {
		b = le.AppendUint16(b, uint16(off))
	}
	return b
}

/**
* readOffsets lee n números de ítem de 2 bytes.
* @param b []byte: con al menos 2n bytes, n int
* @return []page.OffsetNumber
**/
func readOffsets(b []byte, n int) []page.OffsetNumber {
	offs := make([]page.OffsetNumber, n)
	for i := range offs {
		offs[i] = page.OffsetNumber(le.Uint16(b[2*i:]))
	}
	return offs
}

/**
* parseVacuum lee el delta de KindLeafVacuum: algo que hacer, largos exactos, cada lista estrictamente ascendente y
* sin números comunes entre listas, y solo marcas de página conocidas.
* @param delta []byte
* @return pageVacuum, error: ErrBadRecord
**/
func parseVacuum(delta []byte) (pageVacuum, error) {
	if len(delta) < vacuumHeaderSize {
		return pageVacuum{}, badRecord(wal.KindLeafVacuum, "vacuum delta too short")
	}
	nm, nr := int(le.Uint16(delta)), int(le.Uint16(delta[2:]))
	base := vacuumHeaderSize + 2*(nm+nr)
	if len(delta) < base {
		return pageVacuum{}, badRecord(wal.KindLeafVacuum, "bad vacuum delta length")
	}
	pv := pageVacuum{mark: readOffsets(delta[vacuumHeaderSize:], nm), remove: readOffsets(delta[vacuumHeaderSize+2*nm:], nr)}
	if rest := delta[base:]; len(rest) > 0 {
		if err := parseFreeze(rest, &pv); err != nil {
			return pageVacuum{}, err
		}
	}
	if pv.empty() || !ascending(pv.mark) || !ascending(pv.remove) || !ascending(pv.freeze) ||
		shares(pv.mark, pv.remove) || shares(pv.freeze, pv.mark) || shares(pv.freeze, pv.remove) {
		return pageVacuum{}, badRecord(wal.KindLeafVacuum, "vacuum offsets empty, out of order or repeated")
	}
	return pv, nil
}

/**
* parseFreeze lee la parte del congelado del delta: nFreeze(2) pageFlags(2) y los números.
* @param b []byte, pv *pageVacuum
* @return error: ErrBadRecord
**/
func parseFreeze(b []byte, pv *pageVacuum) error {
	if len(b) < freezeHeaderSize {
		return badRecord(wal.KindLeafVacuum, "bad vacuum delta length")
	}
	nf, flags := int(le.Uint16(b)), le.Uint16(b[2:])
	if len(b) != freezeHeaderSize+2*nf || flags&^vacuumSetVisible != 0 || (nf == 0 && flags == 0) {
		return badRecord(wal.KindLeafVacuum, "bad vacuum freeze data")
	}
	pv.freeze = readOffsets(b[freezeHeaderSize:], nf)
	pv.setVisible = flags&vacuumSetVisible != 0
	return nil
}

/**
* shares dice si dos listas ascendentes tienen algún número en común.
* @param a, b []page.OffsetNumber
* @return bool
**/
func shares(a, b []page.OffsetNumber) bool {
	return slices.ContainsFunc(a, func(o page.OffsetNumber) bool {
		_, found := slices.BinarySearch(b, o)
		return found
	})
}

/**
* ascending dice si los números son estrictamente ascendentes y ninguno es 0.
* @param offs []page.OffsetNumber
* @return bool
**/
func ascending(offs []page.OffsetNumber) bool {
	for i, o := range offs {
		if o == 0 || (i > 0 && o <= offs[i-1]) {
			return false
		}
	}
	return true
}

/**
* applyLeafVacuum aplica el delta de KindLeafVacuum a la hoja (la misma función al limpiar y al rehacer): valida todos
* los números, marca EntryFreeing en las versiones con desborde de mark, EntryFrozen en las de freeze (nunca en una
* «liberándose»), page.FlagAllVisible si el delta lo pide, quita las de remove y compacta la página (page.Repair). No
* cambia la clave alta ni los enlaces.
* @param p page.Page, delta []byte
* @return error: ErrBadRecord si un número no cae en una entrada de la hoja
**/
func applyLeafVacuum(p page.Page, delta []byte) error {
	pv, err := parseVacuum(delta)
	if err != nil {
		return err
	}
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf == 0 {
		return badRecord(wal.KindLeafVacuum, "not a leaf")
	}
	n := node{p: p, o: o}
	if err := checkVacuumEntries(n, pv); err != nil {
		return err
	}
	if err := orEntries(p, pv.mark, EntryFreeing); err != nil {
		return err
	}
	if err := orEntries(p, pv.freeze, EntryFrozen); err != nil {
		return err
	}
	if pv.setVisible {
		p.SetFlags(p.Flags() | page.FlagAllVisible)
	}
	if err := p.DeleteItems(pv.remove); err != nil {
		return badRecord(wal.KindLeafVacuum, err.Error())
	}
	return p.Repair()
}

/**
* checkVacuumEntries valida que cada número caiga en una entrada de datos de la hoja, que solo se marquen
* «liberándose» versiones con desborde y que no se congele una «liberándose».
* @param n node, pv pageVacuum
* @return error: ErrBadRecord
**/
func checkVacuumEntries(n node, pv pageVacuum) error {
	all := slices.Concat(pv.mark, pv.remove, pv.freeze)
	for _, off := range all {
		if off < n.firstData() || off > n.maxOff() {
			return badRecord(wal.KindLeafVacuum, "offset outside the leaf's entries")
		}
		e, err := n.entryAt(off)
		if err != nil {
			return badRecord(wal.KindLeafVacuum, err.Error())
		}
		if _, marked := slices.BinarySearch(pv.mark, off); marked && e.flags()&EntryOverflow == 0 {
			return badRecord(wal.KindLeafVacuum, "only an overflow version can be freeing")
		}
		if _, frozen := slices.BinarySearch(pv.freeze, off); frozen && e.freeing() {
			return badRecord(wal.KindLeafVacuum, "a freeing version cannot be frozen")
		}
	}
	return nil
}

/**
* orEntries agrega una marca a las entradas de la hoja con un OR atómico.
* @param p page.Page, offs []page.OffsetNumber, flag EntryFlags
* @return error: ErrBadRecord si una entrada no se puede marcar
**/
func orEntries(p page.Page, offs []page.OffsetNumber, flag EntryFlags) error {
	for _, off := range offs {
		b, _ := p.Item(off)
		if !entry(b).orFlags(flag) {
			return badRecord(wal.KindLeafVacuum, "cannot mark an entry")
		}
	}
	return nil
}
