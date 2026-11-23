package store

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Catálogo de la API pública (fases 1A, 1B, 2 y 3): envolturas delgadas; la implementación vive en los demás archivos del
// paquete con el mismo nombre en minúscula. Ver doc.go para la semántica de cada método.
//
// Disco lleno o lento (doc.go): toda escritura puede devolver ErrDiskFull (sin ErrFailed: no hizo nada y se
// reintenta cuando haya lugar; con ErrFailed: el motor quedó de solo lectura hasta reabrirlo) y ErrThrottled (el WAL
// sin reciclar pasa de 2 × MAX_WAL_SIZE: no hizo nada y se reintenta). Con el motor de solo lectura las lecturas,
// los recorridos y los snapshots siguen.

// ---------------------------------------------------------------------------------------------------------------
// Motor
// ---------------------------------------------------------------------------------------------------------------

/**
* Open abre el motor del directorio dir (lo crea si no existe): toma LOCK y recupera desde el último checkpoint.
* @param dir string
* @return *Engine, error: ErrLocked si otro motor tiene el directorio, ErrCorrupt, ErrDiskFull (sin lugar para el
* checkpoint que cierra la recuperación: se reintenta con lugar)
**/
func Open(dir string) (*Engine, error) {
	return open(vfs.OS(), dir, envConfig())
}

/**
* OpenWith es Open con opciones propias de este motor (Options): lo que no fija sale de las variables de entorno, así
* un proceso abre varios motores con distinta caché, WAL y tiempos.
* @param dir string, o Options
* @return *Engine, error: los de Open
**/
func OpenWith(dir string, o Options) (*Engine, error) {
	return openWith(dir, o, false)
}

/**
* OpenAtTime restaura una copia de Backup hasta una hora (recovery_target_time de PostgreSQL): como OpenAt, con el
* objetivo en el último registro antes del primero con hora posterior a t (una confirmación de lote, un checkpoint o
* el fin de un respaldo; las escrituras sueltas no llevan hora y van con lo que las rodea). Si nada en el WAL es
* posterior a t, rehace todo.
* @param dir string, t time.Time: no antes del fin del respaldo
* @return *Engine, error: ErrRecoveryTarget (sin backup_label o t antes del fin del respaldo: no cambia nada), los de
* Open
**/
func OpenAtTime(dir string, t time.Time) (*Engine, error) {
	cfg := envConfig()
	cfg.stopTime = t
	return open(vfs.OS(), dir, cfg)
}

/**
* OpenAt restaura una copia de Backup hasta el lsn dado (recovery_target_lsn de PostgreSQL): rehace el WAL del
* respaldo más los segmentos archivados que se agregaron a wal/ hasta el registro de ese lsn inclusive, corta lo que
* sigue y abre el motor. Lo confirmado después de lsn no está; lo que estaba a medias en lsn queda deshecho.
* @param dir string, lsn uint64: ≥ el fin del respaldo (BackupInfo.EndLSN)
* @return *Engine, error: ErrRecoveryTarget (sin backup_label, antes del fin del respaldo o después del final del
* WAL: no cambia nada), los de Open
**/
func OpenAt(dir string, lsn uint64) (*Engine, error) {
	if lsn == 0 {
		return nil, fmt.Errorf(MSG_TARGET_ZERO, ErrRecoveryTarget)
	}
	cfg := envConfig()
	cfg.stopLSN = page.LSN(lsn)
	return open(vfs.OS(), dir, cfg)
}

/**
* Close espera a las operaciones en curso, hace el checkpoint final y cierra. Cerrar dos veces no hace nada.
* @return error: el fallo que dejó el motor inutilizable, si lo hubo
**/
func (e *Engine) Close() error {
	return e.close()
}

/**
* Store abre el store name, o lo crea si no existe. Si lo está creando un lote abierto (StoreIn), espera hasta
* LOCK_TIMEOUT a que confirme o deshaga.
* @param name string: UTF-8 válido sin nulos, de 1 a 255 bytes
* @return *Store, error: ErrBadName, ErrStorePending, ErrClosed, ErrFailed
**/
func (e *Engine) Store(name string) (*Store, error) {
	return e.store(name)
}

/**
* DropStore borra el store name con todos sus datos.
* @param name string
* @return error: ErrStoreNotFound, ErrStorePending (un lote abierto lo crea o lo borra), ErrClosed, ErrFailed
**/
func (e *Engine) DropStore(name string) error {
	return e.dropStore(name)
}

/**
* StoreIn abre el store name para el lote tx, o lo crea dentro del lote: hasta el Commit solo tx lo ve (Store desde
* fuera espera, las escrituras de otros dan ErrStorePending y Stores no lo lista); Rollback, el tiempo máximo o un
* corte antes del Commit lo borran con sus archivos. Dos lotes que crean el mismo nombre: el segundo espera al
* primero y, si confirmó, recibe ErrSerialization.
* @param tx *Batch, name string
* @return *Store, error: ErrBadName, ErrSerialization, ErrConflict, ErrTimeout, ErrStoreDropped, ErrStorePending,
* ErrTxDone, ErrOtherEngine, ErrClosed, ErrFailed, ErrDiskFull
**/
func (e *Engine) StoreIn(tx *Batch, name string) (*Store, error) {
	return e.storeIn(tx, name)
}

