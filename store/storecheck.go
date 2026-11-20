package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// errStopWalk corta un recorrido después de anotar en el informe el hallazgo que lo corta.
var errStopWalk = errors.New("store: check walk stopped")

// bitset es un bit por bloque del fork principal: 16 KB por archivo de datos de 1 GB.
type bitset []uint64

/**
* newBitset da un bitset para n bloques, en cero.
* @param n page.BlockNumber
* @return bitset
**/
func newBitset(n page.BlockNumber) bitset {
	return make(bitset, (uint64(n)+63)/64)
}

// has dice si el bit del bloque b está puesto.
func (b bitset) has(blk page.BlockNumber) bool { return b[blk/64]&(1<<(blk%64)) != 0 }

// set pone el bit del bloque b.
func (b bitset) set(blk page.BlockNumber) { b[blk/64] |= 1 << (blk % 64) }

// storeCheck es la revisión de un store: el inventario de sus páginas (de quién es cada una) y el conteo de ids vivos.
// Lo que ocupa en RAM son dos bitsets del tamaño del fork principal (owned y leaves, 32 KB por GB de datos) y una hoja
// de entradas: no crece con las claves.
type storeCheck struct {
	e      *Engine
	s      *Store
	r      *Report
	repair bool

	file    smgr.FileNo
	where   string           // dataFileName(file), para los hallazgos
	n       page.BlockNumber // bloques del fork principal
	owned   bitset           // meta, árbol, cadenas vivas y lista de libres
	leaves  bitset           // las hojas del árbol
	horizon page.XID         // el xid vivo más viejo al empezar: lo visible para todos
	frozen  page.XID         // el frozenXID del store (el de la meta, nunca menos que el del motor al abrir)

	// pagesSafe: ninguna cadena viva, página repetida ni la lista de libres están dañadas, así que las páginas sin
	// dueño de verdad no son de nadie y Repair puede devolverlas a la lista.
	pagesSafe bool

	// El conteo: ids vivos según lo último confirmado, en el orden del árbol (id ascendente, lsn descendente).
	live    int64
	curID   string
	decided bool
}

/**
* newStoreCheck prepara la revisión de un store tomado en escritura.
* @param s *Store, r *Report, repair bool
* @return *storeCheck
**/
func newStoreCheck(s *Store, r *Report, repair bool) *storeCheck {
	return &storeCheck{e: s.e, s: s, r: r, repair: repair, file: s.file, where: dataFileName(s.file), pagesSafe: true}
}

/**
* add anota un hallazgo del store.
* @param kind FindingKind, blk int64: -1 si no es de un bloque, blocks int64, repairable bool, desc string
* @return int: la posición en el informe (para repaired)
**/
func (c *storeCheck) add(kind FindingKind, blk, blocks int64, repairable bool, desc string) int {
	if blk < 0 {
		blocks = 0
	}
	return c.r.add(Finding{Kind: kind, Store: c.s.name, File: c.where, Block: blk, Blocks: blocks,
		Repairable: repairable, Desc: desc})
}

/**
* problem anota un error de lectura como hallazgo irreparable, salvo que haya dejado el motor inutilizable.
* @param err error, kind FindingKind, blk int64, format string: con un %v para err
* @return error: ErrFailed, o nil si quedó en el informe
**/
func (c *storeCheck) problem(err error, kind FindingKind, blk int64, format string) error {
	if ferr := c.e.fatal(err); ferr != nil {
		return ferr
	}
	c.add(kind, blk, 1, false, fmt.Sprintf(format, err))
	return nil
}

/**
* fixed termina un arreglo: si salió bien lo marca hecho; si falló por un daño (una página ilegible) lo deja sin
* hacer; si falló la escritura, el motor queda inutilizable.
* @param i int: la posición del hallazgo, err error
* @return error: ErrFailed
**/
func (c *storeCheck) fixed(i int, err error) error {
	if err == nil {
		c.r.repaired(i)
		return nil
	}
	return c.e.fatal(err)
}

