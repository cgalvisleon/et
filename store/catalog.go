package store

import (
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Catálogo de la API pública (fases 1A, 1B, 2 y 3): envolturas delgadas; la implementación vive en los demás archivos del
// paquete con el mismo nombre en minúscula. Ver doc.go para la semántica de cada método.

// ---------------------------------------------------------------------------------------------------------------
// Motor
// ---------------------------------------------------------------------------------------------------------------

/**
* Open abre el motor del directorio dir (lo crea si no existe): toma LOCK y recupera desde el último checkpoint.
* @param dir string
* @return *Engine, error: ErrLocked si otro motor tiene el directorio, ErrCorrupt
**/
func Open(dir string) (*Engine, error) {
	return open(vfs.OS(), dir, envConfig())
}

/**
* Close espera a las operaciones en curso, hace el checkpoint final y cierra. Cerrar dos veces no hace nada.
* @return error: el fallo que dejó el motor inutilizable, si lo hubo
**/
func (e *Engine) Close() error {
	return e.close()
}

/**
* Store abre el store name, o lo crea si no existe.
* @param name string: UTF-8 válido sin nulos, de 1 a 255 bytes
* @return *Store, error: ErrBadName, ErrClosed, ErrFailed
**/
func (e *Engine) Store(name string) (*Store, error) {
	return e.store(name)
}

/**
* DropStore borra el store name con todos sus datos.
* @param name string
* @return error: ErrStoreNotFound, ErrClosed, ErrFailed
**/
func (e *Engine) DropStore(name string) error {
	return e.dropStore(name)
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
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	return e.checkpoint(control.StateInProduction, false)
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