/**
* DropStoreIn borra el store name al confirmar el lote tx (el Commit espera el fsync, también con CommitDeferred, y
* recién después borra los archivos). Hasta entonces sigue entero; Rollback lo deja como estaba. Uno creado por el
* mismo lote se borra enseguida.
* @param tx *Batch, name string
* @return error: ErrStoreNotFound, ErrBusy (otro lote abierto escribió en él), ErrSerialization, ErrConflict,
* ErrTimeout, ErrStoreDropped (tx ya lo borra), ErrStorePending, ErrTxDone, ErrOtherEngine, ErrClosed, ErrFailed
**/
func (e *Engine) DropStoreIn(tx *Batch, name string) error {
	return e.dropStoreIn(tx, name)
}

/**
* Stores devuelve los nombres de los stores del motor, en orden.
* @return []string
**/
func (e *Engine) Stores() []string {
	return e.storeNames()
}

/**
* NewBatch empieza un lote: una transacción que puede abarcar varios stores del motor, con aislamiento por snapshot,
* y que dura como mucho TRANSACTION_TIMEOUT. Termina con Commit o Rollback.
* @return *Batch: si el motor está cerrado o inutilizable, sus operaciones devuelven ese error
**/
func (e *Engine) NewBatch() *Batch {
	return e.newBatch()
}

/**
* Snapshot toma una foto fija de todos los stores del motor para leer con ReadAt; dura como mucho
* TRANSACTION_TIMEOUT. Termina con Close.
* @return *Snapshot: si el motor está cerrado o inutilizable, ReadAt devuelve ese error
**/
func (e *Engine) Snapshot() *Snapshot {
	return e.snapshotFor(e.cfg.txTimeout)
}

/**
* SnapshotFor es Snapshot con un tiempo máximo propio.
* @param timeout time.Duration: 0 = sin tiempo máximo
* @return *Snapshot
**/
func (e *Engine) SnapshotFor(timeout time.Duration) *Snapshot {
	return e.snapshotFor(timeout)
}

/**
* Checkpoint hace un checkpoint ahora, sin repartirlo en el tiempo.
* @return error: ErrClosed, ErrFailed
**/
func (e *Engine) Checkpoint() error {
	return e.checkpointNow()
}

/**
* Backup copia el motor en caliente al directorio dir (vacío o inexistente), como pg_basebackup: las escrituras, los
* lotes y el DDL siguen mientras copia. Hace un checkpoint (el inicio), copia el catálogo, los archivos de datos y
* los de estados, escribe un registro de fin de respaldo en el WAL y copia el WAL desde el inicio hasta ese registro.
* Open sobre dir restaura la copia: rehace ese WAL y la deja como estaba el motor en el fin del respaldo.
* @param dir string
* @return BackupInfo, error: ErrBackupDir, ErrDiskFull (en el destino o en el WAL; el motor sigue), ErrClosed,
* ErrFailed
**/
func (e *Engine) Backup(dir string) (BackupInfo, error) {
	return e.backup(vfs.OS(), dir)
}

/**
* BackupTo es Backup hacia un flujo (un tar, comprimido o cifrado, o una subida a S3): entrega a sink cada archivo
* de la copia con su largo, sin directorio intermedio. Restaurar es dejar los archivos en un directorio y abrirlo.
* El marcador backup_label sale dos veces, «en curso» al principio y el final al terminar: el segundo pisa al primero
* (en un tar, al extraer), así una copia cortada no se puede abrir.
* @param sink BackupSink
* @return BackupInfo, error: los de Backup y los de sink (el motor sigue)
**/
func (e *Engine) BackupTo(sink BackupSink) (BackupInfo, error) {
	return e.backupTo(sink)
}

/**
* PendingArchive lista los segmentos sellados del WAL que el archivo continuo (WAL_ARCHIVE=on) retiene hasta que el
* archivador los copie, del más viejo al más nuevo y hasta 256 (se vuelve a pedir). Copiar cada uno (sus Size bytes
* alcanzan) y después llamar a SetArchived con su número, en orden.
* @return []WALSegment, error: ErrArchiveOff, ErrClosed, ErrFailed
**/
func (e *Engine) PendingArchive() ([]WALSegment, error) {
	return e.pendingArchive()
}

