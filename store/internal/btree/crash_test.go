package btree

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// errCut simula el corte entre los dos pasos de una partición.
var errCut = errors.New("corte entre KindSplit y KindInsertPivot")

/**
* cutNextSplit hace que la próxima partición quede incompleta (el hook devuelve errCut una sola vez) y devuelve
* cuántas veces se cortó.
* @param tr *BTree
* @return *int
**/
func cutNextSplit(tr *BTree) *int {
	cuts := 0
	tr.afterSplit = func(page.BlockNumber) error {
		if cuts > 0 {
			return nil
		}
		cuts++
		return errCut
	}
	return &cuts
}

/**
* fillUntilCut escribe ids crecientes hasta que una partición se corta.
* @param t testing.TB, tr *BTree, m model, prefix string, start int
* @return int: el próximo número de id
**/
func fillUntilCut(t testing.TB, tr *BTree, m model, prefix string, start int) int {
	t.Helper()
	cuts := cutNextSplit(tr)
	for i := start; ; i++ {
		id := fmt.Sprintf("%s%06d", prefix, i)
		value := []byte(fmt.Sprintf("value %d with padding padding padding padding", i))
		_, err := tr.Write(id, Write{XID: 1, Value: value, Commit: true})
		if errors.Is(err, errCut) {
			// El valor no se escribió: la partición se hizo antes de insertar.
			tr.afterSplit = nil
			return i
		}
		if err != nil {
			t.Fatal(err)
		}
		m[id] = value
		if *cuts > 0 {
			t.Fatal("el corte no devolvió errCut")
		}
	}
}

// TestIncompleteSplit: una partición cortada entre sus dos pasos (de la raíz y de una hoja no raíz) deja un árbol
// válido donde todo se lee (por el enlace a la derecha); sobrevive a un corte del disco y la próxima escritura que
// pasa por ella la termina.
func TestIncompleteSplit(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 24)
	tr := createTree(t, e)
	m := model{}

	// 1. La raíz es una hoja: su partición queda sin raíz nueva.
	next := fillUntilCut(t, tr, m, "a", 0)
	if st := checkTree(t, tr, m); st.Incomplete != 1 || st.Levels != 1 || st.Pages != 2 {
		t.Fatalf("raíz partida a medias: %+v", st)
	}
	next = writeRange(t, tr, m, "a", next, 5)
	if st := checkTree(t, tr, m); st.Incomplete != 0 || st.Levels != 2 {
		t.Fatalf("tras terminar la raíz: %+v", st)
	}

	// 2. Una hoja que no es la raíz, y un corte del disco con la partición a medias.
	next = writeRange(t, tr, m, "a", next, 300)
	next = fillUntilCut(t, tr, m, "a", next)
	if st := checkTree(t, tr, m); st.Incomplete != 1 {
		t.Fatalf("hoja partida a medias: %+v", st)
	}
	redo := checkpoint(t, e)
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 1)
	e2 := openEnv(t, fs, redo, 24)
	tr2 := openTree(t, e2)
	if st := checkTree(t, tr2, m); st.Incomplete != 1 {
		t.Fatalf("tras el corte: %+v", st)
	}
	writeRange(t, tr2, m, "a", next, 5)
	if st := checkTree(t, tr2, m); st.Incomplete != 0 {
		t.Fatalf("tras terminar la hoja: %+v", st)
	}
}

/**
* writeRange escribe n ids crecientes desde start.
* @param t testing.TB, tr *BTree, m model, prefix string, start, n int
* @return int: el próximo número
**/
func writeRange(t testing.TB, tr *BTree, m model, prefix string, start, n int) int {
	t.Helper()
	for i := start; i < start+n; i++ {
		put(t, tr, m, fmt.Sprintf("%s%06d", prefix, i), []byte(fmt.Sprintf("value %d with padding padding padding padding", i)))
	}
	return start + n
}

// op es una escritura confirmada en el WAL, para rearmar el modelo hasta un lsn.
type op struct {
	lsn   page.LSN
	id    string
	value []byte
	tomb  bool
}

/**
* modelUpTo rearma el modelo con las escrituras de lsn ≤ upTo.
* @param ops []op, upTo page.LSN
* @return model
**/
func modelUpTo(ops []op, upTo page.LSN) model {
	m := model{}
	for _, o := range ops {
		if o.lsn > upTo {
			break
		}
		if o.tomb {
			delete(m, o.id)
		} else {
			m[o.id] = o.value
		}
	}
	return m
}

