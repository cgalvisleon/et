package store

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Disco lleno y disco lento (ver «Disco lleno o lento» en doc.go).

const (
	// walBacklogFactor: los escritores esperan mientras el WAL desde el último checkpoint terminado pase de
	// walBacklogFactor × MAX_WAL_SIZE (constante).
	walBacklogFactor = 2

	// maxDeferredLag: una confirmación diferida (o con SYNC_ON_WRITE=false) espera un fsync cuando hay más de estos
	// registros del WAL sin fsync, así un disco lento no deja crecer sin tope lo que un corte puede perder (constante).
	maxDeferredLag = 1 << 13
)

// syncGuard envuelve el disco del motor y anota si algún fsync falló. Un disco lleno solo es «limpio» (la operación
// falla y el motor sigue) si ningún fsync falló: tras uno fallido nada de lo escrito sin fsync es seguro.
type syncGuard struct {
	vfs.FS
	failed atomic.Bool
}

// guardFile es un archivo abierto con syncGuard: su Sync anota el fallo.
type guardFile struct {
	vfs.File
	g *syncGuard
}

/**
* OpenFile abre el archivo del disco envuelto, para que su Sync quede anotado.
* @param name string, flag int
* @return vfs.File, error
**/
func (g *syncGuard) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := g.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return guardFile{File: f, g: g}, nil
}

/**
* SyncDir hace el fsync del directorio y anota si falló.
* @param dir string
* @return error
**/
func (g *syncGuard) SyncDir(dir string) error {
	return g.note(g.FS.SyncDir(dir))
}

/**
* Sync hace el fsync del archivo y anota si falló.
* @return error
**/
func (f guardFile) Sync() error {
	return f.g.note(f.File.Sync())
}

/**
* Flush lleva lo escrito al dispositivo (vfs.Flusher) y anota si falló: un error de escritura cuenta como un fsync
* fallido.
* @return error
**/
func (f guardFile) Flush() error {
	return f.g.note(vfs.FlushFile(f.File))
}

/**
* note anota un fsync fallido y devuelve err tal cual.
* @param err error
* @return error
**/
func (g *syncGuard) note(err error) error {
	if err != nil {
		g.failed.Store(true)
	}
	return err
}

/**
* cleanNoSpace dice si err es un disco lleno que no dejó nada a medias: sale de un lugar que no había escrito nada
* irreversible (extender un archivo de datos, una página del archivo de estados, catalog.tmp), ningún fsync falló, el
* WAL y la caché siguen sanos, y no es la falta de lugar del WAL en medio de una operación (wal.ErrNoRoom, que podría
* dejar una página pedida sin usar).
* @param err error
* @return bool
**/
func (e *Engine) cleanNoSpace(err error) bool {
	return vfs.IsNoSpace(err) && !errors.Is(err, wal.ErrNoRoom) && !errors.Is(err, buffer.ErrFailed) &&
		!errors.Is(err, ErrFailed) && e.log.Failed() == nil && !e.guard.failed.Load()
}

/**
* writeError trata el error de una escritura: un disco lleno limpio vuelve como ErrDiskFull y el motor sigue; todo lo
* demás deja el motor inutilizable (de solo lectura si la causa es de disco lleno).
* @param err error
* @return error
**/
func (e *Engine) writeError(err error) error {
	if e.cleanNoSpace(err) {
		return e.diskFull(err)
	}
	return e.fail(err)
}

/**
* diskFull envuelve en ErrDiskFull el error de una operación que no hizo nada por falta de lugar, y la cuenta.
* @param err error
* @return error
**/
func (e *Engine) diskFull(err error) error {
	e.metrics.diskFull.Add(1)
	return fmt.Errorf(MSG_DISK_FULL, ErrDiskFull, err)
}

/**
* isCleanDiskFull dice si err es un ErrDiskFull de una operación que no hizo nada (el motor sigue).
* @param err error
* @return bool
**/
func isCleanDiskFull(err error) bool {
	return errors.Is(err, ErrDiskFull) && !errors.Is(err, ErrFailed)
}

