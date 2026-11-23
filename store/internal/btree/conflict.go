package btree

import (
	"slices"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// VacuumConflict es lo que un KindLeafVacuum cambia de lo que ve un snapshot de una réplica (store/replica.go), leído
// de la hoja antes de rehacerlo, como el latestRemovedXid que PostgreSQL calcula en el standby: Affects dice si un
// snapshot abierto puede notar el cambio. Los snapshots de una réplica no tienen transacciones en curso (InProgress
// vacío, Own 0): un xid confirmado lo ven si es menor que su Xmax, así que de cada grupo alcanza el menor xid.
type VacuumConflict struct {
	removed []removedVersion // las versiones que se quitan o se marcan «liberándose» y que un snapshot puede ver
	frozen  page.XID         // el mayor xid que se congela (0 = ninguno): un snapshot que no lo veía lo vería
}

// removedVersion es una versión que se quita y que algún snapshot puede ver.
type removedVersion struct {
	xid   page.XID // 0 = la ven todos (congelada o sin xid)
	newer page.XID // el menor xid de las versiones confirmadas más nuevas del mismo id en la hoja; 0 = ninguna
}

/**
* Affects dice si snap puede notar el cambio: ve una versión que se quita y ninguna más nueva del mismo id en la hoja
* (una más nueva en la hoja de la izquierda no se mira: se supone que no la ve), o no veía una que se congela.
* @param snap *xact.Snapshot
* @return bool
**/
func (c VacuumConflict) Affects(snap *xact.Snapshot) bool {
	if c.frozen != 0 && !snap.Visible(c.frozen, xact.StatusCommitted) {
		return true
	}
	for _, rv := range c.removed {
		sees := rv.xid == 0 || snap.Visible(rv.xid, xact.StatusCommitted)
		if sees && (rv.newer == 0 || !snap.Visible(rv.newer, xact.StatusCommitted)) {
			return true
		}
	}
	return false
}

/**
* LeafVacuumConflict lee de la hoja p, antes de rehacer el delta de un KindLeafVacuum, qué versiones quita o marca
* «liberándose» (las abortadas o sin confirmar no cuentan: nadie las ve) con el menor xid confirmado más nuevo de su
* id, y el mayor xid que congela. No cambia la página.
* @param p page.Page: la hoja bloqueada (al menos en compartido), delta []byte
* @param status func(page.XID) (xact.Status, error): el archivo de estados
* @return VacuumConflict, error: un delta o una hoja que no se pueden leer (quien llama entra en conflicto con todos)
**/
func LeafVacuumConflict(p page.Page, delta []byte, status func(page.XID) (xact.Status, error)) (VacuumConflict, error) {
	pv, err := parseVacuum(delta)
	if err != nil {
		return VacuumConflict{}, err
	}
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf == 0 {
		return VacuumConflict{}, corruptAt(0, "leaf vacuum conflict: not a leaf")
	}
	n, err := pageNode(p, o)
	if err != nil {
		return VacuumConflict{}, err
	}
	gone := slices.Concat(pv.mark, pv.remove)
	slices.Sort(gone)
	var c VacuumConflict
	var id []byte
	var newer page.XID // el menor xid confirmado más nuevo del grupo (0 = ninguno)
	var shadowed bool  // hay una más nueva que ven todos: nadie ve las siguientes
	for off := n.firstData(); off <= n.maxOff(); off++ {
		e, err := n.entryAt(off)
		if err != nil {
			return VacuumConflict{}, err
		}
		if id == nil || !e.idEqual(id) {
			id, newer, shadowed = e.appendID(id[:0]), 0, false
		}
		seen, all, err := entrySeen(e, status)
		if err != nil {
			return VacuumConflict{}, err
		}
		if _, removed := slices.BinarySearch(gone, off); removed && seen && !shadowed {
			rv := removedVersion{newer: newer}
			if !all {
				rv.xid = e.xid()
			}
			c.removed = append(c.removed, rv)
		}
		if _, frozen := slices.BinarySearch(pv.freeze, off); frozen {
			c.frozen = max(c.frozen, e.xid())
		}
		switch {
		case !seen:
		case all:
			shadowed = true
		case newer == 0 || e.xid() < newer:
			newer = e.xid()
		}
	}
	return c, nil
}

/**
* entrySeen dice si algún snapshot puede ver la versión (confirmada y no «liberándose») y si la ven todos (congelada o
* sin xid), con sus marcas primero y después el archivo de estados (como statusOf, sin anotar marcas).
* @param e entry, status func(page.XID) (xact.Status, error)
* @return bool: la ve alguno, bool: la ven todos, error
**/
func entrySeen(e entry, status func(page.XID) (xact.Status, error)) (bool, bool, error) {
	flags := e.flags()
	switch {
	case flags&EntryFreeing != 0, flags&EntryXidAborted != 0:
		return false, false, nil
	case flags&EntryFrozen != 0, e.xid() == 0:
		return true, true, nil
	case flags&EntryXidCommitted != 0:
		return true, false, nil
	}
	st, err := status(e.xid())
	return st == xact.StatusCommitted, false, err
}
