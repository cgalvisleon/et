package store

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Flujo lógico de cambios (tarea 46, pregunta 31; ver «Replicación» en doc.go), como el decodificado lógico de
// PostgreSQL: un ChangeReader lee el WAL desde la ranura lógica y entrega, por transacción confirmada y en orden de
// confirmación, los cambios por id (store, id, valor nuevo o borrado) y los DDL. Las transacciones abortadas o que
// nunca confirmaron no salen. Un búfer de reordenamiento junta los cambios de cada xid hasta su confirmación; pasa a
// disco (replslot/) cuando lo que tiene en memoria pasa de logicalWorkMem, empezando por la transacción más grande.

const (
	// logicalWorkMem es la memoria del búfer de reordenamiento de un lector (como logical_decoding_work_mem de
	// PostgreSQL, constante): con más, la transacción más grande pasa a disco.
	logicalWorkMem = 16 << 20

	// spillDirName es el directorio de los archivos de derrame, dentro del motor.
	spillDirName = "replslot"

	// itemHeader es lo fijo de un cambio guardado: kind(1) file(4) a(4) b(4) idLen(2) dataLen(4).
	itemHeader = 19
)

// ChangeOp es la clase de un cambio lógico.
type ChangeOp string

const (
	ChangePut    ChangeOp = "put"    // Put, Insert, Update (sueltos o en un lote): Data es el valor nuevo
	ChangeDelete ChangeOp = "delete" // Remove o Delete: Data nil
	ChangeCreate ChangeOp = "create" // un store nuevo (Store, StoreIn)
	ChangeDrop   ChangeOp = "drop"   // un store borrado (DropStore, DropStoreIn)
	ChangeEmpty  ChangeOp = "empty"  // un store vaciado (Empty)
)

// Change es un cambio de una transacción confirmada.
type Change struct {
	Op    ChangeOp
	Store string // el nombre del store ("" si ya no se conoce: un store borrado antes de abrir el lector)
	ID    string // en put y delete
	Data  []byte // en put; del llamador
}

// TxInfo identifica la transacción de un cambio: su xid (0 en un DDL suelto) y el lsn de su confirmación, que crece
// de una transacción a la siguiente (sirve para descartar repetidos tras un corte del motor).
type TxInfo struct {
	XID       uint64
	CommitLSN uint64
}

// clases de los cambios guardados en el búfer.
const (
	itemChunk  byte = 1 // un trozo de desborde: a = su bloque, b = el siguiente; data = el trozo
	itemWrite  byte = 2 // una versión: a = 1 si es lápida, b = el primer bloque si va en desborde; data = el valor o su largo
	itemCreate byte = 3 // un store creado en el lote: id = el nombre
)

// txBuffer son los cambios de una transacción que todavía no confirmó: primero los que pasaron a disco (file), después
// los de memoria, en el orden del WAL.
type txBuffer struct {
	first    page.LSN // su primer registro: hasta ahí retiene la ranura
	mem      [][]byte
	memBytes int
	file     vfs.File // nil hasta el primer derrame
	path     string
	size     int64

	// DDL en lote: los stores que crea (KindCreateStoreIn) y los que creó y deshizo antes de confirmar (un
	// KindDropStore de xid 0 de dropOwnCreate o de un disco lleno en createIn): al entregar, sus cambios se saltan.
	// Crecen con los stores del lote, no con las claves.
	creates  []smgr.FileNo
	canceled map[smgr.FileNo]bool
}

// ChangeReader lee el flujo lógico de una ranura (Changes). No se usa desde varias goroutines a la vez.
type ChangeReader struct {
	e    *Engine
	slot string

	mu        sync.Mutex
	pos       wal.Position // el próximo registro a leer
	set       bool         // pos ya se ubicó en el WAL
	confirmed page.LSN     // lo confirmado por la ranura al abrir: lo de antes no se entrega
	read      page.LSN     // el último registro procesado
	txs       map[page.XID]*txBuffer
	memBytes  int
	workMem   int
	names     map[smgr.FileNo]string   // número de archivo → nombre: crece con los stores, no con las claves
	creating  map[smgr.FileNo]page.XID // store creado por un lote que todavía no confirmó → su xid
	ghosts    map[smgr.FileNo]bool     // stores de lotes abortados: su KindDropStore de deshacer no se entrega
	dir       string                   // el directorio de derrame
	err       error                    // un error a mitad de una transacción deja el lector inservible

	spilled  int64 // bytes pasados a disco desde que se abrió (pruebas, Stats)
	peakMem  int   // lo más que tuvo en memoria (pruebas)
	nextFile int   // para nombrar los archivos de derrame
}

