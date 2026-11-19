package store

import (
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Catálogo de la API pública (fases 1A y 1B): envolturas delgadas; la implementación vive en los demás archivos del
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
* Vacuum limpia todos los stores, uno a la vez: quita las versiones que ningún lote ni snapshot abierto puede ver y
* libera sus cadenas de desborde. Puede correr con lecturas y escrituras.
* @return error: el del primer store que falle, ErrClosed, ErrFailed
**/
func (e *Engine) Vacuum() error {
	return e.vacuum()
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
	return s.put(id, data)
}

/**
* Remove borra id sin preguntar (si no existe no escribe nada).
* @param id string
* @return error
**/
func (s *Store) Remove(id string) error {
	return s.remove(id)
}

/**
* Insert guarda id solo si no existe.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) Insert(id string, data []byte) (bool, error) {
	return s.insert(id, data)
}

/**
* Update reemplaza id solo si existe y el valor cambió.
* @param id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) Update(id string, data []byte) (bool, error) {
	return s.update(id, data)
}

/**
* Delete borra id solo si existe.
* @param id string
* @return bool: si borró, error
**/
func (s *Store) Delete(id string) (bool, error) {
	return s.delete(id)
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
* Count devuelve la cantidad de ids vivos del store (adelanto de la fase 2: cuenta también las escrituras que están
* confirmándose).
* @return int64
**/
func (s *Store) Count() int64 {
	return s.count()
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
	return b.commit()
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
