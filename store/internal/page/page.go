package page

// LSN es el número de secuencia de un registro del WAL: crece de a uno y sin huecos en todo el WAL. 0 = ninguno.
type LSN uint64

// XID es el número de una transacción, de 64 bits (sin vuelta). 0 = ninguna.
type XID uint64

// BlockNumber es el número de una página dentro de un fork de un store: el bloque n está en el byte n×Size del
// archivo lógico (repartido en archivos de smgr.BlocksPerFile bloques).
type BlockNumber uint32

// OffsetNumber es el número de un ítem dentro de su página, desde 1.
type OffsetNumber uint16

// Page es una página: exactamente Size bytes, alineada a 8 en memoria (la caché las reserva así).
type Page []byte

const (
	// Size es el tamaño de una página (pregunta 19).
	Size = 8192

	// LayoutVersion va en los bits bajos de pd_pagesize_version: la versión con que Init deja una página. Es la
	// versión de la forma de la página, no la del formato: todas las clases de página son de la 1 salvo las hojas
	// del formato 2 (LayoutVersion2, btree: prefijo compartido y cabecera corta).
	LayoutVersion = 1

	// LayoutVersion2 es la de una hoja del formato 2 (btree la pone con SetLayoutVersion al armar la hoja).
	LayoutVersion2 = 2

	// HeaderSize es el largo del encabezado de página.
	HeaderSize = 32

	// ItemIDSize es el largo de un puntero a ítem.
	ItemIDSize = 4

	// ItemAlign es la alineación de la posición de cada ítem.
	ItemAlign = 8

	// OpaqueSize es el largo del espacio especial de las páginas de un store.
	OpaqueSize = 16

	// FirstOffset es el número del primer ítem.
	FirstOffset OffsetNumber = 1
)

// Posiciones de los campos del encabezado.
const (
	OffLSN             = 0
	OffChecksum        = 8
	OffFlags           = 10
	OffLower           = 12
	OffUpper           = 14
	OffSpecial         = 16
	OffPageSizeVersion = 18
	OffPruneXID        = 20
)

// Marcas de pd_flags (las de PostgreSQL).
const (
	FlagHasFreeLines uint16 = 0x0001 // hay punteros LineUnused que se pueden reutilizar
	FlagPageFull     uint16 = 0x0002 // la última inserción no cupo
	FlagAllVisible   uint16 = 0x0004 // todas las versiones de la página son visibles para todos
)

// LineFlags es el estado de un puntero a ítem (lp_flags).
type LineFlags uint8

const (
	LineUnused   LineFlags = 0 // libre, lp_len = 0
	LineNormal   LineFlags = 1 // en uso
	LineRedirect LineFlags = 2 // reservado (HOT en PostgreSQL; no se usa)
	LineDead     LineFlags = 3 // muerto: la limpieza lo puede quitar
)

// Posiciones de los campos del espacio especial, desde pd_special.
const (
	OffOpaquePrev    = 0
	OffOpaqueNext    = 4
	OffOpaqueLevel   = 8
	OffOpaqueFlags   = 12
	OffOpaqueCycleID = 14
)

// Marcas del espacio especial (flags), las de nbtree más las propias del motor.
const (
	OpaqueLeaf            uint16 = 1 << 0  // hoja del árbol
	OpaqueRoot            uint16 = 1 << 1  // raíz del árbol
	OpaqueDeleted         uint16 = 1 << 2  // borrada del árbol (fase 2)
	OpaqueMeta            uint16 = 1 << 3  // página meta (bloque 0)
	OpaqueHalfDead        uint16 = 1 << 4  // vacía pero todavía enlazada (fase 2)
	OpaqueSplitEnd        uint16 = 1 << 5  // la de más a la derecha de un grupo partido
	OpaqueIncompleteSplit uint16 = 1 << 7  // la hermana derecha todavía no tiene su enlace en el padre
	OpaqueOverflow        uint16 = 1 << 9  // página de desborde
	OpaqueFree            uint16 = 1 << 10 // en la lista de páginas libres
)

// Opaque es el espacio especial ya leído.
type Opaque struct {
	Prev    BlockNumber
	Next    BlockNumber
	Level   uint32
	Flags   uint16
	CycleID uint16
}
