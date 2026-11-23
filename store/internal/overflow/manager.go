package overflow

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Manager escribe, lee y libera las cadenas de desborde de todos los stores de un motor. No guarda nada en RAM: todo
// vive en las páginas de la caché. Es seguro usarlo desde varias goroutines.
type Manager struct {
	pool buffer.Pool
	free fsm.FreeList
	log  wal.Log

	// afterFree, si no es nil, corre en FreeTail tras liberar cada página con cuántas lleva; si devuelve error,
	// FreeTail para ahí (solo para las pruebas: simula un corte a mitad de la liberación).
	afterFree func(freed int) error
}

// Comprobación en compilación de que Manager cumple la interfaz.
var _ Chains = (*Manager)(nil)

/**
* New crea el gestor sobre la caché, la lista de libres y el WAL del motor.
* @param pool buffer.Pool, free fsm.FreeList, log wal.Log
* @return *Manager
**/
func New(pool buffer.Pool, free fsm.FreeList, log wal.Log) *Manager {
	return &Manager{pool: pool, free: free, log: log}
}

/**
* Write guarda data en una cadena nueva de file. Escribe de la última página a la primera, así cada una ya conoce a
* su siguiente y nunca hay dos páginas bloqueadas a la vez: por cada trozo pide una página a fsm, la inicia con un
* KindOverflowWrite (BlockWillInit) y la suelta. Si falla a medias, devuelve a la lista lo ya escrito. Lee data solo
* durante la llamada.
* @param file smgr.FileNo, xid page.XID: el store y la transacción dueña de la versión
* @param data []byte: de 1 a MaxValueLen bytes
* @return Ref, error: ErrBadLen, o el error de fsm, de la caché o del WAL
**/
func (m *Manager) Write(file smgr.FileNo, xid page.XID, data []byte) (Ref, error) {
	if len(data) == 0 || len(data) > MaxValueLen {
		return Ref{}, fmt.Errorf(MSG_BAD_LEN, ErrBadLen, len(data), MaxValueLen)
	}
	total := uint32(len(data))
	next := page.BlockNumber(0)
	for seq := pagesOf(total) - 1; seq >= 0; seq-- {
		start := seq * ChunkSize
		block, err := m.writePage(file, xid, next, seq, data[start:start+chunkLen(seq, total)])
		if err != nil {
			if next != 0 {
				err = errors.Join(err, m.freeFrom(file, next, seq+1, total, xid))
			}
			return Ref{}, err
		}
		next = block
	}
	return Ref{First: next, Len: total}, nil
}

/**
* writePage pide una página a fsm y la deja como la página seq de la cadena, siguiendo el orden de un cambio
* (store/doc.go): con la página bloqueada reserva el registro, aplica, pone pd_lsn y la marca sucia; la suelta y
* escribe la reserva. Con BlockWillInit nunca hace falta imagen de página entera, así que Reserve no pide ninguna.
* @param file smgr.FileNo, xid page.XID, next page.BlockNumber, seq int, chunk []byte
* @return page.BlockNumber: la página escrita, error
**/
func (m *Manager) writePage(file smgr.FileNo, xid page.XID, next page.BlockNumber, seq int, chunk []byte) (page.BlockNumber, error) {
	h, err := m.free.Allocate(file, xid)
	if err != nil {
		return 0, err
	}
	block := h.Tag().Block
	rec := &wal.Record{
		XID:  xid,
		Kind: wal.KindOverflowWrite,
		Blocks: []wal.BlockRef{
			{File: file, Fork: smgr.ForkMain, Block: block, Flags: wal.BlockWillInit, Delta: chunk},
		},
		Main: encodeMain(next, seq),
	}
	res, err := m.log.Reserve(rec, nil)
	if err != nil {
		release(h)
		return 0, err
	}
	p := h.Page()
	applyErr := apply(p, xid, next, uint32(seq), chunk)
	if applyErr == nil {
		p.SetLSN(res.LSN())
		h.MarkDirty(res.LSN())
	}
	release(h)
	// Toda reserva se escribe, también si aplicar falló (no debería: el trozo ya se validó).
	if err := errors.Join(applyErr, res.Write()); err != nil {
		return 0, err
	}
	return block, nil
}

/**
* Read devuelve una copia del valor de la cadena ref, validando cada página: que sea de desborde, su posición en la
* cadena, el largo de su trozo y su siguiente. Así un corte, un bucle o un largo que no coincide devuelven
* ErrBadChain sin leer más páginas de las que tiene el valor. Fija una página a la vez, con bloqueo compartido.
* @param file smgr.FileNo, ref Ref
* @return []byte: del llamador, error: ErrBadRef, ErrBadChain o el error de la caché
**/
func (m *Manager) Read(file smgr.FileNo, ref Ref) ([]byte, error) {
	if err := checkRef(file, ref); err != nil {
		return nil, err
	}
	out := make([]byte, ref.Len)
	if err := m.walk(file, ref.First, 0, ref.Len, out, nil); err != nil {
		return nil, err
	}
	return out, nil
}

