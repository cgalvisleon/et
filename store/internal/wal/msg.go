package wal

import "errors"

// Mensajes de error del paquete.
const (
	MSG_BAD_RECORD      = "wal: invalid record: %s"
	MSG_FAILED          = "wal: a previous write or fsync failed, reopen the engine: %w"
	MSG_BAD_SEGMENT     = "wal: invalid segment file name %s"
	MSG_SEGMENT_MISSING = "wal: segment %d is missing (found %d after %d)"
	MSG_SEGMENT_GONE    = "wal: segment %d does not exist"
	MSG_WAL_MISSING     = "wal: no segments but the redo point is %+v"
	MSG_GAP             = "wal: %w: segment %d starts at lsn %d, expected %d"
	MSG_BAD_POSITION    = "wal: invalid position %+v"
	MSG_LSN_AHEAD       = "wal: lsn %d was never reserved (last %d)"
	MSG_SEGMENT_SIZE    = "wal: segment size %d is smaller than %d"
	MSG_WRITTEN_TWICE   = "wal: reservation %d already written"
	MSG_KEEP_AHEAD      = "wal: keep segment %d is past the active segment %d"
	MSG_BAD_IMAGE       = "wal: invalid page image: %s"
)

var (
	// ErrClosed: el WAL ya se cerró.
	ErrClosed = errors.New("wal: closed")

	// ErrCorrupted: faltan registros confirmados (un segmento empieza después de un hueco).
	ErrCorrupted = errors.New("wal: corrupted")
)
