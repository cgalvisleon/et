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

	// ErrBusy: Empty con lotes o snapshots abiertos en el motor (cualquiera de ellos podría ver el store vaciado, y un
	// lote podría tener versiones sin confirmar en él). Se reintenta cuando terminen.
	ErrBusy = errors.New("store: cannot empty the store while batches or snapshots are open")

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
	MSG_BUSY            = "%w: %d open"
)

// Mensajes del informe de Verify y Repair (report.go, verify.go, repair.go).
const (
	MSG_REPORT_TOTALS   = "stores %d, pages %d (tree %d, chains %d, free %d, lost %d), versions %d, live ids %d, orphan files %d\n"
	MSG_REPORT_FINDINGS = "findings %d (repaired %d, irreparable %d)\n"
	MSG_REPORT_OMITTED  = "... and %d more findings\n"

	MSG_CHECK_CONTROL_READ  = "cannot read the control file: %v"
	MSG_CHECK_CONTROL_REDO  = "redo point lsn %d, the engine's is %d"
	MSG_CHECK_CONTROL_CKPT  = "checkpoint lsn %d is before the redo point lsn %d"
	MSG_CHECK_CONTROL_XID   = "frozenXID %d is after nextXID %d"
	MSG_CHECK_CATALOG_READ  = "cannot read the catalog: %v"
	MSG_CHECK_CATALOG_LEN   = "the saved catalog has %d stores, the engine %d"
	MSG_CHECK_CATALOG_STORE = "store %q has file %d in the saved catalog and %d in the engine"
	MSG_CHECK_CATALOG_NEXT  = "the saved catalog's next file number %d is after the engine's %d"
	MSG_CHECK_DATA_DIR      = "cannot list the data directory: %v"
	MSG_CHECK_ORPHAN_FILE   = "%d data files of file number %d, which no store uses"
	MSG_CHECK_UNKNOWN_FILE  = "not a data file name"
	MSG_CHECK_NO_META       = "the store has no meta page: Store creates it empty"
	MSG_CHECK_BLOCKS        = "cannot count the blocks: %v"
	MSG_CHECK_META          = "cannot read the meta page: %v"
	MSG_CHECK_TREE          = "the tree does not verify: %v"
	MSG_CHECK_DOUBLE_TREE   = "tree page reached twice"
	MSG_CHECK_DOUBLE_CHAIN  = "page %d of the overflow chain of %.40q already has another owner"
	MSG_CHECK_DOUBLE_FREE   = "page in the free list already has another owner"
	MSG_CHECK_CHAIN         = "overflow chain of %.40q (lsn %d): %v"
	MSG_CHECK_FREE_LIST     = "the free list is damaged: %v"
	MSG_CHECK_STATUS        = "cannot read the status of xid %d: %v"
	MSG_CHECK_FROZEN        = "%d versions below the store's frozenXID %d are not frozen (first xid %d)"
	MSG_CHECK_FROZEN_NEXT   = "the store's frozenXID %d is after nextXID %d"
	MSG_CHECK_VM_LEAF       = "visibility bits %d on a leaf that is not %s"
	MSG_CHECK_VM_NOT_LEAF   = "visibility bits %d on a block that is not a leaf"
	MSG_CHECK_VM_READ       = "cannot read the visibility map: %v"
	MSG_CHECK_COUNT         = "Count is %d, the walk finds %d live ids"
	MSG_CHECK_LOST          = "pages without an owner: %s"
	MSG_CHECK_ORPHAN_CHAIN  = "overflow pages without a version (orphan chain)"
)
