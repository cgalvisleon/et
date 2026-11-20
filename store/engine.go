package store

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// lockName es el archivo de bloqueo del directorio del motor.
const lockName = "LOCK"

// Engine es un motor abierto: un directorio con su caché, su WAL y sus stores. No hay estado global: un proceso
// puede abrir varios motores (en directorios distintos).
//
// Orden de los locks del motor: closeMu (lectura) → ddlMu → txGate → Store.mu; closeMu (lectura) → Store.mu (lectura) →
// commitMu (lectura) → páginas → writeMu del WAL → counts.mu; páginas → hintGate (lectura); ckptMu → hintGate
// (escritura) → writeMu; ckptMu → commitMu (escritura); ckptMu → ddlMu; Store.vacuumMu → closeMu (lectura) →
// Store.mu (lectura) → páginas; closeMu (lectura) → txGate (lectura); closeMu (lectura) → ckptMu → ddlMu y
// Store.vacuumMu → closeMu (lectura) → Store.mu → commitMu (escritura) (Verify y Repair). failMu, counts.mu y el
// catálogo en memoria son hojas.
type Engine struct {
	dir string
	fs  vfs.FS
	cfg config

	lock       io.Closer
	ctl        control.ControlFile
	catFile    control.CatalogFile
	hasControl bool            // false en un motor nuevo hasta su primer checkpoint (que crea el control)
	cat        *control.Stores // el catálogo como se guarda: contadores a cat.CountLSN()
	counts     *counts         // los contadores vivos

	sm     smgr.Manager
	proxy  *walProxy
	pool   buffer.Pool
	status *xact.Statuses
	log    *wal.WAL
	txs    *xact.Transactions
	free   *fsm.Manager
	chains *overflow.Manager

	// closeMu: las operaciones lo toman en lectura; Close en escritura (espera a las que están en curso).
	closeMu sync.RWMutex
	closed  bool

	failMu  sync.Mutex
	failErr error // el primer fallo de escritura o fsync: el motor queda inutilizable (de solo lectura si es de disco lleno)

	guard *syncGuard // el disco del motor: anota si algún fsync falló (ver space.go)

	// doneWAL es log.Written() en el punto de rehacer del último checkpoint terminado: Written() − doneWAL es el WAL
	// que todavía no se puede reciclar (lo que mira el freno de los escritores, ver space.go).
	doneWAL atomic.Int64

	// catalogBehind: un cambio de stores no pudo guardar el catálogo por disco lleno (storeCatalog); el guardado
	// quedó atrás del de memoria hasta el próximo guardado (el del checkpoint, a más tardar).
	catalogBehind atomic.Bool

	// sigMu y ckptDone: los escritores frenados esperan en ckptDone, que se cierra al terminar cada checkpoint.
	sigMu    sync.Mutex
	ckptDone chan struct{}

	// ddlMu serializa crear, abrir y borrar stores y cada guardado del catálogo.
	ddlMu   sync.Mutex
	handles map[string]*Store // stores abiertos, por nombre (crece con los stores, no con las claves)

	// ckptMu: un checkpoint a la vez.
	ckptMu sync.Mutex

	// commitMu: una operación lo tiene en lectura desde antes de reservar su registro hasta anotar su estado en el
	// archivo de estados; el checkpoint lo toma en escritura tras fijar el punto de rehacer, así ninguna
	// confirmación anterior al punto de rehacer queda fuera del archivo de estados que guarda (delayChkpt de
	// PostgreSQL).
	commitMu sync.RWMutex

	// hintGate es la compuerta de las marcas de visibilidad sin WAL (btree.Config.HintGate de todos los árboles): las
	// lecturas la toman en lectura para decidir y marcar; el paso 1 del checkpoint, en escritura alrededor de
	// BeginCheckpoint.
	hintGate sync.RWMutex

	// txGate y openTx: lotes y snapshots abiertos. newBatch y snapshotFor suman a openTx con txGate en lectura;
	// Empty toma txGate en escritura y exige openTx = 0, así ninguno empieza mientras vacía un store.
	txGate sync.RWMutex
	openTx atomic.Int64

	// writes cuenta las escrituras aplicadas (sueltas y de lotes): la limpieza de fondo mira cuántas hubo desde la
	// última.
	writes atomic.Int64

	lastCheckpoint atomic.Int64 // hora del último checkpoint (Unix en nanosegundos)

	// lastCommit es el mayor lsn de confirmación anotado (durable o diferido): WaitDurable espera el WAL hasta él. Se
	// sube antes de anotar confirmada en el archivo de estados, así lo visible siempre queda cubierto.
	lastCommit atomic.Uint64

	replayed int // registros rehechos por Open, sin contar los checkpoints (pruebas, regla 2)

	frozen frozenXIDs // el frozenXID de cada store leído y el del control al abrir (ver frozen.go)

	timers timers // los temporizadores de los lotes y snapshots abiertos: Close los detiene

	metrics engineMetrics // contadores y latencias de Stats (stats.go): tamaño fijo, solo atómicos

	stop chan struct{} // se cierra en Close: detiene el checkpointer, el escritor del WAL, la limpieza de fondo y OnStats
	bg   sync.WaitGroup

	// afterStep, si no es nil, corre al terminar cada paso del checkpoint (1 a 7, el 7 es el truncado del archivo de
	// estados). Solo para las pruebas: simula un corte entre dos pasos.
	afterStep func(step int)
}

