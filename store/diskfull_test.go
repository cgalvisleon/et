package store

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// diskQuotas son los cupos de disco libre que reciben las pruebas de disco lleno (por semilla): menos de lo que pide
// preasignar un segmento del WAL de prueba (1 MB), alrededor y bastante más, así el disco se llena en distintos
// puntos (el WAL, los archivos de datos, el archivo de estados, catalog.tmp).
var diskQuotas = []int64{32 << 10, 300 << 10, 700 << 10, 1200 << 10}

// fullRun es lo que dejó una carga con el disco lleno: el estado confirmado, el de la operación que dejó el motor de
// solo lectura (nil si ninguna) y cuántas operaciones se rechazaron sin tocar el motor.
type fullRun struct {
	model   state
	pending *state
	full    int
}

/**
* loadUntilFull hace hasta ops operaciones al azar, hasta que maxFull se rechacen con ErrDiskFull o una deje el
* motor de solo lectura. Una rechazada sin ErrFailed no debe cambiar nada; ninguna debe fallar con otro error.
* @param t *testing.T, e *Engine, gen *generator, model state, ops, maxFull int
* @return fullRun
**/
func loadUntilFull(t *testing.T, e *Engine, gen *generator, model state, ops, maxFull int) fullRun {
	t.Helper()
	run := fullRun{model: model}
	for range ops {
		o := gen.next()
		after := run.model.clone()
		want := o.apply(after)
		got, err := o.run(e)
		switch {
		case err == nil:
			if got != want {
				t.Fatalf("%v = %v; model says %v", o, got, want)
			}
			run.model = after
		case !errors.Is(err, ErrDiskFull):
			t.Fatalf("%v failed without ErrDiskFull: %v", o, err)
		case errors.Is(err, ErrFailed):
			t.Logf("%v left the engine read-only: %v", o, err)
			run.pending = &after
			return run
		default:
			if run.full++; run.full == 1 {
				t.Logf("first rejected: %v: %v", o, err)
			}
			if run.full >= maxFull {
				return run
			}
		}
	}
	return run
}

/**
* fillDisk hace 200 operaciones sobre el motor de fsys y un checkpoint, y deja libre en el disco solo quota bytes.
* @param t *testing.T, fsys crashfs.FS, e *Engine, gen *generator, quota int64
* @return state: el modelo
**/
func fillDisk(t *testing.T, fsys crashfs.FS, e *Engine, gen *generator, quota int64) state {
	t.Helper()
	model := state{}
	for range 200 {
		o := gen.next()
		want := o.apply(model)
		if got, err := o.run(e); err != nil || got != want {
			t.Fatalf("%v = %v, %v; model says %v", o, got, err, want)
		}
	}
	mustOK(t, e.Checkpoint())
	fsys.SetCapacity(fsys.Used() + quota)
	return model
}

/**
* mustVerifyClean corre Verify y falla la prueba si encuentra algo.
* @param t *testing.T, e *Engine
**/
func mustVerifyClean(t *testing.T, e *Engine) {
	t.Helper()
	r, err := e.Verify()
	mustOK(t, err)
	if !r.Clean() {
		t.Fatalf("Verify is not clean:\n%s", r)
	}
}

