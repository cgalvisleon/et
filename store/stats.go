package store

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/metric"
)

// defaultStatsInterval es el intervalo de OnStats cuando se pide 0 o menos.
const defaultStatsInterval = time.Second

// Stats es una foto barata del motor (ver «Estadísticas» en doc.go): contadores atómicos leídos uno a uno, sin
// recorrer el árbol ni detener a nadie, así que dos campos pueden no ser del mismo instante. Los contadores son
// desde Open; para tasas, restar dos fotos.
type Stats struct {
	// Caché de páginas
	CacheSize    int   // búferes de 8 KB (STORE_CACHE_SIZE)
	CachePages   int64 // búferes con una página
	CacheDirty   int64 // páginas sucias: cambiadas y todavía no escritas al disco
	CacheHits    int64 // páginas pedidas que ya estaban en la caché
	CacheMisses  int64 // páginas pedidas que hubo que leer del disco (o crear)
	CacheWritten int64 // páginas escritas al disco (checkpoint, escritor de fondo, reemplazo)

	// WAL y checkpoints
	WALSinceCheckpoint int64         // bytes de WAL desde el punto de rehacer del último checkpoint
	LSN                uint64        // todo lsn ≤ LSN está escrito en el WAL
	DurableLSN         uint64        // todo lsn ≤ DurableLSN tiene fsync
	Checkpoints        int64         // checkpoints hechos (también el que cierra la recuperación de Open)
	LastCheckpoint     time.Time     // cuándo empezó el último (cero si ninguno)
	LastCheckpointTook time.Duration // cuánto duró el último (uno de fondo se reparte en el 90 % de su intervalo)

	// Versiones por limpiar y transacciones
	WritesSinceVacuum int64  // escrituras desde la última limpieza: lo que mira VACUUM_THRESHOLD
	Vacuums           int64  // limpiezas del motor (Vacuum y la de fondo)
	OpenBatches       int64  // lotes abiertos
	OpenSnapshots     int64  // snapshots abiertos
	NextXID           uint64 // el próximo xid a asignar
	OldestXID         uint64 // el xid vivo más viejo (el horizonte de la limpieza); NextXID si no hay ninguno
	FrozenXID         uint64 // el frozenXID del motor: las xid menores están congeladas
	FrozenAge         uint64 // OldestXID − FrozenXID: cuántos xid esperan el congelado

	// Operaciones
	Reads   int64 // lecturas puntuales: Read, Exists, GetVersion y sus …In y …At (no los recorridos)
	Writes  int64 // escrituras aplicadas (sueltas y de lotes)
	Commits int64 // confirmaciones: cada escritura suelta aplicada y cada Commit de un lote que escribió

	// Disco lleno o lento
	ReadOnly  bool  // el motor quedó de solo lectura por un disco lleno: hay que cerrarlo y abrirlo
	DiskFull  int64 // operaciones rechazadas con ErrDiskFull sin tocar el motor (se pueden reintentar)
	Throttled int64 // escrituras frenadas: esperaron a un checkpoint porque el WAL pasaba de 2 × MAX_WAL_SIZE

	// Presión (Engine.Pressure)
	Pressure      float64       // qué tan cerca está de frenar a los escritores: 0 = nada, ≥ 1 = ya frena
	PressureCause PressureCause // la razón mayor: wal, deferred, dirty, read_only, failed (vacía si todas son 0)

	// Latencias
	CommitLatency Latency // confirmar: el registro (en un lote), la espera del fsync y el archivo de estados
	ReadLatency   Latency // una lectura puntual entera
	FsyncLatency  Latency // cada fsync del WAL del group commit
}

// Latency resume un histograma log2 de duraciones (store/internal/metric): P50 y P99 son el límite superior de su
// cubeta, con un error de hasta ×2.
type Latency struct {
	Count int64
	Mean  time.Duration
	P50   time.Duration
	P99   time.Duration
	Max   time.Duration
}

// engineMetrics son los contadores y latencias del motor que no tiene otra pieza. Tamaño fijo, solo atómicos: el
// camino caliente nunca toma un candado por ellos.
type engineMetrics struct {
	reads, commits atomic.Int64
	checkpoints    atomic.Int64
	checkpointTook atomic.Int64 // duración del último checkpoint, en ns
	vacuums        atomic.Int64
	vacuumWrites   atomic.Int64 // Engine.writes al empezar la última limpieza
	openBatches    atomic.Int64
	openSnapshots  atomic.Int64
	diskFull       atomic.Int64 // operaciones rechazadas con ErrDiskFull sin tocar el motor
	throttled      atomic.Int64 // escrituras que tuvieron que esperar a un checkpoint (throttle)
	commitLatency  metric.Histogram
	readLatency    metric.Histogram
}

