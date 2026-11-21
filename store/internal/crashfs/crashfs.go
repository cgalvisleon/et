package crashfs

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Mode dice qué hace un corte con lo que no era durable.
type Mode int

const (
	// LoseUnsynced pierde todo lo escrito desde el último Sync de cada archivo y todos los cambios de directorio
	// desde el último SyncDir: el disco queda exactamente en su estado durable.
	LoseUnsynced Mode = iota

	// TornWrites deja sobrevivir al azar parte de lo no durable y rompe las escrituras por sectores: cada sector de
	// cada escritura sin Sync queda o no, por separado, así que de una página de 8 KB puede quedar una mezcla de
	// sectores viejos y nuevos (lo que protegen las páginas enteras en el WAL), y una escritura posterior puede quedar
	// sin una anterior (reordenamiento). El sector es SectorSize o LargeSectorSize, elegido al azar en cada corte.
	TornWrites

	// ReorderUnsynced deja sobrevivir al azar, enteras, algunas de las escrituras sin Sync, en cualquier
	// combinación (una posterior puede quedar sin una anterior). Cada escritura queda entera o no queda.
	ReorderUnsynced
)

// SectorSize es la unidad que un disco escribe de forma atómica en la simulación de escrituras rotas.
const SectorSize = 512

// LargeSectorSize es el sector de los discos de 4 KB; TornWrites lo usa en la mitad de los cortes.
const LargeSectorSize = 4096

var (
	// ErrIO es la falla de E/S que inyectan FailAfter y FailSyncAfter.
	ErrIO = errors.New("crashfs: input/output error")

	// ErrNoSpace es la falla de disco lleno que inyecta SetCapacity: es vfs.ErrNoSpace, así vfs.IsNoSpace la reconoce.
	ErrNoSpace = vfs.ErrNoSpace

	// ErrCrashed es el error de un archivo abierto antes de un corte.
	ErrCrashed = errors.New("crashfs: file opened before a crash")
)

// FS es el sistema de archivos simulado.
type FS interface {
	vfs.FS

	// Crash simula un corte: aplica mode a lo que no era durable, suelta todos los bloqueos, deja inválidos los
	// archivos abiertos (cualquier operación sobre ellos falla con ErrCrashed) y quita las fallas de FailAfter y
	// FailSyncAfter (la capacidad de SetCapacity sigue). La misma historia con la misma seed da el mismo disco.
	// Después se puede volver a abrir el motor encima.
	Crash(mode Mode, seed int64)

	// FailAfter hace que, después de n operaciones de escritura (WriteAt, Truncate) o Sync (Sync, SyncDir) más,
	// todas fallen con ErrIO (para probar que el motor queda inutilizable y no reintenta el fsync). n < 0 lo
	// desactiva.
	FailAfter(n int)

	// FailSyncAfter es como FailAfter pero cuenta solo Sync y SyncDir: las escrituras siguen funcionando.
	FailSyncAfter(n int)

	// SetCapacity fija el tamaño del disco en bytes (la suma del lugar asignado de los archivos con nombre: los bytes
	// escritos, sin los huecos): una escritura que lo pase falla con ErrNoSpace sin escribir nada, también la que cae
	// dentro de un hueco, y crear un archivo con el disco lleno también. Un Truncate que agranda o una escritura más
	// allá del final dejan un hueco, que no ocupa lugar (como ext4 y XFS). Con la capacidad por debajo de lo usado (un disco que prometió lugar y no lo tenía, como un volumen
	// fino o la asignación diferida), además falla con ErrNoSpace el Sync de un archivo con escrituras pendientes, que
	// las olvida como cualquier Sync fallido. Subir la capacidad vuelve a dejar escribir. n <= 0 es sin límite.
	SetCapacity(n int64)

	// Used es lo que ocupa el disco en bytes: la suma del lugar asignado de los archivos con nombre (sin los huecos).
	Used() int64

	// SetDelay agrega una demora a cada escritura (WriteAt, Truncate) y a cada Sync o SyncDir, antes de hacerlos y
	// sin bloquear el resto del disco: un disco lento. 0 la quita.
	SetDelay(write, sync time.Duration)
}
