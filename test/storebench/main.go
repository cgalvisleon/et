// storebench comprueba las dos reglas de et/store (F-23) a la escala que se le pida:
//
//  1. la memoria no crece con el número de claves (solo la caché de páginas, STORE_CACHE_SIZE);
//  2. Open no tarda más con más claves (solo rehace el WAL desde el último checkpoint).
//
// Carga N claves en lotes, cierra, vuelve a abrir midiendo Open, lee claves al azar (frío) y repite unas pocas
// (en caché) para la mediana. Usa un directorio temporal que borra al terminar, salvo con -dir.
//
//	go run ./test/storebench -n 1000000
//	go run ./test/storebench -n 10000000 -batch 20000
//	go run ./test/storebench -n 1000000 -reopen=false   # un solo motor: el RSS es el de una apertura
//	go run ./test/storebench -n 1000000 -dir d -load=false # reabre y lee lo ya cargado en d (repite la medida de Open)
//	go run ./test/storebench -n 0 -rate 2s              # solo Put/s durable vs diferido con 1, 8 y 64 escritores
//	go run ./test/storebench -n 0 -rate 2s -writers 1,8 # otros números de escritores
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cgalvisleon/et/store"
)

func main() {
	n := flag.Int("n", 1_000_000, "claves a cargar")
	batch := flag.Int("batch", 10_000, "claves por lote al cargar")
	size := flag.Int("value", 100, "bytes por valor")
	dir := flag.String("dir", "", "directorio del motor (vacío = uno temporal que se borra)")
	reopen := flag.Bool("reopen", true, "cerrar, volver a abrir y leer (false = solo cargar)")
	doLoad := flag.Bool("load", true, "cargar las n claves (false = usar las que ya hay en -dir)")
	rate := flag.Duration("rate", 0, "si no es 0, mide Put/s durable y diferido durante ese tiempo por medida")
	writers := flag.String("writers", "1,8,64", "escritores de cada medida de -rate, separados por coma")
	flag.Parse()

	path := *dir
	if path == "" {
		tmp, err := os.MkdirTemp("", "storebench-")
		check(err)
		defer os.RemoveAll(tmp)
		path = tmp
	}

	if *rate > 0 {
		writeRates(path, *rate, *size, parseWriters(*writers))
	}
	if *n <= 0 {
		return
	}
	if *doLoad {
		start := time.Now()
		load(path, *n, *batch, *size)
		fmt.Printf("carga: %d claves en %v (%.0f claves/s)\n", *n, since(start), float64(*n)/time.Since(start).Seconds())
		fmt.Printf("disco: %d MB\n", dirSize(path)>>20)
	}
	if !*reopen {
		return
	}

	start := time.Now()
	engine, err := store.Open(path)
	check(err)
	openTime := time.Since(start)
	keys, err := engine.Store("keys")
	check(err)
	fmt.Printf("Open tras Close: %v (Count %d)\n", openTime, keys.Count())

	cold := readRandom(keys, *n, 100_000)
	fmt.Printf("lectura al azar (100.000, fría al principio): p50 %v p99 %v p999 %v\n", cold.p50, cold.p99, cold.p999)
	warm := readHot(keys, 1_000, 200_000)
	fmt.Printf("lectura en caché (1.000 claves × 200): p50 %v p99 %v p999 %v\n", warm.p50, warm.p99, warm.p999)
	report("tras abrir y leer")
	check(engine.Close())
}

/**
* load carga n claves k0000000000… en lotes de batch, con valores de size bytes, y cierra (checkpoint final).
* @param path string, n, batch, size int
**/
func load(path string, n, batch, size int) {
	engine, err := store.Open(path)
	check(err)
	keys, err := engine.Store("keys")
	check(err)
	value := make([]byte, size)
	for from := 0; from < n; from += batch {
		tx := engine.NewBatch()
		for i := from; i < min(from+batch, n); i++ {
			check(keys.PutIn(tx, key(i), value))
		}
		check(tx.Commit())
		if from > 0 && from%(n/10+1) < batch {
			fmt.Printf("  %d%%\n", from*100/n)
		}
	}
	report("tras cargar")
	check(engine.Close())
}

/**
* writeRates mide Put sueltos por segundo, durables (esperan su fsync, group commit) y diferidos (Deferred, con un
* WaitDurable al final que entra en la cuenta), con cada número de escritores de counts, cada medida en su store.
* @param path string, d time.Duration: cuánto dura cada medida, size int: bytes por valor, counts []int: escritores
**/
func writeRates(path string, d time.Duration, size int, counts []int) {
	engine, err := store.Open(path)
	check(err)
	for _, writers := range counts {
		for _, deferred := range []bool{false, true} {
			s, err := engine.Store(fmt.Sprintf("rate-%d-%v", writers, deferred))
			check(err)
			ops, took := writeFor(engine, s, writers, deferred, d, size)
			mode := "durable"
			if deferred {
				mode = "diferido"
			}
			fmt.Printf("Put %s con %d escritores: %.0f ops/s\n", mode, writers, float64(ops)/took.Seconds())
		}
	}
	check(engine.Close())
}

