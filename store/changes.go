package store

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/page"
)

// ChangePos es una posición del flujo de cambios (tarea 75 de F-23): desde Restart se vuelve a leer el WAL y se salta
// lo confirmado hasta Confirmed. La da ChangeReader.Pos o Engine.ChangeHead; ChangesFrom sigue desde ahí.
type ChangePos struct {
	Restart   uint64 // el primer registro que hay que leer (el de la transacción más vieja todavía sin entregar)
	Confirmed uint64 // lo que ya se entregó (o era anterior a la posición): las confirmaciones con lsn ≤ Confirmed se saltan
}

// ChangeFilter elige qué cambios salen del flujo; un campo nil deja pasar todo. Store e ID se miran por cambio (ID
// antes de armar un valor en desborde) y Origin por transacción (TxInfo.Origin). Un DDL suelto tiene origen 0.
type ChangeFilter struct {
	Store  func(store string) bool
	ID     func(store, id string) bool
	Origin func(origin uint16) bool
}

// keepTx dice si una transacción de ese origen sale.
func (f ChangeFilter) keepTx(origin uint16) bool {
	return f.Origin == nil || f.Origin(origin)
}

// keepStore dice si los cambios del store salen.
func (f ChangeFilter) keepStore(store string) bool {
	return f.Store == nil || f.Store(store)
}

// keepID dice si el cambio del id en el store sale.
func (f ChangeFilter) keepID(store, id string) bool {
	return f.keepStore(store) && (f.ID == nil || f.ID(store, id))
}

// headState guarda el primer registro posible de cada transacción que escribe (una cota por debajo), para
// ChangeHead: los lotes y las escrituras sueltas con desborde, cuya cadena va antes que la hoja (tarea 103). Crece con
// las transacciones abiertas, no con las claves (regla 1).
type headState struct {
	mu    sync.Mutex
	first map[page.XID]page.LSN
}

// readerSeq numera los lectores sin ranura para nombrar sus archivos de derrame.
var readerSeq atomic.Uint64

/**
* note anota para xid, si no estaba, una cota del lsn de su primer registro: el siguiente al último reservado.
* @param xid page.XID, last func() page.LSN: el último lsn reservado del WAL
**/
func (h *headState) note(xid page.XID, last func() page.LSN) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.first == nil {
		h.first = map[page.XID]page.LSN{}
	}
	if _, ok := h.first[xid]; !ok {
		h.first[xid] = last() + 1
	}
}

/**
* forget saca a xid de headState, después de su registro final.
* @param xid page.XID
**/
func (h *headState) forget(xid page.XID) {
	h.mu.Lock()
	delete(h.first, xid)
	h.mu.Unlock()
}

/**
* noteFirst anota, en la primera escritura del lote, una cota del lsn de su primer registro. Con b.mu tomado.
**/
func (b *Batch) noteFirst() {
	if b.firstNoted {
		return
	}
	b.firstNoted = true
	b.e.heads.note(b.xid, b.e.log.LastLSN)
}

/**
* forgetFirst saca al lote de headState al terminar, después de escribir su registro final.
**/
func (b *Batch) forgetFirst() {
	if b.firstNoted {
		b.e.heads.forget(b.xid)
	}
}

/**
* changeHead da la posición actual del flujo sin esperar a las transacciones en curso: Confirmed es el último lsn
* reservado (lo confirmado antes queda atrás) y Restart el primer registro posible de las transacciones que escriben
* (lotes y escrituras sueltas con desborde).
* @return ChangePos, error: ErrClosed, ErrFailed
**/
func (e *Engine) changeHead() (ChangePos, error) {
	if err := e.enterSlots(); err != nil {
		return ChangePos{}, err
	}
	defer e.leave()
	h := &e.heads
	h.mu.Lock()
	defer h.mu.Unlock()
	last := e.log.LastLSN()
	restart := last + 1
	for _, first := range h.first {
		restart = min(restart, first)
	}
	return ChangePos{Restart: uint64(restart), Confirmed: uint64(last)}, nil
}

/**
* changesFrom abre un lector sin ranura desde pos: no retiene WAL (si el checkpoint ya recicló pos.Restart, Next da
* ErrWalRemoved) y anda también en una réplica.
* @param pos ChangePos, f ChangeFilter
* @return *ChangeReader, error: ErrClosed, ErrFailed
**/
func (e *Engine) changesFrom(pos ChangePos, f ChangeFilter) (*ChangeReader, error) {
	if err := e.enterSlots(); err != nil {
		return nil, err
	}
	defer e.leave()
	restart := page.LSN(max(pos.Restart, 1))
	r := e.newReader("", restart, page.LSN(pos.Confirmed), f)
	r.spillTag = fmt.Sprintf("~%x", readerSeq.Add(1))
	return r, nil
}

/**
* changesWith abre el lector de la ranura lógica name con un filtro.
* @param name string, f ChangeFilter
* @return *ChangeReader, error: los de Changes
**/
func (e *Engine) changesWith(name string, f ChangeFilter) (*ChangeReader, error) {
	r, err := e.changes(name)
	if err != nil {
		return nil, err
	}
	r.filter = f
	return r, nil
}

/**
* pos es la posición tras lo entregado: Restart la del primer registro de la transacción más vieja que sigue en el
* búfer (o el siguiente a lo leído), Confirmed lo leído (nunca menos que la posición de partida).
* @return ChangePos
**/
func (r *ChangeReader) position() ChangePos {
	r.mu.Lock()
	defer r.mu.Unlock()
	restart := r.read + 1
	for _, t := range r.txs {
		restart = min(restart, t.first)
	}
	return ChangePos{Restart: uint64(restart), Confirmed: uint64(max(r.read, r.confirmed))}
}

/**
* advanceLogical avanza la ranura lógica name hasta pos sin su lector (un reparto que guarda la posición aparte).
* @param name string, pos ChangePos
* @return error: los de AdvanceSlot
**/
func (e *Engine) advanceLogical(name string, pos ChangePos) error {
	return e.advanceSlot(name, page.LSN(pos.Confirmed), page.LSN(pos.Restart), true)
}
