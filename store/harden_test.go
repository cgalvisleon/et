package store

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// Pruebas del endurecimiento del 2026-11-21 (tareas 110 a 114 de F-23): los hallazgos medios N3, N4, R3, R4 y V2 de
// doc/review-store-2026-11-20.md y el disco lleno con huecos de la tarea 107 (crashfs cuenta el lugar asignado).

// TestDiskFullNoHoles (tarea 107, crashfs con huecos): un store que crece de a varias páginas (fsm.Grow extiende de a
// 1/64, hasta 16) llena el disco. Cada escritura que no entra da ErrDiskFull limpio, el motor nunca queda
// inutilizable ni de solo lectura, Verify queda limpio y al liberar lugar las escrituras vuelven. Si smgr.Extend
// dejara huecos, la página sucia que cae en uno daría ENOSPC al escribirse (desalojo o checkpoint), cuando ya no hay
// vuelta atrás: es el fallo que solo vio la VM con ext4.
func TestDiskFullNoHoles(t *testing.T) {
	for _, quota := range []int64{64 << 10, 256 << 10, 900 << 10} {
		t.Run(fmt.Sprint(quota>>10, "KB"), func(t *testing.T) {
			fsys := crashfs.New()
			e := openTest(t, fsys, testConfig())
			defer e.Close()
			s := mustStore(t, e, "big")
			value := make([]byte, 900)
			for i := range 2500 { // ~300 páginas: las extensiones ya son de varias páginas
				mustOK(t, s.Put(fmt.Sprintf("k-%05d", i), value))
			}
			mustOK(t, e.Checkpoint())
			fsys.SetCapacity(fsys.Used() + quota)

			written, full := 2500, 0
			for i := 0; full < 20 && i < 20000; i++ {
				err := s.Put(fmt.Sprintf("n-%05d", i), value)
				switch {
				case err == nil:
					written++
				case errors.Is(err, ErrFailed) || !errors.Is(err, ErrDiskFull):
					t.Fatalf("Put %d with the disk full: %v (want a clean ErrDiskFull)", i, err)
				default:
					full++
					if err := e.Checkpoint(); err != nil && !errors.Is(err, ErrDiskFull) || errors.Is(err, ErrFailed) {
						t.Fatalf("Checkpoint with the disk full: %v", err)
					}
				}
			}
			if full == 0 {
				t.Fatal("the disk never filled")
			}
			if st := e.Stats(); st.ReadOnly {
				t.Fatal("the engine went read-only: a dirty page hit ENOSPC (a hole)")
			}
			if n := s.Count(); n != int64(written) {
				t.Fatalf("Count = %d; want %d", n, written)
			}
			mustVerifyClean(t, e)

			fsys.SetCapacity(0)
			mustOK(t, s.Put("after", value))
			mustOK(t, e.Checkpoint())
			mustVerifyClean(t, e)
		})
	}
}

// TestCloseHurriesCheckpoint (N3): un checkpoint de fondo reparte sus escrituras en el 90 % de CHECKPOINT_TIMEOUT; un
// Checkpoint() del usuario lo espera con closeMu en lectura. Close apura al de fondo antes de esperar closeMu: no
// queda detrás de todo el reparto (antes tardaba casi CHECKPOINT_TIMEOUT con el motor detenido).
func TestCloseHurriesCheckpoint(t *testing.T) {
	cfg := testConfig()
	cfg.checkpointTimeout = 30 * time.Second
	cfg.checkEvery = 5 * time.Millisecond
	e := openTest(t, crashfs.New(), cfg)
	s := mustStore(t, e, "s")
	value := make([]byte, 2000)
	for i := range 200 { // más páginas sucias que el reparto puede escribir de golpe
		mustOK(t, s.Put(fmt.Sprintf("k-%04d", i), value))
	}
	started := make(chan struct{})
	var once sync.Once
	e.afterStep = func(step int) {
		if step == 1 {
			once.Do(func() { close(started) })
		}
	}
	e.lastCheckpoint.Store(0) // el de fondo toca ya (por tiempo)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the background checkpoint did not start")
	}
	user := goErr(e.Checkpoint) // espera ckptMu con closeMu en lectura
	settle()
	start := time.Now()
	mustOK(t, waitErr(t, "Close", goErr(e.Close), 5*time.Second))
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("Close took %v behind the spread checkpoint", took)
	}
	if err := waitErr(t, "Checkpoint", user, 5*time.Second); err != nil && !errors.Is(err, ErrClosed) {
		t.Fatalf("user Checkpoint: %v", err)
	}
}