/**
* open abre (o crea) el motor de dir sobre base (envuelto en un syncGuard): toma LOCK, recupera desde el último
* checkpoint y arranca el checkpointer y la limpieza de fondo (si VACUUM_INTERVAL no es 0).
* @param base vfs.FS, dir string, cfg config
* @return *Engine, error: ErrLocked si otro motor tiene el directorio
**/
func open(base vfs.FS, dir string, cfg config) (*Engine, error) {
	guard := &syncGuard{FS: base}
	fsys := vfs.FS(guard)
	if err := fsys.MkdirAll(dir); err != nil {
		return nil, fmt.Errorf(MSG_OPEN, dir, err)
	}
	lock, err := fsys.Lock(filepath.Join(dir, lockName))
	if errors.Is(err, vfs.ErrLocked) {
		return nil, fmt.Errorf(MSG_LOCKED, ErrLocked, dir, err)
	}
	if err != nil {
		return nil, fmt.Errorf(MSG_OPEN, dir, err)
	}
	e := &Engine{
		dir:     dir,
		fs:      fsys,
		guard:   guard,
		cfg:     cfg,
		lock:    lock,
		ctl:     control.NewControlFile(fsys, dir),
		catFile: control.NewCatalogFile(fsys, dir),
		counts:  newCounts(),
		proxy:   &walProxy{},
		handles: map[string]*Store{},
		stop:    make(chan struct{}),
		frozen:  frozenXIDs{files: map[smgr.FileNo]page.XID{}},
	}
	if err := e.recover(); err != nil {
		return nil, errors.Join(err, e.closeResources())
	}
	e.bg.Add(1)
	go e.checkpointer()
	if cfg.walWriterDelay > 0 {
		e.bg.Add(1)
		go e.walWriter()
	}
	if cfg.vacuumEvery > 0 {
		e.bg.Add(1)
		go e.vacuumer()
	}
	return e, nil
}

/**
* close detiene las operaciones nuevas, espera a las que están en curso (también la limpieza de un store), detiene el
* checkpointer, el escritor del WAL y la limpieza de fondo, hace el checkpoint final (state = cerrado bien) y cierra todo. Si el motor quedó inutilizable no escribe nada y devuelve ese fallo.
* Cerrar dos veces no hace nada.
* @return error
**/
func (e *Engine) close() error {
	e.closeMu.Lock()
	if e.closed {
		e.closeMu.Unlock()
		return nil
	}
	e.closed = true
	e.closeMu.Unlock()

	e.timers.closeAll()
	close(e.stop)
	e.bg.Wait()
	err := e.checkpoint(control.StateShutdown, false)
	return errors.Join(err, e.closeResources())
}

/**
* closeResources cierra las piezas abiertas (las nil se saltan), sin escribir nada.
* @return error
**/
func (e *Engine) closeResources() error {
	var errs []error
	if e.pool != nil {
		errs = append(errs, e.pool.Close())
	}
	if e.log != nil {
		errs = append(errs, e.log.Close())
	}
	if e.status != nil {
		errs = append(errs, e.status.Close())
	}
	if e.sm != nil {
		errs = append(errs, e.sm.Close())
	}
	if e.lock != nil {
		errs = append(errs, e.lock.Close())
	}
	return errors.Join(errs...)
}

/**
* enter deja pasar una operación que escribe si el motor está abierto y usable; quien entra llama a leave al
* terminar. Una caché que falló (al hacer lugar para una lectura, por ejemplo) deja el motor inutilizable aquí.
* @return error: ErrClosed o ErrFailed (con ErrDiskFull si el fallo fue de disco lleno)
**/
func (e *Engine) enter() error {
	e.closeMu.RLock()
	if e.closed {
		e.closeMu.RUnlock()
		return ErrClosed
	}
	if err := e.checkFailed(); err != nil {
		e.closeMu.RUnlock()
		return err
	}
	return nil
}