/**
* run revisa el store: el árbol con sus hojas (conteo, congelado, mapa de visibilidad de cada hoja, cadenas de
* desborde), la lista de libres, los bits del mapa fuera de las hojas, las páginas sin dueño y Count. Si el árbol no
* pasa, el resto no es confiable y no se mira (ni se repara).
* @return error: ErrFailed
**/
func (c *storeCheck) run() error {
	n, err := c.e.sm.NBlocks(c.file, smgr.ForkMain)
	if err != nil {
		return c.problem(err, FindTree, -1, MSG_CHECK_BLOCKS)
	}
	c.n, c.owned, c.leaves = n, newBitset(n), newBitset(n)
	c.r.Stores++
	c.r.Pages += int64(n)
	if n > 0 {
		c.owned.set(btree.MetaBlock)
	}
	c.horizon = c.e.txs.OldestActive()
	meta, err := c.s.tree.FrozenXID()
	if err != nil {
		return c.problem(err, FindTree, int64(btree.MetaBlock), MSG_CHECK_META)
	}
	c.frozen = c.e.frozen.atLeast(meta)
	if next := c.e.txs.NextXID(); meta > next {
		c.add(FindFrozen, int64(btree.MetaBlock), 1, false, fmt.Sprintf(MSG_CHECK_FROZEN_NEXT, meta, next))
	}
	before := c.e.counts.get(c.file)
	st, err := c.s.tree.VerifyWith(btree.VerifyOptions{Strategy: buffer.Ring, Page: c.treePage, Leaf: c.leaf})
	if err != nil {
		if errors.Is(err, errStopWalk) {
			return nil
		}
		return c.problem(err, FindTree, -1, MSG_CHECK_TREE)
	}
	c.r.TreePages += int64(st.Pages)
	c.r.Entries += int64(st.Entries)
	for _, step := range []func() error{c.freeList, c.visibilityMap, c.lostPages} {
		if err := step(); err != nil {
			return err
		}
	}
	return c.count(before)
}

/**
* treePage anota una página del árbol; una que ya tenía dueño corta el recorrido.
* @param blk page.BlockNumber
* @return error: errStopWalk
**/
func (c *storeCheck) treePage(blk page.BlockNumber) error {
	if blk >= c.n || c.owned.has(blk) {
		c.add(FindDouble, int64(blk), 1, false, MSG_CHECK_DOUBLE_TREE)
		c.pagesSafe = false
		return errStopWalk
	}
	c.owned.set(blk)
	return nil
}

/**
* leaf revisa una hoja ya soltada: cuenta sus ids vivos, busca versiones sin congelar por debajo del frozenXID,
* recorre sus cadenas de desborde y comprueba su bit del mapa de visibilidad.
* @param info btree.LeafInfo, entries []btree.Version
* @return error: errStopWalk o ErrFailed
**/
func (c *storeCheck) leaf(info btree.LeafInfo, entries []btree.Version) error {
	c.leaves.set(info.Block)
	unfrozen, first := 0, page.XID(0)
	for i := range entries {
		v := &entries[i]
		if err := c.countEntry(info.Block, v); err != nil {
			return err
		}
		if v.Flags&(btree.EntryFrozen|btree.EntryFreeing) == 0 && v.XID != 0 && v.XID < c.frozen {
			if unfrozen++; first == 0 {
				first = v.XID
			}
		}
		if v.Flags&btree.EntryOverflow != 0 && v.Flags&btree.EntryTombstone == 0 {
			if err := c.chain(info.Block, v); err != nil {
				return err
			}
		}
	}
	if unfrozen > 0 {
		c.add(FindFrozen, int64(info.Block), 1, false, fmt.Sprintf(MSG_CHECK_FROZEN, unfrozen, c.frozen, first))
	}
	return c.leafVisibility(info, entries)
}

/**
* statusOf es el estado de la transacción de una versión como lo ve btree (sus marcas primero; «liberándose» cuenta
* como abortada), sin poner marcas. Un error del archivo de estados queda en el informe y corta el recorrido.
* @param blk page.BlockNumber: la hoja, v *btree.Version
* @return xact.Status, error: errStopWalk o ErrFailed
**/
func (c *storeCheck) statusOf(blk page.BlockNumber, v *btree.Version) (xact.Status, error) {
	switch {
	case v.Flags&btree.EntryFreeing != 0:
		return xact.StatusAborted, nil
	case v.Flags&(btree.EntryXidCommitted|btree.EntryFrozen) != 0:
		return xact.StatusCommitted, nil
	case v.Flags&btree.EntryXidAborted != 0:
		return xact.StatusAborted, nil
	case v.XID == 0:
		return xact.StatusCommitted, nil
	}
	st, _, err := c.e.status.Status(v.XID)
	if err != nil {
		if ferr := c.e.fatal(err); ferr != nil {
			return st, ferr
		}
		c.add(FindStatus, int64(blk), 1, false, fmt.Sprintf(MSG_CHECK_STATUS, v.XID, err))
		return st, errStopWalk
	}
	return st, nil
}