/**
* Free devuelve todas las páginas de la cadena a la lista de libres de fsm. Primero la recorre entera validándola:
* si está dañada no libera nada (una página ajena nunca vuelve a la lista). Después libera de la primera a la última,
* leyendo la siguiente de cada una antes de liberarla.
* @param file smgr.FileNo, ref Ref, xid page.XID: la transacción que limpia
* @return error: ErrBadRef, ErrBadChain o el error de fsm, de la caché o del WAL
**/
func (m *Manager) Free(file smgr.FileNo, ref Ref, xid page.XID) error {
	if err := checkRef(file, ref); err != nil {
		return err
	}
	if err := m.walk(file, ref.First, 0, ref.Len, nil, nil); err != nil {
		return err
	}
	return m.freeFrom(file, ref.First, 0, ref.Len, xid)
}

/**
* FreeTail libera lo que queda de la cadena ref de una versión de owner, de la última página a la primera (pregunta
* 58). Primero junta las páginas que siguen siendo de la versión —de desborde, en su posición, con owner en
* pd_prune_xid y con su trozo— desde la primera, y después las libera desde el final. Un corte a mitad deja liberada
* una cola y entero el resto: llamarla otra vez libera solo lo que falta, y si no queda nada no hace nada. Una página
* ya liberada (o reutilizada por otra versión, que tiene otro xid) no es de owner, así que nunca se libera dos veces.
* @param file smgr.FileNo, ref Ref
* @param owner page.XID: el xid de la versión (distinto de 0), xid page.XID: el de la limpieza
* @return int: páginas liberadas, error: ErrBadRef, ErrBadChain (una página de la versión dañada) o el error de fsm,
* de la caché o del WAL
**/
func (m *Manager) FreeTail(file smgr.FileNo, ref Ref, owner, xid page.XID) (int, error) {
	if err := checkRef(file, ref); err != nil {
		return 0, err
	}
	if owner == 0 {
		return 0, fmt.Errorf(MSG_BAD_REF, ErrBadRef, file, ref.First, ref.Len)
	}
	blocks, err := m.ownedPrefix(file, ref, owner)
	if err != nil {
		return 0, err
	}
	freed := 0
	for i := len(blocks) - 1; i >= 0; i-- {
		if err := m.free.Free(file, blocks[i], xid); err != nil {
			return freed, err
		}
		freed++
		if m.afterFree != nil {
			if err := m.afterFree(freed); err != nil {
				return freed, err
			}
		}
	}
	return freed, nil
}

/**
* ownedPrefix junta, desde la primera, las páginas de la cadena ref que siguen siendo de owner: la cadena entera, o el
* prefijo que dejó una liberación desde el final cortada a mitad.
* @param file smgr.FileNo, ref Ref, owner page.XID
* @return []page.BlockNumber: en orden de la cadena, error
**/
func (m *Manager) ownedPrefix(file smgr.FileNo, ref Ref, owner page.XID) ([]page.BlockNumber, error) {
	var blocks []page.BlockNumber
	block := ref.First
	for seq := 0; seq < pagesOf(ref.Len); seq++ {
		next, owned, err := m.ownPage(file, block, seq, ref.Len, owner, buffer.Normal)
		if err != nil || !owned {
			return blocks, err
		}
		blocks = append(blocks, block)
		block = next
	}
	return blocks, nil
}

/**
* ownPage dice si la página block es todavía la página seq de la cadena de owner (un valor de total bytes) y, si lo
* es, la valida como readPage.
* @param file smgr.FileNo, block page.BlockNumber, seq int, total uint32, owner page.XID
* @param strategy buffer.Strategy: buffer.Ring en los recorridos de Verify
* @return page.BlockNumber: la siguiente, bool: si es de owner, error: ErrBadChain si es suya y está dañada
**/
func (m *Manager) ownPage(file smgr.FileNo, block page.BlockNumber, seq int, total uint32, owner page.XID, strategy buffer.Strategy) (page.BlockNumber, bool, error) {
	h, err := m.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: block}, strategy)
	if err != nil {
		return 0, false, err
	}
	h.Lock(buffer.Share)
	defer release(h)

	p := h.Page()
	if p.IsNew() {
		return 0, false, nil
	}
	o, err := p.Opaque()
	if err != nil || o.Flags != page.OpaqueOverflow || o.Level != uint32(seq) || p.PruneXID() != owner {
		return 0, false, nil
	}
	last := seq == pagesOf(total)-1
	switch {
	case p.Lower() != page.HeaderSize+chunkLen(seq, total):
		return 0, false, badChain(file, block, seq, "chunk length does not match the value length")
	case last && o.Next != 0:
		return 0, false, badChain(file, block, seq, "chain is longer than the value")
	case !last && (o.Next == 0 || o.Next == block):
		return 0, false, badChain(file, block, seq, "chain ends before the value")
	}
	return o.Next, true, nil
}

