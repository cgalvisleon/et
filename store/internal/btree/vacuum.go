package btree

import (
	"bytes"
	"slices"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// vacuumHeaderSize es el encabezado del delta de KindLeafVacuum: nMark(2) nRemove(2).
const vacuumHeaderSize = 2 + 2

// leafPlan es lo que la limpieza hace en una hoja: marcar «liberándose» las versiones muertas con desborde, quitar
// las demás muertas, y las versiones cuyas cadenas hay que liberar (las recién marcadas y las que ya lo estaban).
type leafPlan struct {
	mark   []page.OffsetNumber
	remove []page.OffsetNumber
	chains []Version
}

// vacuumGroup son las versiones de un id que la limpieza está mirando, de la más nueva a la más vieja.
type vacuumGroup struct {
	id        []byte            // copia
	anchored  bool              // ya apareció la versión de referencia: la más nueva confirmada visible para todos
	anchorLSN page.LSN          // su lsn: toda versión del id más vieja está muerta
	tomb      page.OffsetNumber // la versión de referencia, si es una lápida de esta hoja (0 = no)
	behind    bool              // detrás de la versión de referencia queda una versión que no se quita
}

/**
* Vacuum limpia el store (como VACUUM, fase 1): recorre las hojas de izquierda a derecha por sus enlaces, una
* bloqueada a la vez, y quita las versiones que ningún snapshot puede ver: las de transacciones abortadas, las que
* reemplazó una versión confirmada visible para todos (xid < Horizon) y la lápida que queda sin versiones detrás. Las
* que tienen desborde se marcan «liberándose» en el mismo registro (KindLeafVacuum), se libera su cadena con
* opts.Free y recién entonces se quitan (pregunta 58): un corte a mitad deja la versión marcada y la próxima pasada
* sigue desde donde quedó. El espacio se reutiliza dentro de la hoja (las hojas vacías quedan: borrarlas es de la fase
* 2). Puede correr con lecturas y escrituras; quien llama no corre dos limpiezas del mismo store a la vez.
* @param opts VacuumOptions
* @return VacuumStats, error
**/
func (t *BTree) Vacuum(opts VacuumOptions) (VacuumStats, error) {
	var st VacuumStats
	h, n, _, err := t.descend(minusInf, 0, buffer.Exclusive, false)
	if err != nil {
		return st, err
	}
	var carry vacuumGroup
	for {
		blk, next, last, level := n.blk, n.o.Next, n.rightmost(), n.o.Level
		plan, err := t.planLeaf(n, opts, &carry)
		if err != nil {
			h.Release()
			return st, err
		}
		if err := t.vacuumPage(h, plan.mark, plan.remove, opts.XID); err != nil {
			return st, err
		}
		st.Leaves++
		st.Removed += len(plan.remove)
		if err := t.freeChains(blk, plan.chains, opts, &st); err != nil {
			return st, err
		}
		if last {
			return st, nil
		}
		if h, n, err = t.lockNode(next, buffer.Exclusive); err != nil {
			return st, err
		}
		if n.o.Level != level {
			h.Release()
			return st, corruptAt(next, "right sibling at level %d, expected %d", n.o.Level, level)
		}
	}
}

/**
* planLeaf decide, con la hoja bloqueada en exclusivo, qué versiones quitar y cuáles marcar. carry es el último id de
* la hoja anterior (sus versiones pueden seguir en esta) y al volver es el último de esta.
* @param n node, opts VacuumOptions, carry *vacuumGroup
* @return leafPlan, error
**/
func (t *BTree) planLeaf(n node, opts VacuumOptions, carry *vacuumGroup) (leafPlan, error) {
	var plan leafPlan
	var g *vacuumGroup
	for off := n.firstData(); off <= n.maxOff(); off++ {
		e, err := n.entryAt(off)
		if err != nil {
			return leafPlan{}, err
		}
		if g == nil || !bytes.Equal(g.id, e.id()) {
			if g != nil {
				plan.endGroup(g)
			}
			g = startGroup(e.id(), carry)
		}
		if err := t.planEntry(&plan, g, off, e, opts); err != nil {
			return leafPlan{}, err
		}
	}
	if g != nil {
		if !continuesRight(n, g.id) {
			plan.endGroup(g)
		}
		*carry = *g
	}
	slices.Sort(plan.remove)
	return plan, nil
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
* la de referencia (confirmada y visible para todos) o viva (se queda).
* @param plan *leafPlan, g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions
* @return error: el del archivo de estados
**/
func (t *BTree) planEntry(plan *leafPlan, g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions) error {
	if e.freeing() {
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
	switch {
	case err != nil:
		return err
	case st == xact.StatusAborted:
		plan.dead(g, off, e, opts)
	case st == xact.StatusCommitted && (e.flags()&EntryFrozen != 0 || e.xid() < opts.Horizon):
		g.anchored, g.anchorLSN = true, e.lsn()
		if e.tombstone() {
			g.tomb = off
		}
	}
	return nil
}

/**
* dead anota una versión muerta: se quita, o se marca «liberándose» si tiene desborde (y hay con qué liberarlo; si no,
* se queda).
* @param g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions
**/
func (plan *leafPlan) dead(g *vacuumGroup, off page.OffsetNumber, e entry, opts VacuumOptions) {
	switch {
	case e.flags()&EntryOverflow == 0:
		plan.remove = append(plan.remove, off)
	case opts.Free != nil:
		plan.mark = append(plan.mark, off)
		v := e.version()
		v.Flags |= EntryFreeing
		plan.chains = append(plan.chains, v)
	case g.anchored:
		g.behind = true
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
* vacuumPage registra y aplica un KindLeafVacuum sobre la hoja bloqueada en exclusivo (si hay algo que hacer), la
* suelta y escribe el registro.
* @param h buffer.Handle, mark, remove []page.OffsetNumber: en orden ascendente, xid page.XID
* @return error
**/
func (t *BTree) vacuumPage(h buffer.Handle, mark, remove []page.OffsetNumber, xid page.XID) error {
	if len(mark) == 0 && len(remove) == 0 {
		h.Release()
		return nil
	}
	refs := []wal.BlockRef{{Delta: vacuumDelta(mark, remove)}}
	res, err := t.logChange(wal.KindLeafVacuum, xid, 0, refs, []buffer.Handle{h}, nil)
	h.Release()
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
			past, err := n.pastHighKey(k)
			if err != nil {
				h.Release()
				return removed, err
			}
			if !past {
				break
			}
			next := n.o.Next
			if err := t.vacuumPage(h, nil, offs, xid); err != nil {
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
	err = t.vacuumPage(h, nil, offs, xid)
	return removed + len(offs), err
}

/**
* vacuumDelta arma el delta de KindLeafVacuum: nMark(2) nRemove(2), los números a marcar «liberándose» y los números
* a quitar, cada lista en orden ascendente.
* @param mark, remove []page.OffsetNumber
* @return []byte
**/
func vacuumDelta(mark, remove []page.OffsetNumber) []byte {
	b := le.AppendUint16(nil, uint16(len(mark)))
	b = le.AppendUint16(b, uint16(len(remove)))
	for _, off := range append(slices.Clone(mark), remove...) {
		b = le.AppendUint16(b, uint16(off))
	}
	return b
}

/**
* parseVacuum lee el delta de KindLeafVacuum: algo que hacer, largos exactos, cada lista estrictamente ascendente y
* sin números en las dos.
* @param delta []byte
* @return mark, remove []page.OffsetNumber, error: ErrBadRecord
**/
func parseVacuum(delta []byte) ([]page.OffsetNumber, []page.OffsetNumber, error) {
	if len(delta) < vacuumHeaderSize {
		return nil, nil, badRecord(wal.KindLeafVacuum, "vacuum delta too short")
	}
	nm, nr := int(le.Uint16(delta)), int(le.Uint16(delta[2:]))
	if nm+nr == 0 || len(delta) != vacuumHeaderSize+2*(nm+nr) {
		return nil, nil, badRecord(wal.KindLeafVacuum, "bad vacuum delta length")
	}
	offs := make([]page.OffsetNumber, nm+nr)
	for i := range offs {
		offs[i] = page.OffsetNumber(le.Uint16(delta[vacuumHeaderSize+2*i:]))
	}
	mark, remove := offs[:nm], offs[nm:]
	if !ascending(mark) || !ascending(remove) || slices.ContainsFunc(mark, func(o page.OffsetNumber) bool {
		_, found := slices.BinarySearch(remove, o)
		return found
	}) {
		return nil, nil, badRecord(wal.KindLeafVacuum, "vacuum offsets out of order or repeated")
	}
	return mark, remove, nil
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
* los números, marca EntryFreeing en las versiones con desborde de mark, quita las de remove y compacta la página
* (page.Repair). No cambia la clave alta ni los enlaces.
* @param p page.Page, delta []byte
* @return error: ErrBadRecord si un número no cae en una entrada de la hoja
**/
func applyLeafVacuum(p page.Page, delta []byte) error {
	mark, remove, err := parseVacuum(delta)
	if err != nil {
		return err
	}
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf == 0 {
		return badRecord(wal.KindLeafVacuum, "not a leaf")
	}
	n := node{p: p, o: o}
	for _, off := range append(slices.Clone(mark), remove...) {
		if off < n.firstData() || off > n.maxOff() {
			return badRecord(wal.KindLeafVacuum, "offset outside the leaf's entries")
		}
		e, err := n.entryAt(off)
		if err != nil {
			return badRecord(wal.KindLeafVacuum, err.Error())
		}
		if slices.Contains(mark, off) && (e.flags()&EntryOverflow == 0) {
			return badRecord(wal.KindLeafVacuum, "only an overflow version can be freeing")
		}
	}
	for _, off := range mark {
		b, _ := p.Item(off)
		if !entry(b).orFlags(EntryFreeing) {
			return badRecord(wal.KindLeafVacuum, "cannot mark an entry")
		}
	}
	if err := p.DeleteItems(remove); err != nil {
		return badRecord(wal.KindLeafVacuum, err.Error())
	}
	return p.Repair()
}
