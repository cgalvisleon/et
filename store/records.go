package store

import (
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// le es el orden de bytes de todos los formatos del motor.
var le = binary.LittleEndian

const (
	// checkpointMainSize: redoSegment(8) redoOffset(8) redoLSN(8) nextXID(8) oldestActiveXID(8) time(8).
	checkpointMainSize = 48

	// storeMainHeader: fileNo(4) nameLen(2), seguido del nombre (KindCreateStore, KindCreateStoreIn y KindDropStore).
	storeMainHeader = 6

	// emptyMainHeader: oldFileNo(4) newFileNo(4) nameLen(2), seguido del nombre (KindEmptyStore).
	emptyMainHeader = 10

	// commitMainHeader: time(8) storeCount(4), seguido de storeCount × [fileNo(4) delta(8)] (KindCommit).
	commitMainHeader = 12
	commitStoreSize  = 12

	// commitDDLHeader: createCount(4) dropCount(4), seguido de createCount × fileNo(4) y dropCount × fileNo(4): la
	// cola optativa de un KindCommit de un lote que creó o borró stores (tarea 67). Sin DDL no se escribe.
	commitDDLHeader = 8
)

// checkpointInfo son los datos principales de un KindCheckpoint.
type checkpointInfo struct {
	redo    wal.Position
	nextXID page.XID
	oldest  page.XID
	time    int64
	frozen  page.XID // el frozenXID del motor que guarda el control (no va en el registro)
	wal     int64    // log.Written() en el punto de rehacer (no va en el registro; ver checkpointFinished)
}

/**
* encodeCheckpoint arma los datos principales de un KindCheckpoint.
* @param c checkpointInfo
* @return []byte
**/
func encodeCheckpoint(c checkpointInfo) []byte {
	b := make([]byte, checkpointMainSize)
	le.PutUint64(b[0:], c.redo.Segment)
	le.PutUint64(b[8:], uint64(c.redo.Offset))
	le.PutUint64(b[16:], uint64(c.redo.LSN))
	le.PutUint64(b[24:], uint64(c.nextXID))
	le.PutUint64(b[32:], uint64(c.oldest))
	le.PutUint64(b[40:], uint64(c.time))
	return b
}

/**
* decodeCheckpoint lee los datos principales de un KindCheckpoint.
* @param b []byte
* @return checkpointInfo, bool: false si el largo no es el del formato
**/
func decodeCheckpoint(b []byte) (checkpointInfo, bool) {
	if len(b) != checkpointMainSize {
		return checkpointInfo{}, false
	}
	return checkpointInfo{
		redo: wal.Position{
			Segment: le.Uint64(b[0:]),
			Offset:  int64(le.Uint64(b[8:])),
			LSN:     page.LSN(le.Uint64(b[16:])),
		},
		nextXID: page.XID(le.Uint64(b[24:])),
		oldest:  page.XID(le.Uint64(b[32:])),
		time:    int64(le.Uint64(b[40:])),
	}, true
}

/**
* encodeStoreMain arma los datos de KindCreateStore y KindDropStore: el número de archivo y el nombre.
* @param file smgr.FileNo, name string: de 1 a MaxNameLen bytes
* @return []byte
**/
func encodeStoreMain(file smgr.FileNo, name string) []byte {
	b := make([]byte, storeMainHeader+len(name))
	le.PutUint32(b[0:], uint32(file))
	le.PutUint16(b[4:], uint16(len(name)))
	copy(b[storeMainHeader:], name)
	return b
}

/**
* decodeStoreMain lee los datos de KindCreateStore y KindDropStore.
* @param b []byte
* @return smgr.FileNo, string, bool: false si no tiene la forma del formato
**/
func decodeStoreMain(b []byte) (smgr.FileNo, string, bool) {
	if len(b) < storeMainHeader {
		return 0, "", false
	}
	file := smgr.FileNo(le.Uint32(b[0:]))
	n := int(le.Uint16(b[4:]))
	if file == 0 || n == 0 || n > MaxNameLen || len(b) != storeMainHeader+n {
		return 0, "", false
	}
	return file, string(b[storeMainHeader:]), true
}

/**
* encodeEmptyMain arma los datos de KindEmptyStore: el número de archivo viejo, el nuevo y el nombre.
* @param old, file smgr.FileNo, name string: de 1 a MaxNameLen bytes
* @return []byte
**/
func encodeEmptyMain(old, file smgr.FileNo, name string) []byte {
	b := make([]byte, emptyMainHeader+len(name))
	le.PutUint32(b[0:], uint32(old))
	le.PutUint32(b[4:], uint32(file))
	le.PutUint16(b[8:], uint16(len(name)))
	copy(b[emptyMainHeader:], name)
	return b
}

/**
* decodeEmptyMain lee los datos de KindEmptyStore.
* @param b []byte
* @return smgr.FileNo: el viejo, smgr.FileNo: el nuevo, string, bool: false si no tiene la forma del formato
**/
func decodeEmptyMain(b []byte) (smgr.FileNo, smgr.FileNo, string, bool) {
	if len(b) < emptyMainHeader {
		return 0, 0, "", false
	}
	old, file := smgr.FileNo(le.Uint32(b[0:])), smgr.FileNo(le.Uint32(b[4:]))
	n := int(le.Uint16(b[8:]))
	if old == 0 || file == 0 || old == file || n == 0 || n > MaxNameLen || len(b) != emptyMainHeader+n {
		return 0, 0, "", false
	}
	return old, file, string(b[emptyMainHeader:]), true
}

// commitInfo son los datos principales de un KindCommit: lo que suma a Count de cada store, el DDL del lote y su
// origen.
type commitInfo struct {
	deltas  map[smgr.FileNo]int64
	creates []smgr.FileNo // stores que el lote creó: dejan de estar pendientes
	drops   []smgr.FileNo // stores que el lote borra: salen del catálogo y se borran sus archivos
	origin  uint16        // el origen del lote (Batch.SetOrigin); 0 = escrito en este motor
}

// commitOriginSize es el largo del origen al final de un KindCommit con wal.FlagOrigin (tarea 83).
const commitOriginSize = 2

/**
* encodeCommit arma los datos de un KindCommit: la hora, lo que la transacción suma a Count de cada store (fase 1B:
* los lotes; las operaciones sueltas no lo usan) y, si el lote hizo DDL, la cola con los stores creados y borrados.
* Con origen (c.origin ≠ 0) le sigue origin(2) y el registro lleva wal.FlagOrigin (commitFlags).
* @param time int64, c commitInfo
* @return []byte
**/
func encodeCommit(time int64, c commitInfo) []byte {
	size := commitMainHeader + commitStoreSize*len(c.deltas)
	ddl := len(c.creates) + len(c.drops)
	if ddl > 0 {
		size += commitDDLHeader + 4*ddl
	}
	b := make([]byte, size, size+commitOriginSize)
	le.PutUint64(b[0:], uint64(time))
	le.PutUint32(b[8:], uint32(len(c.deltas)))
	off := commitMainHeader
	for file, delta := range c.deltas {
		le.PutUint32(b[off:], uint32(file))
		le.PutUint64(b[off+4:], uint64(delta))
		off += commitStoreSize
	}
	if ddl > 0 {
		le.PutUint32(b[off:], uint32(len(c.creates)))
		le.PutUint32(b[off+4:], uint32(len(c.drops)))
		off += commitDDLHeader
		for _, file := range append(slices.Clone(c.creates), c.drops...) {
			le.PutUint32(b[off:], uint32(file))
			off += 4
		}
	}
	if c.origin != 0 {
		b = le.AppendUint16(b, c.origin)
	}
	return b
}

/**
* commitFlags son las marcas del registro de un KindCommit con c: wal.FlagOrigin si lleva origen.
* @param c commitInfo
* @return wal.Flags
**/
func commitFlags(c commitInfo) wal.Flags {
	if c.origin != 0 {
		return wal.FlagOrigin
	}
	return 0
}

/**
* decodeCommit lee lo que un KindCommit suma a Count de cada store, su cola de DDL (si la tiene) y su origen (con
* wal.FlagOrigin).
* @param b []byte: los datos del registro, flags wal.Flags: sus marcas
* @return commitInfo, error: con detalle si no tiene la forma del formato
**/
func decodeCommit(b []byte, flags wal.Flags) (commitInfo, error) {
	origin := uint16(0)
	if flags&wal.FlagOrigin != 0 {
		if len(b) < commitMainHeader+commitOriginSize {
			return commitInfo{}, fmt.Errorf("commit data has %d bytes with an origin", len(b))
		}
		origin = le.Uint16(b[len(b)-commitOriginSize:])
		b = b[:len(b)-commitOriginSize]
		if origin == 0 {
			return commitInfo{}, fmt.Errorf("commit origin is 0 with FlagOrigin")
		}
	}
	c, err := decodeCommitBody(b)
	c.origin = origin
	return c, err
}

/**
* decodeCommitBody lee la parte de un KindCommit sin el origen: las sumas a Count y la cola de DDL.
* @param b []byte
* @return commitInfo, error
**/
func decodeCommitBody(b []byte) (commitInfo, error) {
	if len(b) < commitMainHeader {
		return commitInfo{}, fmt.Errorf("commit data has %d bytes", len(b))
	}
	n := int(le.Uint32(b[8:]))
	if n > (len(b)-commitMainHeader)/commitStoreSize {
		return commitInfo{}, fmt.Errorf("commit data has %d bytes for %d stores", len(b), n)
	}
	end := commitMainHeader + n*commitStoreSize
	c := commitInfo{deltas: make(map[smgr.FileNo]int64, n)}
	for off := commitMainHeader; off < end; off += commitStoreSize {
		c.deltas[smgr.FileNo(le.Uint32(b[off:]))] += int64(le.Uint64(b[off+4:]))
	}
	if len(b) == end {
		return c, nil
	}
	if len(b)-end < commitDDLHeader {
		return commitInfo{}, fmt.Errorf("commit data has %d bytes after %d stores", len(b)-end, n)
	}
	creates, drops := int(le.Uint32(b[end:])), int(le.Uint32(b[end+4:]))
	tail := len(b) - end - commitDDLHeader
	if creates+drops == 0 || creates > tail/4 || drops > tail/4 || tail != 4*(creates+drops) {
		return commitInfo{}, fmt.Errorf("commit ddl has %d bytes for %d creates and %d drops", tail, creates, drops)
	}
	off := end + commitDDLHeader
	for i := range creates + drops {
		file := smgr.FileNo(le.Uint32(b[off+4*i:]))
		if file == 0 {
			return commitInfo{}, fmt.Errorf("commit ddl names file number 0")
		}
		if i < creates {
			c.creates = append(c.creates, file)
		} else {
			c.drops = append(c.drops, file)
		}
	}
	return c, nil
}

/**
* leafCountDelta es lo que una escritura suelta (KindLeafWrite con FlagCommit) suma a Count.
* @param flags wal.Flags
* @return int64: +1, -1 o 0
**/
func leafCountDelta(flags wal.Flags) int64 {
	switch {
	case flags&wal.FlagCountAdd != 0:
		return 1
	case flags&wal.FlagCountSub != 0:
		return -1
	}
	return 0
}