/**
* latencyOf pasa la foto de un histograma a Latency.
* @param s metric.Summary
* @return Latency
**/
func latencyOf(s metric.Summary) Latency {
	return Latency{Count: s.Count, Mean: s.Mean, P50: s.P50, P99: s.P99, Max: s.Max}
}

/**
* observeRead anota una lectura puntual que empezó en start.
* @param start time.Time
**/
func (e *Engine) observeRead(start time.Time) {
	e.metrics.reads.Add(1)
	e.metrics.readLatency.Since(start)
}

/**
* observeCommit anota una confirmación que empezó en start.
* @param start time.Time
**/
func (e *Engine) observeCommit(start time.Time) {
	e.metrics.commits.Add(1)
	e.metrics.commitLatency.Since(start)
}

/**
* stats arma la foto del motor. Solo lee atómicos, salvo tres candados cortos, uno a la vez, que ya toman el
* checkpointer y la limpieza (el writeMu del WAL para el tamaño desde el punto de rehacer, el de las transacciones en
* curso para el xid más viejo y el del catálogo para el frozenXID). Sirve también con el motor cerrado (los últimos valores).
* @return Stats
**/
func (e *Engine) stats() Stats {
	m := &e.metrics
	cache := e.pool.Stats()
	oldest := e.txs.OldestActive()
	frozen := e.engineFrozen()
	st := Stats{
		CacheSize:    cache.Buffers,
		CachePages:   cache.Used,
		CacheDirty:   cache.Dirty,
		CacheHits:    cache.Hits,
		CacheMisses:  cache.Misses,
		CacheWritten: cache.Written,

		WALSinceCheckpoint: e.log.SizeSince(e.log.RedoPoint()),
		LSN:                uint64(e.log.Published()),
		DurableLSN:         uint64(e.log.Durable()),
		Checkpoints:        m.checkpoints.Load(),
		LastCheckpointTook: time.Duration(m.checkpointTook.Load()),

		WritesSinceVacuum: e.writes.Load() - m.vacuumWrites.Load(),
		Vacuums:           m.vacuums.Load(),
		OpenBatches:       m.openBatches.Load(),
		OpenSnapshots:     m.openSnapshots.Load(),
		NextXID:           uint64(e.txs.NextXID()),
		OldestXID:         uint64(oldest),
		FrozenXID:         uint64(frozen),

		Reads:   m.reads.Load(),
		Writes:  e.writes.Load(),
		Commits: m.commits.Load(),

		ReadOnly:  e.readOnly(),
		DiskFull:  m.diskFull.Load(),
		Throttled: m.throttled.Load(),

		CommitLatency: latencyOf(m.commitLatency.Summary()),
		ReadLatency:   latencyOf(m.readLatency.Summary()),
		FsyncLatency:  latencyOf(e.log.SyncLatency()),
	}
	st.Pressure, st.PressureCause = e.pressure()
	if oldest > frozen {
		st.FrozenAge = uint64(oldest - frozen)
	}
	if m.checkpoints.Load() > 0 {
		st.LastCheckpoint = time.Unix(0, e.lastCheckpoint.Load())
	}
	return st
}

/**
* onStats llama a fn con Stats cada interval, en una goroutine propia y sin ningún candado tomado, hasta stop o
* Close. Close espera a la llamada en curso y después no hay más: fn no debe llamar a Close (se esperaría a sí
* misma). Con el motor cerrado no arranca nada.
* @param interval time.Duration: 0 o menos = defaultStatsInterval, fn func(Stats)
* @return func(): stop, que se puede llamar varias veces y desde fn (no espera a la llamada en curso)
**/
func (e *Engine) onStats(interval time.Duration, fn func(Stats)) func() {
	if interval <= 0 {
		interval = defaultStatsInterval
	}
	e.closeMu.RLock()
	if e.closed {
		e.closeMu.RUnlock()
		return func() {}
	}
	e.bg.Add(1) // con closeMu en lectura: Close todavía no llegó a bg.Wait
	e.closeMu.RUnlock()

	quit := make(chan struct{})
	var once sync.Once
	go e.statsLoop(interval, fn, quit)
	return func() { once.Do(func() { close(quit) }) }
}

/**
* statsLoop es la goroutine de onStats: en cada vuelta mira antes si la detuvieron, así tras stop o Close no empieza
* otra llamada.
* @param interval time.Duration, fn func(Stats), quit chan struct{}: lo cierra stop
**/
func (e *Engine) statsLoop(interval time.Duration, fn func(Stats), quit chan struct{}) {
	defer e.bg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-quit:
			return
		case <-ticker.C:
		}
		select {
		case <-e.stop:
			return
		case <-quit:
			return
		default:
		}
		fn(e.stats())
	}
}