/**
* SetArchived anota que los segmentos hasta seg (inclusive) ya están archivados: desde el próximo checkpoint se
* pueden reciclar. Es durable al volver (sobrevive a Close/Open y a un corte); repetirlo no hace nada.
* @param seg uint64: WALSegment.Segment
* @return error: ErrArchiveOff, ErrNotSealed, ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) SetArchived(seg uint64) error {
	return e.setArchived(seg)
}

/**
* SwitchWAL sella el segmento activo del WAL aunque no esté lleno (pg_switch_wal): pasa a PendingArchive enseguida.
* Funciona también con el archivo apagado.
* @return uint64: el segmento sellado (0 si el activo no tenía registros), error: ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) SwitchWAL() (uint64, error) {
	return e.switchWAL()
}

/**
* WaitDurable espera a que todo lo confirmado antes de llamar (en cualquier store, diferido o no) esté en disco: un
* fsync del WAL, compartido con las demás confirmaciones que esperan (group commit).
* @return error: ErrClosed, ErrFailed
**/
func (e *Engine) WaitDurable() error {
	return e.waitDurable()
}

/**
* Vacuum limpia todos los stores, uno a la vez: quita las versiones que ningún lote ni snapshot abierto puede ver y
* libera sus cadenas de desborde. Puede correr con lecturas y escrituras.
* @return error: el del primer store que falle, ErrClosed, ErrFailed
**/
func (e *Engine) Vacuum() error {
	return e.vacuum()
}

/**
* Verify revisa el motor sin cambiarlo, con el motor abierto: el control, el catálogo, los archivos de datos sin
* store y, de a un store (frenando solo las operaciones de ese store mientras dura), su árbol, sus cadenas de
* desborde, la lista de libres, el mapa de visibilidad, el congelado y Count (ver «Verificación y reparación» en
* doc.go).
* @return Report: los totales y los hallazgos, error: ErrClosed, ErrFailed (el daño va en el informe, no acá)
**/
func (e *Engine) Verify() (Report, error) {
	return e.check(false)
}

/**
* Repair revisa como Verify y arregla, con WAL (seguro ante un corte), lo que se puede sin perder nada sano: borra los
* archivos de datos huérfanos, devuelve a la lista de libres las páginas sin dueño y las de cadenas huérfanas, borra
* los bits del mapa de visibilidad mal puestos y corrige Count. Lo irreparable (una hoja ilegible, una cadena viva
* rota…) queda en el informe sin tocar.
* @return Report: los hallazgos, con Repaired en los arreglados, error: ErrClosed, ErrFailed
**/
func (e *Engine) Repair() (Report, error) {
	return e.check(true)
}

// ---------------------------------------------------------------------------------------------------------------
// Store: operaciones sueltas (cada una es su propia transacción y confirma al volver)
// ---------------------------------------------------------------------------------------------------------------

/**
* Name devuelve el nombre del store.
* @return string
**/
func (s *Store) Name() string {
	return s.name
}

/**
* Put guarda id con data: lo crea o lo reemplaza.
* @param id string: de 1 a 2.000 bytes, data []byte: hasta 64 MiB
* @return error
**/
func (s *Store) Put(id string, data []byte) error {
	return s.put(id, data, false)
}

/**
* Remove borra id sin preguntar (si no existe no escribe nada).
* @param id string
* @return error
**/
func (s *Store) Remove(id string) error {
	return s.remove(id, false)
}

/**
* Insert guarda id solo si no existe.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) Insert(id string, data []byte) (bool, error) {
	return s.insert(id, data, false)
}

/**
* Update reemplaza id solo si existe y el valor cambió.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) Update(id string, data []byte) (bool, error) {
	return s.update(id, data, false)
}

/**
* Delete borra id solo si existe.
* @param id string
* @return bool: si borró, error
**/
func (s *Store) Delete(id string) (bool, error) {
	return s.delete(id, false)
}

/**
* Read devuelve la versión más nueva confirmada de id. El slice devuelto es del llamador.
* @param id string
* @return []byte, bool: si existe, error
**/
func (s *Store) Read(id string) ([]byte, bool, error) {
	return s.read(id)
}

/**
* GetVersion es Read con el número de versión de id: el lsn del registro que escribió la versión devuelta. Cambia con
* cada escritura que aplica (no con un Update sin cambios) y nunca baja; sirve para concurrencia optimista.
* @param id string
* @return []byte: del llamador, uint64: la versión (0 si no existe), bool: si existe, error
**/
func (s *Store) GetVersion(id string) ([]byte, uint64, bool, error) {
	return s.getVersion(id)
}

/**
* Count devuelve la cantidad de ids vivos del store según su contador, en O(1) (cuenta también las escrituras que
* están confirmándose). CountAt cuenta lo que ve un snapshot.
* @return int64
**/
func (s *Store) Count() int64 {
	return s.count()
}

/**
* Exists dice si id existe según lo último confirmado, sin leer ni copiar su valor.
* @param id string
* @return bool, error
**/
func (s *Store) Exists(id string) (bool, error) {
	return s.exists(id)
}

/**
* Empty vacía el store en O(1), como TRUNCATE: un archivo de datos nuevo y el viejo borrado. No es MVCC: con lotes o
* snapshots abiertos en el motor devuelve ErrBusy; espera a las operaciones en curso del store.
* @return error: ErrBusy, ErrStoreDropped, ErrClosed, ErrFailed
**/
func (s *Store) Empty() error {
	return s.empty()
}

