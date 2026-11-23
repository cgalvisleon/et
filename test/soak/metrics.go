package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	logMu   sync.Mutex
	logFile *os.File
)

/**
* openLog abre el .log de la corrida (se agrega al final).
* @param path string
**/
func openLog(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	logFile = f
}

/**
* logf escribe una línea con la hora en el .log y en la salida.
* @param format string, args ...any
**/
func logf(format string, args ...any) {
	line := time.Now().Format("2006-01-02 15:04:05 ") + fmt.Sprintf(format, args...) + "\n"
	logMu.Lock()
	defer logMu.Unlock()
	os.Stdout.WriteString(line)
	if logFile != nil {
		logFile.WriteString(line)
	}
}

/**
* rssMB mide el RSS del proceso con ps (macOS y Linux), en MB; -1 si no pudo.
* @return float64
**/
func rssMB() float64 {
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return -1
	}
	kb, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return -1
	}
	return kb / 1024
}

const mb = 1 << 20

/**
* point es una medición tomada en una verificación (carga detenida, tras un GC).
**/
type point struct {
	at      time.Duration // desde el inicio
	rss     float64       // MB
	heap    float64       // MB: HeapAlloc tras GC
	data    float64       // MB: DirUsage().Data
	wal     float64       // MB: DirUsage().WAL
	xact    float64       // MB: DirUsage().Xact
	live    float64       // MB: ids + valores vivos según el modelo
	ratio   float64       // data / live
	open    time.Duration // Open de esta verificación (0 si no reabrió)
	verify  time.Duration // Engine.Verify
	frozen  uint64        // Stats.FrozenAge
	xidLag  uint64        // NextXID − OldestXID
	reopens int
}

/**
* metrics junta las muestras y escribe metrics.tsv, checks.tsv y summary.md.
**/
type metrics struct {
	r           *run
	start       time.Time
	points      []point
	reopens     int
	checks      int
	lastOps     int64
	lastAt      time.Time
	lastLive    float64
	tsv, chk    *os.File
	interrupted bool
}

/**
* newMetrics crea las salidas con su cabecera.
* @param r *run, start time.Time
* @return *metrics
**/
func newMetrics(r *run, start time.Time) *metrics {
	m := &metrics{r: r, start: start, lastAt: start}
	var err error
	if m.tsv, err = os.Create(filepath.Join(r.cfg.out, "metrics.tsv")); err == nil {
		fmt.Fprintln(m.tsv, "fecha\tseg\tops\tops_s\trss_mb\theap_mb\tdata_mb\twal_mb\ttotal_mb\tdata_vivos\twal_desde_ckpt_mb\tcheckpoints\tvacuums\tpag_borradas\tpag_recicladas\tlotes\tsnapshots\txid_atraso\tfrozen_age\tpresion\tcausa\tfrenadas\tcache_hit\tfsync_p99_ms\tcommit_p99_ms\tconflictos\tcommits")
	}
	if m.chk, err = os.Create(filepath.Join(r.cfg.out, "checks.tsv")); err == nil {
		fmt.Fprintln(m.chk, "fecha\tseg\treabrio\topen_ms\tverify_ms\trss_mb\theap_mb\tdata_mb\twal_mb\txact_mb\tvivos_mb\tdata_vivos\tfrozen_age\txid_atraso")
	}
	return m
}