/**
* countEntry suma al conteo: por id, la versión más nueva confirmada decide (viva si no es una lápida), como Count.
* @param blk page.BlockNumber, v *btree.Version
* @return error
**/
func (c *storeCheck) countEntry(blk page.BlockNumber, v *btree.Version) error {
	if v.ID != c.curID {
		c.curID, c.decided = v.ID, false
	}
	if c.decided {
		return nil
	}
	st, err := c.statusOf(blk, v)
	if err != nil {
		return err
	}
	if st == xact.StatusCommitted {
		c.decided = true
		if v.Flags&btree.EntryTombstone == 0 {
			c.live++
		}
	}
	return nil
}

/**
* chain recorre la cadena de desborde de una versión y anota sus páginas (de una «liberándose», solo lo que todavía
* es suyo). Una cadena dañada o una página que ya tenía dueño quedan en el informe sin cortar el recorrido del árbol.
* @param blk page.BlockNumber: la hoja, v *btree.Version: con EntryOverflow
* @return error: ErrFailed
**/
func (c *storeCheck) chain(blk page.BlockNumber, v *btree.Version) error {
	partial := v.Flags&btree.EntryFreeing != 0
	err := c.e.chains.Pages(c.file, v.Overflow, v.XID, partial, func(b page.BlockNumber) error {
		if b >= c.n || c.owned.has(b) {
			c.add(FindDouble, int64(b), 1, false, fmt.Sprintf(MSG_CHECK_DOUBLE_CHAIN, b, v.ID))
			c.pagesSafe = false
			return errStopWalk
		}
		c.owned.set(b)
		c.r.ChainPages++
		return nil
	})
	if err == nil || errors.Is(err, errStopWalk) {
		return nil
	}
	if ferr := c.e.fatal(err); ferr != nil {
		return ferr
	}
	c.pagesSafe = false
	c.add(FindChain, int64(blk), 1, false, fmt.Sprintf(MSG_CHECK_CHAIN, v.ID, v.LSN, err))
	return nil
}

/**
* leafVisibility comprueba el bit del mapa de visibilidad de una hoja: un bit puesto exige page.FlagAllVisible y
* toda versión visible para todos (congelada, o confirmada con xid menor que el horizonte), y VMAllFrozen exige todas
* congeladas. Uno mal puesto se borra con repair (borrarlo siempre es seguro).
* @param info btree.LeafInfo, entries []btree.Version
* @return error
**/
func (c *storeCheck) leafVisibility(info btree.LeafInfo, entries []btree.Version) error {
	bits, err := c.e.free.Get(c.file, info.Block)
	if err != nil {
		return c.problem(err, FindVisibility, int64(info.Block), MSG_CHECK_VM_READ)
	}
	if bits == 0 {
		return nil
	}
	why, err := c.notAllVisible(info, entries, bits)
	if err != nil || why == "" {
		return err
	}
	return c.badBits(info.Block, fmt.Sprintf(MSG_CHECK_VM_LEAF, bits, why))
}

/**
* notAllVisible dice por qué una hoja no merece sus bits del mapa.
* @param info btree.LeafInfo, entries []btree.Version, bits fsm.VMBits
* @return string: "" si los merece, error
**/
func (c *storeCheck) notAllVisible(info btree.LeafInfo, entries []btree.Version, bits fsm.VMBits) (string, error) {
	if !info.AllVisible {
		return "marked all-visible", nil
	}
	for i := range entries {
		v := &entries[i]
		frozen := v.Flags&btree.EntryFrozen != 0
		if bits&fsm.VMAllFrozen != 0 && !frozen {
			return "all frozen", nil
		}
		if frozen {
			continue
		}
		st, err := c.statusOf(info.Block, v)
		if err != nil {
			return "", err
		}
		if v.Flags&btree.EntryFreeing != 0 || st != xact.StatusCommitted || v.XID >= c.horizon {
			return "all visible", nil
		}
	}
	return "", nil
}

