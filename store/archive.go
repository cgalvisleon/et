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

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Archivo continuo del WAL (fase 3, tarea 66 de F-23; ver «Archivo continuo del WAL y recuperación a un punto en el
// tiempo» en doc.go): como archive_command de PostgreSQL o SetArchived de josefina. Con WAL_ARCHIVE=on ningún
// segmento sellado se recicla hasta que el archivador diga que ya lo copió (SetArchived); lo archivado se guarda en
// archive_status, así sobrevive a Close/Open y a un corte. Un respaldo (Backup) más los segmentos archivados después
// se restauran hasta el final del WAL archivado (Open) o hasta un lsn (OpenAt).

const (
	// archiveStatusName es el estado del archivo; archiveStatusTemp, su versión a medio escribir.
	archiveStatusName = "archive_status"
	archiveStatusTemp = "archive_status.tmp"

	// archiveStatusMagic es "ETAR"; archiveStatusSize, el largo del estado.
	archiveStatusMagic = 0x45544152
	archiveStatusSize  = 32

	// maxPendingArchive es cuántos segmentos devuelve como mucho PendingArchive: el archivador pide de nuevo.
	maxPendingArchive = 256
)

// WALSegment es un segmento sellado del WAL que espera al archivador.
type WALSegment struct {
	Segment  uint64 // número del segmento: el que recibe SetArchived
	Name     string // nombre del archivo dentro de wal/ ("0000000003.seg"): con este nombre va a wal/ al restaurar
	Path     string // ruta del archivo en el directorio del motor
	StartLSN uint64 // lsn de su primer registro
	EndLSN   uint64 // lsn de su último registro
	Size     int64  // bytes con registros válidos desde el inicio: alcanza con copiar estos (ver doc.go)
}

// archiveStatus es lo que se guarda en archive_status.
type archiveStatus struct {
	archived uint64 // el último segmento archivado (0 = ninguno)
	gap      uint64 // primer segmento después de la última pérdida de continuidad (0 = nunca la hubo)
}

// archiveState es el archivo continuo de un motor.
type archiveState struct {
	// mu: SetArchived, el tope del checkpoint y Open. Va antes que el writeMu del WAL (Hold).
	mu     sync.Mutex
	on     bool          // WAL_ARCHIVE al abrir; no cambia
	next   atomic.Uint64 // el primer segmento sin archivar: desde él se retiene; se escribe con mu
	gap    atomic.Uint64 // archiveStatus.gap; se escribe con mu
	unhold func()        // suelta la retención actual (wal.Hold); con mu
}

/**
* moveHold retiene el WAL desde next y suelta la retención anterior. Con a.mu.
* @param log *wal.WAL, next uint64
**/
func (a *archiveState) moveHold(log *wal.WAL, next uint64) {
	unhold := log.Hold(next)
	if a.unhold != nil {
		a.unhold()
	}
	a.unhold = unhold
	a.next.Store(next)
}

/**
* startArchive activa el archivo al abrir (con WAL_ARCHIVE=on), después de rehacer el WAL y antes de escribir nada:
* lee archive_status, retiene desde el primer segmento sin archivar, guarda el estado si cambió y sella el segmento
* activo si tiene registros. Sin estado (primera vez) archiva desde el segmento más viejo que haya; si con el archivo
* apagado se reciclaron segmentos que no se archivaron, anota la pérdida de continuidad (gap). El sellado evita
* agregar registros a un segmento que el archivador ya pudo copiar antes de un corte (Open sigue escribiendo en el
* último segmento con registros, y uno sellado con SwitchWAL tiene lugar).
* @return error
**/
func (e *Engine) startArchive() error {
	if !e.cfg.archive {
		return nil
	}
	saved, found, err := e.readArchiveStatus()
	if err != nil {
		return err
	}
	nums, err := e.log.Segments()
	if err != nil {
		return err
	}
	oldest := nums[0]
	st := archiveStatus{archived: oldest - 1}
	if found {
		st = saved
		if oldest > st.archived+1 { // reciclados sin archivar con el archivo apagado
			st = archiveStatus{archived: oldest - 1, gap: oldest}
		}
	}

	a := &e.archive
	a.mu.Lock()
	defer a.mu.Unlock()
	a.on = true
	a.gap.Store(st.gap)
	a.moveHold(e.log, st.archived+1)
	if !found || st != saved {
		if err := e.writeArchiveStatus(st); err != nil {
			return err
		}
	}
	if _, err := e.log.Switch(); err != nil {
		return err
	}
	return nil
}

/**
* pendingArchive lista los segmentos sellados sin archivar, del más viejo al más nuevo, hasta maxPendingArchive.
* Los lee del WAL (el directorio y lo anotado al sellar), no de una lista en memoria.
* @return []WALSegment, error: ErrArchiveOff, ErrClosed, ErrFailed
**/
func (e *Engine) pendingArchive() ([]WALSegment, error) {
	if err := e.enterRead(); err != nil {
		return nil, err
	}
	defer e.leave()
	a := &e.archive
	if !a.on {
		return nil, ErrArchiveOff
	}
	var out []WALSegment
	active := e.log.ActiveSegment()
	for num := a.next.Load(); num < active && len(out) < maxPendingArchive; num++ {
		info, err := e.log.SegmentInfo(num)
		if err != nil {
			if num < a.next.Load() {
				continue // lo archivaron (y quizá se recicló) mientras tanto
			}
			return out, err
		}
		name := wal.SegmentName(num)
		out = append(out, WALSegment{
			Segment:  num,
			Name:     name,
			Path:     filepath.Join(e.dir, wal.Dir, name),
			StartLSN: uint64(info.First),
			EndLSN:   uint64(info.End.LSN - 1),
			Size:     info.End.Offset,
		})
	}
	return out, nil
}

/**
* setArchived anota que el archivador ya copió los segmentos hasta seg (inclusive): guarda archive_status (durable)
* y recién después mueve la retención, así un corte en el medio solo hace volver a archivar. Lo ya archivado no hace
* nada. Despierta a los escritores frenados por el archivo.
* @param seg uint64
* @return error: ErrArchiveOff, ErrNotSealed (seg no está sellado), ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) setArchived(seg uint64) error {
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	a := &e.archive
	if !a.on {
		return ErrArchiveOff
	}
	if active := e.log.ActiveSegment(); seg >= active {
		return fmt.Errorf(MSG_NOT_SEALED, ErrNotSealed, seg, active)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if seg < a.next.Load() {
		return nil
	}
	if err := e.writeArchiveStatus(archiveStatus{archived: seg, gap: a.gap.Load()}); err != nil {
		return e.writeError(err)
	}
	a.moveHold(e.log, seg+1)
	e.wakeWriters()
	return nil
}

/**
* switchWAL sella el segmento activo aunque no esté lleno (pg_switch_wal), para archivarlo sin esperar.
* @return uint64: el segmento sellado (0 si el activo estaba vacío), error: ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) switchWAL() (uint64, error) {
	if err := e.enter(); err != nil {
		return 0, err
	}
	defer e.leave()
	if err := e.failed(); err != nil {
		return 0, err
	}
	num, err := e.log.Switch()
	switch {
	case err == nil:
		return num, nil
	case errors.Is(err, wal.ErrNoRoom):
		return 0, e.diskFull(err)
	}
	return 0, e.fail(err)
}

/**
* archiveRetained son los bytes de los segmentos sellados que esperan al archivador (cada uno ocupa WAL_SEGMENT_SIZE
* en el disco por la preasignación). Sin candados.
* @return int64
**/
func (e *Engine) archiveRetained() int64 {
	a := &e.archive
	if !a.on {
		return 0
	}
	next, active := a.next.Load(), e.log.ActiveSegment()
	if active <= next {
		return 0
	}
	return int64(active-next) * e.cfg.wal.SegmentSize
}

/**
* archiveBacklog es lo que el archivo suma al freno de los escritores: lo retenido si no hay tope
* (WAL_ARCHIVE_MAX_SIZE = 0); con tope, nada (el tope suelta la retención en vez de frenar).
* @return int64: bytes
**/
func (e *Engine) archiveBacklog() int64 {
	if e.cfg.archiveMaxSize > 0 {
		return 0
	}
	return e.archiveRetained()
}

/**
* enforceArchiveCap es el tope del archivo (como max_slot_wal_keep_size), antes del reciclaje del checkpoint: si lo
* retenido pasa de WAL_ARCHIVE_MAX_SIZE, deja de retener lo anterior al punto de rehacer y anota la pérdida de
* continuidad (gap = ese segmento) en archive_status, antes de soltar nada. Desde ahí la recuperación a un punto en
* el tiempo necesita un respaldo nuevo. Con ckptMu.
* @param redo uint64: el segmento del punto de rehacer del checkpoint
* @return error: el de guardar el estado (un disco lleno limpio deja la retención como estaba)
**/
func (e *Engine) enforceArchiveCap(redo uint64) error {
	a := &e.archive
	if !a.on || e.cfg.archiveMaxSize <= 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if redo <= a.next.Load() || e.archiveRetained() <= e.cfg.archiveMaxSize {
		return nil
	}
	if err := e.writeArchiveStatus(archiveStatus{archived: redo - 1, gap: redo}); err != nil {
		return e.writeError(err)
	}
	a.gap.Store(redo)
	a.moveHold(e.log, redo)
	e.metrics.archiveLost.Add(1)
	return nil
}

/**
* encodeArchiveStatus arma archive_status: magic(4) version(4) archived(8) gap(8) reservado(4) crc(4) de todo lo
* anterior.
* @param st archiveStatus
* @return []byte
**/
func encodeArchiveStatus(st archiveStatus) []byte {
	b := make([]byte, archiveStatusSize)
	le.PutUint32(b[0:], archiveStatusMagic)
	le.PutUint32(b[4:], control.FormatVersion)
	le.PutUint64(b[8:], st.archived)
	le.PutUint64(b[16:], st.gap)
	le.PutUint32(b[28:], crc32.Checksum(b[:28], crcTable))
	return b
}

/**
* decodeArchiveStatus lee archive_status.
* @param b []byte
* @return archiveStatus, bool: false si el largo, el magic, la versión o el crc no son los del formato
**/
func decodeArchiveStatus(b []byte) (archiveStatus, bool) {
	if len(b) != archiveStatusSize || le.Uint32(b[0:]) != archiveStatusMagic ||
		!wal.Readable(le.Uint32(b[4:])) || le.Uint32(b[28:]) != crc32.Checksum(b[:28], crcTable) {
		return archiveStatus{}, false
	}
	return archiveStatus{archived: le.Uint64(b[8:]), gap: le.Uint64(b[16:])}, true
}

/**
* readArchiveStatus lee archive_status del motor, si lo hay. Se escribe de forma atómica, así que uno dañado es
* corrupción.
* @return archiveStatus, bool: si hay estado, error: ErrCorrupt
**/
func (e *Engine) readArchiveStatus() (archiveStatus, bool, error) {
	names, err := e.fs.ReadDir(e.dir)
	if err != nil {
		return archiveStatus{}, false, err
	}
	if !slices.Contains(names, archiveStatusName) {
		return archiveStatus{}, false, nil
	}
	f, err := e.fs.OpenFile(filepath.Join(e.dir, archiveStatusName), os.O_RDONLY)
	if err != nil {
		return archiveStatus{}, false, err
	}
	defer f.Close()
	b := make([]byte, archiveStatusSize+1)
	n, _ := f.ReadAt(b, 0)
	st, ok := decodeArchiveStatus(b[:n])
	if !ok {
		return st, false, fmt.Errorf(MSG_ARCHIVE_STATUS, ErrCorrupt)
	}
	return st, true, nil
}

/**
* writeArchiveStatus guarda archive_status de forma atómica: archive_status.tmp, fsync, rename y fsync del
* directorio. Un disco lleno al escribir el temporal no cambia nada.
* @param st archiveStatus
* @return error
**/
func (e *Engine) writeArchiveStatus(st archiveStatus) error {
	tmp, path := filepath.Join(e.dir, archiveStatusTemp), filepath.Join(e.dir, archiveStatusName)
	f, err := e.fs.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(encodeArchiveStatus(st), 0)
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
	return err
}