/**
* sample escribe una fila de metrics.tsv sin detener la carga (los datos vivos son los de la última verificación).
**/
func (m *metrics) sample() {
	e := m.r.e
	st := e.Stats()
	u, _ := e.DirUsage()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	now := time.Now()
	ops := m.r.ops.Load()
	rate := float64(ops-m.lastOps) / now.Sub(m.lastAt).Seconds()
	m.lastOps, m.lastAt = ops, now
	hit := 0.0
	if st.CacheHits+st.CacheMisses > 0 {
		hit = float64(st.CacheHits) / float64(st.CacheHits+st.CacheMisses)
	}
	ratio := 0.0
	if m.lastLive > 0 {
		ratio = float64(u.Data) / mb / m.lastLive
	}
	if m.tsv != nil {
		fmt.Fprintf(m.tsv, "%s\t%d\t%d\t%.0f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.3f\t%.1f\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.3f\t%s\t%d\t%.4f\t%.2f\t%.2f\t%d\t%d\n",
			now.Format("2006-01-02 15:04:05"), int(now.Sub(m.start).Seconds()), ops, rate, rssMB(), float64(ms.HeapAlloc)/mb,
			float64(u.Data)/mb, float64(u.WAL)/mb, float64(u.Total)/mb, ratio, float64(st.WALSinceCheckpoint)/mb,
			st.Checkpoints, st.Vacuums, st.PagesDeleted, st.PagesRecycled, st.OpenBatches, st.OpenSnapshots,
			st.NextXID-st.OldestXID, st.FrozenAge, st.Pressure, st.PressureCause, st.Throttled, hit,
			float64(st.FsyncLatency.P99)/float64(time.Millisecond), float64(st.CommitLatency.P99)/float64(time.Millisecond),
			m.r.conflicts.Load(), m.r.commits.Load())
	}
	if st.ReadOnly {
		m.r.fail("el motor quedó de solo lectura (disco lleno)")
	}
}

/**
* verify detiene la carga y compara el motor con el modelo: si reopen, antes Close + Open; luego Count y recorrido
* de cada store, suma del banco, conjunto de stores, Verify limpio y una medición (tras GC).
* @param reopen bool
* @return error
**/
func (m *metrics) verify(reopen bool) error {
	r := m.r
	r.gate.Lock()
	defer r.gate.Unlock()
	m.checks++
	p := point{}
	if reopen {
		if err := r.e.Close(); err != nil {
			return fmt.Errorf("Close: %w", err)
		}
		t := time.Now()
		if err := r.open(); err != nil {
			return err
		}
		p.open = time.Since(t)
		m.reopens++
	}
	if err := m.compare(); err != nil {
		return err
	}
	t := time.Now()
	rep, err := r.e.Verify()
	p.verify = time.Since(t)
	if err != nil {
		return fmt.Errorf("Verify: %w", err)
	}
	if !rep.Clean() {
		return fmt.Errorf("Verify sucio:\n%s", rep.String())
	}

	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	u, err := r.e.DirUsage()
	if err != nil {
		return fmt.Errorf("DirUsage: %w", err)
	}
	st := r.e.Stats()
	_, liveBytes := r.model.liveStats()
	p.at = time.Since(m.start)
	p.rss, p.heap = rssMB(), float64(ms.HeapAlloc)/mb
	p.data, p.wal, p.xact = float64(u.Data)/mb, float64(u.WAL)/mb, float64(u.Xact)/mb
	p.live = float64(liveBytes) / mb
	p.ratio = p.data / p.live
	p.frozen, p.xidLag = st.FrozenAge, st.NextXID-st.OldestXID
	p.reopens = m.reopens
	m.lastLive = p.live
	m.points = append(m.points, p)
	if m.chk != nil {
		fmt.Fprintf(m.chk, "%s\t%d\t%v\t%d\t%d\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.3f\t%d\t%d\n", time.Now().Format("2006-01-02 15:04:05"),
			int(p.at.Seconds()), reopen, p.open.Milliseconds(), p.verify.Milliseconds(), p.rss, p.heap, p.data, p.wal, p.xact,
			p.live, p.ratio, p.frozen, p.xidLag)
	}
	logf("verificación %d (reabrió=%v open=%v verify=%v): ops=%d rss=%.0f MB heap=%.1f MB data=%.1f MB wal=%.0f MB vivos=%.1f MB data/vivos=%.2f frozen_age=%d vacuums=%d checkpoints=%d conflictos=%d",
		m.checks, reopen, p.open.Round(time.Millisecond), p.verify.Round(time.Millisecond), r.ops.Load(), p.rss, p.heap, p.data, p.wal,
		p.live, p.ratio, p.frozen, st.Vacuums, st.Checkpoints, r.conflicts.Load())
	if p.at >= r.cfg.warmup && p.ratio > r.cfg.maxRatio {
		return fmt.Errorf("data/ ÷ datos vivos = %.2f > %.2f (-max-ratio)", p.ratio, r.cfg.maxRatio)
	}
	if frozenMax := uint64(134217728); p.frozen > frozenMax {
		return fmt.Errorf("FrozenAge = %d > %d (un archivo de estados): el congelado no avanza", p.frozen, frozenMax)
	}
	return nil
}

