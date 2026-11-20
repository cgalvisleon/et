package store

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Ranuras de retención del WAL (tarea 46, pregunta 48; ver «Replicación» en doc.go): como las replication slots de
// PostgreSQL con max_slot_wal_keep_size. Cada réplica o consumidor lógico tiene una ranura con el lsn que ya
// confirmó; el checkpoint no recicla los segmentos desde el de la ranura más atrasada (wal.Hold) salvo que lo retenido
// pase MAX_SLOT_WAL_KEEP_SIZE, que invalida la ranura. Se guardan en el archivo slots de la raíz.

const (
	// slotsFileName es el archivo de las ranuras; slotsTemp, su versión a medio escribir.
	slotsFileName = "slots"
	slotsTemp     = "slots.tmp"

	// slotsMagic es "ETSL"; slotsHeaderSize, el encabezado; slotEntrySize, lo fijo de cada ranura.
	slotsMagic      = 0x4554534C
	slotsHeaderSize = 16
	slotEntrySize   = 24

	// maxSlots es cuántas ranuras puede tener un motor; maxSlotName, el largo máximo de un nombre.
	maxSlots    = 16
	maxSlotName = 64

	// slotLogical y slotInvalid son las marcas de una ranura en el archivo.
	slotLogical = 1 << 0
	slotInvalid = 1 << 1
)

// SlotInfo describe una ranura de retención (Slots).
type SlotInfo struct {
	Name         string
	Logical      bool   // de un consumidor lógico (Changes); si no, de una réplica física (WalSince + ApplyWal)
	RestartLSN   uint64 // el lsn más viejo que la ranura retiene: el WAL desde su segmento no se recicla
	ConfirmedLSN uint64 // hasta dónde confirmó el consumidor (AdvanceSlot, ChangeReader.Confirm)
	Invalid      bool   // pasó MAX_SLOT_WAL_KEEP_SIZE: ya no retiene nada; hace falta empezar de nuevo
	Retained     int64  // bytes de WAL que retiene (los segmentos desde el de RestartLSN, enteros)
	Lag          uint64 // registros escritos que el consumidor todavía no confirmó
}

// slot es una ranura en memoria: lo guardado más la retención que tiene puesta.
type slot struct {
	name      string
	logical   bool
	invalid   bool
	restart   page.LSN // el primer lsn que necesita
	confirmed page.LSN // el último lsn que confirmó
	seg       uint64   // el segmento retenido (el de restart)
	unhold    func()   // suelta la retención (wal.Hold); nil si no retiene
}

// slotState son las ranuras del motor. mu va antes que el writeMu del WAL (Hold) y es una hoja respecto de los demás
// locks del motor: se toma con closeMu en lectura o con ckptMu.
type slotState struct {
	mu          sync.Mutex
	list        []*slot
	dirty       bool         // hay avances sin guardar (se guardan en el checkpoint y en Close)
	invalidated atomic.Int64 // ranuras invalidadas por el tope desde Open
}

/**
* find busca una ranura por nombre. Con mu.
* @param name string
* @return *slot: nil si no está
**/
func (st *slotState) find(name string) *slot {
	for _, s := range st.list {
		if s.name == name {
			return s
		}
	}
	return nil
}

/**
* hold retiene el segmento del lsn restart y suelta la retención anterior. Un lsn que ya no está en el WAL invalida
* la ranura. Con mu.
* @param log *wal.WAL, s *slot
* @return error: el de leer el WAL
**/
func (s *slot) hold(log *wal.WAL) error {
	if s.invalid {
		return nil
	}
	seg, err := log.SegmentOf(s.restart)
	if errors.Is(err, wal.ErrRemoved) {
		s.release()
		s.invalid = true
		return nil
	}
	if err != nil {
		return err
	}
	if s.unhold != nil && seg == s.seg {
		return nil
	}
	unhold := log.Hold(seg)
	s.release()
	s.seg, s.unhold = seg, unhold
	return nil
}