// TestDiskFull llena el disco a mitad de una carga de Put sueltos, lotes, DDL, checkpoints y limpiezas, con cupos
// distintos: cada operación que no entra devuelve ErrDiskFull sin cambiar nada (o deja el motor de solo lectura), las
// lecturas siguen, Verify queda limpio y Count coincide; al liberar espacio las escrituras vuelven sin reabrir, y tras
// cerrar y abrir está todo lo confirmado.
func TestDiskFull(t *testing.T) {
	for seed := range 8 {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			t.Parallel()
			fsys := crashfs.New()
			gen := newGenerator(uint64(seed) + 100)
			e := openTest(t, fsys, testConfig())
			model := fillDisk(t, fsys, e, gen, diskQuotas[seed%len(diskQuotas)])
			run := loadUntilFull(t, e, gen, model, 4000, 30)
			if run.pending != nil {
				checkReadOnly(t, fsys, e, run)
				return
			}
			if run.full == 0 {
				t.Fatal("the disk never filled")
			}
			if st := e.Stats(); st.DiskFull == 0 || st.ReadOnly {
				t.Fatalf("Stats after a clean disk full: DiskFull %d, ReadOnly %v", st.DiskFull, st.ReadOnly)
			}

			// Con el disco lleno las lecturas siguen, Count coincide y el motor está sano.
			if got := engineState(t, e); !got.equal(run.model) {
				t.Fatalf("reads with the disk full:\n%s", diffState(got, run.model))
			}
			mustVerifyClean(t, e)

			// Al liberar espacio las escrituras vuelven sin reabrir.
			fsys.SetCapacity(0)
			again := loadUntilFull(t, e, gen, run.model, 300, 1)
			if again.full != 0 || again.pending != nil {
				t.Fatalf("after freeing space: %d rejected, read-only %v", again.full, again.pending != nil)
			}
			mustOK(t, e.Checkpoint())
			mustVerifyClean(t, e)
			mustOK(t, e.Close())

			e = openTest(t, fsys, testConfig())
			defer e.Close()
			recoveredOneOf(t, e, again.model, nil)
			mustVerifyClean(t, e)
		})
	}
}

/**
* checkReadOnly comprueba un motor que quedó de solo lectura por el disco lleno: las lecturas siguen (ven lo
* confirmado, o también la operación en curso), las escrituras dan ErrDiskFull con ErrFailed, Close no se cuelga, y
* al reabrir con lugar está lo confirmado.
* @param t *testing.T, fsys crashfs.FS, e *Engine, run fullRun
**/
func checkReadOnly(t *testing.T, fsys crashfs.FS, e *Engine, run fullRun) {
	t.Helper()
	if !e.Stats().ReadOnly {
		t.Fatal("Stats.ReadOnly is false")
	}
	got := readOnlyState(t, e)
	if !got.equal(run.model) && !got.equal(*run.pending) {
		t.Fatalf("reads of the read-only engine:\n%s", diffState(got, run.model))
	}
	for _, name := range e.Stores() {
		err := mustStore(t, e, name).Put("x", []byte("y"))
		if !errors.Is(err, ErrDiskFull) || !errors.Is(err, ErrFailed) {
			t.Fatalf("put on the read-only engine: %v", err)
		}
	}
	if err := e.NewBatch().Commit(); err != nil && !errors.Is(err, ErrDiskFull) {
		t.Fatalf("batch on the read-only engine: %v", err)
	}
	closeWithin(t, e, 10*time.Second)

	fsys.SetCapacity(0)
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	recoveredOneOf(t, e, run.model, run.pending)
	checkAfterCrash(t, e)
}

// TestDiskFullFsync: un fsync que falla por disco lleno (la capacidad bajó por debajo de lo usado, como un volumen
// fino) deja el motor de solo lectura: la operación y las siguientes dan ErrDiskFull con ErrFailed, las lecturas
// siguen, Close no se cuelga, y al reabrir con lugar está todo lo confirmado.
func TestDiskFullFsync(t *testing.T) {
	fsys := crashfs.New()
	gen := newGenerator(7)
	e := openTest(t, fsys, testConfig())
	model := fillDisk(t, fsys, e, gen, 1<<30)
	mustOK(t, e.log.Room()) // el WAL ya tiene su lugar: la falta de espacio la encuentra el fsync
	fsys.SetCapacity(fsys.Used() - 1)
	s := mustStore(t, e, "a")
	// Un Put que tiene que extender un archivo de datos se rechaza sin tocar nada; el primero que escribe la hoja
	// encuentra el fsync que falla.
	for i := range 20 {
		after := model.clone()
		o := op{kind: "put", store: "a", id: fmt.Sprintf("after-%d", i), value: "v"}
		o.apply(after)
		err := s.Put(o.id, []byte(o.value))
		switch {
		case !errors.Is(err, ErrDiskFull):
			t.Fatalf("put with a failing fsync: %v", err)
		case errors.Is(err, ErrFailed):
			checkReadOnly(t, fsys, e, fullRun{model: model, pending: &after})
			return
		}
	}
	t.Fatal("no put reached the failing fsync")
}

