package crashfs

import "github.com/cgalvisleon/et/store/internal/vfs"

// Mode dice qué hace un corte con lo que no era durable.
type Mode int

const (
	// LoseUnsynced pierde todo lo escrito desde el último Sync de cada archivo y los cambios de directorio desde el
	// último SyncDir. Las escrituras anteriores al corte quedan enteras o no quedan.
	LoseUnsynced Mode = iota

	// TornWrites además rompe, al azar, las escrituras sin Sync en sectores de SectorSize: de una página de 8 KB
	// puede quedar una mezcla de sectores viejos y nuevos (lo que protegen las páginas enteras en el WAL).
	TornWrites
)

// SectorSize es la unidad que un disco escribe de forma atómica en la simulación de escrituras rotas.
const SectorSize = 512

// FS es el sistema de archivos simulado.
type FS interface {
	vfs.FS

	// Crash simula un corte: aplica mode a lo que no era durable, suelta todos los bloqueos y deja inválidos los
	// archivos abiertos (cualquier operación sobre ellos falla). Después se puede volver a abrir el motor encima.
	Crash(mode Mode, seed int64)

	// FailAfter hace que, después de n operaciones de escritura o Sync más, todas fallen con un error de E/S (para
	// probar que el motor queda inutilizable y no reintenta el fsync). n < 0 lo desactiva.
	FailAfter(n int)
}
