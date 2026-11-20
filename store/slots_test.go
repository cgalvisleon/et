package store

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* slotConfig es testConfig con segmentos de 256 KB y un WAL chico: cada prueba llena y recicla varios.
* @return config
**/
func slotConfig() config {
	cfg := testConfig()
	cfg.wal.SegmentSize = 256 << 10
	cfg.wal.Spares = 1
	cfg.maxWALSize = 64 << 20 // el freno no molesta: las pruebas miran la retención
	return cfg
}

/**
* writeBurst escribe n ids de ~1 KB en el store y hace un checkpoint (que recicla lo que se puede).
* @param t testing.TB, e *Engine, s *Store, tag string, n int
**/
func writeBurst(t testing.TB, e *Engine, s *Store, tag string, n int) {
	t.Helper()
	value := make([]byte, 1000)
	for i := range n {
		mustOK(t, s.Put(fmt.Sprintf("%s-%04d", tag, i), value))
	}
	mustOK(t, e.Checkpoint())
}

/**
* slotNamed busca una ranura en Slots.
* @param t testing.TB, e *Engine, name string
* @return SlotInfo
**/
func slotNamed(t testing.TB, e *Engine, name string) SlotInfo {
	t.Helper()
	for _, s := range e.Slots() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("slot %q not found in %+v", name, e.Slots())
	return SlotInfo{}
}

/**
* readFrom exige que el WAL desde lsn se pueda leer (WalSince + Next) o, si gone, que dé ErrWalRemoved.
* @param t testing.TB, e *Engine, lsn uint64, gone bool
**/
func readFrom(t testing.TB, e *Engine, lsn uint64, gone bool) {
	t.Helper()
	rd, err := e.WalSince(lsn)
	mustOK(t, err)
	entries, err := rd.Next(1 << 16)
	switch {
	case gone && !errors.Is(err, ErrWalRemoved):
		t.Fatalf("WalSince(%d): %d entries, %v; want ErrWalRemoved", lsn, len(entries), err)
	case !gone && (err != nil || len(entries) == 0 || entries[0].LSN != lsn+1):
		t.Fatalf("WalSince(%d): %d entries, %v", lsn, len(entries), err)
	}
}

// TestSlotRetains: una ranura retiene el WAL desde lo que confirmó, también tras varios checkpoints (sin ranura se
// recicla); AdvanceSlot lo suelta y el próximo checkpoint recicla. Errores de la API: nombre repetido, inexistente,
// nombre largo; Stats y Pressure muestran lo retenido.
func TestSlotRetains(t *testing.T) {
	e := openTest(t, crashfs.New(), slotConfig())
	defer e.Close()
	s := mustStore(t, e, "a")
	writeBurst(t, e, s, "pre", 50)
	info, err := e.CreateSlot("r1")
	mustOK(t, err)
	if _, err := e.CreateSlot("r1"); !errors.Is(err, ErrSlotExists) {
		t.Fatalf("duplicated slot: %v", err)
	}
	if _, err := e.CreateSlot(string(make([]byte, 65))); !errors.Is(err, ErrSlotLimit) {
		t.Fatalf("long name: %v", err)
	}
	if err := e.AdvanceSlot("nope", 1); !errors.Is(err, ErrSlotNotFound) {
		t.Fatalf("unknown slot: %v", err)
	}
	start := info.ConfirmedLSN
	for i := range 6 {
		writeBurst(t, e, s, fmt.Sprintf("b%d", i), 300)
	}
	readFrom(t, e, start, false) // retenido
	st := e.Stats()
	if st.Slots != 1 || st.SlotRetained < 4*e.cfg.wal.SegmentSize || st.SlotLag == 0 {
		t.Fatalf("stats: slots %d retained %d lag %d", st.Slots, st.SlotRetained, st.SlotLag)
	}
	if p, cause := e.pressure(); cause != PressureSlot || p <= 0 {
		t.Fatalf("pressure %v %q, want slot", p, cause)
	}
	segs := len(walSegments(t, e.fs))

	mustOK(t, e.AdvanceSlot("r1", uint64(e.log.Durable())))
	if err := e.AdvanceSlot("r1", uint64(e.log.Published())+10); !errors.Is(err, ErrWalGap) {
		t.Fatalf("advance past the WAL: %v", err)
	}
	writeBurst(t, e, s, "after", 10)
	if after := len(walSegments(t, e.fs)); after >= segs {
		t.Fatalf("segments %d → %d: nothing recycled after advancing", segs, after)
	}
	readFrom(t, e, start, true)
	mustOK(t, e.DropSlot("r1"))
	if err := e.DropSlot("r1"); !errors.Is(err, ErrSlotNotFound) {
		t.Fatalf("drop twice: %v", err)
	}
	if len(e.Slots()) != 0 || e.log.Holds() != 0 {
		t.Fatalf("slots %+v, holds %d", e.Slots(), e.log.Holds())
	}
}