// ---------------------------------------------------------------------------------------------------------------
// Store: escrituras diferidas (confirman sin esperar el fsync del WAL; Engine.WaitDurable las hace durables)
// ---------------------------------------------------------------------------------------------------------------

/**
* Deferred da las escrituras sueltas del store que confirman sin esperar el fsync del WAL: se ven enseguida y un corte
* puede perderlas enteras (nunca a medias) hasta que Engine.WaitDurable vuelva.
* @return Deferred
**/
func (s *Store) Deferred() Deferred {
	return Deferred{s: s}
}

/**
* Put es Store.Put sin esperar el fsync.
* @param id string, data []byte
* @return error
**/
func (d Deferred) Put(id string, data []byte) error {
	return d.s.put(id, data, true)
}

/**
* Remove es Store.Remove sin esperar el fsync.
* @param id string
* @return error
**/
func (d Deferred) Remove(id string) error {
	return d.s.remove(id, true)
}

/**
* Insert es Store.Insert sin esperar el fsync.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (d Deferred) Insert(id string, data []byte) (bool, error) {
	return d.s.insert(id, data, true)
}

/**
* Update es Store.Update sin esperar el fsync.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (d Deferred) Update(id string, data []byte) (bool, error) {
	return d.s.update(id, data, true)
}

/**
* Delete es Store.Delete sin esperar el fsync.
* @param id string
* @return bool: si borró, error
**/
func (d Deferred) Delete(id string) (bool, error) {
	return d.s.delete(id, true)
}

// ---------------------------------------------------------------------------------------------------------------
// Store: recorridos sueltos (cada uno toma un snapshot propio y lo suelta al terminar, también ante error o pánico
// de fn; lo que ve no cambia mientras recorre). fn no tiene nada tomado mientras corre: puede leer y escribir en el
// motor, y el recorrido no ve lo escrito después de su snapshot.
// ---------------------------------------------------------------------------------------------------------------

/**
* Keys devuelve los ids vivos en orden.
* @param asc bool, offset int: ids que saltea, limit int: ≤ 0 = sin límite
* @return []string, error
**/
func (s *Store) Keys(asc bool, offset, limit int) ([]string, error) {
	return s.keys(scanRange{asc: asc, offset: offset, limit: limit})
}

/**
* KeysAfter devuelve hasta limit ids mayores que after, en orden ascendente, sin recorrer los anteriores: sirve para
* paginar (el último de una página es el after de la siguiente).
* @param after string: "" = desde el primero, limit int: ≤ 0 = sin límite
* @return []string, error
**/
func (s *Store) KeysAfter(after string, limit int) ([]string, error) {
	return s.keysAfter(after, limit)
}

/**
* KeysPrefix devuelve los ids vivos que empiezan con prefix, en orden.
* @param prefix string, asc bool, offset, limit int: ≤ 0 = sin límite
* @return []string, error
**/
func (s *Store) KeysPrefix(prefix string, asc bool, offset, limit int) ([]string, error) {
	return s.keys(prefixRange(prefix, asc, offset, limit))
}

/**
* ForEach recorre los ids vivos en orden con su valor (una copia, del llamador).
* @param fn func(id string, data []byte) (bool, error): false para parar; su error corta y se devuelve
* @param asc bool, offset int: ids que saltea, limit int: ≤ 0 = sin límite
* @return error
**/
func (s *Store) ForEach(fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEach(scanRange{asc: asc, offset: offset, limit: limit}, scanValues, fn)
}

/**
* ForEachPrefix es ForEach sobre los ids que empiezan con prefix.
* @param prefix string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Store) ForEachPrefix(prefix string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEach(prefixRange(prefix, asc, offset, limit), scanValues, fn)
}

/**
* ForEachRange es ForEach sobre los ids de [from, to).
* @param from string: incluido ("" = desde el primero), to string: excluido ("" = hasta el último)
* @param fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Store) ForEachRange(from, to string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEach(scanRange{from: from, to: to, asc: asc, offset: offset, limit: limit}, scanValues, fn)
}

/**
* ForEachBorrowed recorre los ids vivos en orden ascendente sin una copia por valor: data es un búfer del recorrido
* que se reutiliza y vale solo dentro de fn (copiarlo si hace falta después).
* @param fn func(id string, data []byte) (bool, error): false para parar
* @return error
**/
func (s *Store) ForEachBorrowed(fn func(id string, data []byte) (bool, error)) error {
	return s.forEach(scanRange{asc: true}, scanBorrowed, fn)
}

// ---------------------------------------------------------------------------------------------------------------
// Lote
// ---------------------------------------------------------------------------------------------------------------

/**
* Commit confirma el lote: todo o nada, también tras una caída, con un solo fsync (SYNC_ON_WRITE). Si el lote había
* abortado solo (serialización, conflicto o tiempo), lo deshace y devuelve ese error.
* @return error: ErrTxDone si ya terminó, ErrSerialization, ErrConflict, ErrTimeout, ErrClosed, ErrFailed
**/
func (b *Batch) Commit() error {
	return b.commit(false)
}