/**
* badBits anota un bit del mapa mal puesto y, con repair, lo borra (KindVMClear).
* @param blk page.BlockNumber, desc string
* @return error: ErrFailed
**/
func (c *storeCheck) badBits(blk page.BlockNumber, desc string) error {
	i := c.add(FindVisibility, int64(blk), 1, true, desc)
	if !c.repair {
		return nil
	}
	return c.fixed(i, c.e.free.Clear(c.file, blk))
}

/**
* freeList recorre la lista de libres y anota sus páginas. Una lista que apunta a una página que no es libre, que se
* sale del archivo o que repite una página es irreparable (y frena la reparación de las páginas sin dueño).
* @return error
**/
func (c *storeCheck) freeList() error {
	err := c.e.free.WalkFree(c.file, c.n, func(b page.BlockNumber) error {
		if c.owned.has(b) {
			c.add(FindDouble, int64(b), 1, false, MSG_CHECK_DOUBLE_FREE)
			c.pagesSafe = false
			return errStopWalk
		}
		c.owned.set(b)
		c.r.FreePages++
		return nil
	})
	if err == nil || errors.Is(err, errStopWalk) {
		return nil
	}
	c.pagesSafe = false
	return c.problem(err, FindFreeList, -1, MSG_CHECK_FREE_LIST)
}

/**
* visibilityMap busca bits del mapa en bloques que no son hojas (o que no existen) y, con repair, los borra.
* @return error
**/
func (c *storeCheck) visibilityMap() error {
	err := c.e.free.ForEachVM(c.file, func(blk page.BlockNumber, bits fsm.VMBits) error {
		if blk < c.n && c.leaves.has(blk) {
			return nil
		}
		return c.badBits(blk, fmt.Sprintf(MSG_CHECK_VM_NOT_LEAF, bits))
	})
	if err == nil || errors.Is(err, ErrFailed) {
		return err
	}
	return c.problem(err, FindVisibility, -1, MSG_CHECK_VM_READ)
}

// lostRun son bloques seguidos sin dueño de la misma clase.
type lostRun struct {
	first page.BlockNumber
	count page.BlockNumber
	class string
}

// Clases de las páginas sin dueño (las de desborde son cadenas huérfanas).
const (
	lostUnreadable = "unreadable"
	lostZeroed     = "zeroed (never written)"
	lostEmpty      = "empty (taken from the free list and never used)"
	lostFree       = "free but not in the free list"
	lostOverflow   = "overflow"
	lostTree       = "unreachable tree page"
	lostOther      = "unknown content"
)

/**
* lostPages agrupa las páginas sin dueño en tramos seguidos de la misma clase y anota cada tramo; con repair (si
* nada lo hace dudoso, pagesSafe) las devuelve a la lista de libres.
* @return error
**/
func (c *storeCheck) lostPages() error {
	var run lostRun
	for blk := page.BlockNumber(1); blk < c.n; blk++ {
		if c.owned.has(blk) {
			continue
		}
		class, err := c.classify(blk)
		if err != nil {
			return err
		}
		if run.count > 0 && run.class == class && run.first+run.count == blk {
			run.count++
			continue
		}
		if err := c.lostRun(run); err != nil {
			return err
		}
		run = lostRun{first: blk, count: 1, class: class}
	}
	return c.lostRun(run)
}

/**
* classify dice qué tiene una página sin dueño.
* @param blk page.BlockNumber
* @return string: la clase, error: ErrFailed
**/
func (c *storeCheck) classify(blk page.BlockNumber) (string, error) {
	h, err := c.e.pool.Pin(buffer.Tag{File: c.file, Fork: smgr.ForkMain, Block: blk}, buffer.Ring)
	if err != nil {
		if ferr := c.e.fatal(err); ferr != nil {
			return "", ferr
		}
		return lostUnreadable, nil
	}
	defer h.Release()
	h.Lock(buffer.Share)
	defer h.Unlock()
	p := h.Page()
	if p.IsNew() {
		return lostZeroed, nil
	}
	o, err := p.Opaque()
	switch {
	case err != nil:
		return lostOther, nil
	case o.Flags&page.OpaqueOverflow != 0:
		return lostOverflow, nil
	case o.Flags&page.OpaqueFree != 0:
		return lostFree, nil
	case o.Flags == 0 && o.Next == 0 && o.Level == 0:
		return lostEmpty, nil
	case o.Flags&page.OpaqueMeta == 0:
		return lostTree, nil
	}
	return lostOther, nil
}

