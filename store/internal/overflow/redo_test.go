package overflow

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// entry es un valor de una historia de prueba: cuándo se escribió y cuándo se liberó (0: nunca).
type entry struct {
	ref     Ref
	value   []byte
	written int
	freed   int
}

// history es una historia al azar sobre el store de prueba: escribe valores de 1 a 5 páginas y libera algunos.
type history struct {
	rng     *rand.Rand
	entries []*entry
	ops     int
}

/**
* step hace una operación al azar con e: escribir un valor nuevo (dos de cada tres veces) o liberar uno vivo.
* @param t testing.TB, e *env
**/
func (h *history) step(t testing.TB, e *env) {
	t.Helper()
	h.ops++
	live := h.liveAt(h.ops)
	if len(live) > 0 && h.rng.Intn(3) == 0 {
		v := live[h.rng.Intn(len(live))]
		if err := e.ov.Free(testFile, v.ref, page.XID(h.ops)); err != nil {
			t.Fatalf("Free: %v", err)
		}
		v.freed = h.ops
		return
	}
	value := randomValue(h.rng, inlineLimit+1+h.rng.Intn(5*ChunkSize))
	ref, err := e.ov.Write(testFile, page.XID(h.ops), value)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	h.entries = append(h.entries, &entry{ref: ref, value: value, written: h.ops})
}

// liveAt devuelve los valores escritos hasta la operación op y no liberados antes de ella.
func (h *history) liveAt(op int) []*entry {
	var live []*entry
	for _, v := range h.entries {
		if v.written <= op && (v.freed == 0 || v.freed > op) {
			live = append(live, v)
		}
	}
	return live
}

/**
* checkState comprueba el store tras la operación op: cada valor vivo (salvo los que skip excluye) se lee igual, y
* ninguna de sus páginas está en la lista de libres.
* @param t testing.TB, e *env, live []*entry, skip func(*entry) bool
**/
func checkState(t testing.TB, e *env, live []*entry, skip func(*entry) bool) {
	t.Helper()
	free := map[page.BlockNumber]bool{}
	for _, b := range freeList(t, e.pool) {
		free[b] = true
	}
	for _, v := range live {
		if skip != nil && skip(v) {
			continue
		}
		got, err := e.ov.Read(testFile, v.ref)
		if err != nil {
			t.Fatalf("Read del valor de la operación %d: %v", v.written, err)
		}
		if !bytes.Equal(got, v.value) {
			t.Fatalf("el valor de la operación %d no coincide", v.written)
		}
		for _, b := range chainBlocks(t, e.pool, v.ref) {
			if free[b] {
				t.Fatalf("el bloque %d del valor de la operación %d está en la lista de libres", b, v.written)
			}
		}
	}
}

// TestRedoIdempotent: rehacer el WAL sobre el disco sin páginas llega a los mismos valores; rehacerlo otra vez
// encima no cambia nada; y rehacerlo sobre el disco ya al día tampoco.
func TestRedoIdempotent(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, 64)
	createStore(t, e.sm, testFile)
	h := &history{rng: rand.New(rand.NewSource(10))}
	for i := range 30 {
		if i == 15 {
			if _, err := e.log.BeginCheckpoint(nil); err != nil {
				t.Fatal(err)
			}
		}
		h.step(t, e)
	}
	live := h.liveAt(h.ops)
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}

	// Disco anterior: solo la meta (la caché es chica, así que parte de las páginas sí llegó al disco).
	e.crash(crashfs.LoseUnsynced, 0)
	e2 := openEnv(t, fs, 64)
	checkState(t, e2, live, nil)

	// Rehacer dos veces sobre las mismas páginas.
	if _, err := wal.NewReader(fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		return redoAll(e2.pool, e2.sm, rec)
	}); err != nil {
		t.Fatal(err)
	}
	checkState(t, e2, live, nil)

	// Rehacer sobre el disco al día.
	if err := e2.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if err := e2.sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
	e2.crash(crashfs.LoseUnsynced, 0)
	e3 := openEnv(t, fs, 64)
	checkState(t, e3, live, nil)

	// Y el motor sigue escribiendo.
	h.step(t, e3)
	checkState(t, e3, h.liveAt(h.ops), nil)
}