/**
* CommitLSN es el lsn del KindCommit del lote tras un Commit (o CommitDeferred) que escribió: el TxInfo.CommitLSN con
* que sale en el flujo de cambios. 0 antes de confirmar, si no escribió o si se deshizo.
* @return uint64
**/
func (b *Batch) CommitLSN() uint64 {
	return b.commitLSNOf()
}

/**
* SetOrigin marca el lote con un origen (el nodo de donde vino lo que escribe, en una réplica lógica): va en su
* KindCommit y sale en TxInfo.Origin del flujo de cambios, para que quien replica no vuelva a publicar lo que aplicó
* (como replication origins de PostgreSQL). 0 = sin origen (escrito en este motor). Se llama antes de Commit.
* @param origin uint16
* @return error: ErrTxDone si el lote ya terminó, o por qué abortó
**/
func (b *Batch) SetOrigin(origin uint16) error {
	return b.setOrigin(origin)
}

/**
* CommitDeferred confirma el lote como Commit pero sin esperar el fsync del WAL: se ve enseguida y un corte puede
* perderlo entero (nunca a medias) hasta que Engine.WaitDurable vuelva.
* @return error: los de Commit
**/
func (b *Batch) CommitDeferred() error {
	return b.commit(true)
}

/**
* Rollback deshace el lote sin dejar rastro visible. Sirve también tras un error que abortó el lote.
* @return error: ErrTxDone si ya se confirmó o se deshizo
**/
func (b *Batch) Rollback() error {
	return b.rollback()
}

// ---------------------------------------------------------------------------------------------------------------
// Store: dentro de un lote (ven la foto del lote más sus propios cambios)
// ---------------------------------------------------------------------------------------------------------------

/**
* PutIn guarda id con data dentro del lote tx.
* @param tx *Batch, id string, data []byte
* @return error: ErrSerialization si otra transacción confirmó un cambio de id después de que tx empezó
**/
func (s *Store) PutIn(tx *Batch, id string, data []byte) error {
	return s.putIn(tx, id, data)
}

/**
* RemoveIn borra id dentro del lote tx sin preguntar.
* @param tx *Batch, id string
* @return error
**/
func (s *Store) RemoveIn(tx *Batch, id string) error {
	return s.removeIn(tx, id)
}

/**
* InsertIn guarda id dentro del lote tx solo si no existe para él.
* @param tx *Batch, id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) InsertIn(tx *Batch, id string, data []byte) (bool, error) {
	return s.insertIn(tx, id, data)
}

/**
* UpdateIn reemplaza id dentro del lote tx solo si existe para él y el valor cambió.
* @param tx *Batch, id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) UpdateIn(tx *Batch, id string, data []byte) (bool, error) {
	return s.updateIn(tx, id, data)
}

/**
* DeleteIn borra id dentro del lote tx solo si existe para él.
* @param tx *Batch, id string
* @return bool: si borró, error
**/
func (s *Store) DeleteIn(tx *Batch, id string) (bool, error) {
	return s.deleteIn(tx, id)
}

/**
* ExistsIn dice si id existe para el lote tx (su propio cambio, o la foto tomada al crearlo), sin leer su valor.
* @param tx *Batch, id string
* @return bool, error
**/
func (s *Store) ExistsIn(tx *Batch, id string) (bool, error) {
	return s.existsIn(tx, id)
}

/**
* GetVersionIn es ReadIn con la versión de id (ver GetVersion); la de un cambio propio del lote es la que tendrá al
* confirmar.
* @param tx *Batch, id string
* @return []byte, uint64: 0 si no existe, bool, error
**/
func (s *Store) GetVersionIn(tx *Batch, id string) ([]byte, uint64, bool, error) {
	return s.getVersionIn(tx, id)
}

/**
* ReadIn devuelve id como lo ve el lote tx: su propio cambio, o la foto tomada al crearlo.
* @param tx *Batch, id string
* @return []byte: del llamador, bool: si existe, error
**/
func (s *Store) ReadIn(tx *Batch, id string) ([]byte, bool, error) {
	cur, found, err := s.readIn(tx, id)
	return cur.value, found, err
}

// ---------------------------------------------------------------------------------------------------------------
// Snapshot
// ---------------------------------------------------------------------------------------------------------------

/**
* Close cierra el snapshot y deja de retener el horizonte de la limpieza. Cerrarlo dos veces no hace nada.
* @return error
**/
func (v *Snapshot) Close() error {
	return v.close()
}

/**
* ReadAt devuelve id como lo ve el snapshot v.
* @param v *Snapshot, id string
* @return []byte: del llamador, bool: si existe, error: ErrTxDone si se cerró, ErrTimeout si pasó su tiempo
**/
func (s *Store) ReadAt(v *Snapshot, id string) ([]byte, bool, error) {
	return s.readAt(v, id)
}

