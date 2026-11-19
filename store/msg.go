package store

import "errors"

// Errores públicos del motor. Se comparan con errors.Is; los de las piezas internas van envueltos con %w.
var (
	// ErrLocked: otro motor (de este u otro proceso) tiene abierto el directorio. Envuelve vfs.ErrLocked.
	ErrLocked = errors.New("store: the directory is open by another engine")

	// ErrClosed: el motor ya se cerró.
	ErrClosed = errors.New("store: engine closed")

	// ErrFailed: una escritura o un fsync falló y el motor quedó inutilizable; se recupera cerrando y abriendo.
	ErrFailed = errors.New("store: a write or fsync failed, close and reopen the engine")

	// ErrBadID: el id no mide de 1 a MaxIDLen bytes.
	ErrBadID = errors.New("store: id must be 1 to 2000 bytes")

	// ErrValueTooLarge: el valor pasa de MaxValueLen.
	ErrValueTooLarge = errors.New("store: value is larger than 64 MiB")

	// ErrBadName: el nombre de un store no es UTF-8 válido sin nulos, de 1 a 255 bytes.
	ErrBadName = errors.New("store: store name must be valid UTF-8 without NUL, 1 to 255 bytes")

	// ErrStoreNotFound: DropStore de un store que no existe.
	ErrStoreNotFound = errors.New("store: store not found")

	// ErrStoreDropped: el store se borró con DropStore; su *Store ya no sirve.
	ErrStoreDropped = errors.New("store: store was dropped")

	// ErrConflict: otra transacción tuvo el id más de LOCK_TIMEOUT. En un lote, el lote queda abortado.
	ErrConflict = errors.New("store: timeout waiting for another transaction on the same id")

	// ErrSerialization: un lote escribió un id que otra transacción cambió y confirmó después de que el lote empezó
	// (aislamiento por snapshot, pregunta 34). El lote queda abortado: solo admite Rollback.
	ErrSerialization = errors.New("store: could not serialize: the id was changed by another transaction after the batch started")

	// ErrTimeout: el lote o el snapshot pasó su tiempo máximo (TRANSACTION_TIMEOUT o el de SnapshotFor); el lote
	// quedó abortado y el snapshot cerrado.
	ErrTimeout = errors.New("store: the batch or snapshot passed its time limit")

	// ErrTxDone: el lote ya se confirmó o se deshizo, o el snapshot ya se cerró.
	ErrTxDone = errors.New("store: the batch or snapshot is already finished")

	// ErrOtherEngine: el lote o el snapshot es de otro motor que el store.
	ErrOtherEngine = errors.New("store: the batch or snapshot belongs to another engine")

	// ErrCorrupt: el WAL, el control o el catálogo no son coherentes (falta el registro del checkpoint, un registro
	// del motor está mal formado…).
	ErrCorrupt = errors.New("store: corrupted engine")
)

// Mensajes con detalle.
const (
	MSG_LOCKED          = "%w: %s: %w"
	MSG_FAILED          = "%w: %w"
	MSG_OPEN            = "store: open %s: %w"
	MSG_NO_CHECKPOINT   = "%w: the checkpoint record at lsn %d is missing from the WAL"
	MSG_BAD_RECORD      = "%w: wal record lsn %d kind %d: %s"
	MSG_NOT_SUPPORTED   = "%w: wal record lsn %d kind %d is not supported yet"
	MSG_STORE_NAME      = "%w: %q"
	MSG_REDO            = "store: redo lsn %d kind %d: %w"
	MSG_BAD_CONTROL_NEW = "%w: the control file has no valid copy but the catalog has stores: %w"
)
