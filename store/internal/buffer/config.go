package buffer

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

const (
	// NumPartitions es la cantidad de particiones del mapa Tag → búfer (NUM_BUFFER_PARTITIONS de PostgreSQL).
	NumPartitions = 128

	// DefaultBgWriterDelay es la pausa entre rondas del escritor de fondo (bgwriter_delay de PostgreSQL).
	DefaultBgWriterDelay = 200 * time.Millisecond

	// BgWriterMaxPages es el tope de páginas que escribe el escritor de fondo por ronda (bgwriter_lru_maxpages).
	BgWriterMaxPages = 100

	// BgWriterDirtyTarget: con más de esta fracción de la caché sucia, el escritor de fondo escribe también páginas
	// en uso (sin pins), no solo las que el reemplazo usaría ya: sin reemplazo la manecilla no gira y las sucias
	// quedaban hasta el próximo checkpoint (la presión «dirty» en 0,9; tarea 97 de F-23).
	BgWriterDirtyTarget = 0.5

	// BgWriterMaxDirtyPages es el tope de esa segunda pasada por ronda (8 MB cada BgWriterDelay).
	BgWriterMaxDirtyPages = 1000
)

// Errores de la caché. Se comparan con errors.Is.
var (
	// ErrClosed: la caché ya se cerró.
	ErrClosed = errors.New("buffer: caché cerrada")

	// ErrNoBuffers: todos los búferes están fijados; no hay ninguno para reemplazar.
	ErrNoBuffers = errors.New("buffer: no hay búferes libres (todos fijados)")

	// ErrFailed: falló la escritura de una página o la espera al WAL; la caché quedó inutilizable hasta reabrir el
	// motor. Se devuelve envolviendo además el error original.
	ErrFailed = errors.New("buffer: la caché quedó inutilizable")

	// ErrConfig: falta el gestor de archivos o el WAL, o la cantidad de búferes no sirve.
	ErrConfig = errors.New("buffer: configuración inválida")
)

// Config es lo que necesita Open.
type Config struct {
	Smgr    smgr.Manager // los archivos de datos
	Flusher wal.Flusher  // el WAL, para cumplir «WAL antes que la página»

	// Buffers es la cantidad de búferes de page.Size; 0 = EnvBuffers(). Las pruebas usan pocos.
	Buffers int

	// BgWriterDelay es la pausa del escritor de fondo; 0 = DefaultBgWriterDelay, negativo = sin escritor de fondo.
	BgWriterDelay time.Duration
}

/**
* EnvBuffers lee STORE_CACHE_SIZE (MB, 128 por omisión) y lo pasa a cantidad de búferes (al menos uno).
* @return int
**/
func EnvBuffers() int {
	mb := max(envar.GetInt64("STORE_CACHE_SIZE", DefaultCacheSize>>20), 1)
	return int(mb << 20 / page.Size)
}

/**
* ringLen es la cantidad de búferes del anillo: RingSize, pero nunca más de un octavo de la caché (como
* PostgreSQL), y al menos uno.
* @param buffers int: tamaño de la caché
* @return int
**/
func ringLen(buffers int) int {
	return max(min(RingSize/page.Size, buffers/8), 1)
}
