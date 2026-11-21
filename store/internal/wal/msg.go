package wal

import "errors"

// Mensajes de error del paquete.
const (
	MSG_BAD_RECORD      = "wal: invalid record: %s"
	MSG_FAILED          = "wal: a previous write or fsync failed, reopen the engine: %w"
	MSG_BAD_SEGMENT     = "wal: invalid segment file name %s"
	MSG_SEGMENT_MISSING = "wal: segment %d is missing (found %d after %d)"
	MSG_SEGMENT_GONE    = "wal: %w: segment %d does not exist"
	MSG_LSN_GONE        = "wal: %w: lsn %d is before the oldest segment"
	MSG_LSN_MISSING     = "wal: lsn %d is not in the WAL (it ends at lsn %d)"
	MSG_APPEND_ORDER    = "wal: %w: record lsn %d at %+v, the next goes at segment %d offset %d lsn %d"
	MSG_APPEND_RAW      = "wal: %w: not one valid record"
	MSG_WAL_MISSING     = "wal: no segments but the redo point is %+v"
	MSG_GAP             = "wal: %w: segment %d starts at lsn %d, expected %d"
	MSG_BAD_POSITION    = "wal: invalid position %+v"
	MSG_LSN_AHEAD       = "wal: lsn %d was never reserved (last %d)"
	MSG_SEGMENT_SIZE    = "wal: segment size %d is smaller than %d"
	MSG_WRITTEN_TWICE   = "wal: reservation %d already written"
	MSG_KEEP_AHEAD      = "wal: keep segment %d is past the active segment %d"
	MSG_BAD_IMAGE       = "wal: invalid page image: %s"
	MSG_NO_ROOM         = "%w: %d bytes ready, %d needed: %w"
	MSG_NOT_SEALED      = "wal: segment %d is not sealed (the active segment is %d)"
	MSG_SEGMENT_EMPTY   = "wal: %w: sealed segment %d has no valid first record"
	MSG_SEALED_CUT      = "wal: %w: invalid record in sealed segment %d at offset %d (lsn %d), its records end at offset %d"
	MSG_SEALED_AFTER    = "wal: %w: sealed segment %d is past the end of the valid WAL at %+v"
	MSG_SEALED_NEXT     = "wal: %w: segment %d is sealed but segment %d has no valid header for lsn %d"
	MSG_HEADER_VERSION  = "wal: %w: segment %s is format version %d, this binary reads version %d"
	MSG_HEADER_BAD      = "wal: %w: segment %d has an invalid header for position %+v"
	MSG_HEADER_RAW      = "wal: %w: not a valid segment header"
)

var (
	// ErrClosed: el WAL ya se cerró.
	ErrClosed = errors.New("wal: closed")

	// ErrCorrupted: faltan registros confirmados (un segmento empieza después de un hueco).
	ErrCorrupted = errors.New("wal: corrupted")

	// ErrNoRoom: no se pudo preasignar lugar en el disco para el WAL (Room, o una reserva que tuvo que preasignar);
	// envuelve el error de disco lleno. No deja el WAL inutilizable.
	ErrNoRoom = errors.New("wal: no room on disk for the WAL")

	// ErrRemoved: lo pedido está en un segmento que ya no existe (se recicló o se borró).
	ErrRemoved = errors.New("wal: segment removed")

	// ErrOutOfOrder: Append recibió un registro que no es el siguiente (lsn o posición) del WAL de la réplica.
	ErrOutOfOrder = errors.New("wal: record out of order")
)