// TestDiskFullDeferred: con escrituras diferidas pendientes, un fsync que falla por disco lleno hace que WaitDurable
// devuelva ErrDiskFull (con ErrFailed); al reabrir con lugar está todo lo que era durable y, de las diferidas, un
// prefijo en el orden en que se escribieron.
func TestDiskFullDeferred(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	s := mustStore(t, e, "s")
	mustOK(t, s.Put("durable", []byte("1")))
	const n = 50
	for i := range n {
		mustOK(t, s.Deferred().Put(fmt.Sprintf("d-%03d", i), deferredValue(i)))
	}
	if e.log.Durable() >= e.log.Published() {
		t.Fatal("the deferred puts are already durable")
	}
	fsys.SetCapacity(fsys.Used() - 1)
	err := e.WaitDurable()
	if !errors.Is(err, ErrDiskFull) || !errors.Is(err, ErrFailed) {
		t.Fatalf("WaitDurable with a failing fsync: %v", err)
	}
	if v, found, err := s.Read("durable"); err != nil || !found || string(v) != "1" {
		t.Fatalf("read on the read-only engine = %q %v %v", v, found, err)
	}
	if err := s.Deferred().Put("more", nil); !errors.Is(err, ErrDiskFull) {
		t.Fatalf("deferred put on the read-only engine: %v", err)
	}
	closeWithin(t, e, 10*time.Second)

	fsys.SetCapacity(0)
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	s = mustStore(t, e, "s")
	if _, found, err := s.Read("durable"); err != nil || !found {
		t.Fatalf("the durable put is gone: %v %v", found, err)
	}
	kept := 0
	for i := range n {
		_, found, err := s.Read(fmt.Sprintf("d-%03d", i))
		mustOK(t, err)
		switch {
		case found && kept < i:
			t.Fatalf("deferred %d survived but %d did not", i, kept)
		case found:
			kept++
		}
	}
	t.Logf("%d of %d deferred puts survived", kept, n)
	mustVerifyClean(t, e)
}

// TestDiskFullCrash: un corte justo después de un ErrDiskFull no pierde nada confirmado, con los tres modos de
// crashfs. Al reabrir con el disco todavía lleno, Open recupera o falla con ErrDiskFull (nunca otra cosa); con lugar,
// recupera todo.
func TestDiskFullCrash(t *testing.T) {
	modes := []crashfs.Mode{crashfs.LoseUnsynced, crashfs.TornWrites, crashfs.ReorderUnsynced}
	for seed := range 6 {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			t.Parallel()
			fsys := crashfs.New()
			proc := newProc(fsys)
			gen := newGenerator(uint64(seed) + 200)
			e := openTest(t, proc, testConfig())
			model := fillDisk(t, fsys, e, gen, diskQuotas[seed%len(diskQuotas)])
			run := loadUntilFull(t, e, gen, model, 4000, 1)
			if run.full == 0 && run.pending == nil {
				t.Fatal("the disk never filled")
			}
			proc.kill(modes[seed%len(modes)], int64(seed))
			closeWithin(t, e, 10*time.Second)

			if e, err := open(newProc(fsys), testDir, testConfig()); err == nil {
				recoveredOneOf(t, e, run.model, run.pending)
				closeWithin(t, e, 10*time.Second)
			} else if !errors.Is(err, ErrDiskFull) {
				t.Fatalf("open with the disk still full: %v", err)
			}
			fsys.SetCapacity(0)
			e = openTest(t, newProc(fsys), testConfig())
			defer e.Close()
			recoveredOneOf(t, e, run.model, run.pending)
			checkAfterCrash(t, e)
		})
	}
}

/**
* readOnlyState lee el contenido entero de un motor de solo lectura, como engineState pero sin comparar Count: la
* operación que lo dejó así pudo sumar a Count (bajo el writeMu del WAL) sin llegar a confirmar.
* @param t *testing.T, e *Engine
* @return state
**/
func readOnlyState(t *testing.T, e *Engine) state {
	t.Helper()
	got := state{}
	for _, name := range e.Stores() {
		ids := map[string]string{}
		mustOK(t, mustStore(t, e, name).ForEach(func(id string, data []byte) (bool, error) {
			ids[id] = string(data)
			return true, nil
		}, true, 0, 0))
		got[name] = ids
	}
	return got
}
