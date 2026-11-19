package fsm

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// probes son los bloques del mapa que tocan las historias de prueba (bordes de byte y de página).
var probes = []page.BlockNumber{0, 3, 4, VMBlocksPerPage - 1, VMBlocksPerPage, 2*VMBlocksPerPage + 1}

// history es una historia de operaciones al azar sobre el store 1: pide, devuelve, pone y borra bits.
type history struct {
	rng  *rand.Rand
	held []page.BlockNumber
}

/**
* step hace una operación al azar con e.
* @param t testing.TB, e *env
**/
func (h *history) step(t testing.TB, e *env) {
	t.Helper()
	switch op := h.rng.Intn(10); {
	case op < 4 || len(h.held) == 0:
		h.held = append(h.held, allocate(t, e.fsm, 1))
	case op < 7:
		i := h.rng.Intn(len(h.held))
		free(t, e.fsm, 1, h.held[i])
		h.held = slices.Delete(h.held, i, i+1)
	case op < 9:
		bits := []VMBits{VMAllVisible, VMAllFrozen}[h.rng.Intn(2)]
		if err := e.fsm.Set(1, probes[h.rng.Intn(len(probes))], bits); err != nil {
			t.Fatal(err)
		}
	default:
		if err := e.fsm.Clear(1, probes[h.rng.Intn(len(probes))]); err != nil {
			t.Fatal(err)
		}
	}
}

// TestRedoReachesSameState: rehacer el WAL sobre el disco anterior (solo la meta) llega al estado de antes; rehacerlo
// otra vez encima no cambia nada; y rehacerlo sobre el disco ya al día (con todas las páginas escritas) tampoco. A
// mitad de la historia empieza un checkpoint, así que parte de los registros lleva imágenes enteras.
func TestRedoReachesSameState(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs)
	createStore(t, e.sm, 1)
	h := &history{rng: rand.New(rand.NewSource(1))}
	for i := range 120 {
		if i == 60 {
			if _, err := e.log.BeginCheckpoint(nil); err != nil {
				t.Fatal(err)
			}
		}
		h.step(t, e)
	}
	for _, b := range h.held[1:] { // que la lista termine con páginas
		free(t, e.fsm, 1, b)
	}
	want := stateOf(t, e, 1, probes)
	if len(want.list) == 0 {
		t.Fatal("la historia no dejó páginas libres: no prueba nada")
	}
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}

	// Disco anterior: solo la meta (nada de la caché llegó al disco) y el WAL.
	fs.Crash(crashfs.LoseUnsynced, 0)
	e2 := openEnv(t, fs)
	if got := stateOf(t, e2, 1, probes); !got.equal(want) {
		t.Fatalf("tras rehacer: %+v, se esperaba %+v", got, want)
	}

	// Rehacer dos veces sobre las mismas páginas.
	if _, err := wal.NewReader(fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		return Redo(e2.pool, e2.sm, rec)
	}); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, e2, 1, probes); !got.equal(want) {
		t.Fatalf("tras rehacer dos veces: %+v, se esperaba %+v", got, want)
	}

	// Rehacer sobre el disco al día: todos los registros encuentran su cambio ya hecho.
	if err := e2.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if err := e2.sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e3 := openEnv(t, fs)
	if got := stateOf(t, e3, 1, probes); !got.equal(want) {
		t.Fatalf("tras rehacer sobre el disco al día: %+v, se esperaba %+v", got, want)
	}

	// Y el motor sigue: la próxima página sale de la cabeza de la lista.
	if got := allocate(t, e3.fsm, 1); got != want.list[0] {
		t.Fatalf("Allocate tras recuperar = %d, se esperaba %d", got, want.list[0])
	}
}

// TestRedoRejectsBadRecords: un registro de fsm con forma inválida no toca páginas.
func TestRedoRejectsBadRecords(t *testing.T) {
	e := openEnv(t, crashfs.New())
	createStore(t, e.sm, 1)
	meta := wal.BlockRef{File: 1, Fork: smgr.ForkMain, Block: 0, Delta: encodeBlock(3)}
	bad := []wal.Record{
		{Kind: wal.KindFreePush, Blocks: []wal.BlockRef{meta}},
		{Kind: wal.KindFreePush, Blocks: []wal.BlockRef{meta, {File: 1, Block: 3, Delta: encodeBlock(0)}}},
		{Kind: wal.KindFreePop, Blocks: []wal.BlockRef{meta, {File: 2, Block: 3, Flags: wal.BlockWillInit}}},
		{Kind: wal.KindVMSet, Blocks: []wal.BlockRef{{File: 1, Fork: smgr.ForkVM, Block: 1, Delta: encodeVM(5, 1)}}},
		{Kind: wal.KindVMSet, Blocks: []wal.BlockRef{{File: 1, Fork: smgr.ForkVM, Block: 0, Delta: encodeVM(5, 4)}}},
		{Kind: wal.KindLeafWrite},
	}
	for i, rec := range bad {
		rec.LSN = 100
		if err := Redo(e.pool, e.sm, &rec); !errors.Is(err, ErrBadRecord) {
			t.Fatalf("registro %d: %v, se esperaba ErrBadRecord", i, err)
		}
	}
}

// TestCrashRecovery: en un disco que se corta (pierde, reordena o rompe por sectores lo que no tuvo fsync), con parte de las páginas
// escritas al disco y parte del WAL durable, rehacer deja exactamente el estado de después de la última operación
// que quedó en el WAL, y nunca menos de lo que era durable.
func TestCrashRecovery(t *testing.T) {
	modes := map[string]crashfs.Mode{"LoseUnsynced": crashfs.LoseUnsynced, "ReorderUnsynced": crashfs.ReorderUnsynced, "TornWrites": crashfs.TornWrites}
	for name, mode := range modes {
		for seed := int64(0); seed < 24; seed++ { // con TornWrites, pocas semillas no siempre rompen una página
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := openEnv(t, fs)
				createStore(t, e.sm, 1)
				h := &history{rng: rand.New(rand.NewSource(seed))}

				const ops = 60
				states := []state{stateOf(t, e, 1, probes)}
				lsns := []page.LSN{e.log.Published()}
				for i := range ops {
					h.step(t, e)
					states = append(states, stateOf(t, e, 1, probes))
					lsns = append(lsns, e.log.Published())
					switch i {
					case ops / 3: // un checkpoint a medias: las páginas siguientes llevan imagen
						if _, err := e.log.BeginCheckpoint(nil); err != nil {
							t.Fatal(err)
						}
					case ops / 2: // la mitad de las páginas va al disco (y el WAL hasta ellas es durable)
						if err := e.pool.flush(func(buffer.Tag) bool { return h.rng.Intn(2) == 0 }); err != nil {
							t.Fatal(err)
						}
					case 2 * ops / 3:
						if err := e.log.WaitDurable(e.log.Published()); err != nil {
							t.Fatal(err)
						}
					}
				}
				durable := e.log.Durable()

				fs.Crash(mode, seed)
				e2 := openEnv(t, fs)
				recovered := e2.log.Published()
				if recovered < durable {
					t.Fatalf("se recuperó hasta el lsn %d y era durable hasta %d", recovered, durable)
				}
				k := 0
				for k+1 < len(lsns) && lsns[k+1] <= recovered {
					k++
				}
				if got := stateOf(t, e2, 1, probes); !got.equal(states[k]) {
					t.Fatalf("tras el corte (lsn %d, operación %d): %+v, se esperaba %+v", recovered, k, got, states[k])
				}
			})
		}
	}
}
