package crashfs

import (
	"maps"
	"math/rand/v2"
	"slices"
)

// Crash simula un corte (ver FS.Crash y el modelo en doc.go).
func (m *memFS) Crash(mode Mode, seed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := newCrash(mode, seed)
	done := map[*inode]bool{}
	// Se recorre en orden fijo para que la misma seed dé el mismo resultado.
	for _, name := range slices.Sorted(maps.Keys(m.dirs)) {
		d := m.dirs[name]
		d.crash(c.dirPrefix(len(d.pending)))
		for _, entry := range slices.Sorted(maps.Keys(d.live)) {
			node := d.live[entry]
			if !done[node] {
				node.reset(c.survivor(node))
				done[node] = true
			}
		}
	}

	m.epoch++
	m.locks = map[string]bool{}
	m.failAfter = -1
	m.failSyncAfter = -1
}

// crash son las decisiones al azar de un corte.
type crash struct {
	mode   Mode
	rng    *rand.Rand
	keep   float64 // probabilidad de que sobreviva cada operación o sector no durable
	sector int64
}

// newCrash prepara un corte reproducible a partir de seed.
func newCrash(mode Mode, seed int64) *crash {
	c := &crash{mode: mode, rng: rand.New(rand.NewPCG(uint64(seed), 0x63726173686673)), sector: SectorSize}
	if mode != LoseUnsynced {
		c.keep = c.rng.Float64()
	}
	if mode == TornWrites && c.rng.IntN(2) == 1 {
		c.sector = LargeSectorSize
	}
	return c
}

// lucky dice si una operación o un sector no durable sobrevive.
func (c *crash) lucky() bool {
	return c.mode != LoseUnsynced && c.rng.Float64() < c.keep
}

// dirPrefix dice cuántos de los n cambios pendientes de un directorio sobreviven.
func (c *crash) dirPrefix(n int) int {
	if c.mode == LoseUnsynced || n == 0 {
		return 0
	}
	return c.rng.IntN(n + 1)
}

// survivor calcula el contenido de un archivo tras el corte: lo durable más lo que sobreviva de lo pendiente.
func (c *crash) survivor(node *inode) []byte {
	data := append([]byte(nil), node.durable...)
	for _, op := range node.pending {
		if c.mode != TornWrites || op.truncate {
			if c.lucky() {
				data = op.apply(data)
			}
			continue
		}
		for _, piece := range op.sectors(c.sector) {
			if c.lucky() {
				data = piece.apply(data)
			}
		}
	}
	return data
}
