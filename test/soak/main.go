// soak es la prueba de resistencia de et/store (tarea 51 de F-23): carga mixta continua durante horas o días con
// un modelo exacto de lo esperado, y cada cierto tiempo la carga se detiene para comparar el motor con el modelo,
// pasar Verify, cerrar y abrir, y medir memoria y disco. Solo usa la API pública de store.
//
// Carga (todo a la vez, hasta -duration):
//   - escritores de datos (-workers) sobre -stores stores de -keys claves: Put, Put diferido, Insert, Update,
//     Delete, Remove, Read, GetVersion/Exists, ForEachRange, KeysPrefix, lotes confirmados (Commit o
//     CommitDeferred) y deshechos (Rollback), y snapshots que no deben ver lo escrito después. Cada clave tiene un
//     solo dueño, así el modelo (generación por clave; el valor se deriva de ella) es exacto sin candados;
//   - banco: -bank escritores transfieren entre -accounts cuentas en lotes que chocan (ErrSerialization o
//     ErrConflict: se deshacen y se cuentan); un auditor suma las cuentas en un snapshot cada 2 s (debe dar
//     siempre lo mismo) y cada -long-every mantiene uno abierto -long-hold (retiene la limpieza);
//   - DDL cada -ddl-every: stores creados, llenados, vaciados (Empty) y borrados, sueltos y en lotes;
//   - la limpieza y los checkpoints son los automáticos del motor.
//
// Cada -check: con la carga detenida, Count y un recorrido completo de cada store contra el modelo, la suma del
// banco, el conjunto de stores y Engine.Verify limpio; luego GC y una muestra de memoria y disco. Cada -reopen,
// además Close + Open. Cada -sample, una fila de métricas sin detener nada. Al final, la prueba de crecimiento
// (ver growth.go). Sale 0 si todo bien, 1 si falla.
//
//	go run ./test/soak -dir /tmp/soak/db -out /tmp/soak -duration 10m -check 1m -reopen 3m -warmup 2m
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/cgalvisleon/et/store"
)

/**
* config son las opciones de la prueba (flags).
**/
type config struct {
	dir, out                            string
	duration, warmup                    time.Duration
	sample, check, reopen               time.Duration
	ddlEvery, longEvery, longHold       time.Duration
	stores, keys, workers, bank         int
	accounts, valueSize, largePct       int
	rate                                int
	cacheMB, maxWALMB                   int
	seed                                uint64
	growPct, maxRatio                   float64
	floorRSSMB, floorHeapMB, floorWALMB float64
	fresh                               bool
}