// TestRedoRejectsBadRecords: un KindOverflowWrite con forma inválida no toca páginas.
func TestRedoRejectsBadRecords(t *testing.T) {
	e := newEnv(t, 16)
	good := wal.BlockRef{File: testFile, Fork: smgr.ForkMain, Block: 3, Flags: wal.BlockWillInit, Delta: []byte("x")}
	with := func(change func(b *wal.BlockRef)) []wal.BlockRef {
		b := good
		change(&b)
		return []wal.BlockRef{b}
	}
	bad := []wal.Record{
		{Kind: wal.KindLeafWrite, Blocks: []wal.BlockRef{good}, Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: []wal.BlockRef{good, good}, Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: []wal.BlockRef{good}},
		{Kind: wal.KindOverflowWrite, Blocks: with(func(b *wal.BlockRef) { b.Fork = smgr.ForkVM }), Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: with(func(b *wal.BlockRef) { b.Block = 0 }), Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: with(func(b *wal.BlockRef) { b.Flags = 0 }), Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: with(func(b *wal.BlockRef) { b.Delta = nil }), Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: with(func(b *wal.BlockRef) { b.Delta = make([]byte, ChunkSize+1) }), Main: encodeMain(0, 0)},
		{Kind: wal.KindOverflowWrite, Blocks: []wal.BlockRef{good}, Main: encodeMain(3, 0)},
	}
	for i, rec := range bad {
		rec.LSN = 100
		if err := Redo(e.pool, e.sm, &rec); !errors.Is(err, ErrBadRecord) {
			t.Fatalf("registro %d: %v, se esperaba ErrBadRecord", i, err)
		}
	}
	if n, _ := e.sm.NBlocks(testFile, smgr.ForkMain); n != 1 {
		t.Fatalf("un registro inválido extendió el archivo a %d bloques", n)
	}
}

// TestCrashRecovery: en un disco que se corta (pierde, reordena o rompe por sectores lo que no tuvo fsync), con
// parte de las páginas escritas al disco y parte del WAL durable, rehacer deja todos los valores de la última
// operación que quedó entera en el WAL, y nunca menos de lo que era durable. Solo se excluye un valor cuya
// liberación quedó a medias (ver «a confirmar» en doc.go).
func TestCrashRecovery(t *testing.T) {
	modes := map[string]crashfs.Mode{"LoseUnsynced": crashfs.LoseUnsynced, "ReorderUnsynced": crashfs.ReorderUnsynced, "TornWrites": crashfs.TornWrites}
	for name, mode := range modes {
		for seed := int64(0); seed < 16; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := openEnv(t, fs, 48)
				createStore(t, e.sm, testFile)
				h := &history{rng: rand.New(rand.NewSource(seed))}

				const ops = 40
				lsns := []page.LSN{e.log.Published()}
				for i := range ops {
					h.step(t, e)
					lsns = append(lsns, e.log.Published())
					switch i {
					case ops / 3: // un checkpoint a medias
						if _, err := e.log.BeginCheckpoint(nil); err != nil {
							t.Fatal(err)
						}
					case ops / 2: // las páginas sucias al disco, sin fsync de los archivos de datos
						if err := e.pool.FlushAll(nil); err != nil {
							t.Fatal(err)
						}
					case 2 * ops / 3:
						if err := e.log.WaitDurable(e.log.Published()); err != nil {
							t.Fatal(err)
						}
					}
				}
				durable := e.log.Durable()

				e.crash(mode, seed)
				e2 := openEnv(t, fs, 48)
				recovered := e2.log.Published()
				if recovered < durable {
					t.Fatalf("se recuperó hasta el lsn %d y era durable hasta %d", recovered, durable)
				}
				k := 0
				for k+1 < len(lsns) && lsns[k+1] <= recovered {
					k++
				}

				halfFreed := func(v *entry) bool { return v.freed == k+1 }
				checkState(t, e2, h.liveAt(k), halfFreed)
			})
		}
	}
}
