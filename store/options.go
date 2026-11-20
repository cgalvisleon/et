package store

import (
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Options es la configuración de un motor para OpenWith y OpenReplicaWith (tarea 84 de F-23): un proceso puede abrir
// varios motores con distinta caché, WAL y tiempos (josefina abre el de datos y uno local chico). Un campo en cero
// toma el valor de su variable de entorno (ver «Variables de entorno» en doc.go), así Options{} es lo mismo que Open.
// En los que 0 apaga algo (TransactionTimeout, WALWriterDelay, VacuumInterval, ArchiveMaxMB, SlotKeepMB,
// WALBuffersKB), un valor negativo lo apaga.
type Options struct {
	CacheMB      int // STORE_CACHE_SIZE: la caché de páginas, en MB
	WALSegmentMB int // WAL_SEGMENT_SIZE: el largo de un segmento del WAL, en MB (al menos 1)
	MaxWALMB     int // MAX_WAL_SIZE: el WAL desde el último checkpoint que dispara otro, en MB
	WALBuffersKB int // WAL_BUFFERS: los registros que esperan en memoria antes de escribirse, en KB (< 0: sin búfer)

	SyncOnWrite *bool // SYNC_ON_WRITE: si una confirmación espera su fsync (nil: la variable)

	CheckpointTimeout  time.Duration // CHECKPOINT_TIMEOUT
	LockTimeout        time.Duration // LOCK_TIMEOUT
	TransactionTimeout time.Duration // TRANSACTION_TIMEOUT (< 0: sin tiempo máximo)
	WALWriterDelay     time.Duration // WAL_WRITER_DELAY (< 0: sin escritor de fondo del WAL)
	VacuumInterval     time.Duration // VACUUM_INTERVAL (< 0: sin limpieza de fondo)
	VacuumThreshold    int64         // VACUUM_THRESHOLD: escrituras entre dos limpiezas de fondo

	Archive      *bool // WAL_ARCHIVE: el archivo continuo del WAL (nil: la variable)
	ArchiveMaxMB int   // WAL_ARCHIVE_MAX_SIZE, en MB (< 0: sin tope)
	SlotKeepMB   int   // MAX_SLOT_WAL_KEEP_SIZE, en MB (< 0: sin tope)
}

/**
* apply pone en cfg (la del entorno) los campos de o que no están en cero.
* @param cfg *config
**/
func (o Options) apply(cfg *config) {
	if o.CacheMB > 0 {
		cfg.buffers = max(o.CacheMB<<20/page.Size, 1)
	}
	if o.MaxWALMB > 0 {
		cfg.maxWALSize = int64(o.MaxWALMB) << 20
	}
	if o.WALSegmentMB > 0 || o.MaxWALMB > 0 {
		segment := cfg.wal.SegmentSize
		if o.WALSegmentMB > 0 {
			segment = max(int64(o.WALSegmentMB)<<20, wal.MinSegmentSize)
		}
		cfg.wal.SegmentSize = segment
		cfg.wal.Spares = int(max(cfg.maxWALSize/2/segment, 1))
	}
	if o.WALBuffersKB != 0 {
		cfg.wal.Buffer = max(int64(o.WALBuffersKB), 0) << 10
	}
	if o.SyncOnWrite != nil {
		cfg.syncOnWrite = *o.SyncOnWrite
	}
	if o.Archive != nil {
		cfg.archive = *o.Archive
	}
	setDuration(&cfg.checkpointTimeout, o.CheckpointTimeout, false)
	setDuration(&cfg.lockTimeout, o.LockTimeout, false)
	setDuration(&cfg.txTimeout, o.TransactionTimeout, true)
	setDuration(&cfg.walWriterDelay, o.WALWriterDelay, true)
	setDuration(&cfg.vacuumEvery, o.VacuumInterval, true)
	if o.VacuumThreshold > 0 {
		cfg.vacuumWrites = o.VacuumThreshold
	}
	setMB(&cfg.archiveMaxSize, o.ArchiveMaxMB)
	setMB(&cfg.slotKeepSize, o.SlotKeepMB)
}

/**
* setDuration pone v en dst si no es cero; uno negativo es 0 donde 0 apaga (off), y se ignora donde no.
* @param dst *time.Duration, v time.Duration, off bool
**/
func setDuration(dst *time.Duration, v time.Duration, off bool) {
	switch {
	case v > 0:
		*dst = v
	case v < 0 && off:
		*dst = 0
	}
}

/**
* setMB pone mb MB en dst si no es cero; uno negativo es 0 (sin tope).
* @param dst *int64, mb int
**/
func setMB(dst *int64, mb int) {
	switch {
	case mb > 0:
		*dst = int64(mb) << 20
	case mb < 0:
		*dst = 0
	}
}

/**
* openWith abre el motor de dir con la configuración del entorno cambiada por o.
* @param dir string, o Options, replica bool
* @return *Engine, error
**/
func openWith(dir string, o Options, replica bool) (*Engine, error) {
	cfg := envConfig()
	o.apply(&cfg)
	cfg.replica = replica
	return open(vfs.OS(), dir, cfg)
}