// TestSlotCap: con MAX_SLOT_WAL_KEEP_SIZE la ranura que se atrasa más que el tope se invalida en un checkpoint: el
// WAL se recicla, el motor sigue sano, Stats lo cuenta y AdvanceSlot y WalSince dan ErrWalRemoved (la réplica
// necesita un respaldo nuevo). Sin tope, la ranura frena a los escritores (ErrThrottled) en vez de perder WAL.
func TestSlotCap(t *testing.T) {
	cfg := slotConfig()
	cfg.slotKeepSize = 1 << 20
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	s := mustStore(t, e, "a")
	info, err := e.CreateSlot("late")
	mustOK(t, err)
	for i := range 8 {
		writeBurst(t, e, s, fmt.Sprintf("b%d", i), 300)
	}
	got := slotNamed(t, e, "late")
	st := e.Stats()
	if !got.Invalid || got.Retained != 0 || st.SlotsInvalid != 1 || st.SlotsInvalidated != 1 {
		t.Fatalf("slot %+v, stats invalid %d invalidated %d", got, st.SlotsInvalid, st.SlotsInvalidated)
	}
	if err := e.AdvanceSlot("late", info.ConfirmedLSN+1); !errors.Is(err, ErrWalRemoved) {
		t.Fatalf("advance an invalidated slot: %v", err)
	}
	readFrom(t, e, info.ConfirmedLSN, true)
	if n := len(walSegments(t, e.fs)); n > 8 {
		t.Fatalf("%d segments kept after invalidating", n)
	}
	writeBurst(t, e, s, "still", 20)
	checkClean(t, e)

	cfg = slotConfig()
	cfg.maxWALSize = 1 << 20 // freno en 2 MB
	cfg.lockTimeout = 200 * time.Millisecond
	cfg.checkpointTimeout = 50 * time.Millisecond // el checkpoint de fondo no se reparte en una hora
	f := openTest(t, crashfs.New(), cfg)
	defer f.Close()
	s = mustStore(t, f, "a")
	_, err = f.CreateSlot("brake")
	mustOK(t, err)
	value := make([]byte, 1000)
	for i := 0; ; i++ {
		err := s.Put(fmt.Sprintf("k%05d", i), value)
		if errors.Is(err, ErrThrottled) {
			break
		}
		mustOK(t, err)
		if i > 20000 {
			t.Fatal("a slot without a cap never braked the writers")
		}
	}
	mustOK(t, f.AdvanceSlot("brake", uint64(f.log.Durable())))
	mustOK(t, f.Checkpoint())
	mustOK(t, s.Put("free", value))
}

// TestSlotPersist: las ranuras y sus avances sobreviven a Close/Open y a un corte (en los tres modos de crashfs)
// después del checkpoint que los guardó; el WAL que retienen sigue ahí al reabrir.
func TestSlotPersist(t *testing.T) {
	base := crashfs.New()
	cfg := slotConfig()
	e := openTest(t, base, cfg)
	s := mustStore(t, e, "a")
	_, err := e.CreateSlot("r1")
	mustOK(t, err)
	_, err = e.CreateSlot("r2")
	mustOK(t, err)
	writeBurst(t, e, s, "one", 200)
	mid := uint64(e.log.Durable())
	mustOK(t, e.AdvanceSlot("r1", mid))
	mustOK(t, e.Close())

	e = openTest(t, base, cfg)
	if got := slotNamed(t, e, "r1"); got.ConfirmedLSN != mid || got.RestartLSN != mid+1 || got.Invalid {
		t.Fatalf("r1 after reopening: %+v", got)
	}
	readFrom(t, e, slotNamed(t, e, "r2").ConfirmedLSN, false)
	mustOK(t, e.Close())

	for name, mode := range map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	} {
		t.Run(name, func(t *testing.T) {
			proc := newProc(base)
			e := openTest(t, proc, cfg)
			s := mustStore(t, e, "a")
			writeBurst(t, e, s, name, 200)
			saved := uint64(e.log.Durable())
			mustOK(t, e.AdvanceSlot("r1", saved))
			mustOK(t, e.Checkpoint()) // lo guarda
			writeBurst(t, e, s, name+"x", 100)
			mustOK(t, e.AdvanceSlot("r1", uint64(e.log.Durable()))) // sin guardar: puede perderse
			proc.kill(mode, 3)
			e.Close()

			e = openTest(t, vfs.FS(newProc(base)), cfg)
			defer e.Close()
			got := slotNamed(t, e, "r1")
			if got.ConfirmedLSN < saved || got.Invalid {
				t.Fatalf("r1 after the crash: %+v, saved %d", got, saved)
			}
			readFrom(t, e, got.ConfirmedLSN, false)
			readFrom(t, e, slotNamed(t, e, "r2").ConfirmedLSN, false)
			mustOK(t, e.Close())
		})
	}
}