/**
* enterRead es enter para una operación que solo lee: también deja pasar con el motor de solo lectura (un fallo de
* disco lleno).
* @return error: ErrClosed o ErrFailed
**/
func (e *Engine) enterRead() error {
	e.closeMu.RLock()
	if e.closed {
		e.closeMu.RUnlock()
		return ErrClosed
	}
	if err := e.checkFailed(); err != nil && !e.readOnly() {
		e.closeMu.RUnlock()
		return err
	}
	return nil
}

/**
* checkFailed devuelve el fallo del motor; si la caché falló y el motor todavía no, lo deja inutilizable con ese
* fallo.
* @return error
**/
func (e *Engine) checkFailed() error {
	if e.pool != nil {
		if err := e.pool.Failed(); err != nil {
			return e.fail(err)
		}
	}
	return e.failed()
}

/**
* readOnly dice si el motor quedó de solo lectura: inutilizable por un fallo de disco lleno.
* @return bool
**/
func (e *Engine) readOnly() bool {
	e.failMu.Lock()
	defer e.failMu.Unlock()
	return e.failErr != nil && vfs.IsNoSpace(e.failErr)
}

/**
* leave termina una operación que pasó por enter.
**/
func (e *Engine) leave() {
	e.closeMu.RUnlock()
}

/**
* fail deja el motor inutilizable con err (si no lo estaba ya) y devuelve el error público.
* @param err error: la causa
* @return error: ErrFailed envolviendo el primer fallo
**/
func (e *Engine) fail(err error) error {
	e.failMu.Lock()
	if e.failErr == nil {
		e.failErr = err
	}
	e.failMu.Unlock()
	return e.failed()
}

/**
* failed devuelve ErrFailed con la causa si el motor quedó inutilizable, o nil. Si la causa es de disco lleno, el
* error es también ErrDiskFull.
* @return error
**/
func (e *Engine) failed() error {
	e.failMu.Lock()
	defer e.failMu.Unlock()
	err := e.failErr
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, ErrFailed):
		err = fmt.Errorf(MSG_FAILED, ErrFailed, err)
	}
	if vfs.IsNoSpace(err) && !errors.Is(err, ErrDiskFull) {
		err = fmt.Errorf(MSG_DISK_FULL, ErrDiskFull, err)
	}
	return err
}

/**
* readError trata el error de una lectura: si viene de una caché o un WAL que fallaron (una lectura puede tener que
* escribir una página sucia para hacerse lugar), el motor queda inutilizable; si no (una página dañada), se devuelve
* tal cual.
* @param err error
* @return error
**/
func (e *Engine) readError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, buffer.ErrFailed) || e.log.Failed() != nil {
		return e.fail(err)
	}
	return err
}

/**
* logRecord escribe un registro del motor sin bloques (crear o borrar un store, checkpoint) y espera su fsync.
* @param kind wal.Kind, main []byte
* @return page.LSN del registro, error
**/
func (e *Engine) logRecord(kind wal.Kind, main []byte) (page.LSN, error) {
	res, err := e.log.Reserve(&wal.Record{Kind: kind, Main: main}, nil)
	if err != nil {
		return 0, err
	}
	lsn := res.LSN()
	if err := res.Write(); err != nil {
		return 0, err
	}
	return lsn, e.log.WaitDurable(lsn)
}

/**
* openTxn anota un lote o snapshot abierto (Empty no corre mientras haya alguno). Con closeMu en lectura.
* @param kind *atomic.Int64: el contador de Stats (metrics.openBatches o metrics.openSnapshots)
**/
func (e *Engine) openTxn(kind *atomic.Int64) {
	e.txGate.RLock()
	e.openTx.Add(1)
	e.txGate.RUnlock()
	kind.Add(1)
}

/**
* closeTxn anota que un lote o snapshot terminó.
* @param kind *atomic.Int64: el mismo de openTxn
**/
func (e *Engine) closeTxn(kind *atomic.Int64) {
	kind.Add(-1)
	e.openTx.Add(-1)
}

/**
* writable dice si se puede empezar a escribir (con el motor ya tomado con enter o enterRead): no falló y el WAL
* tiene lugar preasignado (admit).
* @return error: ErrFailed o ErrDiskFull
**/
func (e *Engine) writable() error {
	if err := e.failed(); err != nil {
		return err
	}
	return e.admit()
}