/**
* release suelta la retención de la ranura, si tiene.
**/
func (s *slot) release() {
	if s.unhold != nil {
		s.unhold()
		s.unhold = nil
	}
}

/**
* info arma la descripción pública de una ranura. Con mu.
* @param e *Engine
* @return SlotInfo
**/
func (s *slot) info(e *Engine) SlotInfo {
	out := SlotInfo{Name: s.name, Logical: s.logical, RestartLSN: uint64(s.restart), ConfirmedLSN: uint64(s.confirmed),
		Invalid: s.invalid}
	if !s.invalid {
		out.Retained = e.segmentsFrom(s.seg)
		if written := e.log.Published(); written > s.confirmed {
			out.Lag = uint64(written - s.confirmed)
		}
	}
	return out
}

/**
* segmentsFrom son los bytes de WAL desde el segmento seg hasta el activo (cada uno ocupa WAL_SEGMENT_SIZE por la
* preasignación).
* @param seg uint64
* @return int64
**/
func (e *Engine) segmentsFrom(seg uint64) int64 {
	active := e.log.ActiveSegment()
	if seg > active {
		return 0
	}
	return int64(active-seg+1) * e.cfg.wal.SegmentSize
}

/**
* validSlotName dice si name sirve de nombre de ranura: UTF-8 válido sin nulos, de 1 a maxSlotName bytes.
* @param name string
* @return bool
**/
func validSlotName(name string) bool {
	return len(name) <= maxSlotName && validName(name)
}

/**
* createSlot crea una ranura que retiene el WAL desde el próximo registro. Una lógica espera antes a que terminen
* las transacciones que estaban en curso (hasta TRANSACTION_TIMEOUT), como PostgreSQL al buscar su punto
* consistente: su ConfirmedLSN queda en el WAL escrito después de eso, y el decodificado no entrega nada anterior.
* @param name string, logical bool
* @return SlotInfo, error: ErrSlotExists, ErrSlotLimit, ErrBusy (lógica: no terminaron), ErrReadOnly (lógica en una
* réplica), ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) createSlot(name string, logical bool) (SlotInfo, error) {
	if !validSlotName(name) {
		return SlotInfo{}, fmt.Errorf(MSG_SLOT_NAME, ErrSlotLimit, name)
	}
	if logical && e.replica != nil {
		return SlotInfo{}, ErrReadOnly
	}
	if err := e.enterSlots(); err != nil {
		return SlotInfo{}, err
	}
	defer e.leave()
	s, err := e.addSlot(name, logical)
	if err != nil || !logical {
		return s, err
	}
	if err := e.waitConsistent(name); err != nil {
		return SlotInfo{}, errors.Join(err, e.dropSlotLocked(name))
	}
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	if found := st.find(name); found != nil {
		found.confirmed = max(found.confirmed, e.log.Published())
		if err := e.saveSlotsLocked(); err != nil {
			return SlotInfo{}, e.writeError(err)
		}
		return found.info(e), nil
	}
	return SlotInfo{}, fmt.Errorf(MSG_SLOT_NAME, ErrSlotNotFound, name)
}

/**
* addSlot agrega y guarda una ranura que retiene el WAL desde el próximo registro.
* @param name string, logical bool
* @return SlotInfo, error: ErrSlotExists, ErrSlotLimit, ErrDiskFull, ErrFailed
**/
func (e *Engine) addSlot(name string, logical bool) (SlotInfo, error) {
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.find(name) != nil {
		return SlotInfo{}, fmt.Errorf(MSG_SLOT_NAME, ErrSlotExists, name)
	}
	if len(st.list) >= maxSlots {
		return SlotInfo{}, fmt.Errorf(MSG_SLOT_NAME, ErrSlotLimit, name)
	}
	written := e.log.Published()
	s := &slot{name: name, logical: logical, restart: written + 1, confirmed: written}
	if err := s.hold(e.log); err != nil {
		return SlotInfo{}, e.fail(err)
	}
	st.list = append(st.list, s)
	if err := e.saveSlotsLocked(); err != nil {
		e.removeSlotLocked(s)
		return SlotInfo{}, e.writeError(err)
	}
	return s.info(e), nil
}

