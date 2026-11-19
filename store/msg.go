package store

// Mensajes de error del paquete.
const (
	MSG_ID_REQUIRED     = "store: id is required"
	MSG_ID_TOO_LONG     = "store: id is longer than %d bytes"
	MSG_DATA_TOO_LONG   = "store: data is longer than %d bytes"
	MSG_CLOSED          = "store: store is closed"
	MSG_FAILED          = "store: a previous write failed, reopen the store: %w"
	MSG_LOCKED          = "store: already open by another process: %s"
	MSG_CORRUPTED       = "store: corrupted record in sealed segment %s at offset %d"
	MSG_BAD_SEGMENT     = "store: invalid segment file name %s"
	MSG_SEGMENT_MISSING = "store: segment %d is missing (found %d after %d)"
)