/**
* changes abre un lector del flujo lógico de la ranura lógica name, desde lo que retiene (RestartLSN): entrega lo
* confirmado después de ConfirmedLSN. Borra los archivos de derrame que haya dejado otro lector de la ranura.
* @param name string
* @return *ChangeReader, error: ErrSlotNotFound, ErrWalRemoved (invalidada), ErrClosed, ErrFailed
**/
func (e *Engine) changes(name string) (*ChangeReader, error) {
	if err := e.enterSlots(); err != nil {
		return nil, err
	}
	defer e.leave()
	restart, confirmed, err := e.slotRestart(name, true)
	if err != nil {
		return nil, err
	}
	r := &ChangeReader{
		e: e, slot: name, pos: wal.Position{LSN: restart}, confirmed: confirmed, read: restart - 1,
		txs: map[page.XID]*txBuffer{}, workMem: e.cfg.decodeWorkMem, names: map[smgr.FileNo]string{},
		creating: map[smgr.FileNo]page.XID{}, ghosts: map[smgr.FileNo]bool{}, dir: filepath.Join(e.dir, spillDirName),
	}
	if r.workMem <= 0 {
		r.workMem = logicalWorkMem
	}
	for _, entry := range e.cat.Catalog().Entries {
		r.names[entry.File] = entry.Name
	}
	if err := r.removeSpills(); err != nil {
		return nil, err
	}
	return r, nil
}

/**
* next procesa el WAL durable desde donde iba y llama a fn con cada cambio de cada transacción que confirma, en orden
* de confirmación, hasta entregar maxTx transacciones (≤ 0 = hasta alcanzar el WAL). fn corre sin candados del motor
* salvo closeMu en lectura: puede leer, no debe cerrar el motor.
* @param maxTx int, fn func(tx TxInfo, c Change) error
* @return int: transacciones entregadas, error: el de fn (el lector queda inservible), ErrWalRemoved, ErrClosed
**/
func (r *ChangeReader) next(maxTx int, fn func(tx TxInfo, c Change) error) (int, error) {
	e := r.e
	if err := e.enterRead(); err != nil {
		return 0, err
	}
	defer e.leave()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return 0, r.err
	}
	stop := e.log.Durable()
	if r.pos.LSN > stop {
		return 0, nil
	}
	if !r.set {
		pos, err := e.log.Locate(r.pos.LSN)
		if err != nil {
			return 0, removedError(r.pos.LSN, err)
		}
		r.pos, r.set = pos, true
	}
	sent := 0
	for r.pos.LSN <= stop && (maxTx <= 0 || sent < maxTx) {
		before := r.pos
		_, err := e.log.Read(r.pos, stop, defaultWalBatch, func(at wal.Position, raw []byte, rec *wal.Record) error {
			if maxTx > 0 && sent >= maxTx {
				return errStopRead
			}
			n, err := r.process(rec, fn)
			if err != nil {
				return err
			}
			sent += n
			r.pos = wal.Position{Segment: at.Segment, Offset: at.Offset + int64(len(raw)), LSN: at.LSN + 1}
			r.read = at.LSN
			return nil
		})
		switch {
		case errors.Is(err, errStopRead):
			return sent, nil
		case err != nil:
			r.err = removedError(r.pos.LSN, err)
			return sent, r.err
		case r.pos == before:
			r.err = fmt.Errorf(MSG_WAL_REMOVED, ErrWalRemoved, r.pos.LSN, wal.ErrRemoved)
			return sent, r.err
		}
	}
	return sent, nil
}

// errStopRead corta un recorrido del lector al llegar a maxTx.
var errStopRead = errors.New("store: enough transactions")

