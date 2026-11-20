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
	// lote podría tener versiones sin confirmar en él), o DropStoreIn de un store en el que escribió otro lote que
	// sigue abierto. Se reintenta cuando terminen.
	ErrBusy = errors.New("store: busy: open batches or snapshots use the store")

	// ErrStorePending: el store lo está creando o borrando un lote abierto (DDL en lote). Store espera hasta
	// LOCK_TIMEOUT a que ese lote termine y después da este error; las escrituras de otros en él y DropStore lo dan
	// enseguida. Se reintenta cuando el lote termine.
	ErrStorePending = errors.New("store: the store is being created or dropped by an open batch")

	// ErrDiskFull: el disco se llenó. Si viene solo, la operación no hizo nada y el motor sigue: se puede reintentar
	// cuando haya lugar. Si viene con ErrFailed, el disco se llenó donde ya no se podía volver atrás (una escritura
	// del WAL ya reservada o un fsync): el motor quedó de solo lectura (las lecturas siguen, las escrituras dan este
	// error) hasta cerrarlo y abrirlo, que recupera todo lo confirmado (ver «Disco lleno o lento» en doc.go).
	ErrDiskFull = errors.New("store: no space left on device")

	// ErrThrottled: el disco no da abasto: el WAL desde el último checkpoint pasa de 2 × MAX_WAL_SIZE y no bajó en
	// LOCK_TIMEOUT. La operación no hizo nada; se reintenta.
	// Con el archivo continuo sin tope (WAL_ARCHIVE_MAX_SIZE = 0) cuenta también el WAL que espera al archivador.
	ErrThrottled = errors.New("store: the disk is not keeping up: too much WAL since the last checkpoint")

	// ErrArchiveOff: PendingArchive o SetArchived con el archivo continuo apagado (WAL_ARCHIVE=off al abrir).
	ErrArchiveOff = errors.New("store: the WAL archive is off (WAL_ARCHIVE)")

	// ErrNotSealed: SetArchived de un segmento que todavía no está sellado (el activo o uno futuro).
	ErrNotSealed = errors.New("store: the WAL segment is not sealed yet")

	// ErrRecoveryTarget: OpenAt con un objetivo que no se puede alcanzar: sin copia de Backup (backup_label), antes
	// del fin del respaldo o del checkpoint del control, o después del final del WAL que hay. No cambia nada.
	ErrRecoveryTarget = errors.New("store: the recovery target lsn cannot be reached")

	// ErrBackupDir: el directorio de destino de Backup no está vacío.
	ErrBackupDir = errors.New("store: the backup directory is not empty")

	// ErrBackupIncomplete: Open de una copia de Backup que no terminó o cuyo WAL no llega hasta el registro de fin
	// del respaldo: la copia no es consistente y Open no la abre (ni la cambia).
	ErrBackupIncomplete = errors.New("store: incomplete backup: its WAL does not reach the backup end")

	// ErrReadOnly: una escritura (o Checkpoint de un lote, Vacuum, Repair, Backup…) en una réplica (OpenReplica):
	// la réplica solo aplica el WAL del primario. Se escribe en el primario.
	ErrReadOnly = errors.New("store: the engine is a read-only replica")

	// ErrNotReplica: ApplyWal en un motor que no se abrió con OpenReplica, u OpenReplica de un directorio sin motor
	// (una réplica empieza de una copia de Backup).
	ErrNotReplica = errors.New("store: the engine is not a replica")

	// ErrWalGap: ApplyWal recibió un registro que no es el siguiente del WAL de la réplica (falta alguno, o su
	// posición no es la del primario). No aplicó nada de ahí en adelante: se pide de nuevo desde ReplayLSN.
	ErrWalGap = errors.New("store: the WAL entry does not follow the replica's WAL")

	// ErrWalRemoved: el WAL pedido (WalSince, Changes, una ranura) ya se recicló, o la ranura se invalidó por pasar
	// MAX_SLOT_WAL_KEEP_SIZE. La réplica necesita una copia nueva (Backup); el consumidor lógico, empezar de nuevo.
	ErrWalRemoved = errors.New("store: the requested WAL was already removed")

	// ErrSlotNotFound: la ranura no existe.
	ErrSlotNotFound = errors.New("store: replication slot not found")

	// ErrSlotExists: CreateSlot con un nombre que ya tiene una ranura.
	ErrSlotExists = errors.New("store: replication slot already exists")

	// ErrSlotLimit: CreateSlot con todas las ranuras en uso (maxSlots), o un nombre que no mide de 1 a 64 bytes.
	ErrSlotLimit = errors.New("store: too many replication slots, or a bad slot name")

	// ErrRecoveryConflict: un snapshot de la réplica se canceló porque el aplicador necesitaba rehacer un cambio que
	// el snapshot no debía ver (una confirmación, una limpieza o un DDL) y esperó más de replicaConflictDelay. Se
	// reintenta con un snapshot nuevo.
	ErrRecoveryConflict = errors.New("store: the replica snapshot was canceled to apply the primary's WAL")

	// ErrCorrupt: el WAL, el control o el catálogo no son coherentes (falta el registro del checkpoint, un registro
	// del motor está mal formado…).
	ErrCorrupt = errors.New("store: corrupted engine")
)

