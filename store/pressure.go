package store

// Presión del motor (ver «Presión y uso de disco» en doc.go).

// PressureCause es la razón que domina en Pressure: la que está más cerca de frenar a los escritores.
type PressureCause string

const (
	PressureNone     PressureCause = ""          // ninguna razón pesa (todas en 0)
	PressureWAL      PressureCause = "wal"       // el WAL sin reciclar frente al tope de throttle
	PressureDeferred PressureCause = "deferred"  // los registros sin fsync frente al tope de deferredBrake
	PressureDirty    PressureCause = "dirty"     // las páginas sucias frente al tamaño de la caché
	PressureReadOnly PressureCause = "read_only" // el motor quedó de solo lectura por un disco lleno
	PressureFailed   PressureCause = "failed"    // el motor quedó inutilizable por otro fallo
)

/**
* pressure dice qué tan cerca está el motor de frenar a los escritores: el máximo de tres razones (WAL sin reciclar /
* walBacklogFactor × MAX_WAL_SIZE, registros sin fsync / maxDeferredLag, páginas sucias / búferes de la caché), o 1 si
* el motor quedó de solo lectura o inutilizable. Solo lee atómicos (y failMu, una hoja): no frena a nadie.
* @return float64: 0 = sin presión, ≥ 1 = ya frena; PressureCause: la razón mayor
**/
func (e *Engine) pressure() (float64, PressureCause) {
	if e.failed() != nil {
		if e.readOnly() {
			return 1, PressureReadOnly
		}
		return 1, PressureFailed
	}
	value, cause := 0.0, PressureNone
	consider := func(ratio float64, c PressureCause) {
		if ratio > value {
			value, cause = ratio, c
		}
	}
	consider(ratioOf(e.walBacklog(), e.cfg.maxWALSize*walBacklogFactor), PressureWAL)
	consider(ratioOf(int64(e.log.Published())-int64(e.log.Durable()), maxDeferredLag), PressureDeferred)
	cache := e.pool.Stats()
	consider(ratioOf(cache.Dirty, int64(cache.Buffers)), PressureDirty)
	return value, cause
}

/**
* ratioOf es n / limit, 0 si alguno no es positivo.
* @param n, limit int64
* @return float64
**/
func ratioOf(n, limit int64) float64 {
	if n <= 0 || limit <= 0 {
		return 0
	}
	return float64(n) / float64(limit)
}
