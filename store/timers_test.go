package store

import (
	"runtime"
	"testing"
	"weak"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// TestCloseReleasesEngine: un snapshot y un lote que quedan abiertos al cerrar el motor no lo retienen en memoria.
// Antes, su temporizador de TRANSACTION_TIMEOUT seguía armado tras Close y retenía el motor entero (caché de
// páginas incluida) hasta vencer: TestModelCrash con -count alto llegó a agotar la memoria del equipo.
func TestCloseReleasesEngine(t *testing.T) {
	e := openTest(t, newProc(crashfs.New()), testConfig())
	e.Snapshot() // quedan abiertos a propósito
	e.NewBatch()
	alive := weak.Make(e)
	mustOK(t, e.Close())
	if n := len(e.timers.live); n != 0 {
		t.Fatalf("%d timers still armed after Close", n)
	}
	e = nil
	for range 3 {
		runtime.GC()
	}
	if alive.Value() != nil {
		t.Fatal("the closed engine is still retained")
	}
}

// TestTimersAfterClose: con el motor cerrado no se arman temporizadores nuevos.
func TestTimersAfterClose(t *testing.T) {
	var ts timers
	ts.closeAll()
	if tm := ts.after(1, func() {}); tm != nil {
		t.Fatal("a timer was armed after closeAll")
	}
	ts.stop(nil) // no debe fallar
}