/**
* process pasa un registro por el búfer de reordenamiento: guarda los cambios de su transacción, o la entrega si es su
* confirmación, o la descarta si abortó.
* @param rec *wal.Record, fn func(tx TxInfo, c Change) error
* @return int: transacciones entregadas (0 o 1), error
**/
func (r *ChangeReader) process(rec *wal.Record, fn func(tx TxInfo, c Change) error) (int, error) {
	switch rec.Kind {
	case wal.KindOverflowWrite:
		block, next, _, chunk, err := overflow.DecodeWrite(rec)
		if err != nil || rec.XID == 0 {
			return 0, err
		}
		return 0, r.add(rec, encodeItem(itemChunk, rec.Blocks[0].File, uint32(block), uint32(next), "", chunk))
	case wal.KindLeafWrite:
		return r.leafWrite(rec, fn)
	case wal.KindCreateStoreIn:
		file, name, ok := decodeStoreMain(rec.Main)
		if !ok {
			return 0, fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad store data")
		}
		r.names[file] = name
		if err := r.add(rec, encodeItem(itemCreate, file, 0, 0, name, nil)); err != nil {
			return 0, err
		}
		t := r.tx(rec.XID, rec.LSN)
		t.creates = append(t.creates, file)
		r.creating[file] = rec.XID
		return 0, nil
	case wal.KindCommit:
		info, err := decodeCommit(rec.Main)
		if err != nil {
			return 0, fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, err.Error())
		}
		return r.commit(rec.XID, rec.LSN, info.drops, fn)
	case wal.KindAbort:
		r.discard(rec.XID)
	case wal.KindCreateStore, wal.KindDropStore, wal.KindEmptyStore:
		return r.storeDDL(rec, fn)
	case wal.KindCheckpoint:
		r.prune(rec)
	}
	return 0, nil
}

/**
* leafWrite guarda una versión nueva en el búfer de su transacción; una escritura suelta (FlagCommit) se entrega
* enseguida.
* @param rec *wal.Record, fn func(tx TxInfo, c Change) error
* @return int, error
**/
func (r *ChangeReader) leafWrite(rec *wal.Record, fn func(tx TxInfo, c Change) error) (int, error) {
	v, err := btree.DecodeLeafWrite(rec)
	if err != nil || rec.XID == 0 {
		return 0, err
	}
	var tomb, first uint32
	data := v.Value
	switch {
	case v.Flags&btree.EntryTombstone != 0:
		tomb, data = 1, nil
	case v.Flags&btree.EntryOverflow != 0:
		first, data = uint32(v.Overflow.First), le.AppendUint32(nil, v.Overflow.Len)
	}
	if err := r.add(rec, encodeItem(itemWrite, rec.Blocks[0].File, tomb, first, v.ID, data)); err != nil {
		return 0, err
	}
	if rec.Flags&wal.FlagCommit == 0 {
		return 0, nil
	}
	return r.commit(rec.XID, rec.LSN, nil, fn)
}

/**
* storeDDL entrega un DDL suelto (crear, borrar o vaciar un store) como una transacción propia, en su lugar del WAL, y
* pone al día los nombres.
* @param rec *wal.Record, fn func(tx TxInfo, c Change) error
* @return int, error
**/
func (r *ChangeReader) storeDDL(rec *wal.Record, fn func(tx TxInfo, c Change) error) (int, error) {
	var c Change
	switch rec.Kind {
	case wal.KindEmptyStore:
		old, file, name, ok := decodeEmptyMain(rec.Main)
		if !ok {
			return 0, fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad empty store data")
		}
		delete(r.names, old)
		r.names[file] = name
		c = Change{Op: ChangeEmpty, Store: name}
	default:
		file, name, ok := decodeStoreMain(rec.Main)
		if !ok {
			return 0, fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad store data")
		}
		if rec.Kind == wal.KindDropStore && r.undoesBatchCreate(file) {
			return 0, nil
		}
		c = Change{Op: ChangeCreate, Store: name}
		r.names[file] = name
		if rec.Kind == wal.KindDropStore {
			c.Op = ChangeDrop
			delete(r.names, file)
		}
	}
	if rec.LSN <= r.confirmed {
		return 0, nil
	}
	if err := r.guard(fn(TxInfo{CommitLSN: uint64(rec.LSN)}, c)); err != nil {
		return 0, err
	}
	return 1, nil
}