/**
* waitConsistent espera, sin candados, a que terminen las transacciones en curso al crear una ranura lógica (sus
* cambios pueden haber empezado antes de lo que la ranura retiene), hasta TRANSACTION_TIMEOUT.
* @param name string
* @return error: ErrBusy si no terminan
**/
func (e *Engine) waitConsistent(name string) error {
	limit := e.cfg.txTimeout
	if limit <= 0 {
		limit = e.cfg.lockTimeout
	}
	deadline := time.Now().Add(limit)
	for _, xid := range e.txs.Snapshot(0).InProgress {
		if e.txs.Wait(xid, max(time.Until(deadline), time.Millisecond)) != nil {
			return fmt.Errorf(MSG_SLOT_BUSY, ErrBusy, name)
		}
	}
	return nil
}

/**
* dropSlotLocked borra la ranura name (si está) y guarda, tomando mu.
* @param name string
* @return error
**/
func (e *Engine) dropSlotLocked(name string) error {
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	if s := st.find(name); s != nil {
		e.removeSlotLocked(s)
		return e.saveSlotsLocked()
	}
	return nil
}

/**
* removeSlotLocked saca una ranura de la lista y suelta su retención. Con mu.
* @param s *slot
**/
func (e *Engine) removeSlotLocked(s *slot) {
	s.release()
	e.slots.list = slices.DeleteFunc(e.slots.list, func(x *slot) bool { return x == s })
}

/**
* dropSlot borra una ranura: deja de retener el WAL (el próximo checkpoint lo recicla).
* @param name string
* @return error: ErrSlotNotFound, ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) dropSlot(name string) error {
	if err := e.enterSlots(); err != nil {
		return err
	}
	defer e.leave()
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.find(name)
	if s == nil {
		return fmt.Errorf(MSG_SLOT_NAME, ErrSlotNotFound, name)
	}
	e.removeSlotLocked(s)
	if err := e.saveSlotsLocked(); err != nil {
		return e.writeError(err)
	}
	return nil
}

/**
* slotsInfo describe las ranuras, por nombre.
* @return []SlotInfo
**/
func (e *Engine) slotsInfo() []SlotInfo {
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]SlotInfo, 0, len(st.list))
	for _, s := range st.list {
		out = append(out, s.info(e))
	}
	slices.SortFunc(out, func(a, b SlotInfo) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return out
}

/**
* advanceSlot anota que el consumidor de la ranura ya tiene todo hasta confirmed y que necesita el WAL desde restart
* (0 = confirmed + 1, una réplica física). Queda en memoria (mueve la retención enseguida) y se guarda en el próximo
* checkpoint o en Close: tras un corte del motor la ranura puede volver al valor del último checkpoint, y el
* consumidor recibe de nuevo lo que ya tenía (la réplica lo salta por su lsn). Nunca retrocede.
* @param name string, confirmed, restart page.LSN, logical bool: la clase que se espera
* @return error: ErrSlotNotFound (o de la otra clase), ErrWalRemoved (invalidada), ErrClosed, ErrFailed
**/
func (e *Engine) advanceSlot(name string, confirmed, restart page.LSN, logical bool) error {
	if err := e.enterSlots(); err != nil {
		return err
	}
	defer e.leave()
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.find(name)
	switch {
	case s == nil || s.logical != logical:
		return fmt.Errorf(MSG_SLOT_NAME, ErrSlotNotFound, name)
	case s.invalid:
		return fmt.Errorf(MSG_SLOT_INVALID, ErrWalRemoved, name)
	case confirmed > e.log.Published():
		return fmt.Errorf(MSG_SLOT_BACKWARD, ErrWalGap, name, s.confirmed, confirmed, e.log.Published())
	}
	if restart == 0 {
		restart = confirmed + 1
	}
	if confirmed <= s.confirmed && restart <= s.restart {
		return nil
	}
	s.confirmed = max(s.confirmed, confirmed)
	s.restart = max(s.restart, min(restart, s.confirmed+1))
	st.dirty = true
	seg := s.seg
	if err := s.hold(e.log); err != nil {
		return e.fail(err)
	}
	if s.seg != seg {
		e.wakeWriters() // los frenados por lo que retenía la ranura vuelven a mirar
	}
	return nil
}

