// Ejemplo de uso de et/store: abre un store, guarda registros con Put (uno por uno y desde varias goroutines),
// lo cierra, lo vuelve a abrir y muestra los archivos de segmento que quedaron en el directorio.
//
//	go run ./cmd/store                      # en un directorio temporal que se borra al terminar
//	go run ./cmd/store -dir ./data -n 50000 # conserva los datos en ./data
//	SYNC_ON_WRITE=false go run ./cmd/store  # sin esperar el fsync en cada escritura
//
// Por ahora el store solo tiene Open, Put y Close (F-23); Read y los demás llegan en las fases siguientes.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/store"
)

func main() {
	dir := flag.String("dir", "", "directorio del store (vacío: uno temporal que se borra al terminar)")
	n := flag.Int("n", 10000, "registros que escriben las goroutines en total")
	writers := flag.Int("w", 8, "goroutines que escriben a la vez")
	size := flag.Int("size", 100, "bytes de cada valor")
	flag.Parse()

	path, cleanup := storeDir(*dir)
	defer cleanup()

	s := open(path)
	putOne(s)
	putParallel(s, *n, *writers, *size)
	closeStore(s)

	// Al reabrir, el store rehace su índice recorriendo los segmentos.
	start := time.Now()
	s = open(path)
	fmt.Printf("\nReabierto en %v\n", time.Since(start).Round(time.Microsecond))
	closeStore(s)

	listSegments(path)
}

/**
* storeDir: El directorio del store: el pedido, o uno temporal que se borra al terminar.
* @param dir string
* @return string, func() (limpieza)
**/
func storeDir(dir string) (string, func()) {
	if dir != "" {
		return dir, func() {}
	}
	tmp, err := os.MkdirTemp("", "et-store-")
	if err != nil {
		log.Fatal(err)
	}
	return tmp, func() { os.RemoveAll(tmp) }
}

/**
* open: Abre el store o termina el programa.
* @param path string
* @return *store.Store
**/
func open(path string) *store.Store {
	s, err := store.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

/**
* closeStore: Cierra el store o termina el programa.
* @param s *store.Store
**/
func closeStore(s *store.Store) {
	if err := s.Close(); err != nil {
		log.Fatal(err)
	}
}

/**
* putOne: Put sobre un id nuevo y sobre el mismo id otra vez: el segundo lo reemplaza sin leer lo que había.
* @param s *store.Store
**/
func putOne(s *store.Store) {
	fmt.Println("Put de un registro y su reemplazo")
	for _, v := range []string{`{"name":"Ana"}`, `{"name":"Ana María"}`} {
		if err := s.Put("customer:001", []byte(v)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  customer:001 ← %s\n", v)
	}

	fmt.Println("Errores que Put rechaza")
	if err := s.Put("", []byte("x")); err != nil {
		fmt.Printf("  id vacío: %v\n", err)
	}
}

/**
* putParallel: n registros de size bytes escritos por writers goroutines a la vez; muestra el tiempo y las ops/s.
* @param s *store.Store, n, writers, size int
**/
func putParallel(s *store.Store, n, writers, size int) {
	fmt.Printf("\n%d Put de %d bytes desde %d goroutines (SYNC_ON_WRITE=%s)\n", n, size, writers, syncOnWrite())
	value := bytes.Repeat([]byte("v"), size)
	start := time.Now()
	var wg sync.WaitGroup
	for w := range writers {
		wg.Go(func() {
			for i := w; i < n; i += writers {
				if err := s.Put(fmt.Sprintf("item:%08d", i), value); err != nil {
					log.Fatal(err)
				}
			}
		})
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("  %v, %.0f ops/s\n", elapsed.Round(time.Millisecond), float64(n)/elapsed.Seconds())
}

/**
* syncOnWrite: El valor efectivo de SYNC_ON_WRITE, para mostrarlo.
* @return string
**/
func syncOnWrite() string {
	if v := os.Getenv("SYNC_ON_WRITE"); v != "" {
		return v
	}
	return "true"
}

/**
* listSegments: Muestra los archivos del directorio del store y su tamaño.
* @param path string
**/
func listSegments(path string) {
	fmt.Printf("\nArchivos en %s\n", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		info, err := os.Stat(filepath.Join(path, e.Name()))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %-18s %10d bytes\n", e.Name(), info.Size())
	}
}
