// powercut prueba et/store con fallos de disco reales (tareas 33 y 50 de F-23). Lo lanzan los scripts de
// test/linuxvm dentro de la VM Linux: powercut-lazyfs.sh (cortes de energía con LazyFS), powercut-flakey.sh
// (escrituras perdidas y errores de E/S con dm-flakey) y diskfull.sh (un loop ext4 chico que se llena).
//
// Cada escritor tiene su store y anota en un registro fuera del disco que se corta cada intento (I), cada
// operación confirmada y durable (C: Put o Commit que volvió bien; las diferidas solo tras WaitDurable) y cada
// rechazo sin efecto (F). Tras el corte, -mode verify reabre y exige que cada store sea el estado tras un prefijo de
// las operaciones de su escritor que llegue al menos hasta la última confirmada (así nada confirmado falta y un lote
// está entero o no está), que Count coincida y que Engine.Verify dé limpio.
//
//	powercut -mode load -dir /mnt/x/db -log /tmp/r.log -seed 7            # hasta que lo maten (kill -9)
//	powercut -mode verify -dir /mnt/x/db -log /tmp/r.log
//	powercut -mode ioerror -dir /mnt/x/db -log /tmp/r.log -seed 7 -wait 60s
//	powercut -mode diskfull -dir /mnt/x/db -log /tmp/r.log -filler /mnt/x/relleno [-closefull]
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store"
)

func main() {
	mode := flag.String("mode", "load", "load | verify | ioerror | diskfull")
	dir := flag.String("dir", "", "directorio del motor (en el disco que se corta)")
	logPath := flag.String("log", "", "registro de confirmaciones (en un disco que NO se corta)")
	seed := flag.Uint64("seed", 1, "semilla de las operaciones y los valores")
	workers := flag.Int("workers", 4, "escritores (uno por store)")
	duration := flag.Duration("duration", 0, "load: parar solo tras este tiempo (0 = hasta que lo maten)")
	wait := flag.Duration("wait", 60*time.Second, "ioerror: cuánto esperar a que los escritores fallen")
	filler := flag.String("filler", "", "diskfull: archivo de relleno que se borra para liberar lugar")
	closeFull := flag.Bool("closefull", false, "diskfull: Close y Open con el disco todavía lleno antes de liberar")
	flag.Parse()
	if *dir == "" || *logPath == "" {
		fmt.Fprintln(os.Stderr, "faltan -dir y -log")
		os.Exit(2)
	}

	var ok bool
	switch *mode {
	case "load":
		ok = runLoad(*dir, *logPath, *seed, *workers, "load", *duration)
	case "verify":
		ok = runVerify(*dir, *logPath)
	case "ioerror":
		ok = runIOError(*dir, *logPath, *seed, *workers, *wait)
	case "diskfull":
		ok = runDiskFull(*dir, *logPath, *seed, *workers, *filler, *closeFull)
	default:
		fmt.Fprintln(os.Stderr, "modo desconocido:", *mode)
		os.Exit(2)
	}
	if !ok {
		os.Exit(1)
	}
}

/**
* start abre el motor y el registro y crea los escritores; escribe la cabecera del registro.
* @param dir, logPath string
* @param seed uint64
* @param n int
* @param profile string
* @return *store.Engine, *journal, []*worker
**/
func start(dir, logPath string, seed uint64, n int, profile string) (*store.Engine, *journal, []*worker) {
	e, err := store.Open(dir)
	if err != nil {
		fatal("Open: %v", err)
	}
	j, err := openJournal(logPath)
	if err != nil {
		fatal("registro: %v", err)
	}
	j.line("H %d %d %s", seed, n, profile)
	ws := make([]*worker, n)
	for g := range ws {
		if ws[g], err = newWorker(e, j, seed, profile, g); err != nil {
			fatal("Store %s: %v", storeName(g), err)
		}
	}
	return e, j, ws
}