/**
* GetVersionAt es ReadAt con la versión de id (ver GetVersion).
* @param v *Snapshot, id string
* @return []byte, uint64: 0 si no existe, bool, error: ErrTxDone si se cerró, ErrTimeout si pasó su tiempo
**/
func (s *Store) GetVersionAt(v *Snapshot, id string) ([]byte, uint64, bool, error) {
	return s.getVersionAt(v, id)
}

/**
* ExistsAt dice si id existe para el snapshot v, sin leer su valor.
* @param v *Snapshot, id string
* @return bool, error: ErrTxDone si se cerró, ErrTimeout si pasó su tiempo
**/
func (s *Store) ExistsAt(v *Snapshot, id string) (bool, error) {
	return s.existsAt(v, id)
}

/**
* ForEachAt recorre los ids que ve el snapshot v, en orden, con su valor (una copia, del llamador). Las lápidas no
* se ven; las versiones viejas que v todavía ve sí, aunque haya otras más nuevas.
* @param v *Snapshot, fn func(id string, data []byte) (bool, error): false para parar
* @param asc bool, offset int: ids que saltea, limit int: ≤ 0 = sin límite
* @return error: el de fn, ErrTxDone o ErrTimeout si v termina durante el recorrido
**/
func (s *Store) ForEachAt(v *Snapshot, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEachAt(v, scanRange{asc: asc, offset: offset, limit: limit}, fn)
}

/**
* ForEachPrefixAt es ForEachAt sobre los ids que empiezan con prefix.
* @param v *Snapshot, prefix string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Store) ForEachPrefixAt(v *Snapshot, prefix string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEachAt(v, prefixRange(prefix, asc, offset, limit), fn)
}

/**
* ForEachRangeAt es ForEachAt sobre los ids de [from, to) ("" = sin límite de ese lado).
* @param v *Snapshot, from, to string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Store) ForEachRangeAt(v *Snapshot, from, to string, fn func(id string, data []byte) (bool, error), asc bool, offset, limit int) error {
	return s.forEachAt(v, scanRange{from: from, to: to, asc: asc, offset: offset, limit: limit}, fn)
}

/**
* ForEachBorrowedAt es ForEachBorrowed con el snapshot v: data vale solo dentro de fn.
* @param v *Snapshot, fn func(id string, data []byte) (bool, error)
* @return error
**/
func (s *Store) ForEachBorrowedAt(v *Snapshot, fn func(id string, data []byte) (bool, error)) error {
	return s.scanAt(v, scanRange{asc: true}, scanBorrowed, fn)
}

/**
* CountAt cuenta los ids que ve el snapshot v. Recorre el store entero (O(n), sin leer valores): para el contador en
* O(1), Count.
* @param v *Snapshot
* @return int64, error
**/
func (s *Store) CountAt(v *Snapshot) (int64, error) {
	return s.countAt(v)
}

// ---------------------------------------------------------------------------------------------------------------
// Estadísticas
// ---------------------------------------------------------------------------------------------------------------

/**
* Stats devuelve una foto barata del motor: caché, WAL y checkpoints, versiones por limpiar, transacciones abiertas,
* contadores por operación y latencias (p50, p99). No recorre el árbol ni frena a nadie.
* @return Stats
**/
func (e *Engine) Stats() Stats {
	return e.stats()
}

/**
* OnStats llama a fn con Stats cada interval, sin ningún candado tomado, hasta stop o Close. Close espera a la
* llamada en curso y después no hay más; fn no debe llamar a Close.
* @param interval time.Duration: 0 o menos = 1 s, fn func(Stats)
* @return func(): stop (se puede llamar varias veces, también desde fn)
**/
func (e *Engine) OnStats(interval time.Duration, fn func(Stats)) func() {
	return e.onStats(interval, fn)
}

// ---------------------------------------------------------------------------------------------------------------
// Presión y uso de disco
// ---------------------------------------------------------------------------------------------------------------

/**
* Pressure dice qué tan cerca está el motor de frenar a los escritores, como pebble.Pressure de josefina: el máximo
* entre el WAL sin reciclar y el tope de throttle, los registros sin fsync y el de las diferidas, y las páginas sucias
* y la caché; 1 con el motor de solo lectura o inutilizable. Barato: solo atómicos. Stats trae también la causa.
* @return float64: 0 = sin presión, ≥ 1 = ya frena
**/
func (e *Engine) Pressure() float64 {
	value, _ := e.pressure()
	return value
}

/**
* DirUsage mide lo que ocupa el motor en disco (datos, WAL con lo preasignado, archivos de estados, control y
* catálogo) y cada store, con el largo de los archivos, sin leer páginas.
* @return DiskUsage, error: ErrClosed
**/
func (e *Engine) DirUsage() (DiskUsage, error) {
	return e.dirUsage()
}

/**
* DirUsage son los bytes en disco del store (su árbol con el desborde y su mapa de visibilidad), sin leer páginas.
* @return int64, error: ErrClosed, ErrStoreDropped
**/
func (s *Store) DirUsage() (int64, error) {
	return s.dirUsage()
}

