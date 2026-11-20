package wal

import (
	"errors"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
)

// preallocOptions: segmentos chicos con preasignación.
func preallocOptions(spares int) Options {
	opts := smallOptions(spares)
	opts.Preallocate = true
	return opts
}

// TestPreallocate: con preasignación cada segmento queda escrito entero (con ceros) antes de recibir registros, Room
// deja preparado el siguiente pasada la mitad del activo, y la historia se lee entera al reabrir (los ceros son el
// final del WAL), también tras un corte en los tres modos.
func TestPreallocate(t *testing.T) {
	const dir, confirmed, total = "/db", 150, 220
	for name, mode := range crashModes {
		t.Run(name, func(t *testing.T) {
			fsys := crashfs.New()
			opts := preallocOptions(1)
			w, _, _ := openTest(t, fsys, dir, opts, Start)
			for i := 1; i <= confirmed; i++ {
				if err := w.Room(); err != nil {
					t.Fatal(err)
				}
				appendRecord(t, w, testRecord(i))
				if size := vfsSize(t, fsys, segPath(dir, w.active.num)); size != opts.SegmentSize {
					t.Fatalf("registro %d: el segmento activo mide %d, no %d", i, size, opts.SegmentSize)
				}
			}
			if !w.nextReady || w.active.num < 2 {
				t.Fatalf("segmento activo %d, siguiente listo %v", w.active.num, w.nextReady)
			}
			if err := w.WaitDurable(confirmed); err != nil {
				t.Fatal(err)
			}
			for i := confirmed + 1; i <= total; i++ {
				appendRecord(t, w, testRecord(i))
			}

			fsys.Crash(mode, 3)
			w.Close()
			w, end, recs := openTest(t, fsys, dir, opts, Start)
			if len(recs) < confirmed {
				t.Fatalf("tras el corte: %d registros, confirmados %d", len(recs), confirmed)
			}
			checkHistory(t, recs, len(recs))
			n := len(recs)
			for i := n + 1; i <= n+40; i++ { // se sigue escribiendo a continuación
				appendRecord(t, w, testRecord(i))
			}
			if end.LSN != page.LSN(n+1) {
				t.Fatalf("fin %+v con %d registros", end, n)
			}
			w.Close()
			_, _, recs = openTest(t, fsys, dir, opts, Start)
			checkHistory(t, recs, n+40)
		})
	}
}

// TestRoomDiskFull: con el disco lleno Room devuelve ErrNoRoom sin dejar el WAL inutilizable; los registros siguen
// entrando en lo ya preasignado, una reserva que necesita un segmento nuevo devuelve ErrNoRoom (sin asignar lsn), y al
// subir la capacidad todo vuelve a andar sin reabrir.
func TestRoomDiskFull(t *testing.T) {
	const dir = "/db"
	fsys := crashfs.New()
	opts := preallocOptions(0)
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	if err := w.Room(); err != nil {
		t.Fatal(err)
	}
	fsys.SetCapacity(fsys.Used() + 1000) // no alcanza para preasignar el siguiente

	i, roomFailed := 0, false
	var reserveErr error
	for reserveErr == nil {
		i++
		if err := w.Room(); err != nil {
			if !errors.Is(err, ErrNoRoom) {
				t.Fatalf("Room: %v", err)
			}
			roomFailed = true
		}
		rec := testRecord(i)
		var r Reservation
		if r, reserveErr = w.Reserve(&rec, nil); reserveErr == nil {
			if err := r.Write(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !roomFailed || !errors.Is(reserveErr, ErrNoRoom) || w.Failed() != nil {
		t.Fatalf("Room falló %v, Reserve %v, WAL fallado %v", roomFailed, reserveErr, w.Failed())
	}
	last := page.LSN(i - 1)
	if w.lastLSN.Load() != uint64(last) {
		t.Fatalf("la reserva rechazada asignó un lsn: último %d, se esperaba %d", w.lastLSN.Load(), last)
	}
	if err := w.WaitDurable(last); err != nil {
		t.Fatal(err)
	}

	fsys.SetCapacity(0)
	if err := w.Room(); err != nil {
		t.Fatalf("Room con lugar: %v", err)
	}
	for j := i; j < i+60; j++ {
		appendRecord(t, w, testRecord(j))
	}
	w.Close()
	_, _, recs := openTest(t, fsys, dir, opts, Start)
	checkHistory(t, recs, i+59)
}

// TestPrepareFsyncFails: si falla un fsync de la preasignación del siguiente segmento, el WAL queda inutilizable (no
// es ErrNoRoom) y no se reintenta: un fsync fallido nunca pasa por éxito más tarde.
func TestPrepareFsyncFails(t *testing.T) {
	fsys := crashfs.New()
	opts := preallocOptions(0)
	w, _, _ := openTest(t, fsys, "/db", opts, Start)
	for i := 1; w.active.size <= opts.SegmentSize/2; i++ {
		appendRecord(t, w, testRecord(i))
	}
	fsys.FailSyncAfter(0)
	err := w.Room()
	if err == nil || errors.Is(err, ErrNoRoom) || w.Failed() == nil {
		t.Fatalf("Room con fsync fallido = %v, WAL fallado %v", err, w.Failed())
	}
	rec := testRecord(1000)
	if _, err := w.Reserve(&rec, nil); err == nil {
		t.Fatal("Reserve tras el fsync fallido no falló")
	}
}