/**
* runLoad corre la carga hasta que lo maten (o hasta duration) y, si llega al final, cierra bien.
* @param dir, logPath string
* @param seed uint64
* @param n int
* @param profile string
* @param duration time.Duration
* @return bool
**/
func runLoad(dir, logPath string, seed uint64, n int, profile string, duration time.Duration) bool {
	e, j, ws := start(dir, logPath, seed, n, profile)
	defer j.close()
	stop := make(chan struct{})
	var maintErrs atomic.Int64
	go maintenance(e, stop, &maintErrs)
	if duration > 0 {
		time.AfterFunc(duration, func() { close(stop) })
	}
	fmt.Println("LOAD-STARTED")
	ok := true
	runWorkers(ws, stop, 0, func(w *worker, err error) {
		fmt.Printf("LOAD-ERROR %v\n", err)
		ok = false
	})
	for _, w := range ws {
		if w.err == nil {
			ok = w.waitDurable() == nil && ok
		}
	}
	if err := e.Close(); err != nil {
		fmt.Printf("LOAD-ERROR Close: %v\n", err)
		ok = false
	}
	fmt.Printf("LOAD-DONE ops=%d errores_mantenimiento=%d\n", totalOps(ws), maintErrs.Load())
	return ok
}

/**
* runVerify reabre el motor tras el corte y lo compara con el registro.
* @param dir, logPath string
* @return bool
**/
func runVerify(dir, logPath string) bool {
	rec, err := readRecord(logPath)
	if err != nil {
		fmt.Printf("VERIFY-FAIL registro: %v\n", err)
		return false
	}
	t := time.Now()
	e, err := store.Open(dir)
	if err != nil {
		fmt.Printf("VERIFY-FAIL Open: %v\n", err)
		return false
	}
	opened := time.Since(t)
	sum := checkEngine(e, rec, true)
	if err := e.Close(); err != nil {
		sum.problems = append(sum.problems, fmt.Sprintf("Close: %v", err))
	}
	if !sum.ok() {
		fmt.Printf("VERIFY-FAIL open=%v %s\n", opened.Round(time.Millisecond), sum)
		return false
	}
	fmt.Printf("VERIFY-OK open=%v %s\n", opened.Round(time.Millisecond), sum)
	return true
}

/**
* runIOError corre la carga mientras el script cambia el disco a error_writes: espera a que todos los escritores
* paren por un error, prueba qué hace el motor fallado (escribir, leer, Close) sin pánico ni cuelgue.
* @param dir, logPath string
* @param seed uint64
* @param n int
* @param wait time.Duration
* @return bool
**/
func runIOError(dir, logPath string, seed uint64, n int, wait time.Duration) bool {
	e, j, ws := start(dir, logPath, seed, n, "load")
	defer j.close()
	stop := make(chan struct{})
	var maintErrs atomic.Int64
	go maintenance(e, stop, &maintErrs)
	fmt.Println("LOAD-STARTED")

	var mu sync.Mutex
	failed := map[string]int{}
	timer := time.AfterFunc(wait, func() { close(stop) })
	runWorkers(ws, stop, 0, func(w *worker, err error) {
		mu.Lock()
		failed[errClass(err)]++
		mu.Unlock()
	})
	timedOut := !timer.Stop()
	if !timedOut {
		close(stop)
	}

	ok := true
	stopped := 0
	for _, w := range ws {
		if w.err != nil {
			stopped++
		}
	}
	fmt.Printf("IOERROR escritores_parados=%d/%d por_clase=%v espera_vencida=%v ops=%d\n", stopped, n, failed, timedOut, totalOps(ws))
	if stopped == 0 {
		fmt.Println("IOERROR-FAIL ningún escritor vio el error de E/S")
		ok = false
	}

	s := ws[0].s
	putErr := withTimeout(10*time.Second, func() error { return s.Put("sonda", []byte("x")) })
	fmt.Printf("IOERROR Put después: %s\n", errClass(putErr))
	if putErr == nil && stopped > 0 {
		fmt.Println("IOERROR-FAIL el motor aceptó una escritura después de fallar")
		ok = false
	}
	readErr := withTimeout(10*time.Second, func() error { _, _, err := s.Read("k0001"); return err })
	fmt.Printf("IOERROR Read después: %s ReadOnly=%v\n", errClass(readErr), e.Stats().ReadOnly)
	waitErr := withTimeout(10*time.Second, e.WaitDurable)
	fmt.Printf("IOERROR WaitDurable después: %s\n", errClass(waitErr))
	closeErr := withTimeout(30*time.Second, e.Close)
	fmt.Printf("IOERROR Close: %s\n", errClass(closeErr))
	for _, err := range []error{putErr, readErr, waitErr, closeErr} {
		if errors.Is(err, errHang) {
			fmt.Println("IOERROR-FAIL una llamada se colgó")
			ok = false
			break
		}
	}
	if ok {
		fmt.Println("IOERROR-OK")
	}
	return ok
}

