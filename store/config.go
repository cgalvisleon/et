package store

import (
	"strings"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
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

	// freezeMinAge (FREEZE_MIN_AGE, constante): 0 = se congela toda versión confirmada visible para todos la primera
	// vez que la limpieza pasa por su hoja (ver «Congelado» en doc.go).
	freezeMinAge = 0

	// freezeTableAge (FREEZE_TABLE_AGE, constante): con el frozenXID de un store un archivo de estados entero detrás
	// del horizonte, la limpieza es agresiva (no salta las hojas solo todo-visibles).
	freezeTableAge = xact.XidsPerFile

	// vacuumXids: la limpieza de fondo corre también cada medio archivo de estados de xids asignados.
	vacuumXids = xact.XidsPerFile / 2

	// replicaConflictDelay: cuánto espera el aplicador de una réplica a los snapshots abiertos antes de cancelarlos
	// para rehacer un cambio que no deben ver (como max_standby_streaming_delay de PostgreSQL, constante).
	replicaConflictDelay = 30 * time.Second
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
	walWriterDelay    time.Duration // WAL_WRITER_DELAY (ms): cada cuánto el escritor del WAL hace durable lo confirmado sin fsync (0 = apagado)
	vacuumEvery       time.Duration // VACUUM_INTERVAL: cada cuánto mira la limpieza de fondo (0 = apagada)
	vacuumWrites      int64         // VACUUM_THRESHOLD: escrituras desde la última limpieza de fondo para correr otra
	vacuumXids        uint64        // xids asignados desde la última limpieza de fondo para correr otra
	freezeMinAge      uint64        // se congela lo confirmado con xid < horizonte − freezeMinAge (FREEZE_MIN_AGE)
	freezeTableAge    uint64        // limpieza agresiva si el frozenXID del store está así de detrás (FREEZE_TABLE_AGE)
	archive           bool          // WAL_ARCHIVE: archivo continuo del WAL (retiene los segmentos hasta SetArchived)
	archiveMaxSize    int64         // WAL_ARCHIVE_MAX_SIZE, en bytes: tope del WAL retenido por el archivo (0 = sin tope: frena)
	stopLSN           page.LSN      // objetivo de la recuperación de OpenAt (0 = rehacer todo el WAL)
	slotKeepSize      int64         // MAX_SLOT_WAL_KEEP_SIZE, en bytes: tope del WAL que retiene una ranura (0 = sin tope: frena)
	replica           bool          // OpenReplica: el motor es una réplica de solo lectura que aplica el WAL del primario
	conflictDelay     time.Duration // replicaConflictDelay (las pruebas lo acortan)
	decodeWorkMem     int           // memoria del búfer de reordenamiento de un ChangeReader (0 = logicalWorkMem)

	// Solo pruebas: páginas por archivo de estados (0 = xact.PagesPerFile) y primer xid de un motor nuevo (0 = 1).
	xactFilePages int
	firstXID      page.XID
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
		walWriterDelay:    time.Duration(max(envar.GetInt64("WAL_WRITER_DELAY", 200), 0)) * time.Millisecond,
		vacuumEvery:       time.Duration(max(envar.GetInt64("VACUUM_INTERVAL", 60), 0)) * time.Second,
		vacuumWrites:      max(envar.GetInt64("VACUUM_THRESHOLD", 1000), 0),
		vacuumXids:        vacuumXids,
		freezeMinAge:      freezeMinAge,
		freezeTableAge:    freezeTableAge,
		archive:           archiveOn(envar.GetStr("WAL_ARCHIVE", "off")),
		archiveMaxSize:    max(envar.GetInt64("WAL_ARCHIVE_MAX_SIZE", 0), 0) << 20,
		slotKeepSize:      max(envar.GetInt64("MAX_SLOT_WAL_KEEP_SIZE", 4096), 0) << 20,
		conflictDelay:     replicaConflictDelay,
	}
}

/**
* archiveOn interpreta WAL_ARCHIVE: on, true, 1 o yes lo activan; cualquier otro valor lo deja apagado.
* @param value string
* @return bool
**/
func archiveOn(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "yes":
		return true
	}
	return false
}

/**
* walTrigger es el volumen de WAL desde el punto de rehacer que dispara un checkpoint.
* @return int64
**/
func (c config) walTrigger() int64 {
	return int64(float64(c.maxWALSize) / walTriggerDivisor)
}
