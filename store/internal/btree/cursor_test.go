package btree

import (
	"bytes"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// TestCursorOptions: el Cursor con Ring, KeysOnly y Borrow, hacia los dos lados, entrega los mismos ids que el
// modelo; con Borrow los valores son los del modelo mientras dura la hoja; con KeysOnly no hay valores; Exists
// coincide con el modelo. Corre sobre una caché chica (el anillo reutiliza sus búferes).
func TestCursorOptions(t *testing.T) {
	e := openEnv(t, crashfs.New(), wal.Start, 24)
	tr := createTree(t, e)
	m := model{}
	rng := rand.New(rand.NewSource(7))
	for range 2500 {
		id := randomID(rng, 6)
		if rng.Intn(5) == 0 {
			if _, live := m[id]; live {
				remove(t, tr, m, id)
			}
			continue
		}
		put(t, tr, m, id, randomValue(rng, 80))
	}
	ids := m.sortedIDs()
	back := slices.Clone(ids)
	slices.Reverse(back)

	for _, opts := range []ScanOptions{
		{Ring: true}, {KeysOnly: true}, {Borrow: true, Ring: true},
		{Backward: true, Ring: true}, {Backward: true, KeysOnly: true}, {Backward: true, Borrow: true},
	} {
		t.Run(fmt.Sprintf("%+v", opts), func(t *testing.T) {
			want := ids
			if opts.Backward {
				want = back
			}
			got := cursorIDs(t, tr, m, opts)
			if !slices.Equal(got, want) {
				t.Fatalf("%d ids, se esperaban %d (primer distinto %s)", len(got), len(want), firstDiff(got, want))
			}
		})
	}
	for range 500 {
		id := randomID(rng, 6)
		_, want := m[id]
		got, err := tr.Exists(id, nil)
		if err != nil || got != want {
			t.Fatalf("Exists(%q) = %v, %v; se esperaba %v", id, got, err, want)
		}
	}
}

/**
* cursorIDs recorre con opts y comprueba los valores de cada hoja contra el modelo antes de pedir la siguiente.
* @param t testing.TB, tr *BTree, m model, opts ScanOptions
* @return []string
**/
func cursorIDs(t testing.TB, tr *BTree, m model, opts ScanOptions) []string {
	t.Helper()
	c := tr.NewCursor(opts)
	var got []string
	for {
		batch, more, err := c.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		for _, v := range batch {
			switch {
			case opts.KeysOnly && v.Value != nil:
				t.Fatalf("KeysOnly devolvió el valor de %q", short(v.ID))
			case !opts.KeysOnly && !bytes.Equal(v.Value, m[v.ID]):
				t.Fatalf("%q = %q, se esperaba %q", short(v.ID), v.Value, m[v.ID])
			}
			got = append(got, v.ID)
		}
		if !more {
			return got
		}
	}
}