// TestDropStoreWaitsBatchWriters (N4): un DropStore suelto no borra un store en el que escribió un lote abierto (sus
// escrituras se perdían en silencio al confirmar): espera al lote hasta LOCK_TIMEOUT y si no termina da ErrBusy, como
// DropStoreIn. Si el lote termina durante la espera, el borrado sigue.
func TestDropStoreWaitsBatchWriters(t *testing.T) {
	cfg := testConfig()
	cfg.lockTimeout = 300 * time.Millisecond
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	s := mustStore(t, e, "s")
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "k", []byte("v")))
	if err := e.DropStore("s"); !errors.Is(err, ErrBusy) {
		t.Fatalf("DropStore with an open batch that wrote in it: %v, want ErrBusy", err)
	}
	mustOK(t, tx.Commit())
	if data, ok, err := s.Read("k"); err != nil || !ok || string(data) != "v" {
		t.Fatalf("Read after the batch committed: %q, %v", data, err)
	}

	tx = e.NewBatch()
	mustOK(t, s.PutIn(tx, "k2", []byte("v")))
	drop := goErr(func() error { return e.DropStore("s") })
	time.Sleep(50 * time.Millisecond) // dentro de LOCK_TIMEOUT
	mustOK(t, tx.Commit())
	mustOK(t, waitErr(t, "DropStore", drop, 5*time.Second))
	if names := e.Stores(); len(names) != 0 {
		t.Fatalf("stores after the drop: %v", names)
	}
}

// TestChangesReadInsideFnWithClose (R3): el fn de ChangeReader.Next puede leer; con Close esperando, una lectura
// dentro de fn ya no queda detrás del Lock pendiente de closeMu (antes Close y el lector se colgaban para siempre).
func TestChangesReadInsideFnWithClose(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	s := mustStore(t, e, "s")
	_, err := e.CreateLogicalSlot("feed")
	mustOK(t, err)
	mustOK(t, s.Put("a", []byte("1")))
	mustOK(t, s.Put("b", []byte("2")))
	r, err := e.Changes("feed")
	mustOK(t, err)

	inFn := make(chan struct{})
	var closing chan error
	next := goErr(func() error {
		_, err := r.Next(0, func(tx TxInfo, c Change) error {
			if closing == nil {
				close(inFn)
				closing = goErr(e.Close)
				settle() // Close llega a esperar
			}
			if _, _, err := s.Read("a"); err != nil && !errors.Is(err, ErrClosed) {
				return err
			}
			return nil
		})
		return err
	})
	<-inFn
	if err := waitErr(t, "Next", next, 5*time.Second); err != nil && !errors.Is(err, ErrClosed) {
		t.Fatalf("Next: %v", err)
	}
	mustOK(t, waitErr(t, "Close", closing, 5*time.Second))
}

// TestLogicalSlotWaitWithClose (R4): CreateLogicalSlot espera a los lotes en curso (hasta TRANSACTION_TIMEOUT) sin
// closeMu: un Close en medio no queda detrás de la espera ni congela el motor; la creación da ErrClosed y la ranura
// no queda.
func TestLogicalSlotWaitWithClose(t *testing.T) {
	cfg := testConfig()
	cfg.txTimeout = 30 * time.Second
	fsys := crashfs.New()
	e := openTest(t, fsys, cfg)
	s := mustStore(t, e, "s")
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "k", []byte("v")))
	create := goErr(func() error { _, err := e.CreateLogicalSlot("feed"); return err })
	settle()
	start := time.Now()
	mustOK(t, waitErr(t, "Close", goErr(e.Close), 5*time.Second))
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("Close took %v behind the logical slot wait", took)
	}
	if err := waitErr(t, "CreateLogicalSlot", create, 5*time.Second); !errors.Is(err, ErrClosed) {
		t.Fatalf("CreateLogicalSlot during Close: %v, want ErrClosed", err)
	}
	e = openTest(t, fsys, cfg)
	defer e.Close()
	if slots := e.Slots(); len(slots) != 0 {
		t.Fatalf("slots after reopening: %+v", slots)
	}
}

// vacuumLoad deja en el store s de e n claves de ~900 bytes con una versión muerta cada una (cientos de hojas que
// limpiar) y un checkpoint.
func vacuumLoad(t *testing.T, e *Engine, s *Store, n int) {
	t.Helper()
	value := make([]byte, 900)
	for round := range 2 {
		value[0] = byte(round)
		for i := range n {
			mustOK(t, s.Put(fmt.Sprintf("k-%05d", i), value))
		}
	}
	mustOK(t, e.Checkpoint())
}

// TestVacuumYieldsToDDL (V2): la limpieza de un store tiene Store.mu en lectura; un DropStore lo pide en escritura con
// ddlMu tomado, y detrás de ddlMu queda el checkpoint. La limpieza mira en cada hoja si un DDL espera y se corta:
// DropStore y el checkpoint no esperan toda la pasada (antes, O(hojas) con el disco lento).
func TestVacuumYieldsToDDL(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	defer e.Close()
	s := mustStore(t, e, "big")
	vacuumLoad(t, e, s, 5000)
	mustStore(t, e, "other")

	fsys.SetDelay(3*time.Millisecond, 0) // cada página que la limpieza ensucia y desaloja cuesta 3 ms
	defer fsys.SetDelay(0, 0)
	vacuum := goErr(e.Vacuum)
	time.Sleep(200 * time.Millisecond)
	start := time.Now()
	mustOK(t, waitErr(t, "DropStore", goErr(func() error { return e.DropStore("big") }), 30*time.Second))
	if took := time.Since(start); took > time.Second {
		t.Fatalf("DropStore waited %v for the vacuum pass", took)
	}
	mustOK(t, waitErr(t, "Checkpoint", goErr(e.Checkpoint), 30*time.Second))
	mustOK(t, waitErr(t, "Vacuum", vacuum, 30*time.Second))
	fsys.SetDelay(0, 0)
	mustVerifyClean(t, e)
}