/**
* compare compara cada store con el modelo (Count y recorrido completo), el banco y el conjunto de stores.
* @return error
**/
func (m *metrics) compare() error {
	r := m.r
	md := r.model
	for s, st := range r.data {
		want := md.liveCount(s)
		if c := st.Count(); c != want {
			return fmt.Errorf("Count %s = %d, el modelo %d", dataName(s), c, want)
		}
		var seen int64
		prev := -1
		err := st.ForEachBorrowed(func(id string, data []byte) (bool, error) {
			i, ok := indexOf(id)
			if !ok || i >= md.keys || i <= prev {
				return false, fmt.Errorf("id inesperado %q tras %d", id, prev)
			}
			prev = i
			seen++
			return true, md.expect(s, i, data, true)
		})
		if err != nil {
			return fmt.Errorf("recorrido de %s: %w", dataName(s), err)
		}
		if seen != want {
			return fmt.Errorf("recorrido de %s: %d ids, el modelo %d", dataName(s), seen, want)
		}
	}
	if err := auditBank(r, 0); err != nil {
		return err
	}
	want := []string{bankName}
	for s := range r.data {
		want = append(want, dataName(s))
	}
	if got := sortedNames(r.e.Stores()); strings.Join(got, ",") != strings.Join(sortedNames(want), ",") {
		return fmt.Errorf("Stores() = %v, se esperaba %v", got, sortedNames(want))
	}
	return nil
}

/**
* median da la mediana.
* @param xs []float64
* @return float64
**/
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

/**
* growth es la prueba de crecimiento de una métrica: mediana del primer tercio contra la del último tercio de las
* verificaciones tras el calentamiento. Falla si crece más de grow % y además más que floor (unidades de la métrica).
**/
type growth struct {
	name        string
	unit        string
	first, last float64
	pct, floor  float64
	fail        bool
}

/**
* growthChecks evalúa RSS, heap, data/vivos, wal/ y xact/; ok=false si hay menos de 6 verificaciones tras el
* calentamiento (sin datos suficientes, no es falla).
* @return []growth, bool
**/
func (m *metrics) growthChecks() ([]growth, bool) {
	var pts []point
	for _, p := range m.points {
		if p.at >= m.r.cfg.warmup {
			pts = append(pts, p)
		}
	}
	if len(pts) < 6 {
		return nil, false
	}
	third := len(pts) / 3
	cfg := m.r.cfg
	defs := []struct {
		name, unit string
		floor      float64
		get        func(point) float64
	}{
		{"RSS del proceso", "MB", cfg.floorRSSMB, func(p point) float64 { return p.rss }},
		{"heap de Go (tras GC)", "MB", cfg.floorHeapMB, func(p point) float64 { return p.heap }},
		{"data/ ÷ datos vivos", "×", 0.10, func(p point) float64 { return p.ratio }},
		{"wal/", "MB", cfg.floorWALMB, func(p point) float64 { return p.wal }},
		// xact/ se recorta por archivos enteros de 32 MB: el piso son dos archivos (uno llenándose y el que espera
		// al checkpoint), para no confundir el llenado de un archivo con una fuga.
		{"xact/", "MB", 64, func(p point) float64 { return p.xact }},
	}
	var out []growth
	for _, d := range defs {
		var a, b []float64
		for _, p := range pts[:third] {
			a = append(a, d.get(p))
		}
		for _, p := range pts[len(pts)-third:] {
			b = append(b, d.get(p))
		}
		g := growth{name: d.name, unit: d.unit, first: median(a), last: median(b), floor: d.floor}
		if g.first > 0 {
			g.pct = (g.last - g.first) / g.first * 100
		}
		g.fail = g.pct > cfg.growPct && g.last-g.first > d.floor
		out = append(out, g)
	}
	return out, true
}