/**
* lostRun anota un tramo de páginas sin dueño y, con repair, las devuelve a la lista (fsm Reclaim, KindFreePush).
* @param run lostRun
* @return error
**/
func (c *storeCheck) lostRun(run lostRun) error {
	if run.count == 0 {
		return nil
	}
	c.r.LostPages += int64(run.count)
	kind, desc := FindLostPage, fmt.Sprintf(MSG_CHECK_LOST, run.class)
	if run.class == lostOverflow {
		kind, desc = FindOrphanChain, MSG_CHECK_ORPHAN_CHAIN
	}
	i := c.add(kind, int64(run.first), int64(run.count), c.pagesSafe, desc)
	if !c.repair || !c.pagesSafe {
		return nil
	}
	for blk := run.first; blk < run.first+run.count; blk++ {
		if err := c.e.free.Reclaim(c.file, blk, 0); err != nil {
			return c.e.fatal(err)
		}
	}
	c.r.repaired(i)
	return nil
}

/**
* count compara el conteo del recorrido con Count. El recorrido corrió sin frenar las confirmaciones de lotes (que no
* toman el store), así que si no coincide con el contador de antes ni con el de después se repite exacto: con
* commitMu en escritura (ninguna confirmación en curso en el motor) y, con repair, se corrige ahí mismo.
* @param before int64: el contador al empezar el recorrido
* @return error
**/
func (c *storeCheck) count(before int64) error {
	defer func() { c.r.LiveIDs += c.live }()
	if c.live == before || c.live == c.e.counts.get(c.file) {
		return nil
	}
	c.e.commitMu.Lock()
	defer c.e.commitMu.Unlock()
	if err := c.recount(); err != nil {
		if errors.Is(err, errStopWalk) {
			return nil
		}
		return c.problem(err, FindTree, -1, MSG_CHECK_TREE)
	}
	counter := c.e.counts.get(c.file)
	if c.live == counter {
		return nil
	}
	i := c.add(FindCount, -1, 0, true, fmt.Sprintf(MSG_CHECK_COUNT, counter, c.live))
	if !c.repair {
		return nil
	}
	if err := c.e.fixCount(c.file, c.live-counter); err != nil {
		return c.e.fail(err)
	}
	c.r.repaired(i)
	return nil
}

/**
* recount vuelve a contar los ids vivos recorriendo solo las hojas.
* @return error
**/
func (c *storeCheck) recount() error {
	c.live, c.curID, c.decided = 0, "", false
	_, err := c.s.tree.VerifyWith(btree.VerifyOptions{Strategy: buffer.Ring,
		Leaf: func(info btree.LeafInfo, entries []btree.Version) error {
			for i := range entries {
				if err := c.countEntry(info.Block, &entries[i]); err != nil {
					return err
				}
			}
			return nil
		}})
	return err
}

/**
* fixCount corrige el contador de Count de un store con una confirmación propia: un KindCommit de un xid nuevo, sin
* versiones, que suma delta al store (bajo writeMu, como un lote) y que la recuperación rehace igual que cualquier
* confirmación. Espera su fsync. Con commitMu en escritura.
* @param file smgr.FileNo, delta int64
* @return error
**/
func (e *Engine) fixCount(file smgr.FileNo, delta int64) error {
	xid := e.txs.Begin()
	defer e.txs.End(xid)
	rec := &wal.Record{XID: xid, Kind: wal.KindCommit,
		Main: encodeCommit(time.Now().UnixNano(), commitInfo{deltas: map[smgr.FileNo]int64{file: delta}})}
	res, err := e.log.Reserve(rec, func(page.LSN) { e.counts.add(file, delta) })
	if err != nil {
		return err
	}
	lsn := res.LSN()
	if err := res.Write(); err != nil {
		return err
	}
	e.noteCommit(lsn)
	if err := e.log.WaitDurable(lsn); err != nil {
		return err
	}
	return e.status.SetStatus(xid, xact.StatusCommitted, lsn)
}