// TestCrashRecovery: en un disco que se corta en los tres modos de crashfs (pierde, reordena o rompe por sectores lo
// que no tuvo fsync), con un checkpoint a mitad, páginas que llegan al disco al ser expulsadas de una caché chica y
// parte del WAL durable, rehacer desde el último checkpoint deja exactamente lo que quedó en el WAL (nunca menos de
// lo durable), con la estructura válida, y el árbol sigue aceptando escrituras.
func TestCrashRecovery(t *testing.T) {
	modes := []struct {
		name string
		mode crashfs.Mode
	}{{"LoseUnsynced", crashfs.LoseUnsynced}, {"ReorderUnsynced", crashfs.ReorderUnsynced}, {"TornWrites", crashfs.TornWrites}}
	for _, md := range modes {
		for seed := range int64(6) {
			t.Run(fmt.Sprintf("%s/seed%d", md.name, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := openEnv(t, fs, wal.Start, 20)
				tr := createTree(t, e)
				redo := checkpoint(t, e)
				rng := rand.New(rand.NewSource(seed))
				var ops []op
				live := model{}
				const n = 500
				for i := range n {
					id := randomID(rng, 12)
					o := op{id: id}
					var res Result
					if _, ok := live[id]; ok && rng.Intn(6) == 0 {
						o.tomb, res = true, remove(t, tr, live, id)
					} else {
						o.value = randomValue(rng, 200)
						res = put(t, tr, live, id, o.value)
					}
					o.lsn = res.LSN
					ops = append(ops, o)
					switch {
					case i == n/2:
						redo = checkpoint(t, e)
					case rng.Intn(60) == 0:
						if err := e.log.WaitDurable(e.log.Published()); err != nil {
							t.Fatal(err)
						}
					}
				}
				durable := e.log.Durable()

				fs.Crash(md.mode, seed)
				e2 := openEnv(t, fs, redo, 20)
				recovered := e2.log.Published()
				if recovered < durable {
					t.Fatalf("se recuperó hasta el lsn %d y era durable hasta %d", recovered, durable)
				}
				tr2 := openTree(t, e2)
				m := modelUpTo(ops, recovered)
				checkTree(t, tr2, m)
				rng2 := rand.New(rand.NewSource(seed + 100))
				for range 100 {
					put(t, tr2, m, randomID(rng2, 12), randomValue(rng2, 200))
				}
				checkTree(t, tr2, m)
			})
		}
	}
}

// TestRedoTwice: rehacer el WAL otra vez sobre las páginas ya rehechas (y sobre el disco ya al día) no cambia nada.
func TestRedoTwice(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 20)
	tr := createTree(t, e)
	m := model{}
	rng := rand.New(rand.NewSource(9))
	for range 1500 {
		put(t, tr, m, randomID(rng, 10), randomValue(rng, 100))
	}
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e2 := openEnv(t, fs, wal.Start, 20)
	checkTree(t, openTree(t, e2), m)

	if _, err := wal.NewReader(fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		return redoAny(e2.pool, e2.sm, rec)
	}); err != nil {
		t.Fatal(err)
	}
	checkTree(t, openTree(t, e2), m)

	if err := e2.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if err := e2.sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e3 := openEnv(t, fs, wal.Start, 20)
	checkTree(t, openTree(t, e3), m)
}

// TestRedoRejectsBadRecords: un registro de btree con forma inválida no toca páginas.
func TestRedoRejectsBadRecords(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 16)
	createTree(t, e)
	block := func(blk page.BlockNumber, flags wal.BlockFlags, delta []byte) wal.BlockRef {
		return wal.BlockRef{File: testFile, Fork: smgr.ForkMain, Block: blk, Flags: flags, Delta: delta}
	}
	bad := []wal.Record{
		{Kind: wal.KindLeafWrite},
		{Kind: wal.KindLeafWrite, Blocks: []wal.BlockRef{block(0, 0, []byte{1, 0, 1})}},
		{Kind: wal.KindLeafWrite, Blocks: []wal.BlockRef{block(1, 0, []byte{1})}},
		{Kind: wal.KindLeafWrite, Blocks: []wal.BlockRef{block(1, 0, []byte{1, 0, 1, 2, 3})}},
		{Kind: wal.KindSplit, Blocks: []wal.BlockRef{block(1, 0, nil), block(2, 0, nil)}},
		{Kind: wal.KindInitStore, Blocks: []wal.BlockRef{block(0, wal.BlockWillInit, nil), block(1, wal.BlockWillInit, nil)}},
		{Kind: wal.KindNewRoot, Blocks: []wal.BlockRef{block(3, wal.BlockWillInit, nil), block(1, 0, nil), block(1, 0, nil)}},
		{Kind: wal.KindLeafVacuum, Blocks: []wal.BlockRef{block(1, 0, nil)}},
		{Kind: wal.KindFreePush},
	}
	for i, rec := range bad {
		rec.LSN = 1 << 40
		if err := Redo(e.pool, e.sm, &rec); !errors.Is(err, ErrBadRecord) {
			t.Fatalf("registro %d: %v, se esperaba ErrBadRecord", i, err)
		}
	}
	checkTree(t, openTree(t, e), model{})
}

// TestCorruptPageOnRead: una hoja con la suma correcta pero un puntero a ítem fuera de lugar se rechaza al leerla
// (page.CheckItems), con ErrCorrupt y sin pánico.
func TestCorruptPageOnRead(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 16)
	tr := createTree(t, e)
	put(t, tr, model{}, "k", []byte("v"))
	redo := checkpoint(t, e)

	buf := make(page.Page, page.Size)
	if err := e.sm.Read(testFile, smgr.ForkMain, rootBlock, buf); err != nil {
		t.Fatal(err)
	}
	le.PutUint32(buf[page.HeaderSize:], le.Uint32(buf[page.HeaderSize:])^0x7FFF) // lp_off del ítem 1
	buf.SetChecksum(rootBlock)
	if err := e.sm.Write(testFile, smgr.ForkMain, rootBlock, buf); err != nil {
		t.Fatal(err)
	}
	if err := e.sm.SyncAll(); err != nil {
		t.Fatal(err)
	}
	fs.Crash(crashfs.LoseUnsynced, 0)
	e2 := openEnv(t, fs, redo, 16) // nada que rehacer: la hoja dañada queda como está
	tr2 := openTree(t, e2)
	if _, _, err := tr2.Lookup("k", nil); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Lookup sobre la hoja dañada: %v, se esperaba ErrCorrupt", err)
	}
	if _, err := tr2.Write("k", Write{XID: 1, Value: []byte("w")}); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Write sobre la hoja dañada: %v, se esperaba ErrCorrupt", err)
	}
}