/**
* writeFor escribe con writers goroutines durante d y devuelve cuántos Put hizo y cuánto tardó (con WaitDurable).
* @param engine *store.Engine, s *store.Store, writers int, deferred bool, d time.Duration, size int
* @return int64, time.Duration
**/
func writeFor(engine *store.Engine, s *store.Store, writers int, deferred bool, d time.Duration, size int) (int64, time.Duration) {
	var ops atomic.Int64
	var wg sync.WaitGroup
	value := make([]byte, size)
	start := time.Now()
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; time.Since(start) < d; i++ {
				id := fmt.Sprintf("w%d-%d", w, i)
				if deferred {
					check(s.Deferred().Put(id, value))
				} else {
					check(s.Put(id, value))
				}
				ops.Add(1)
			}
		}()
	}
	wg.Wait()
	check(engine.WaitDurable())
	return ops.Load(), time.Since(start)
}

// latency es el resumen de una serie de lecturas.
type latency struct{ p50, p99, p999 time.Duration }

/**
* readRandom lee count claves al azar entre las n cargadas.
* @param s *store.Store, n, count int
* @return latency
**/
func readRandom(s *store.Store, n, count int) latency {
	times := make([]time.Duration, 0, count)
	for range count {
		id := key(rand.IntN(n))
		start := time.Now()
		_, found, err := s.Read(id)
		times = append(times, time.Since(start))
		check(err)
		if !found {
			log.Fatalf("falta %s", id)
		}
	}
	return summarize(times)
}

/**
* readHot lee una y otra vez las mismas hot claves (quedan en la caché) count veces en total.
* @param s *store.Store, hot, count int
* @return latency
**/
func readHot(s *store.Store, hot, count int) latency {
	for i := range hot {
		_, _, err := s.Read(key(i))
		check(err)
	}
	times := make([]time.Duration, 0, count)
	for i := range count {
		id := key(i % hot)
		start := time.Now()
		_, _, err := s.Read(id)
		times = append(times, time.Since(start))
		check(err)
	}
	return summarize(times)
}

/**
* summarize da la mediana y los percentiles 99 y 99,9.
* @param times []time.Duration
* @return latency
**/
func summarize(times []time.Duration) latency {
	slices.Sort(times)
	n := len(times)
	return latency{p50: times[n/2], p99: times[n*99/100], p999: times[n*999/1000]}
}

/**
* parseWriters lee la lista de escritores de -writers ("1,8,64").
* @param list string
* @return []int
**/
func parseWriters(list string) []int {
	var counts []int
	for _, part := range strings.Split(list, ",") {
		w, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || w < 1 {
			log.Fatalf("-writers: %q no es un número de escritores", part)
		}
		counts = append(counts, w)
	}
	return counts
}

/**
* report imprime el heap vivo tras un GC, el RSS actual y el RSS máximo del proceso.
* @param when string
**/
func report(when string) {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	var ru syscall.Rusage
	check(syscall.Getrusage(syscall.RUSAGE_SELF, &ru))
	maxRSS := ru.Maxrss // bytes en macOS, KB en Linux
	if runtime.GOOS == "linux" {
		maxRSS *= 1024
	}
	fmt.Printf("memoria %s: heap vivo %d MB, RSS actual %d MB, RSS máximo %d MB\n", when, m.HeapAlloc>>20,
		currentRSS()>>20, maxRSS>>20)
}

/**
* currentRSS da el RSS actual del proceso: en linux de /proc/self/status (VmRSS), si no preguntando a ps (en KB).
* @return int64: bytes; 0 si no se pudo leer
**/
func currentRSS() int64 {
	if status, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(status), "\n") {
			if rest, ok := strings.CutPrefix(line, "VmRSS:"); ok {
				kb, _ := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "kB")), 10, 64)
				return kb << 10
			}
		}
	}
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		return 0
	}
	kb, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return kb << 10
}

/**
* dirSize suma los bytes de los archivos bajo path.
* @param path string
* @return int64
**/
func dirSize(path string) int64 {
	var total int64
	entries, err := os.ReadDir(path)
	check(err)
	for _, e := range entries {
		p := path + "/" + e.Name()
		if e.IsDir() {
			total += dirSize(p)
			continue
		}
		info, err := e.Info()
		check(err)
		total += info.Size()
	}
	return total
}

func key(i int) string { return fmt.Sprintf("k%010d", i) }

func since(t time.Time) time.Duration { return time.Since(t).Round(time.Millisecond) }

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