/**
* admit es la entrada de lugar de una operación que escribe, antes de escribir nada: el WAL tiene que tener al menos
* medio segmento preasignado (wal.Room). Con el disco lleno la operación se rechaza con ErrDiskFull y el motor sigue;
* cuando vuelve a haber lugar, la próxima entrada lo preasigna.
* @return error: ErrDiskFull, o ErrFailed si un fsync de la preasignación falló
**/
func (e *Engine) admit() error {
	err := e.log.Room()
	switch {
	case err == nil:
		return nil
	case errors.Is(err, wal.ErrNoRoom):
		return e.diskFull(err)
	}
	return e.fail(err)
}

/**
* walBacklog es el WAL escrito desde el punto de rehacer del último checkpoint terminado: el que todavía no se puede
* reciclar.
* @return int64: bytes
**/
func (e *Engine) walBacklog() int64 {
	return e.log.Written() - e.doneWAL.Load()
}

/**
* retainedBacklog es lo que mira el freno: el mayor entre el WAL desde el último checkpoint terminado, el que espera
* al archivador (sin tope de archivo) y el que retiene la ranura más atrasada (sin MAX_SLOT_WAL_KEEP_SIZE).
* @return int64: bytes
**/
func (e *Engine) retainedBacklog() int64 {
	return max(e.walBacklog(), e.archiveBacklog(), e.slotBacklog())
}

/**
* throttle frena a un escritor mientras el WAL sin reciclar (retainedBacklog) pase de walBacklogFactor ×
* MAX_WAL_SIZE: espera a que termine un checkpoint o avance el archivador, hasta LOCK_TIMEOUT (≤ 0 = sin límite).
* Se llama sin nada tomado (ni closeMu): Close lo despierta.
* @return error: ErrThrottled al pasar el plazo, ErrClosed si el motor se cierra
**/
func (e *Engine) throttle() error {
	limit := e.cfg.maxWALSize * walBacklogFactor
	if e.retainedBacklog() <= limit {
		return nil
	}
	e.metrics.throttled.Add(1)
	var timeout <-chan time.Time
	if e.cfg.lockTimeout > 0 {
		timer := time.NewTimer(e.cfg.lockTimeout)
		defer timer.Stop()
		timeout = timer.C
	}
	for {
		done := e.checkpointSignal()
		backlog := e.retainedBacklog()
		if backlog <= limit {
			return nil
		}
		select {
		case <-done:
		case <-e.stop:
			return ErrClosed
		case <-timeout:
			return fmt.Errorf(MSG_THROTTLED, ErrThrottled, backlog, limit)
		}
	}
}

/**
* checkpointSignal devuelve el canal que se cierra al terminar el próximo checkpoint (o al avanzar el archivador).
* @return <-chan struct{}
**/
func (e *Engine) checkpointSignal() <-chan struct{} {
	e.sigMu.Lock()
	defer e.sigMu.Unlock()
	if e.ckptDone == nil {
		e.ckptDone = make(chan struct{})
	}
	return e.ckptDone
}

/**
* checkpointFinished anota el WAL ya reciclable (el de antes del punto de rehacer del checkpoint que terminó) y
* despierta a los escritores frenados.
* @param walBytes int64: log.Written() en su punto de rehacer
**/
func (e *Engine) checkpointFinished(walBytes int64) {
	e.doneWAL.Store(walBytes)
	e.wakeWriters()
}

/**
* wakeWriters despierta a los escritores frenados (throttle) para que vuelvan a mirar el WAL retenido.
**/
func (e *Engine) wakeWriters() {
	e.sigMu.Lock()
	defer e.sigMu.Unlock()
	if e.ckptDone != nil {
		close(e.ckptDone)
		e.ckptDone = nil
	}
}

/**
* deferredBrake es el freno de las confirmaciones sin fsync: si quedaron más de maxDeferredLag registros del WAL sin
* fsync, espera el fsync hasta lsn − maxDeferredLag (un solo fsync que hace durable todo lo publicado).
* @param lsn page.LSN: el de la confirmación
* @return error: el del fsync
**/
func (e *Engine) deferredBrake(lsn page.LSN) error {
	if uint64(lsn) <= uint64(e.log.Durable())+maxDeferredLag {
		return nil
	}
	return e.log.WaitDurable(lsn - maxDeferredLag)
}