// ---------------------------------------------------------------------------------------------------------------
// Replicación (tarea 46): réplica física, ranuras de retención y flujo lógico de cambios
// ---------------------------------------------------------------------------------------------------------------

/**
* OpenReplica abre como réplica una copia de Backup del primario (o una réplica ya abierta antes): recupera como Open
* pero no deshace lo que sigue en curso en el primario, y queda de solo lectura para el usuario (las escrituras,
* NewBatch, Vacuum, Repair y Backup dan ErrReadOnly) aplicando lo que llega con ApplyWal. Open sobre el mismo
* directorio la promueve a primario.
* @param dir string
* @return *Engine, error: ErrNotReplica (directorio sin motor), los de Open
**/
func OpenReplica(dir string) (*Engine, error) {
	cfg := envConfig()
	cfg.replica = true
	return open(vfs.OS(), dir, cfg)
}

/**
* OpenReplicaWith es OpenReplica con opciones propias (Options).
* @param dir string, o Options
* @return *Engine, error: los de OpenReplica
**/
func OpenReplicaWith(dir string, o Options) (*Engine, error) {
	return openWith(dir, o, true)
}

/**
* Promote vuelve primario a la réplica sin cerrarla (failover): deja de aplicar WAL (ApplyWal da ErrNotReplica), lo
* que seguía en curso en el primario queda abortado, rota el WAL (lo que escribe como primario va a un segmento nuevo
* del formato actual; con WAL_ARCHIVE=on arranca el archivo) y hace el checkpoint de fin de recuperación. Desde ahí
* acepta escrituras; las lecturas y snapshots abiertos siguen. Es lo mismo que cerrarla y abrirla con Open, sin cortar.
* @return error: ErrNotReplica (no es una réplica, o ya la promovieron), ErrDiskFull (sigue siendo réplica), ErrClosed,
* ErrFailed
**/
func (e *Engine) Promote() error {
	return e.promote()
}

/**
* WalSince abre un lector del WAL desde el registro siguiente a lsn (el ReplayLSN de la réplica): Next entrega los
* registros ya durables de a tandas. No retiene el WAL: para eso, una ranura.
* @param lsn uint64
* @return *WalReader, error: ErrClosed, ErrFailed
**/
func (e *Engine) WalSince(lsn uint64) (*WalReader, error) {
	return e.walSince(lsn)
}

/**
* Next devuelve los registros siguientes ya durables, hasta maxBytes (al menos uno si hay); ninguno si no hay nada
* nuevo todavía (se vuelve a pedir).
* @param maxBytes int: ≤ 0 = 1 MiB
* @return []WalEntry, error: ErrWalRemoved (el WAL desde ahí se recicló: la réplica necesita un Backup nuevo)
**/
func (r *WalReader) Next(maxBytes int) ([]WalEntry, error) {
	return r.next(maxBytes)
}

/**
* ApplyWal aplica en la réplica los registros del primario, en orden: salta los ya aplicados y no aplica nada desde
* el primero que no sigue. No espera el fsync (WaitDurable lo hace).
* @param entries []WalEntry
* @return error: ErrNotReplica, ErrWalGap, ErrCorrupt, ErrClosed, ErrFailed
**/
func (e *Engine) ApplyWal(entries []WalEntry) error {
	return e.applyWal(entries)
}

/**
* ApplyWalEntry es ApplyWal de un solo registro.
* @param entry WalEntry
* @return error
**/
func (e *Engine) ApplyWalEntry(entry WalEntry) error {
	return e.applyWal([]WalEntry{entry})
}

/**
* ReplayLSN es el último lsn que aplicó la réplica (0 en un primario): desde ahí pide WalSince y, tras WaitDurable,
* lo que confirma con AdvanceSlot en el primario.
* @return uint64
**/
func (e *Engine) ReplayLSN() uint64 {
	return e.replayLSN()
}

/**
* CreateSlot crea una ranura de retención para una réplica física: el WAL desde el próximo registro no se recicla
* hasta que AdvanceSlot lo confirme, o hasta pasar MAX_SLOT_WAL_KEEP_SIZE (la ranura se invalida). Crearla antes del
* Backup de la réplica.
* @param name string: de 1 a 64 bytes
* @return SlotInfo, error: ErrSlotExists, ErrSlotLimit, ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) CreateSlot(name string) (SlotInfo, error) {
	return e.createSlot(name, false)
}

/**
* CreateLogicalSlot crea una ranura para un consumidor lógico (Changes): espera a que terminen las transacciones en
* curso (hasta TRANSACTION_TIMEOUT) y entrega lo confirmado después.
* @param name string
* @return SlotInfo, error: los de CreateSlot, ErrBusy (no terminaron), ErrReadOnly (en una réplica)
**/
func (e *Engine) CreateLogicalSlot(name string) (SlotInfo, error) {
	return e.createSlot(name, true)
}

/**
* DropSlot borra una ranura: el próximo checkpoint recicla lo que retenía.
* @param name string
* @return error: ErrSlotNotFound, ErrDiskFull, ErrClosed, ErrFailed
**/
func (e *Engine) DropSlot(name string) error {
	return e.dropSlot(name)
}