/**
* undoesBatchCreate dice si el KindDropStore (de xid 0) de file deshace un store creado dentro de un lote: uno que el
* lote todavía abierto creó y borró (dropOwnCreate, o un disco lleno en createIn: sus cambios se cancelan dentro del
* lote) o uno de un lote que abortó (undoDDL). Ese borrado no es un DDL suelto: el consumidor nunca vio el store.
* @param file smgr.FileNo
* @return bool
**/
func (r *ChangeReader) undoesBatchCreate(file smgr.FileNo) bool {
	if xid, ok := r.creating[file]; ok && r.txs[xid] != nil {
		t := r.txs[xid]
		if t.canceled == nil {
			t.canceled = map[smgr.FileNo]bool{}
		}
		t.canceled[file] = true
		delete(r.creating, file)
		delete(r.names, file)
		return true
	}
	if r.ghosts[file] {
		delete(r.ghosts, file)
		delete(r.names, file)
		return true
	}
	return false
}

/**
* tx devuelve el búfer de la transacción xid, o lo crea con su primer registro en lsn.
* @param xid page.XID, lsn page.LSN
* @return *txBuffer
**/
func (r *ChangeReader) tx(xid page.XID, lsn page.LSN) *txBuffer {
	t := r.txs[xid]
	if t == nil {
		t = &txBuffer{first: lsn}
		r.txs[xid] = t
	}
	return t
}

/**
* add guarda un cambio en el búfer de la transacción del registro; si la memoria pasa de workMem, la transacción
* más grande pasa a disco.
* @param rec *wal.Record, item []byte
* @return error
**/
func (r *ChangeReader) add(rec *wal.Record, item []byte) error {
	t := r.tx(rec.XID, rec.LSN)
	t.mem = append(t.mem, item)
	t.memBytes += len(item)
	r.memBytes += len(item)
	r.peakMem = max(r.peakMem, r.memBytes)
	for r.memBytes > r.workMem {
		if err := r.spill(r.largest()); err != nil {
			return err
		}
	}
	return nil
}

/**
* largest es la transacción con más cambios en memoria.
* @return *txBuffer
**/
func (r *ChangeReader) largest() *txBuffer {
	var big *txBuffer
	for _, t := range r.txs {
		if big == nil || t.memBytes > big.memBytes {
			big = t
		}
	}
	return big
}