/**
* summary escribe summary.md; running = la prueba sigue (resumen parcial). Devuelve si pasó.
* @param running bool
* @return bool
**/
func (m *metrics) summary(running bool) bool {
	r, cfg := m.r, m.r.cfg
	elapsed := time.Since(m.start)
	gs, enough := m.growthChecks()
	grew := false
	for _, g := range gs {
		grew = grew || g.fail
	}
	failure := r.failed()
	state := "OK"
	switch {
	case running:
		state = "EN CURSO"
	case failure != nil:
		state = "FALLA"
	case grew:
		state = "FALLA (crecimiento)"
	case m.interrupted:
		state = "INTERRUMPIDA"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# soak de et/store — %s\n\n", state)
	fmt.Fprintf(&b, "- inicio %s, transcurrido %v de %v (calentamiento %v)\n", m.start.Format("2006-01-02 15:04:05"), elapsed.Round(time.Second), cfg.duration, cfg.warmup)
	fmt.Fprintf(&b, "- carga: %d stores × %d claves, %d escritores, banco %d × %d cuentas, valor %d B (+%d %% de 4–24 KB), tope %d op/s, cache %d MB, maxwal %d MB, semilla %d\n",
		cfg.stores, cfg.keys, cfg.workers, cfg.bank, cfg.accounts, cfg.valueSize, cfg.largePct, cfg.rate, cfg.cacheMB, cfg.maxWALMB, cfg.seed)
	ops := r.ops.Load()
	fmt.Fprintf(&b, "- operaciones %d (%.0f/s), lotes confirmados %d, deshechos %d, choques del banco %d, auditorías %d, ciclos DDL %d, Empty con ErrBusy %d, frenadas %d\n",
		ops, float64(ops)/elapsed.Seconds(), r.commits.Load(), r.rollbacks.Load(), r.conflicts.Load(), r.audits.Load(), r.ddl.Load(), r.busy.Load(), r.throttled.Load())
	fmt.Fprintf(&b, "- verificaciones %d (todas contra el modelo + Verify limpio), reaperturas %d\n", m.checks, m.reopens)
	if failure != nil {
		fmt.Fprintf(&b, "\n**Falla:** %v\n", failure)
	}
	if n := len(m.points); n > 0 {
		var maxOpen, maxVerify time.Duration
		for _, p := range m.points {
			maxOpen, maxVerify = max(maxOpen, p.open), max(maxVerify, p.verify)
		}
		p := m.points[n-1]
		fmt.Fprintf(&b, "\nÚltima medición (%v): RSS %.0f MB, heap %.1f MB, data/ %.1f MB, wal/ %.0f MB, xact/ %.1f MB, vivos %.1f MB, data/vivos %.2f, FrozenAge %d, atraso de xid %d. Open máx %v, Verify máx %v.\n",
			p.at.Round(time.Second), p.rss, p.heap, p.data, p.wal, p.xact, p.live, p.ratio, p.frozen, p.xidLag, maxOpen.Round(time.Millisecond), maxVerify.Round(time.Millisecond))
	}
	fmt.Fprintf(&b, "\n## Crecimiento (mediana del primer tercio contra el último, tras el calentamiento; falla si > %.0f %% y > piso)\n\n", cfg.growPct)
	if !enough {
		b.WriteString("Sin datos suficientes (menos de 6 verificaciones tras el calentamiento).\n")
	} else {
		b.WriteString("| métrica | primer tercio | último tercio | cambio | piso | resultado |\n|---|---|---|---|---|---|\n")
		for _, g := range gs {
			res := "ok"
			if g.fail {
				res = "**CRECE**"
			}
			fmt.Fprintf(&b, "| %s | %.2f %s | %.2f %s | %+.1f %% | %.2f %s | %s |\n", g.name, g.first, g.unit, g.last, g.unit, g.pct, g.floor, g.unit, res)
		}
	}
	b.WriteString("\nDetalle: `soak.log`, `metrics.tsv` (cada -sample), `checks.tsv` (cada verificación).\n")
	os.WriteFile(filepath.Join(cfg.out, "summary.md"), []byte(b.String()), 0o644)
	if !running {
		logf("resultado: %s", state)
		if m.tsv != nil {
			m.tsv.Close()
		}
		if m.chk != nil {
			m.chk.Close()
		}
	}
	return state == "OK"
}