// Mensajes con detalle.
const (
	MSG_LOCKED          = "%w: %s: %w"
	MSG_FAILED          = "%w: %w"
	MSG_FAILED_FULL     = "%w: %w: %w"
	MSG_DISK_FULL       = "%w: %w"
	MSG_THROTTLED       = "%w: %d bytes of WAL not recyclable (since the last checkpoint or not archived), limit %d"
	MSG_OPEN            = "store: open %s: %w"
	MSG_NO_CHECKPOINT   = "%w: the checkpoint record at lsn %d is missing from the WAL"
	MSG_BAD_RECORD      = "%w: wal record lsn %d kind %d: %s"
	MSG_NOT_SUPPORTED   = "%w: wal record lsn %d kind %d is not supported yet"
	MSG_STORE_NAME      = "%w: %q"
	MSG_REDO            = "store: redo lsn %d kind %d: %w"
	MSG_BAD_CONTROL_NEW = "%w: the control file has no valid copy but the catalog has stores: %w"
	MSG_BUSY            = "%w: %d open"
	MSG_BUSY_WRITERS    = "%w: %d open batches wrote in store %q"
	MSG_STORE_PENDING   = "%w: %q"

	MSG_BACKUP            = "store: backup to %s: %w"
	MSG_BACKUP_DIR        = "%w: %s"
	MSG_BACKUP_LABEL      = "%w: the backup_label file is damaged"
	MSG_BACKUP_UNFINISHED = "%w: the backup did not finish (backup_label without an end)"
	MSG_BACKUP_END_RECORD = "%w: wal record lsn %d kind %d is not this backup's end"
	MSG_BACKUP_WAL        = "%w: cannot read the WAL up to the backup end lsn %d: %w"
	MSG_BACKUP_SHORT      = "%w: the WAL ends at lsn %d, before the backup end lsn %d"

	MSG_NOT_SEALED       = "%w: segment %d (the active segment is %d)"
	MSG_ARCHIVE_STATUS   = "%w: the archive_status file is damaged"
	MSG_TARGET_ZERO      = "%w: lsn 0"
	MSG_TARGET_NO_BACKUP = "%w: OpenAt needs a Backup copy (backup_label)"
	MSG_TARGET_EARLY     = "%w: lsn %d is before lsn %d (the backup end or the control checkpoint)"
	MSG_TARGET_LATE      = "%w: the WAL ends at lsn %d, before lsn %d"
	MSG_TARGET_WAL       = "%w: cannot read the WAL up to lsn %d: %w"

	MSG_REPLICA_FRESH = "%w: OpenReplica needs a Backup copy in %s"
	MSG_WAL_GAP       = "%w: entry lsn %d, the replica expects lsn %d"
	MSG_WAL_REMOVED   = "%w: lsn %d: %w"
	MSG_SLOT_NAME     = "%w: %q"
	MSG_SLOT_INVALID  = "%w: slot %q was invalidated (MAX_SLOT_WAL_KEEP_SIZE)"
	MSG_SLOT_FILE     = "%w: the slots file is damaged"
	MSG_SLOT_BUSY     = "%w: transactions older than the logical slot %q did not finish"
	MSG_SLOT_BACKWARD = "%w: slot %q is at lsn %d, cannot confirm lsn %d (ahead of the WAL at %d)"
	MSG_SPILL         = "store: logical decoding spill file: %w"
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
