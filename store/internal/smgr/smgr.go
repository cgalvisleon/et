package smgr

import "github.com/cgalvisleon/et/store/internal/page"

// FileNo es el número de archivo de un store, asignado por el catálogo y nunca reutilizado. 0 = ninguno.
type FileNo uint32

// Fork es una de las secuencias de bloques de un store.
type Fork uint8

const (
	ForkMain Fork = 0 // árbol, desborde y lista de libres
	ForkVM   Fork = 1 // mapa de visibilidad (fase 1B/2)
)

const (
	// BlocksPerFile es la cantidad de bloques de cada archivo de datos: 1 GB / page.Size (pregunta 27).
	BlocksPerFile = (1 << 30) / page.Size

	// DataDir es el directorio de los archivos de datos dentro del motor.
	DataDir = "data"
)

// Manager guarda y lee bloques de los archivos de datos. Es seguro usarlo desde varias goroutines; las escrituras
// de un mismo bloque las serializa quien llama (la caché).
type Manager interface {
	// Create crea el primer archivo vacío del fork (idempotente) y hace fsync de data/.
	Create(file FileNo, fork Fork) error

	// Exists dice si el fork tiene al menos un archivo.
	Exists(file FileNo, fork Fork) (bool, error)

	// NBlocks es la cantidad de bloques del fork (el largo de su último archivo cuenta).
	NBlocks(file FileNo, fork Fork) (page.BlockNumber, error)

	// Extend escribe buf como el bloque block, que debe ser NBlocks (o mayor: los intermedios quedan en ceros),
	// creando el archivo siguiente cuando hace falta (con fsync de data/).
	Extend(file FileNo, fork Fork, block page.BlockNumber, buf page.Page) error

	// Read lee el bloque en buf. Un bloque más allá del final es un error; uno que existe y está en ceros se
	// devuelve en ceros (página nueva).
	Read(file FileNo, fork Fork, block page.BlockNumber, buf page.Page) error

	// Write escribe buf en un bloque existente. No es durable hasta Sync.
	Write(file FileNo, fork Fork, block page.BlockNumber, buf page.Page) error

	// Sync hace fsync de los archivos del fork escritos desde su último Sync.
	Sync(file FileNo, fork Fork) error

	// SyncAll hace fsync de todos los archivos escritos desde el último SyncAll (al terminar un checkpoint).
	SyncAll() error

	// Unlink borra todos los archivos de todos los forks del store y hace fsync de data/.
	Unlink(file FileNo) error

	// Close cierra los descriptores abiertos.
	Close() error
}