/**
* spill pasa a disco los cambios en memoria de t, al final de su archivo de derrame (lo crea la primera vez): cada uno
* como largo(4) + el cambio.
* @param t *txBuffer
* @return error
**/
func (r *ChangeReader) spill(t *txBuffer) error {
	if t.file == nil {
		if err := r.e.fs.MkdirAll(r.dir); err != nil {
			return fmt.Errorf(MSG_SPILL, err)
		}
		r.nextFile++
		t.path = filepath.Join(r.dir, fmt.Sprintf("%x-%d.spill", r.slot, r.nextFile))
		f, err := r.e.fs.OpenFile(t.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
		if err != nil {
			return fmt.Errorf(MSG_SPILL, err)
		}
		t.file = f
	}
	buf := make([]byte, 0, t.memBytes+4*len(t.mem))
	for _, item := range t.mem {
		buf = append(le.AppendUint32(buf, uint32(len(item))), item...)
	}
	if _, err := t.file.WriteAt(buf, t.size); err != nil {
		return fmt.Errorf(MSG_SPILL, err)
	}
	t.size += int64(len(buf))
	r.spilled += int64(len(buf))
	r.memBytes -= t.memBytes
	t.mem, t.memBytes = nil, 0
	return nil
}

/**
* commit entrega la transacción xid confirmada en lsn (si es posterior a lo confirmado por la ranura y tiene
* cambios) y la saca del búfer.
* @param xid page.XID, lsn page.LSN, drops []smgr.FileNo: los stores que borra al confirmar, fn
* @return int: 1 si la entregó, error
**/
func (r *ChangeReader) commit(xid page.XID, lsn page.LSN, drops []smgr.FileNo, fn func(tx TxInfo, c Change) error) (int, error) {
	t := r.txs[xid]
	defer r.discard(xid)
	if t != nil {
		for _, file := range t.creates {
			delete(r.creating, file) // confirmados: ya no son de un lote abierto
		}
	}
	if lsn <= r.confirmed || t == nil && len(drops) == 0 {
		return 0, nil
	}
	info := TxInfo{XID: uint64(xid), CommitLSN: uint64(lsn)}
	sent := false
	emit := func(c Change) error {
		sent = true
		return r.guard(fn(info, c))
	}
	if t != nil {
		if err := r.replay(t, emit); err != nil {
			return 0, err
		}
	}
	for _, file := range drops {
		if err := emit(Change{Op: ChangeDrop, Store: r.names[file]}); err != nil {
			return 0, err
		}
		delete(r.names, file)
	}
	if !sent {
		return 0, nil
	}
	return 1, nil
}

/**
* replay recorre los cambios de t en orden (los de disco y después los de memoria) y entrega cada versión, armando el
* valor de las de desborde con sus trozos.
* @param t *txBuffer, emit func(Change) error
* @return error
**/
func (r *ChangeReader) replay(t *txBuffer, emit func(Change) error) error {
	chunks := map[uint32]chunkRef{} // los trozos de la cadena en curso: un valor a la vez
	visit := func(item []byte) error {
		kind, file, a, b, id, data, ok := decodeItem(item)
		if !ok {
			return fmt.Errorf(MSG_SPILL, ErrCorrupt)
		}
		if t.canceled[file] {
			return nil // un store que el lote creó y deshizo antes de confirmar
		}
		switch kind {
		case itemChunk:
			chunks[a] = chunkRef{next: b, data: data}
			return nil
		case itemCreate:
			return emit(Change{Op: ChangeCreate, Store: id})
		}
		c := Change{Op: ChangePut, Store: r.names[smgr.FileNo(file)], ID: id, Data: data}
		switch {
		case a == 1:
			c.Op, c.Data = ChangeDelete, nil
		case b != 0:
			value, err := assemble(chunks, b, le.Uint32(data))
			if err != nil {
				return err
			}
			c.Data = value
			clear(chunks)
		}
		return emit(c)
	}
	if t.file != nil {
		rd := bufio.NewReaderSize(io.NewSectionReader(t.file, 0, t.size), 1<<16)
		for {
			var n [4]byte
			if _, err := io.ReadFull(rd, n[:]); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf(MSG_SPILL, err)
			}
			item := make([]byte, le.Uint32(n[:]))
			if _, err := io.ReadFull(rd, item); err != nil {
				return fmt.Errorf(MSG_SPILL, err)
			}
			if err := visit(item); err != nil {
				return err
			}
		}
	}
	for _, item := range t.mem {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

// chunkRef es un trozo de una cadena de desborde guardado en el búfer.
type chunkRef struct {
	next uint32
	data []byte
}

/**
* assemble arma un valor en desborde siguiendo su cadena desde first con los trozos de la transacción.
* @param chunks map[uint32]chunkRef, first uint32, size uint32: el largo del valor
* @return []byte, error: ErrCorrupt si falta un trozo
**/
func assemble(chunks map[uint32]chunkRef, first, size uint32) ([]byte, error) {
	value := make([]byte, 0, size)
	for block := first; uint32(len(value)) < size; {
		c, ok := chunks[block]
		if !ok || block == 0 {
			return nil, fmt.Errorf(MSG_SPILL, ErrCorrupt)
		}
		value = append(value, c.data...)
		block = c.next
	}
	if uint32(len(value)) != size {
		return nil, fmt.Errorf(MSG_SPILL, ErrCorrupt)
	}
	return value, nil
}

/**
* guard deja el lector inservible si fn falló (la transacción quedó entregada a medias).
* @param err error
* @return error
**/
func (r *ChangeReader) guard(err error) error {
	if err != nil {
		r.err = err
	}
	return err
}

/**
* discard saca una transacción del búfer y borra su archivo de derrame. Los stores que creó y no confirmó pasan a
* ghosts (el registro que los deshace no se entrega).
* @param xid page.XID
**/
func (r *ChangeReader) discard(xid page.XID) {
	t := r.txs[xid]
	if t == nil {
		return
	}
	delete(r.txs, xid)
	for _, file := range t.creates {
		if r.creating[file] == xid { // el lote no confirmó: su KindDropStore de deshacer no se entrega
			delete(r.creating, file)
			r.ghosts[file] = true
		}
	}
	r.memBytes -= t.memBytes
	if t.file != nil {
		t.file.Close()
		r.e.fs.Remove(t.path)
	}
}

/**
* prune descarta, en cada KindCheckpoint, las transacciones con xid menor que el más viejo en curso de ese
* checkpoint que nunca confirmaron ni abortaron: murieron con un corte del primario (o no escribieron nada).
* @param rec *wal.Record
**/
func (r *ChangeReader) prune(rec *wal.Record) {
	info, ok := decodeCheckpoint(rec.Main)
	if !ok {
		return
	}
	bound := info.oldest
	if bound == 0 {
		bound = info.nextXID
	}
	for xid := range r.txs {
		if xid < bound {
			r.discard(xid)
		}
	}
}

/**
* confirm anota en la ranura que el consumidor ya procesó todo lo entregado: confirmado hasta el último registro
* leído y retenido desde el primer registro de la transacción más vieja todavía en el búfer.
* @return error: los de AdvanceSlot
**/
func (r *ChangeReader) confirm() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	restart := r.read + 1
	for _, t := range r.txs {
		restart = min(restart, t.first)
	}
	return r.e.advanceSlot(r.slot, r.read, restart, true)
}

/**
* close suelta el búfer y borra los archivos de derrame. Cerrar dos veces no hace nada.
* @return error
**/
func (r *ChangeReader) close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for xid := range r.txs {
		r.discard(xid)
	}
	if r.err == nil {
		r.err = ErrTxDone
	}
	return nil
}

