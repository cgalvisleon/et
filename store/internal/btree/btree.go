package btree

import (
	"fmt"

	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/xact"
)

const (
	// MaxIDLen es el largo máximo de un id (pregunta 37).
	MaxIDLen = 2000

	// EntryHeaderSize es el largo de los campos fijos de una entrada de hoja.
	EntryHeaderSize = 28

	// OverflowRefSize es el largo de la referencia a desborde dentro de una entrada.
	OverflowRefSize = 4

	// MaxEntrySize es el largo máximo de una entrada de hoja.
	MaxEntrySize = 2032

	// PivotHeaderSize es el largo de los campos fijos de un pivote.
	PivotHeaderSize = 16

	// MetaSize es el largo del contenido de la página meta.
	MetaSize = 48

	// MetaMagic identifica la página meta de un store.
	MetaMagic = page.MetaMagic // "ETBT"

	// MetaBlock es el bloque de la página meta.
	MetaBlock = page.MetaBlock
)

// Posiciones de los campos de una entrada de hoja.
const (
	OffEntryFlags    = 0
	OffEntryIDLen    = 4
	OffEntryLSN      = 8
	OffEntryXID      = 16
	OffEntryValueLen = 24
)

// EntryFlags son las marcas de una entrada de hoja (uint32 en su byte 0).
type EntryFlags uint32

const (
	EntryTombstone    EntryFlags = 1 << 0 // borrado: sin valor
	EntryOverflow     EntryFlags = 1 << 1 // el valor está en desborde
	EntryXidCommitted EntryFlags = 1 << 2 // marca: su transacción confirmó y el WAL es durable hasta la confirmación
	EntryXidAborted   EntryFlags = 1 << 3 // marca: su transacción abortó
	EntryFrozen       EntryFlags = 1 << 4 // congelada: confirmada y visible para todos, sin consultar el archivo de estados
	EntryFreeing      EntryFlags = 1 << 5 // versión muerta cuya cadena de desborde se está liberando (pregunta 58)
)

// hintFlags son las marcas de visibilidad que pone una lectura (KindSetHints).
const hintFlags = EntryXidCommitted | EntryXidAborted

// PivotHasLSN: el pivote lleva el lsn.
const PivotHasLSN uint16 = 1 << 0

// Version es una entrada de hoja ya leída.
type Version struct {
	ID       string
	LSN      page.LSN
	XID      page.XID
	Flags    EntryFlags
	Value    []byte       // copia del valor en línea; nil en desborde y en lápidas
	Overflow overflow.Ref // con EntryOverflow
}

// Action es lo que decide la condición de una escritura.
type Action int

const (
	Apply Action = 1 // escribir la versión
	Skip  Action = 2 // no escribir (la condición no se cumple)
)

// Latest es lo que ve una escritura en la hoja bloqueada: la versión viva más nueva que cuenta para ella (la propia
// de su transacción o la última confirmada).
type Latest struct {
	Version Version
	Live    bool // hay versión y no es una lápida
}

// Write es una versión a escribir.
type Write struct {
	XID       page.XID
	Tombstone bool
	Value     []byte        // en línea
	Overflow  *overflow.Ref // o ya escrito en desborde
	Commit    bool          // operación suelta: el registro lleva FlagCommit

	// Snapshot de la transacción (nil en una operación suelta, Read Committed): si la última confirmada no es
	// visible para él, Write devuelve el error de serialización.
	Snapshot *xact.Snapshot

	// Check decide con la hoja bloqueada (nil = Apply siempre: Put, Remove). No puede bloquear ni esperar.
	Check func(latest Latest) Action

	// UnderLock corre bajo el writeMu del WAL al reservar el registro de la hoja, con su lsn y cómo cambia la
	// cantidad de ids vivos (+1, -1 o 0): así el motor suma los contadores de Count en orden de lsn.
	UnderLock func(lsn page.LSN, countDelta int)
}

// Result es lo que hizo una escritura.
type Result struct {
	Applied    bool
	LSN        page.LSN
	CountDelta int // +1 agregó un id vivo, -1 quitó uno, 0 ninguno
}

// WaitError: otra transacción en curso tiene una versión más nueva del id. Write soltó la hoja; quien llama espera
// con xact.Manager.Wait y vuelve a llamar.
type WaitError struct {
	XID page.XID
}

func (e *WaitError) Error() string {
	return fmt.Sprintf("btree: id is being written by transaction %d", e.XID)
}

// VacuumOptions es lo que necesita la limpieza de un store.
type VacuumOptions struct {
	// Horizon es el xid vivo más viejo (xact.Manager.OldestActive): toda transacción con xid menor terminó y todo
	// snapshot abierto ve sus versiones confirmadas.
	Horizon page.XID

	// FreezeLimit: se congelan las versiones confirmadas con xid < FreezeLimit (0 = no congela; se acota a Horizon).
	FreezeLimit page.XID

	// Aggressive: salta por el mapa de visibilidad solo las hojas todo-congeladas (no las todo-visibles), así la
	// pasada puede subir el frozenXID del store (VacuumStats.Complete).
	Aggressive bool

	// XID es el de la limpieza: va en sus registros del WAL (0 vale).
	XID page.XID

	// Free libera lo que queda de la cadena de desborde de una versión muerta, de la última página a la primera
	// (overflow.Manager.FreeTail con v.Overflow y v.XID). Tiene que poder repetirse tras un corte. nil = las versiones
	// con desborde no se tocan.
	Free func(v Version) error
}

// VacuumStats es lo que hizo una limpieza.
type VacuumStats struct {
	Leaves  int // hojas recorridas
	Removed int // versiones quitadas de las hojas
	Freed   int // cadenas de desborde liberadas
	Frozen  int // versiones congeladas
	Skipped int // hojas saltadas por el mapa de visibilidad

	// SkippedVisible son las hojas saltadas por ser todo-visibles sin ser todo-congeladas (solo sin Aggressive).
	SkippedVisible int

	// Complete: la pasada terminó sin error y no saltó ninguna hoja que no fuera todo-congelada, así que toda versión
	// del store con xid < FrozenXID está congelada o quitada (o «liberándose», que nunca consulta su estado).
	Complete bool

	// FrozenXID es el menor xid de una versión que quedó sin congelar en las hojas recorridas, acotado por Horizon:
	// el frozenXID que puede guardar el store si Complete.
	FrozenXID page.XID
}

// Tree es el árbol de un store.
type Tree interface {
	// Lookup devuelve la versión que ve snap (nil = la más nueva confirmada, Read suelto), sin copiar el valor
	// de desborde; false si no hay ninguna o es una lápida.
	Lookup(id string, snap *xact.Snapshot) (Version, bool, error)

	// Write escribe una versión nueva de id (ver el orden de un cambio en store/doc.go).
	Write(id string, w Write) (Result, error)

	// Scan recorre en orden de id, desde from, la versión que ve snap de cada id. fn devuelve false para parar; no
	// puede escribir en el store. BTree tiene además ScanBackward (en orden descendente).
	Scan(from string, snap *xact.Snapshot, fn func(v Version) (bool, error)) error
}