/**
* runDiskFull llena el disco con la carga hasta ErrDiskFull, comprueba que las lecturas siguen, borra el relleno y
* comprueba que las escrituras vuelven solas (error limpio) o que Close + Open recupera todo (solo lectura).
* @param dir, logPath string
* @param seed uint64
* @param n int
* @param filler string
* @param closeFull bool: cerrar y abrir con el disco todavía lleno antes de liberar
* @return bool
**/
func runDiskFull(dir, logPath string, seed uint64, n int, filler string, closeFull bool) bool {
	e, j, ws := start(dir, logPath, seed, n, "big")
	defer j.close()
	ok := true
	fail := func(format string, args ...any) {
		fmt.Printf("DISKFULL-FAIL "+format+"\n", args...)
		ok = false
	}

	// 1. Llenar: cada escritor para en su primer ErrDiskFull (limpio o no) y avisa a los demás.
	stop := make(chan struct{})
	var once sync.Once
	var firstFull atomic.Value
	full := func(err error) {
		if errors.Is(err, store.ErrDiskFull) {
			firstFull.CompareAndSwap(nil, errClass(err))
			once.Do(func() { close(stop) })
		}
	}
	for _, w := range ws {
		w.j = j
	}
	t := time.Now()
	deadline := time.AfterFunc(15*time.Minute, func() { once.Do(func() { close(stop) }) })
	var wg sync.WaitGroup
	for _, w := range ws {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				before := e.Stats().DiskFull
				err := w.step()
				if err != nil {
					w.err = err
					full(err)
					return
				}
				if e.Stats().DiskFull > before {
					full(store.ErrDiskFull)
				}
			}
		}()
	}
	wg.Wait()
	deadline.Stop()
	st := e.Stats()
	cause, _ := firstFull.Load().(string)
	fmt.Printf("DISKFULL lleno en %v: ops=%d primer=%q rechazos_limpios=%d ReadOnly=%v\n",
		time.Since(t).Round(time.Millisecond), totalOps(ws), cause, st.DiskFull, st.ReadOnly)
	if cause == "" {
		fail("el disco no se llenó en 15 min")
	}

	// 2. Las lecturas siguen (sin Count ni Verify si quedó de solo lectura).
	rec, err := readRecord(logPath)
	if err != nil {
		fatal("registro: %v", err)
	}
	sum := checkEngine(e, rec, !st.ReadOnly)
	fmt.Printf("DISKFULL lecturas con el disco lleno: %s\n", sum)
	if !sum.ok() {
		fail("lecturas con el disco lleno")
	}

	// 3. Con closeFull, Close y Open con el disco todavía lleno: pueden fallar con ErrDiskFull, nunca colgarse ni
	// perder nada.
	reopen := st.ReadOnly
	if closeFull {
		closeErr := withTimeout(30*time.Second, e.Close)
		fmt.Printf("DISKFULL Close con el disco lleno: %s\n", errClass(closeErr))
		var full *store.Engine
		openErr := withTimeout(60*time.Second, func() (err error) { full, err = store.Open(dir); return })
		fmt.Printf("DISKFULL Open con el disco lleno: %s\n", errClass(openErr))
		if full != nil {
			sum := checkEngine(full, rec, false)
			fmt.Printf("DISKFULL lecturas tras Open con el disco lleno: %s\n", sum)
			if !sum.ok() {
				fail("lecturas tras Open con el disco lleno")
			}
			fmt.Printf("DISKFULL Close otra vez: %s\n", errClass(withTimeout(30*time.Second, full.Close)))
		}
		if errors.Is(closeErr, errHang) || errors.Is(openErr, errHang) {
			fail("Close u Open se colgó con el disco lleno")
			return false
		}
		reopen = true
	}

	// 4. Liberar lugar.
	if filler != "" {
		if err := os.Remove(filler); err != nil {
			fail("no se pudo borrar el relleno: %v", err)
		}
	}

	// 5. Con error limpio las escrituras vuelven sin reabrir; en solo lectura (o tras closeFull), Close + Open.
	if reopen {
		if !closeFull {
			closeErr := withTimeout(30*time.Second, e.Close)
			fmt.Printf("DISKFULL solo lectura: Close = %s\n", errClass(closeErr))
			if errors.Is(closeErr, errHang) {
				fail("Close se colgó")
			}
		}
		if e, err = store.Open(dir); err != nil {
			fail("Open tras liberar lugar: %v", err)
			return false
		}
	}
	for _, w := range ws {
		w.err = nil
		if err := w.reopen(e); err != nil {
			fail("Store %s tras liberar: %v", storeName(w.g), err)
			return false
		}
	}
	retry := atomic.Int64{}
	runWorkers(ws, nil, 150, func(w *worker, err error) {
		retry.Add(1)
		fmt.Printf("DISKFULL escritura tras liberar: %v\n", err)
	})
	for _, w := range ws {
		if w.err == nil {
			w.waitDurable()
		}
	}
	rejected := e.Stats().DiskFull
	if !reopen {
		rejected -= st.DiskFull // el mismo motor: sin los rechazos del llenado (uno reabierto cuenta desde cero)
	}
	if retry.Load() > 0 || rejected > 0 || e.Stats().ReadOnly {
		fail("las escrituras no volvieron tras liberar lugar (errores=%d, rechazos_limpios=%d)", retry.Load(), rejected)
	}
	fmt.Printf("DISKFULL escrituras tras liberar: 150 por escritor, ops=%d rechazos_limpios=%d\n", totalOps(ws), rejected)

	// 6. Todo confirmado, Count y Verify limpio; de nuevo tras Close + Open.
	rec, _ = readRecord(logPath)
	if sum := checkEngine(e, rec, true); !sum.ok() {
		fail("tras liberar: %s", sum)
	} else {
		fmt.Printf("DISKFULL tras liberar: %s\n", sum)
	}
	if err := e.Close(); err != nil {
		fail("Close: %v", err)
	}
	if e, err = store.Open(dir); err != nil {
		fail("Open final: %v", err)
		return false
	}
	if sum := checkEngine(e, rec, true); !sum.ok() {
		fail("tras reabrir: %s", sum)
	} else {
		fmt.Printf("DISKFULL tras reabrir: %s\n", sum)
	}
	e.Close()
	if ok {
		fmt.Println("DISKFULL-OK")
	}
	return ok
}