/**
* removeSpills borra los archivos de derrame que dejó un lector anterior de la misma ranura (un corte, o un lector
* sin cerrar).
* @return error
**/
func (r *ChangeReader) removeSpills() error {
	names, err := r.e.fs.ReadDir(r.dir)
	if err != nil {
		return nil // no hay directorio: nada que borrar
	}
	prefix := fmt.Sprintf("%x-", r.slot)
	for _, name := range names {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".spill") {
			if err := r.e.fs.Remove(filepath.Join(r.dir, name)); err != nil {
				return fmt.Errorf(MSG_SPILL, err)
			}
		}
	}
	return nil
}

/**
* encodeItem arma un cambio del búfer: kind(1) file(4) a(4) b(4) idLen(2) dataLen(4) id data.
* @param kind byte, file smgr.FileNo, a, b uint32, id string, data []byte
* @return []byte
**/
func encodeItem(kind byte, file smgr.FileNo, a, b uint32, id string, data []byte) []byte {
	out := make([]byte, itemHeader, itemHeader+len(id)+len(data))
	out[0] = kind
	le.PutUint32(out[1:], uint32(file))
	le.PutUint32(out[5:], a)
	le.PutUint32(out[9:], b)
	le.PutUint16(out[13:], uint16(len(id)))
	le.PutUint32(out[15:], uint32(len(data)))
	return append(append(out, id...), data...)
}

/**
* decodeItem lee un cambio del búfer.
* @param item []byte
* @return byte, smgr.FileNo, uint32, uint32, string, []byte, bool: false si no tiene la forma
**/
func decodeItem(item []byte) (byte, smgr.FileNo, uint32, uint32, string, []byte, bool) {
	if len(item) < itemHeader {
		return 0, 0, 0, 0, "", nil, false
	}
	idLen, dataLen := int(le.Uint16(item[13:])), int(le.Uint32(item[15:]))
	if len(item) != itemHeader+idLen+dataLen {
		return 0, 0, 0, 0, "", nil, false
	}
	id := string(item[itemHeader : itemHeader+idLen])
	return item[0], smgr.FileNo(le.Uint32(item[1:])), le.Uint32(item[5:]), le.Uint32(item[9:]), id,
		item[itemHeader+idLen:], true
}
