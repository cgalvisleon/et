package control

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

const (
	// FileName es el archivo de control dentro del directorio del motor.
	FileName = "control"

	// CatalogFileName es el catálogo de stores; CatalogTempName, su versión a medio escribir.
	CatalogFileName = "catalog"
	CatalogTempName = "catalog.tmp"

	// FormatVersion es la versión del formato en disco del motor (control, catálogo, páginas y WAL): la del encabezado
	// de los segmentos del WAL, así hay un solo número.
	FormatVersion = wal.FormatVersion

	ControlMagic = 0x45544354 // "ETCT"
	CatalogMagic = 0x45544347 // "ETCG"

	// CopySize es el largo de cada copia del control: menos que un sector, como pg_control.
	CopySize = 512

	// SlotCount y SlotSize: ranuras de retención del WAL reservadas en el control (fase 3, pregunta 48).
	SlotCount   = 8
	SlotSize    = 32
	SlotNameLen = 24

	CatalogHeaderSize = 32
	CatalogEntrySize  = 24 // sin el nombre

	// MaxNameLen es el largo máximo del nombre de un store, en bytes.
	MaxNameLen = 255
)

// State es el estado del motor anotado en el control.
type State uint32

const (
	StateShutdown     State = 1 // cerrado con un checkpoint final: no hay nada que reproducir
	StateInProduction State = 2 // abierto (o caído): Open reproduce desde el punto de rehacer
	StateInRecovery   State = 3 // reproduciendo el WAL
)

// Slot es una ranura de retención del WAL (fase 3).
type Slot struct {
	Name       string
	RestartLSN page.LSN
}

// Control es el contenido de una copia del archivo de control.
type Control struct {
	Seq             uint64
	State           State
	CheckpointTime  int64
	Redo            wal.Position
	CheckpointLSN   page.LSN
	NextXID         page.XID
	OldestActiveXID page.XID
	FrozenXID       page.XID
	Slots           [SlotCount]Slot
}

// Entry es un store del catálogo.
type Entry struct {
	Name      string
	File      smgr.FileNo
	MetaBlock page.BlockNumber
	Count     int64
	Flags     uint32
}

// Catalog es el contenido del catálogo.
type Catalog struct {
	NextFileNo smgr.FileNo
	CountLSN   page.LSN
	Entries    []Entry
}

// ControlFile lee y escribe el archivo de control.
type ControlFile interface {
	// Read devuelve la copia válida más nueva; error si no hay ninguna.
	Read() (Control, error)

	// Write escribe c con seq = el último + 1 en la copia que no tiene la última versión, y hace Sync.
	Write(c Control) error

	// Create escribe las dos copias de un motor nuevo (seq 1 y 2 con el mismo contenido) y hace Sync y SyncDir.
	Create(c Control) error
}

// CatalogFile lee y escribe el catálogo.
type CatalogFile interface {
	// Load lee el catálogo; un catalog.tmp sobrante se ignora y se borra.
	Load() (Catalog, error)

	// Read lee y valida el catálogo como Load, sin tocar nada (ni un catalog.tmp sobrante): Verify del motor.
	Read() (Catalog, error)

	// Save reemplaza el catálogo de forma atómica (tmp, Sync, rename, SyncDir).
	Save(c Catalog) error
}