/**
* parseFlags lee los flags.
* @return config
**/
func parseFlags() config {
	var c config
	flag.StringVar(&c.dir, "dir", "", "directorio del motor (se borra al empezar si -fresh)")
	flag.StringVar(&c.out, "out", ".", "carpeta de salida: soak.log, metrics.tsv, checks.tsv, summary.md")
	flag.DurationVar(&c.duration, "duration", 72*time.Hour, "cuánto dura la carga")
	flag.DurationVar(&c.warmup, "warmup", 0, "calentamiento sin evaluar crecimiento (0 = la sexta parte de -duration, hasta 6 h)")
	flag.DurationVar(&c.sample, "sample", time.Minute, "cada cuánto una fila de metrics.tsv (sin detener la carga)")
	flag.DurationVar(&c.check, "check", 10*time.Minute, "cada cuánto se compara con el modelo y se pasa Verify (carga detenida)")
	flag.DurationVar(&c.reopen, "reopen", 30*time.Minute, "cada cuánto, además, Close + Open")
	flag.DurationVar(&c.ddlEvery, "ddl-every", 20*time.Second, "cada cuánto un ciclo de DDL")
	flag.DurationVar(&c.longEvery, "long-every", 5*time.Minute, "cada cuánto el auditor mantiene un snapshot largo")
	flag.DurationVar(&c.longHold, "long-hold", 10*time.Second, "cuánto dura el snapshot largo")
	flag.IntVar(&c.stores, "stores", 4, "stores de datos")
	flag.IntVar(&c.keys, "keys", 50000, "claves posibles por store de datos")
	flag.IntVar(&c.workers, "workers", 4, "escritores de datos")
	flag.IntVar(&c.bank, "bank", 3, "escritores del banco")
	flag.IntVar(&c.accounts, "accounts", 32, "cuentas del banco (pocas = más choques)")
	flag.IntVar(&c.valueSize, "value", 256, "tamaño medio de un valor normal (±25 %)")
	flag.IntVar(&c.largePct, "large", 5, "% de valores grandes, de 4 a 24 KB (cadenas de desborde)")
	flag.IntVar(&c.rate, "rate", 2000, "operaciones por segundo entre todos los escritores (0 = sin tope)")
	flag.IntVar(&c.cacheMB, "cache", 0, "STORE_CACHE_SIZE en MB (0 = la variable de entorno)")
	flag.IntVar(&c.maxWALMB, "maxwal", 0, "MAX_WAL_SIZE en MB (0 = la variable de entorno)")
	flag.Uint64Var(&c.seed, "seed", 1, "semilla")
	flag.Float64Var(&c.growPct, "grow", 20, "crecimiento máximo admitido (%) entre el primer y el último tercio tras el calentamiento")
	flag.Float64Var(&c.maxRatio, "max-ratio", 3, "máximo de disco de data/ ÷ datos vivos en cualquier verificación tras el calentamiento")
	flag.Float64Var(&c.floorRSSMB, "floor-rss", 64, "crecimiento de RSS por debajo del cual no se falla (MB)")
	flag.Float64Var(&c.floorHeapMB, "floor-heap", 16, "crecimiento del heap de Go por debajo del cual no se falla (MB)")
	flag.Float64Var(&c.floorWALMB, "floor-wal", 0, "crecimiento de wal/ por debajo del cual no se falla (MB; 0 = MAX_WAL_SIZE: el WAL oscila hasta ahí entre checkpoints)")
	flag.BoolVar(&c.fresh, "fresh", true, "borrar -dir al empezar")
	flag.Parse()
	if c.dir == "" {
		fmt.Fprintln(os.Stderr, "falta -dir")
		os.Exit(2)
	}
	if c.warmup == 0 {
		c.warmup = min(c.duration/6, 6*time.Hour)
	}
	if c.floorWALMB == 0 {
		c.floorWALMB = float64(c.maxWALMB)
		if c.floorWALMB == 0 {
			c.floorWALMB = float64(envInt("MAX_WAL_SIZE", 1024))
		}
	}
	if c.accounts < 2 {
		c.accounts = 2
	}
	return c
}

/**
* envInt lee una variable entera del entorno.
* @param name string, def int
* @return int
**/
func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