/**
* slotRestart devuelve la ranura name para leer desde ella: su RestartLSN y su ConfirmedLSN.
* @param name string, logical bool: la clase que se espera
* @return page.LSN, page.LSN, error: ErrSlotNotFound, ErrWalRemoved (invalidada)
**/
func (e *Engine) slotRestart(name string, logical bool) (page.LSN, page.LSN, error) {
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.find(name)
	switch {
	case s == nil || s.logical != logical:
		return 0, 0, fmt.Errorf(MSG_SLOT_NAME, ErrSlotNotFound, name)
	case s.invalid:
		return 0, 0, fmt.Errorf(MSG_SLOT_INVALID, ErrWalRemoved, name)
	}
	return s.restart, s.confirmed, nil
}

/**
* enterSlots deja pasar una operación de ranuras: anda también en una réplica (las ranuras son de la replicación, no
* de los datos), pero no con el motor inutilizable.
* @return error: ErrClosed, ErrFailed
**/
func (e *Engine) enterSlots() error {
	if err := e.enterRead(); err != nil {
		return err
	}
	if err := e.failed(); err != nil {
		e.leave()
		return err
	}
	return nil
}

/**
* checkpointSlots es el paso 6 del checkpoint para las ranuras, antes de reciclar: invalida las que retienen más de
* MAX_SLOT_WAL_KEEP_SIZE (sueltan su retención) y guarda el archivo con los avances, así lo que se recicla después
* sale de lo guardado. Con ckptMu.
* @return error: el de guardar (un disco lleno limpio deja todo retenido como estaba)
**/
func (e *Engine) checkpointSlots() error {
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.list) == 0 && !st.dirty {
		return nil
	}
	if limit := e.cfg.slotKeepSize; limit > 0 {
		for _, s := range st.list {
			if !s.invalid && e.segmentsFrom(s.seg) > limit {
				s.release()
				s.invalid = true
				st.dirty = true
				st.invalidated.Add(1)
			}
		}
	}
	if !st.dirty {
		return nil
	}
	if err := e.saveSlotsLocked(); err != nil {
		return e.writeError(err)
	}
	return nil
}

/**
* slotBacklog es lo que las ranuras suman al freno de los escritores: sin tope (MAX_SLOT_WAL_KEEP_SIZE = 0), lo que
* retiene la más atrasada; con tope, nada (el tope invalida en vez de frenar).
* @return int64: bytes
**/
func (e *Engine) slotBacklog() int64 {
	if e.cfg.slotKeepSize > 0 {
		return 0
	}
	return e.slotRetained()
}

/**
* slotRetained son los bytes de WAL que retiene la ranura válida más atrasada.
* @return int64
**/
func (e *Engine) slotRetained() int64 {
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	var most int64
	for _, s := range st.list {
		if !s.invalid {
			most = max(most, e.segmentsFrom(s.seg))
		}
	}
	return most
}

/**
* loadSlots lee el archivo de ranuras al abrir (con el WAL ya abierto) y pone sus retenciones; una cuyo lsn ya no
* está en el WAL queda invalidada.
* @return error: ErrCorrupt si el archivo está dañado
**/
func (e *Engine) loadSlots() error {
	list, err := e.readSlots()
	if err != nil || len(list) == 0 {
		return err
	}
	st := &e.slots
	st.mu.Lock()
	defer st.mu.Unlock()
	st.list = list
	for _, s := range list {
		if err := s.hold(e.log); err != nil {
			return err
		}
	}
	return nil
}