// errHang marca una llamada que no volvió a tiempo.
var errHang = errors.New("colgada: no volvió a tiempo")

/**
* withTimeout corre fn y devuelve su error, o errHang si no vuelve en d (la goroutine queda colgada).
* @param d time.Duration
* @param fn func() error
* @return error
**/
func withTimeout(d time.Duration, fn func() error) error {
	ch := make(chan error, 1)
	go func() { ch <- fn() }()
	select {
	case err := <-ch:
		return err
	case <-time.After(d):
		return errHang
	}
}

/**
* errClass resume un error para el informe: qué errores públicos de store lleva y el texto.
* @param err error
* @return string
**/
func errClass(err error) string {
	if err == nil {
		return "nil"
	}
	tags := ""
	for _, c := range []struct {
		name string
		err  error
	}{{"ErrFailed", store.ErrFailed}, {"ErrDiskFull", store.ErrDiskFull}, {"ErrClosed", store.ErrClosed},
		{"ErrThrottled", store.ErrThrottled}, {"ErrCorrupt", store.ErrCorrupt}} {
		if errors.Is(err, c.err) {
			tags += c.name + "+"
		}
	}
	if tags == "" {
		tags = "otro+"
	}
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		msg = msg[i+2:] // la causa final (el texto entero repite cada capa)
	}
	return fmt.Sprintf("%s(%s)", tags[:len(tags)-1], msg)
}

/**
* totalOps suma las operaciones con efecto de los escritores.
* @param ws []*worker
* @return int64
**/
func totalOps(ws []*worker) int64 {
	var n int64
	for _, w := range ws {
		n += w.ops
	}
	return n
}

/**
* fatal imprime el error y sale con 3 (no es un fallo del motor sino de la prueba).
* @param format string
* @param args ...any
**/
func fatal(format string, args ...any) {
	fmt.Printf("POWERCUT-FATAL "+format+"\n", args...)
	os.Exit(3)
}
