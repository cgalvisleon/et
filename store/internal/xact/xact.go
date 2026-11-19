package xact

import (
	"errors"
	"slices"
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
)

const (
	// Dir es el directorio del archivo de estados dentro del motor.
	Dir = "xact"

	// PageSize es el tamaño de una página del archivo de estados.
	PageSize = 8192

	// XidsPerPage: 4 xids por byte.
	XidsPerPage = PageSize * 4

	// PagesPerFile es la cantidad de páginas de cada archivo.
	PagesPerFile = 4096

	// XidsPerFile es la cantidad de xids de cada archivo.
	XidsPerFile = XidsPerPage * PagesPerFile

	// GroupSize es la cantidad de xids que comparten un lsn de confirmación en memoria.
	GroupSize = 32
)

// Status es el estado de una transacción (2 bits).
type Status uint8

const (
	StatusInProgress Status = 0 // en curso (o nunca terminó: ver el horizonte de recuperación)
	StatusCommitted  Status = 1
	StatusAborted    Status = 2
	StatusReserved   Status = 3 // no se usa; leerlo es corrupción
)

// ErrWaitTimeout: Wait pasó su tiempo máximo y la otra transacción sigue en curso.
var ErrWaitTimeout = errors.New("xact: timeout waiting for another transaction")

// Snapshot es la foto de las transacciones en un momento (ver doc.go, «Visibilidad»).
type Snapshot struct {
	Own        page.XID   // la transacción que lo usa; 0 para un snapshot de solo lectura
	Xmin       page.XID   // el xid más viejo en curso al tomarlo (o Xmax si ninguno)
	Xmax       page.XID   // el siguiente xid a asignar al tomarlo
	InProgress []page.XID // los que estaban en curso, ordenados
}

// Visible dice si una versión con xid y estado status es visible para el snapshot: es de la propia transacción, o
// es anterior a Xmax, no estaba en curso al tomarlo y está confirmada.
func (s Snapshot) Visible(xid page.XID, status Status) bool {
	if s.Own != 0 && xid == s.Own {
		return true
	}
	if xid >= s.Xmax {
		return false
	}
	if _, found := slices.BinarySearch(s.InProgress, xid); found {
		return false
	}
	return status == StatusCommitted
}

// HintAllowed dice si se puede anotar la marca de confirmada: el estado es confirmada y el WAL es durable hasta el
// lsn de confirmación de su grupo (0 = ya durable; pregunta 49).
func HintAllowed(status Status, commitLSN, durable page.LSN) bool {
	return status == StatusCommitted && commitLSN <= durable
}

// StatusFile es el archivo de estados con su caché.
type StatusFile interface {
	// Status devuelve el estado de xid y el lsn de confirmación de su grupo (0 si no se conoce: ya durable).
	Status(xid page.XID) (Status, page.LSN, error)

	// SetStatus anota el estado final de xid con el lsn de su registro de confirmación o aborto.
	SetStatus(xid page.XID, status Status, lsn page.LSN) error

	// Flush escribe al disco las páginas cambiadas (respetando el WAL durable) y hace fsync (en cada checkpoint).
	Flush() error

	// Truncate borra los archivos con todos sus xids menores que before (fase 2, congelado).
	Truncate(before page.XID) error
}

// Manager lleva las transacciones en curso.
type Manager interface {
	// Begin asigna el xid siguiente y anota la transacción en curso.
	Begin() page.XID

	// End la quita de las en curso y despierta a quien la espera. Se llama después de SetStatus.
	End(xid page.XID)

	// Snapshot toma la foto actual para la transacción own (0 = solo lectura).
	Snapshot(own page.XID) Snapshot

	// TakeSnapshot toma una foto de solo lectura y la registra en el horizonte hasta ReleaseSnapshot(id).
	TakeSnapshot() (Snapshot, uint64)

	// ReleaseSnapshot quita del horizonte el snapshot de solo lectura id.
	ReleaseSnapshot(id uint64)

	// Wait espera a que xid termine, hasta timeout (ErrWaitTimeout). Nunca se llama con una página bloqueada.
	Wait(xid page.XID, timeout time.Duration) error

	// NextXID es el siguiente xid a asignar.
	NextXID() page.XID

	// OldestActive es el xid en curso más viejo, contando los Xmin de los snapshots abiertos (horizonte de la
	// limpieza); NextXID si no hay ninguno.
	OldestActive() page.XID
}
