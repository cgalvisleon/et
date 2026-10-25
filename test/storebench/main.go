package main

/**
* Carga concurrente directa sobre et/store (sin HTTP): -w workers en lazo
* cerrado, -r de ellos leen (Get al azar) y el resto inserta, durante -d.
*
*	SYNC_ON_WRITE=true go run ./test/storebench -d 30s -w 1000 -r 0
*
* -think pausa a cada lector entre lecturas (como peticiones que llegan por la red).
* PROF=<archivo> guarda un perfil de CPU.
**/

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store"
)

func main() {
	dur := flag.Duration("d", 30*time.Second, "")
	workers := flag.Int("w", 1000, "")
	readers := flag.Int("r", -1, "lectores (-1 = la mitad)")
	think := flag.Duration("think", 0, "pausa de cada lector entre lecturas (simula la red)")
	flag.Parse()
	if *readers < 0 {
		*readers = *workers / 2
	}
	dir, _ := os.MkdirTemp("", "storebench")
	defer os.RemoveAll(dir)
	s, err := store.Open(dir+"/d", dir+"/w", "bench", store.ReadWrite)
	if err != nil {
		panic(err)
	}
	data := []byte(`{"n":1,"payload":"` + strings.Repeat("x", 200) + `"}`)
	var seq atomic.Int64
	for range 10000 {
		s.Insert(fmt.Sprintf("b%012d", seq.Add(1)), data)
	}
	if path := os.Getenv("PROF"); path != "" {
		pf, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(pf)
		defer pprof.StopCPUProfile()
	}
	var mu sync.Mutex
	var rl, wl []int64
	stop := time.Now().Add(*dur)
	var wg sync.WaitGroup
	for w := range *workers {
		wg.Go(func() {
			var loc []int64
			for time.Now().Before(stop) {
				t := time.Now()
				if w < *readers {
					s.Get(fmt.Sprintf("b%012d", 1+rand.Int64N(seq.Load())))
				} else {
					s.Insert(fmt.Sprintf("b%012d", seq.Add(1)), data)
				}
				loc = append(loc, time.Since(t).Microseconds())
				if *think > 0 && w < *readers {
					time.Sleep(*think)
				}
			}
			mu.Lock()
			if w < *readers {
				rl = append(rl, loc...)
			} else {
				wl = append(wl, loc...)
			}
			mu.Unlock()
		})
	}
	wg.Wait()
	s.Close()
	for _, x := range []struct {
		n string
		l []int64
	}{{"read", rl}, {"write", wl}} {
		if len(x.l) == 0 {
			continue
		}
		slices.Sort(x.l)
		var sum int64
		for _, v := range x.l {
			sum += v
		}
		p := func(q float64) float64 { return float64(x.l[int(q*float64(len(x.l)-1))]) / 1000 }
		fmt.Printf("%-5s ops/s %8.0f avg %8.3f ms p50 %8.3f p99 %8.3f max %8.3f\n", x.n, float64(len(x.l))/dur.Seconds(), float64(sum)/float64(len(x.l))/1000, p(.5), p(.99), p(1))
	}
}
