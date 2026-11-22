package btree

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Pruebas de la tarea 108 de F-23: el llenado de las hojas (partir en el punto de inserción ascendente y podar las
// versiones muertas antes de partir).

/**
* leafFill es el llenado medio de las hojas del árbol: los bytes ocupados por los ítems de datos (con su puntero)
* sobre la capacidad de la página, sin contar la de más a la derecha (está a medio llenar por construcción).
* @param t testing.TB, tr *BTree
* @return float64, int: el llenado y cuántas hojas
**/
func leafFill(t testing.TB, tr *BTree) (float64, int) {
	t.Helper()
	h, n, _, err := tr.descend(minusInf, 0, buffer.Share, false)
	if err != nil {
		t.Fatal(err)
	}
	used, leaves := 0, 0
	for {
		if !n.rightmost() {
			for off := n.firstData(); off <= n.maxOff(); off++ {
				it, err := n.item(off)
				if err != nil {
					t.Fatal(err)
				}
				used += page.SpaceFor(len(it))
			}
			leaves++
		}
		next := n.o.Next
		h.Release()
		if next == 0 {
			break
		}
		if h, n, err = tr.lockNode(next, buffer.Share); err != nil {
			t.Fatal(err)
		}
	}
	if leaves == 0 {
		return 0, 0
	}
	return float64(used) / float64(leaves*capacity), leaves
}

// groupID es un id de índice de josefina: el valor (el grupo) y un sufijo creciente (el jid).
func groupID(group, seq int) string {
	return fmt.Sprintf("v%02d|%08d", group, seq)
}

// TestSplitAscendingGroups: con inserciones al final de varios grupos de duplicados de sufijo creciente (un índice
// valor + jid), las hojas quedan casi llenas (partir en el punto de inserción); con inserciones al azar, mitad y mitad
// como antes (el llenado de un árbol al azar, ~70 %). Cada grupo deja su hoja de cola a medio llenar (recibe las
// inserciones, como la de más a la derecha) y a veces un resto del comienzo del grupo siguiente: por eso la cota baja
// con la cantidad de grupos (unas 13 hojas por grupo con 20).
func TestSplitAscendingGroups(t *testing.T) {
	for _, c := range []struct {
		groups int
		min    float64
	}{{1, 0.88}, {5, 0.85}, {20, 0.8}} {
		t.Run(fmt.Sprintf("grupos=%d", c.groups), func(t *testing.T) {
			e := openEnv(t, crashfs.New(), wal.Start, 256)
			tr := createTree(t, e)
			rng := rand.New(rand.NewSource(int64(c.groups)))
			m := model{}
			for seq := range 2000 * c.groups {
				put(t, tr, m, groupID(rng.Intn(c.groups), seq), nil)
			}
			fill, leaves := leafFill(t, tr)
			t.Logf("%d grupos: %d hojas, llenado %.0f %%", c.groups, leaves, 100*fill)
			if fill < c.min {
				t.Fatalf("llenado %.2f, se esperaba al menos %.2f", fill, c.min)
			}
			checkTree(t, tr, m)
		})
	}
	t.Run("al azar", func(t *testing.T) {
		e := openEnv(t, crashfs.New(), wal.Start, 256)
		tr := createTree(t, e)
		rng := rand.New(rand.NewSource(9))
		m := model{}
		for range 20_000 {
			put(t, tr, m, fmt.Sprintf("%016x", rng.Uint64()), nil)
		}
		fill, leaves := leafFill(t, tr)
		t.Logf("al azar: %d hojas, llenado %.0f %%", leaves, 100*fill)
		if fill < 0.6 || fill > 0.8 {
			t.Fatalf("llenado %.2f con inserciones al azar, se esperaba entre 0,6 y 0,8", fill)
		}
		checkTree(t, tr, m)
	})
}

/**
* updateLoop reescribe al azar ids de un conjunto fijo, cada vez con un xid nuevo, y devuelve el modelo.
* @param t testing.TB, tr *BTree, rng *rand.Rand, ids, rounds int, xid *page.XID
* @return model
**/
func updateLoop(t testing.TB, tr *BTree, rng *rand.Rand, ids, rounds int, xid *page.XID) model {
	t.Helper()
	m := model{}
	for range rounds {
		*xid++
		id := fmt.Sprintf("doc%05d", rng.Intn(ids))
		value := randomValue(rng, 200)
		if rng.Intn(10) == 0 {
			if _, err := tr.Write(id, Write{XID: *xid, Tombstone: true, Commit: true}); err != nil {
				t.Fatal(err)
			}
			delete(m, id)
			continue
		}
		if _, err := tr.Write(id, Write{XID: *xid, Value: value, Commit: true}); err != nil {
			t.Fatal(err)
		}
		m[id] = value
	}
	return m
}

// TestPruneBeforeSplit: con el horizonte por delante de todo lo escrito, las actualizaciones de un conjunto fijo de
// ids podan las versiones muertas de la hoja en lugar de partirla (el árbol queda del tamaño de lo vivo); con el
// horizonte atrás (un snapshot viejo abierto) no se poda nada y el árbol crece como antes.
func TestPruneBeforeSplit(t *testing.T) {
	grow := func(horizon func(xid page.XID) page.XID) (*BTree, int, model) {
		e := openEnv(t, crashfs.New(), wal.Start, 256)
		cfg := e.config()
		xid := page.XID(10)
		cfg.Horizon = func() page.XID { return horizon(xid) }
		tr, err := Create(cfg)
		if err != nil {
			t.Fatal(err)
		}
		m := updateLoop(t, tr, rand.New(rand.NewSource(3)), 300, 20_000, &xid)
		st := checkTree(t, tr, m)
		return tr, st.Pages, m
	}
	trPrune, pagesPrune, _ := grow(func(xid page.XID) page.XID { return xid + 1 })
	trKeep, pagesKeep, _ := grow(func(page.XID) page.XID { return 1 })
	t.Logf("con poda: %d páginas, %d versiones quitadas; sin poda (horizonte atrás): %d páginas, %d quitadas",
		pagesPrune, trPrune.Pruned(), pagesKeep, trKeep.Pruned())
	if trPrune.Pruned() == 0 || trKeep.Pruned() != 0 {
		t.Fatalf("versiones podadas: %d con el horizonte adelante (se esperaban > 0), %d con el horizonte atrás "+
			"(se esperaban 0)", trPrune.Pruned(), trKeep.Pruned())
	}
	if pagesPrune*10 > pagesKeep {
		t.Fatalf("con poda %d páginas, sin poda %d: se esperaba al menos 10 veces menos", pagesPrune, pagesKeep)
	}
}

/**
* pagesOf copia las páginas del árbol desde la caché, con la suma en cero (se calcula al escribir al disco).
* @param t testing.TB, tr *BTree
* @return map[page.BlockNumber][]byte
**/
func pagesOf(t testing.TB, tr *BTree) map[page.BlockNumber][]byte {
	t.Helper()
	out := map[page.BlockNumber][]byte{}
	for _, blk := range treeBlocks(t, tr) {
		h, n, err := tr.lockNode(blk, buffer.Share)
		if err != nil {
			t.Fatal(err)
		}
		b := bytes.Clone(n.p[:page.Size])
		h.Release()
		b[page.OffChecksum], b[page.OffChecksum+1] = 0, 0
		out[blk] = b
	}
	return out
}

// TestPruneRedoIdentical: lo que dejan las podas (y las particiones) en las páginas es lo mismo, byte a byte, que lo
// que deja rehacer el WAL desde el principio sobre archivos vacíos (lo que hace una réplica), y rehacerlo dos veces
// deja lo mismo.
func TestPruneRedoIdentical(t *testing.T) {
	fs := crashfs.New()
	e := openEnv(t, fs, wal.Start, 4096)
	cfg := e.config()
	xid := page.XID(10)
	cfg.Horizon = func() page.XID { return xid + 1 }
	tr, err := Create(cfg)
	if err != nil {
		t.Fatal(err)
	}
	m := updateLoop(t, tr, rand.New(rand.NewSource(4)), 400, 8_000, &xid)
	if tr.Pruned() == 0 {
		t.Fatal("no se podó nada")
	}
	want := pagesOf(t, tr)
	if err := e.log.WaitDurable(e.log.Published()); err != nil {
		t.Fatal(err)
	}
	same := func(tr *BTree) {
		t.Helper()
		got := pagesOf(t, tr)
		if len(got) != len(want) {
			t.Fatalf("%d páginas, se esperaban %d", len(got), len(want))
		}
		for blk, b := range want {
			if !bytes.Equal(got[blk], b) {
				t.Fatalf("la página %d no es igual a la del primario", blk)
			}
		}
		checkTree(t, tr, m)
	}
	fs.Crash(crashfs.LoseUnsynced, 0) // las páginas nunca llegaron al disco: todo sale del WAL
	e2 := openEnv(t, fs, wal.Start, 4096)
	tr2 := openTree(t, e2)
	same(tr2)
	if _, err := wal.NewReader(fs, testDir).Scan(wal.Start, func(rec *wal.Record) error {
		return redoAny(e2.pool, e2.sm, rec)
	}); err != nil {
		t.Fatal(err)
	}
	same(tr2)
}

// TestPruneHorizonUnlocked: la poda antes de partir lee el horizonte (Config.Horizon, que en el motor toma el mutex de
// xact) con la hoja suelta: mientras se calcula, otro puede leer cualquier hoja del árbol.
func TestPruneHorizonUnlocked(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 256)
	cfg := e.config()
	xid := page.XID(10)
	var tr *BTree
	calls, locked := 0, 0
	cfg.Horizon = func() page.XID {
		calls++
		if locked > 0 {
			return xid + 1
		}
		done := make(chan error, 1)
		go func() { done <- readAllLeaves(tr) }()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(500 * time.Millisecond):
			locked++ // una hoja sigue bloqueada en exclusivo
		}
		return xid + 1
	}
	var err error
	if tr, err = Create(cfg); err != nil {
		t.Fatal(err)
	}
	m := updateLoop(t, tr, rand.New(rand.NewSource(5)), 100, 1_500, &xid)
	if calls == 0 || tr.Pruned() == 0 {
		t.Fatalf("%d lecturas del horizonte, %d versiones podadas: se esperaba alguna de cada una", calls, tr.Pruned())
	}
	if locked > 0 {
		t.Fatalf("%d de %d lecturas del horizonte con una hoja bloqueada", locked, calls)
	}
	checkTree(t, tr, m)
}

/**
* readAllLeaves recorre las hojas de izquierda a derecha, cada una bloqueada en compartido.
* @param tr *BTree
* @return error
**/
func readAllLeaves(tr *BTree) error {
	h, n, _, err := tr.descend(minusInf, 0, buffer.Share, false)
	for err == nil {
		next := n.o.Next
		h.Release()
		if next == 0 {
			return nil
		}
		h, n, err = tr.lockNode(next, buffer.Share)
	}
	return err
}
