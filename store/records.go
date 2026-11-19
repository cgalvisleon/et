package store

import (
	"encoding/binary"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// le es el orden de bytes de todos los formatos del motor.
var le = binary.LittleEndian

const (
	// checkpointMainSize: redoSegment(8) redoOffset(8) redoLSN(8) nextXID(8) oldestActiveXID(8) time(8).
	checkpointMainSize = 48

	// storeMainHeader: fileNo(4) nameLen(2), seguido del nombre (KindCreateStore y KindDropStore).
	storeMainHeader = 6

	// commitMainHeader: time(8) storeCount(4), seguido de storeCount × [fileNo(4) delta(8)] (KindCommit).
	commitMainHeader = 12
	commitStoreSize  = 12
)

// checkpointInfo son los datos principales de un KindCheckpoint.
type checkpointInfo struct {
	redo    wal.Position
	nextXID page.XID
	oldest  page.XID
	time    int64
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
* encodeCommit arma los datos de un KindCommit: la hora y lo que la transacción suma a Count de cada store (fase 1B:
* los lotes; las operaciones sueltas no lo usan).
* @param time int64, deltas map[smgr.FileNo]int64
* @return []byte
**/
func encodeCommit(time int64, deltas map[smgr.FileNo]int64) []byte {
	b := make([]byte, commitMainHeader+commitStoreSize*len(deltas))
	le.PutUint64(b[0:], uint64(time))
	le.PutUint32(b[8:], uint32(len(deltas)))
	off := commitMainHeader
	for file, delta := range deltas {
		le.PutUint32(b[off:], uint32(file))
		le.PutUint64(b[off+4:], uint64(delta))
		off += commitStoreSize
	}
	return b
}

/**
* decodeCommit lee lo que un KindCommit suma a Count de cada store.
* @param b []byte
* @return map[smgr.FileNo]int64, error: con detalle si no tiene la forma del formato
**/
func decodeCommit(b []byte) (map[smgr.FileNo]int64, error) {
	if len(b) < commitMainHeader {
		return nil, fmt.Errorf("commit data has %d bytes", len(b))
	}
	n := int(le.Uint32(b[8:]))
	if n > (len(b)-commitMainHeader)/commitStoreSize || len(b) != commitMainHeader+n*commitStoreSize {
		return nil, fmt.Errorf("commit data has %d bytes for %d stores", len(b), n)
	}
	deltas := make(map[smgr.FileNo]int64, n)
	for off := commitMainHeader; off < len(b); off += commitStoreSize {
		deltas[smgr.FileNo(le.Uint32(b[off:]))] += int64(le.Uint64(b[off+4:]))
	}
	return deltas, nil
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
