package store

import (
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/wal"
)

const (
	// MaxIDLen es el largo máximo de un id, en bytes.
	MaxIDLen = btree.MaxIDLen

	// MaxValueLen es el largo máximo de un valor (64 MiB).
	MaxValueLen = overflow.MaxValueLen

	// MaxNameLen es el largo máximo del nombre de un store, en bytes.
	MaxNameLen = 255

	// completionTarget reparte la escritura de un checkpoint en el 90 % de su intervalo
	// (CHECKPOINT_COMPLETION_TARGET de PostgreSQL, constante).
	completionTarget = 0.9

	// walTriggerDivisor: el checkpoint por volumen salta a MAX_WAL_SIZE / (1 + completionTarget).
	walTriggerDivisor = 1 + completionTarget
)

// config es lo configurable del motor: Open la lee del entorno y las pruebas la fijan a mano.
type config struct {
	syncOnWrite       bool          // SYNC_ON_WRITE: una confirmación espera el fsync del WAL
	checkpointTimeout time.Duration // CHECKPOINT_TIMEOUT
	maxWALSize        int64         // MAX_WAL_SIZE, en bytes
	lockTimeout       time.Duration // LOCK_TIMEOUT
	txTimeout         time.Duration // TRANSACTION_TIMEOUT: lotes y Snapshot (0 = sin tiempo máximo)
	wal               wal.Options   // WAL_SEGMENT_SIZE y repuestos
	buffers           int           // STORE_CACHE_SIZE, en búferes de 8 KB
	bgWriterDelay     time.Duration // pausa del escritor de fondo de la caché (0 = la de buffer)
	checkEvery        time.Duration // cada cuánto mira el checkpointer si toca un checkpoint
	vacuumEvery       time.Duration // VACUUM_INTERVAL: cada cuánto mira la limpieza de fondo (0 = apagada)
	vacuumWrites      int64         // VACUUM_THRESHOLD: escrituras desde la última limpieza de fondo para correr otra
}

/**
* envConfig lee la configuración de las variables de entorno (ver «Variables de entorno» en doc.go).
* @return config
**/
func envConfig() config {
	return config{
		syncOnWrite:       envar.GetBool("SYNC_ON_WRITE", true),
		checkpointTimeout: time.Duration(max(envar.GetInt64("CHECKPOINT_TIMEOUT", 300), 1)) * time.Second,
		maxWALSize:        max(envar.GetInt64("MAX_WAL_SIZE", 1024), 1) << 20,
		lockTimeout:       time.Duration(max(envar.GetInt64("LOCK_TIMEOUT", 5), 0)) * time.Second,
		txTimeout:         time.Duration(max(envar.GetInt64("TRANSACTION_TIMEOUT", 600), 0)) * time.Second,
		wal:               wal.EnvOptions(),
		buffers:           buffer.EnvBuffers(),
		checkEvery:        time.Second,
		vacuumEvery:       time.Duration(max(envar.GetInt64("VACUUM_INTERVAL", 60), 0)) * time.Second,
		vacuumWrites:      max(envar.GetInt64("VACUUM_THRESHOLD", 1000), 0),
	}
}

/**
* walTrigger es el volumen de WAL desde el punto de rehacer que dispara un checkpoint.
* @return int64
**/
func (c config) walTrigger() int64 {
	return int64(float64(c.maxWALSize) / walTriggerDivisor)
}