/**
* encodeSlots arma el archivo de ranuras: magic(4) version(4) count(4) reservado(4); por ranura nameLen(1) flags(1)
* reservado(6) restartLSN(8) confirmedLSN(8) y el nombre; al final crc(4) de todo lo anterior.
* @param list []*slot
* @return []byte
**/
func encodeSlots(list []*slot) []byte {
	b := make([]byte, slotsHeaderSize, 512)
	le.PutUint32(b[0:], slotsMagic)
	le.PutUint32(b[4:], control.FormatVersion)
	le.PutUint32(b[8:], uint32(len(list)))
	for _, s := range list {
		var flags byte
		if s.logical {
			flags |= slotLogical
		}
		if s.invalid {
			flags |= slotInvalid
		}
		entry := make([]byte, slotEntrySize)
		entry[0], entry[1] = byte(len(s.name)), flags
		le.PutUint64(entry[8:], uint64(s.restart))
		le.PutUint64(entry[16:], uint64(s.confirmed))
		b = append(append(b, entry...), s.name...)
	}
	return le.AppendUint32(b, crc32.Checksum(b, crcTable))
}

/**
* decodeSlots lee el archivo de ranuras.
* @param b []byte
* @return []*slot, bool: false si no tiene la forma del formato
**/
func decodeSlots(b []byte) ([]*slot, bool) {
	if len(b) < slotsHeaderSize+4 || le.Uint32(b[0:]) != slotsMagic || le.Uint32(b[4:]) != control.FormatVersion {
		return nil, false
	}
	body := b[:len(b)-4]
	if le.Uint32(b[len(b)-4:]) != crc32.Checksum(body, crcTable) {
		return nil, false
	}
	count := int(le.Uint32(b[8:]))
	if count > maxSlots {
		return nil, false
	}
	list := make([]*slot, 0, count)
	off := slotsHeaderSize
	for range count {
		if off+slotEntrySize > len(body) {
			return nil, false
		}
		n, flags := int(body[off]), body[off+1]
		end := off + slotEntrySize + n
		if end > len(body) || flags&^(slotLogical|slotInvalid) != 0 {
			return nil, false
		}
		name := string(body[off+slotEntrySize : end])
		if !validSlotName(name) {
			return nil, false
		}
		list = append(list, &slot{
			name:      name,
			logical:   flags&slotLogical != 0,
			invalid:   flags&slotInvalid != 0,
			restart:   page.LSN(le.Uint64(body[off+8:])),
			confirmed: page.LSN(le.Uint64(body[off+16:])),
		})
		off = end
	}
	if off != len(body) {
		return nil, false
	}
	return list, true
}

/**
* readSlots lee el archivo de ranuras, si lo hay. Se escribe de forma atómica, así que uno dañado es corrupción.
* @return []*slot, error: ErrCorrupt
**/
func (e *Engine) readSlots() ([]*slot, error) {
	names, err := e.fs.ReadDir(e.dir)
	if err != nil || !slices.Contains(names, slotsFileName) {
		return nil, err
	}
	f, err := e.fs.OpenFile(filepath.Join(e.dir, slotsFileName), os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	size, err := f.Size()
	if err != nil {
		return nil, err
	}
	b := make([]byte, min(size, 1<<16))
	n, _ := f.ReadAt(b, 0)
	list, ok := decodeSlots(b[:n])
	if !ok {
		return nil, fmt.Errorf(MSG_SLOT_FILE, ErrCorrupt)
	}
	return list, nil
}

/**
* saveSlotsLocked guarda las ranuras de forma atómica: slots.tmp, fsync, rename y fsync del directorio. Con mu.
* @return error: un disco lleno al escribir el temporal no cambia nada
**/
func (e *Engine) saveSlotsLocked() error {
	tmp, path := filepath.Join(e.dir, slotsTemp), filepath.Join(e.dir, slotsFileName)
	f, err := e.fs.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(encodeSlots(e.slots.list), 0)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = e.fs.Rename(tmp, path)
	}
	if err == nil {
		err = e.fs.SyncDir(e.dir)
	}
	if err == nil {
		e.slots.dirty = false
	}
	return err
}
