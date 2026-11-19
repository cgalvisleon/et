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
	MetaMagic = 0x45544254 // "ETBT"

	// MetaBlock es el bloque de la página meta.
	MetaBlock page.BlockNumber = 0
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
	EntryFrozen       EntryFlags = 1 << 4 // marca: visible para todos (fase 2)
)

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

	// UnderLock se pasa a wal.Log.Reserve (los contadores de una operación suelta).
	UnderLock func(lsn page.LSN)
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

// Tree es el árbol de un store.
type Tree interface {
	// Lookup devuelve la versión que ve snap (nil = la más nueva confirmada, Read suelto), sin copiar el valor
	// de desborde; false si no hay ninguna o es una lápida.
	Lookup(id string, snap *xact.Snapshot) (Version, bool, error)

	// Write escribe una versión nueva de id (ver el orden de un cambio en store/doc.go).
	Write(id string, w Write) (Result, error)

	// Scan recorre en orden de id, desde from, la versión que ve snap de cada id (fase 2). fn devuelve false para
	// parar; no puede escribir en el store.
	Scan(from string, snap *xact.Snapshot, fn func(v Version) (bool, error)) error
}