func main() {
	cfg := parseFlags()
	if err := os.MkdirAll(cfg.out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	openLog(filepath.Join(cfg.out, "soak.log"))
	r := &run{cfg: cfg, stop: make(chan struct{}), model: newModel(cfg.seed, cfg.stores, cfg.keys, cfg.valueSize, cfg.largePct)}
	ok := soak(r)
	if !ok {
		os.Exit(1)
	}
}

/**
* options da las opciones del motor según los flags (cero = la variable de entorno).
* @return store.Options
**/
func (r *run) options() store.Options {
	return store.Options{CacheMB: r.cfg.cacheMB, MaxWALMB: r.cfg.maxWALMB}
}

/**
* open abre el motor y sus stores fijos (con la carga detenida).
* @return error
**/
func (r *run) open() error {
	e, err := store.OpenWith(r.cfg.dir, r.options())
	if err != nil {
		return fmt.Errorf("Open: %w", err)
	}
	r.e = e
	r.data = make([]*store.Store, r.cfg.stores)
	for s := range r.data {
		if r.data[s], err = e.Store(dataName(s)); err != nil {
			return fmt.Errorf("Store %s: %w", dataName(s), err)
		}
	}
	r.bank, err = e.Store(bankName)
	if err != nil {
		return fmt.Errorf("Store %s: %w", bankName, err)
	}
	return nil
}

/**
* setup crea el motor vacío, abre las cuentas y precarga el 70 % de las claves (lotes diferidos + WaitDurable), así
* el régimen llega antes.
* @return error
**/
func (r *run) setup() error {
	if r.cfg.fresh {
		if err := os.RemoveAll(r.cfg.dir); err != nil {
			return err
		}
	}
	if err := r.open(); err != nil {
		return err
	}
	if len(r.e.Stores()) != r.cfg.stores+1 || r.bank.Count() != 0 {
		return fmt.Errorf("el directorio %s no estaba vacío (usa -fresh)", r.cfg.dir)
	}
	tx := r.e.NewBatch()
	for a := 0; a < r.cfg.accounts; a++ {
		if err := r.bank.PutIn(tx, accountOf(a), balance(initialBalance)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	rnd := rand.New(rand.NewPCG(r.cfg.seed, 99))
	m := r.model
	for s, st := range r.data {
		tx := r.e.NewBatch()
		n := 0
		for i := 0; i < m.keys; i++ {
			if rnd.IntN(10) >= 7 {
				continue
			}
			if err := st.PutIn(tx, keyOf(i), m.value(s, i, 1)); err != nil {
				return err
			}
			m.gen[s][i], m.live[s][i] = 1, true
			if n++; n%500 == 0 {
				if err := tx.CommitDeferred(); err != nil {
					return err
				}
				tx = r.e.NewBatch()
			}
		}
		if err := tx.CommitDeferred(); err != nil {
			return err
		}
	}
	return r.e.WaitDurable()
}

/**
* soak corre la prueba completa y escribe el resumen.
* @param r *run
* @return bool
**/
func soak(r *run) bool {
	cfg := r.cfg
	start := time.Now()
	logf("soak: dir=%s duración=%v calentamiento=%v check=%v reopen=%v stores=%d×%d claves workers=%d banco=%d×%d cuentas valor=%d B (+%d %% grandes) rate=%d/s cache=%d MB maxwal=%d MB semilla=%d",
		cfg.dir, cfg.duration, cfg.warmup, cfg.check, cfg.reopen, cfg.stores, cfg.keys, cfg.workers, cfg.bank, cfg.accounts, cfg.valueSize, cfg.largePct, cfg.rate, cfg.cacheMB, cfg.maxWALMB, cfg.seed)
	met := newMetrics(r, start)
	if err := r.setup(); err != nil {
		r.fail("preparación: %v", err)
		met.summary(false)
		return false
	}
	n, b := r.model.liveStats()
	logf("precarga: %d claves vivas, %d MB", n, b>>20)
	if err := met.verify(false); err != nil {
		r.fail("%v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	interrupted := false

	var wg sync.WaitGroup
	total := cfg.workers + cfg.bank
	var interval time.Duration
	if cfg.rate > 0 {
		interval = time.Second * time.Duration(total) / time.Duration(cfg.rate)
	}
	for w := 0; w < cfg.workers; w++ {
		wr := &writer{r: r, id: w, n: cfg.workers, rnd: rand.New(rand.NewPCG(cfg.seed, uint64(w)))}
		wg.Add(1)
		go func() { defer wg.Done(); wr.loop(&pacer{interval: interval}) }()
	}
	for b := 0; b < cfg.bank; b++ {
		wg.Add(1)
		go func(id int) { defer wg.Done(); bankLoop(r, id, &pacer{interval: interval}) }(b)
	}
	wg.Add(2)
	go func() { defer wg.Done(); auditLoop(r) }()
	go func() { defer wg.Done(); ddlLoop(r) }()

	end := time.After(cfg.duration)
	sample := time.NewTicker(cfg.sample)
	check := time.NewTicker(cfg.check)
	defer sample.Stop()
	defer check.Stop()
	lastReopen := time.Now()
loop:
	for {
		select {
		case <-end:
			break loop
		case s := <-sig:
			logf("señal %v: termino", s)
			interrupted = true
			break loop
		case <-r.stop:
			break loop
		case <-sample.C:
			met.sample()
		case <-check.C:
			reopen := time.Since(lastReopen) >= cfg.reopen-time.Second
			if reopen {
				lastReopen = time.Now()
			}
			if err := met.verify(reopen); err != nil {
				r.fail("%v", err)
				break loop
			}
			met.summary(true)
		}
	}
	r.halt()
	wg.Wait()
	if r.failed() == nil {
		if err := met.verify(true); err != nil { // última: Close + Open, modelo y Verify
			r.fail("%v", err)
		}
	}
	if r.e != nil {
		if err := r.e.Close(); err != nil && r.failed() == nil {
			r.fail("Close: %v", err)
		}
	}
	met.interrupted = interrupted
	return met.summary(false)
}