/**
* Slots describe las ranuras, por nombre: lsn confirmado y retenido, atraso, WAL retenido y si se invalidó.
* @return []SlotInfo
**/
func (e *Engine) Slots() []SlotInfo {
	return e.slotsInfo()
}

/**
* AdvanceSlot anota que la réplica de la ranura ya tiene durable todo hasta lsn: el WAL anterior se puede reciclar.
* Se guarda en el próximo checkpoint (tras un corte del primario puede volver atrás hasta ese valor: la réplica salta
* lo repetido). Nunca retrocede.
* @param name string, lsn uint64
* @return error: ErrSlotNotFound, ErrWalRemoved (invalidada), ErrWalGap (lsn después del WAL), ErrClosed, ErrFailed
**/
func (e *Engine) AdvanceSlot(name string, lsn uint64) error {
	return e.advanceSlot(name, page.LSN(lsn), 0, false)
}

/**
* Changes abre un lector del flujo lógico de la ranura lógica name (CreateLogicalSlot): entrega por transacción
* confirmada y en orden de confirmación lo confirmado después de su ConfirmedLSN.
* @param name string
* @return *ChangeReader, error: ErrSlotNotFound, ErrWalRemoved (invalidada), ErrClosed, ErrFailed
**/
func (e *Engine) Changes(name string) (*ChangeReader, error) {
	return e.changes(name)
}

/**
* ChangesWith es Changes con un filtro: solo salen los cambios que f deja pasar (por store, por id y por origen).
* @param name string, f ChangeFilter
* @return *ChangeReader, error: los de Changes
**/
func (e *Engine) ChangesWith(name string, f ChangeFilter) (*ChangeReader, error) {
	return e.changesWith(name, f)
}

/**
* ChangesFrom abre un lector sin ranura desde pos (de ChangeHead o de ChangeReader.Pos, guardada por quien lee): no
* retiene WAL, así que si el checkpoint ya recicló pos.Restart, Next da ErrWalRemoved. Sirve para eventos en vivo y
* para copiar a otra base, y anda también en una réplica. Confirm no corre en él (ErrSlotNotFound): se guarda Pos.
* @param pos ChangePos, f ChangeFilter
* @return *ChangeReader, error: ErrClosed, ErrFailed
**/
func (e *Engine) ChangesFrom(pos ChangePos, f ChangeFilter) (*ChangeReader, error) {
	return e.changesFrom(pos, f)
}

/**
* ChangeHead es la posición actual del flujo, sin esperar a las transacciones en curso: un lector desde ella entrega
* lo que confirme después (también lo de los lotes que ya estaban escribiendo).
* @return ChangePos, error: ErrClosed, ErrFailed
**/
func (e *Engine) ChangeHead() (ChangePos, error) {
	return e.changeHead()
}

/**
* AdvanceLogicalSlot avanza la ranura lógica name hasta pos sin abrir su lector (quien reparte el flujo guarda la
* posición de cada consumidor y avanza la ranura al mínimo). Nunca retrocede.
* @param name string, pos ChangePos
* @return error: los de AdvanceSlot
**/
func (e *Engine) AdvanceLogicalSlot(name string, pos ChangePos) error {
	return e.advanceLogical(name, pos)
}

/**
* Pos es la posición del lector tras lo entregado: ChangesFrom(Pos(), …) sigue sin repetir ni saltar nada, con el
* mismo filtro. Se pide entre llamadas a Next.
* @return ChangePos
**/
func (r *ChangeReader) Pos() ChangePos {
	return r.position()
}

/**
* Next lee el WAL durable y llama a fn con cada cambio de cada transacción que confirma, en orden de confirmación,
* hasta maxTx transacciones (≤ 0 = hasta alcanzar el WAL). Las abortadas o sin confirmar no salen; una transacción
* grande pasa a disco mientras espera su confirmación. fn no debe cerrar el motor; si devuelve un error, el lector
* queda inservible (se abre otro con Changes y sigue desde la ranura).
* @param maxTx int, fn func(tx TxInfo, c Change) error
* @return int: transacciones entregadas, error: el de fn, ErrWalRemoved, ErrClosed
**/
func (r *ChangeReader) Next(maxTx int, fn func(tx TxInfo, c Change) error) (int, error) {
	return r.next(maxTx, fn)
}

/**
* Confirm anota en la ranura que todo lo entregado ya se procesó: el próximo lector de la ranura sigue desde ahí, sin
* repetir ni saltar nada (tras un corte del motor puede repetir lo posterior a su último checkpoint: TxInfo.CommitLSN
* sirve para descartarlo).
* @return error: los de AdvanceSlot
**/
func (r *ChangeReader) Confirm() error {
	return r.confirm()
}

/**
* Close suelta el búfer del lector y borra sus archivos de derrame.
* @return error
**/
func (r *ChangeReader) Close() error {
	return r.close()
}
