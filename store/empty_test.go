package store

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// TestEmpty: Empty deja el store vacío (también Count) con un archivo nuevo, el mismo *Store sigue sirviendo, los
// demás stores no cambian, y todo sigue igual tras reabrir y tras un corte sin checkpoint (la recuperación rehace
// KindEmptyStore).
func TestEmpty(t *testing.T) {
	for _, mode := range []crashfs.Mode{crashfs.LoseUnsynced, crashfs.ReorderUnsynced, crashfs.TornWrites} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			base := crashfs.New()
			proc := newProc(base)
			e := openTest(t, proc, testConfig())
			s, other := mustStore(t, e, "s"), mustStore(t, e, "other")
			fillStore(t, s, 300)
			mustOK(t, other.Put("x", []byte("o")))
			mustOK(t, e.Checkpoint())
			fillStore(t, s, 50) // sin checkpoint: la recuperación las rehace antes del vaciado
			old := s.file
			mustOK(t, s.Empty())
			if s.file == old || s.Count() != 0 {
				t.Fatalf("Empty: archivo %d → %d, Count %d", old, s.file, s.Count())
			}
			checkBool(t, "exists k0001", false)(s.Exists("k0001"))
			keys, err := s.Keys(true, 0, 0)
			mustOK(t, err)
			if len(keys) != 0 {
				t.Fatalf("Keys tras Empty = %v", keys)
			}
			mustOK(t, s.Put("after", bytes.Repeat([]byte("a"), 9000)))
			mustOK(t, s.Put("small", []byte("s")))
			if names, _ := base.ReadDir(testDir + "/data"); slices.ContainsFunc(names, func(n string) bool {
				return bytes.HasPrefix([]byte(n), []byte(fmt.Sprintf("%010d", old)))
			}) {
				t.Fatalf("los archivos viejos siguen: %v", names)
			}
			proc.kill(mode, 1)
			closeWithin(t, e, 10*time.Second)

			e = openTest(t, newProc(base), testConfig())
			defer e.Close()
			want := map[string][]byte{"after": bytes.Repeat([]byte("a"), 9000), "small": []byte("s")}
			checkContent(t, mustStore(t, e, "s"), want, []string{"k0001", "k0299"})
			checkContent(t, mustStore(t, e, "other"), map[string][]byte{"x": []byte("o")}, nil)
			if n := mustStore(t, e, "s").Count(); n != 2 {
				t.Fatalf("Count tras recuperar = %d", n)
			}
			verifyTrees(t, e)
		})
	}
}

// TestEmptyBusy: con un lote o un snapshot abierto, Empty devuelve ErrBusy y no cambia nada; al terminar, vacía.
// Empty de un store borrado devuelve ErrStoreDropped.
func TestEmptyBusy(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	mustOK(t, s.Put("a", []byte("1")))

	v := e.Snapshot()
	if err := s.Empty(); !errors.Is(err, ErrBusy) {
		t.Fatalf("Empty con snapshot: %v", err)
	}
	mustOK(t, v.Close())
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "b", []byte("2")))
	if err := s.Empty(); !errors.Is(err, ErrBusy) {
		t.Fatalf("Empty con lote: %v", err)
	}
	mustOK(t, tx.Commit())
	if s.Count() != 2 {
		t.Fatalf("Count = %d", s.Count())
	}
	tx = e.NewBatch() // un lote abortado por tiempo o deshecho también se descuenta
	mustOK(t, tx.Rollback())
	mustOK(t, s.Empty())
	if s.Count() != 0 {
		t.Fatalf("Count tras Empty = %d", s.Count())
	}
	mustOK(t, e.DropStore("s"))
	if err := s.Empty(); !errors.Is(err, ErrStoreDropped) {
		t.Fatalf("Empty de un store borrado: %v", err)
	}
}

// TestEmptyRecords: KindEmptyStore va y vuelve, y los mal formados se rechazan.
func TestEmptyRecords(t *testing.T) {
	old, file, name, ok := decodeEmptyMain(encodeEmptyMain(3, 9, "ñandú"))
	if !ok || old != 3 || file != 9 || name != "ñandú" {
		t.Fatalf("empty main = %d %d %q %v", old, file, name, ok)
	}
	for _, bad := range [][]byte{nil, encodeEmptyMain(0, 9, "x"), encodeEmptyMain(4, 4, "x"), encodeEmptyMain(3, 9, "x")[:10]} {
		if _, _, _, ok := decodeEmptyMain(bad); ok {
			t.Fatalf("decodeEmptyMain aceptó %v", bad)
		}
	}
	if wal.KindEmptyStore != 7 {
		t.Fatal("KindEmptyStore cambió de número")
	}
}