/**
* freeFrom libera la cadena (o su cola) que empieza en first, la página seq de un valor de total bytes.
* @param file smgr.FileNo, first page.BlockNumber, seq int, total uint32, xid page.XID
* @return error
**/
func (m *Manager) freeFrom(file smgr.FileNo, first page.BlockNumber, seq int, total uint32, xid page.XID) error {
	return m.walk(file, first, seq, total, nil, func(block page.BlockNumber) error {
		return m.free.Free(file, block, xid)
	})
}

/**
* walk recorre la cadena desde block, que es la página from de un valor de total bytes, hasta la última. Por cada
* página la valida (readPage), copia su trozo en dst si no es nil y, ya soltada la página, llama a visit si no es
* nil. Nunca visita más páginas de las que tiene el valor.
* @param file smgr.FileNo, block page.BlockNumber, from int, total uint32
* @param dst []byte: el valor entero (total bytes) o nil
* @param visit func(block page.BlockNumber) error: o nil
* @return error
**/
func (m *Manager) walk(file smgr.FileNo, block page.BlockNumber, from int, total uint32, dst []byte, visit func(page.BlockNumber) error) error {
	for seq := from; seq < pagesOf(total); seq++ {
		var chunk []byte
		if dst != nil {
			chunk = dst[seq*ChunkSize : seq*ChunkSize+chunkLen(seq, total)]
		}
		next, err := m.readPage(file, block, seq, total, chunk)
		if err != nil {
			return err
		}
		if visit != nil {
			if err := visit(block); err != nil {
				return err
			}
		}
		block = next
	}
	return nil
}

/**
* readPage fija la página block con bloqueo compartido, comprueba que sea la página seq de la cadena de un valor de
* total bytes y copia su trozo en dst (si no es nil).
* @param file smgr.FileNo, block page.BlockNumber, seq int, total uint32, dst []byte
* @return page.BlockNumber: la siguiente (0 en la última), error: ErrBadChain o el error de la caché
**/
func (m *Manager) readPage(file smgr.FileNo, block page.BlockNumber, seq int, total uint32, dst []byte) (page.BlockNumber, error) {
	if block == 0 {
		return 0, badChain(file, block, seq, "chain ends before the value")
	}
	h, err := m.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: block}, buffer.Normal)
	if err != nil {
		return 0, err
	}
	h.Lock(buffer.Share)
	defer release(h)

	p := h.Page()
	o, err := p.Opaque()
	switch {
	case p.IsNew() || err != nil || o.Flags != page.OpaqueOverflow || p.LayoutVersion() != page.LayoutVersion:
		return 0, badChain(file, block, seq, "not an overflow page")
	case o.Level != uint32(seq):
		return 0, badChain(file, block, seq, fmt.Sprintf("page says it is number %d", o.Level))
	case p.Lower() != page.HeaderSize+chunkLen(seq, total):
		return 0, badChain(file, block, seq, "chunk length does not match the value length")
	case seq == pagesOf(total)-1 && o.Next != 0:
		return 0, badChain(file, block, seq, "chain is longer than the value")
	}
	if dst != nil {
		copy(dst, p[page.HeaderSize:p.Lower()])
	}
	return o.Next, nil
}

/**
* checkRef valida una referencia antes de seguirla.
* @param file smgr.FileNo, ref Ref
* @return error: ErrBadRef
**/
func checkRef(file smgr.FileNo, ref Ref) error {
	if ref.First == 0 || ref.Len == 0 || ref.Len > MaxValueLen {
		return fmt.Errorf(MSG_BAD_REF, ErrBadRef, file, ref.First, ref.Len)
	}
	return nil
}

// badChain arma el error de una cadena dañada con el lugar y el motivo.
func badChain(file smgr.FileNo, block page.BlockNumber, seq int, why string) error {
	return fmt.Errorf(MSG_BAD_CHAIN, ErrBadChain, file, block, seq, why)
}

// release suelta el bloqueo y el pin de h.
func release(h buffer.Handle) {
	h.Unlock()
	h.Release()
}
