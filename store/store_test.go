package store

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestRoundTrip: toda la API de la fase 1A, con valores chicos y grandes, y lo mismo tras reabrir.
func TestRoundTrip(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	s := mustStore(t, e, "users")
	big := bytes.Repeat([]byte("0123456789"), 3000) // 30 KB: desborde

	mustOK(t, s.Put("a", []byte("1")))
	mustOK(t, s.Put("big", big))
	checkBool(t, "insert a", false)(s.Insert("a", []byte("x")))
	checkBool(t, "insert b", true)(s.Insert("b", []byte("2")))
	checkBool(t, "insert big2", true)(s.Insert("big2", big))
	checkBool(t, "insert big2 again", false)(s.Insert("big2", big))
	checkBool(t, "update same", false)(s.Update("a", []byte("1")))
	checkBool(t, "update a", true)(s.Update("a", []byte("10")))
	checkBool(t, "update missing", false)(s.Update("zz", []byte("1")))
	checkBool(t, "delete missing", false)(s.Delete("zz"))
	checkBool(t, "delete b", true)(s.Delete("b"))
	mustOK(t, s.Remove("zz"))
	mustOK(t, s.Put("empty", nil))

	want := map[string][]byte{"a": []byte("10"), "big": big, "big2": big, "empty": {}}
	checkContent(t, s, want, []string{"b", "zz"})
	mustOK(t, e.Checkpoint())
	if names := e.Stores(); len(names) != 1 || names[0] != "users" {
		t.Fatalf("stores = %v", names)
	}
	mustOK(t, e.Close())

	e = openTest(t, fsys, testConfig())
	defer e.Close()
	checkContent(t, mustStore(t, e, "users"), want, []string{"b", "zz"})
	verifyTrees(t, e)
}

// TestManyStores: varios stores con las mismas claves no se mezclan, y siguen igual tras reabrir.
func TestManyStores(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	for i := range 5 {
		s := mustStore(t, e, fmt.Sprintf("s%d", i))
		for k := range 300 {
			mustOK(t, s.Put(fmt.Sprintf("k%04d", k), []byte(fmt.Sprintf("%d-%d", i, k))))
		}
	}
	mustOK(t, e.Close())
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	for i := range 5 {
		s := mustStore(t, e, fmt.Sprintf("s%d", i))
		if s.Count() != 300 {
			t.Fatalf("s%d count = %d", i, s.Count())
		}
		for k := range 300 {
			got, ok, err := s.Read(fmt.Sprintf("k%04d", k))
			if err != nil || !ok || string(got) != fmt.Sprintf("%d-%d", i, k) {
				t.Fatalf("s%d k%d = %q %v %v", i, k, got, ok, err)
			}
		}
	}
	verifyTrees(t, e)
}

// TestLimits: id de 1 a 2.000 B, valor hasta 64 MiB (por desborde) y nombres de store.
func TestLimits(t *testing.T) {
	e, err := open(vfs.OS(), t.TempDir(), testConfig())
	mustOK(t, err)
	defer e.Close()
	s := mustStore(t, e, "limits")

	long := strings.Repeat("i", MaxIDLen)
	mustOK(t, s.Put(long, []byte("v")))
	if got, ok, err := s.Read(long); err != nil || !ok || string(got) != "v" {
		t.Fatalf("read long id = %q %v %v", got, ok, err)
	}
	for _, id := range []string{"", long + "x"} {
		if err := s.Put(id, nil); !errors.Is(err, ErrBadID) {
			t.Fatalf("put id of %d bytes: %v", len(id), err)
		}
		if _, _, err := s.Read(id); !errors.Is(err, ErrBadID) {
			t.Fatalf("read id of %d bytes: %v", len(id), err)
		}
	}

	huge := make([]byte, MaxValueLen)
	for i := range huge {
		huge[i] = byte(i * 7)
	}
	mustOK(t, s.Put("huge", huge))
	if got, ok, err := s.Read("huge"); err != nil || !ok || !bytes.Equal(got, huge) {
		t.Fatalf("read huge: %d bytes %v %v", len(got), ok, err)
	}
	if err := s.Put("huge", make([]byte, MaxValueLen+1)); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("put too large: %v", err)
	}

	for _, name := range []string{"", strings.Repeat("n", MaxNameLen+1), "a\x00b", "\xff"} {
		if _, err := e.Store(name); !errors.Is(err, ErrBadName) {
			t.Fatalf("store %q: %v", name, err)
		}
	}
	mustStore(t, e, strings.Repeat("n", MaxNameLen))
	mustStore(t, e, "ñandú")
}

// TestLockTwice: un segundo motor sobre el mismo directorio falla con ErrLocked (que envuelve vfs.ErrLocked).
func TestLockTwice(t *testing.T) {
	for name, fsys := range map[string]vfs.FS{"os": vfs.OS(), "crashfs": crashfs.New()} {
		t.Run(name, func(t *testing.T) {
			dir := testDir
			if name == "os" {
				dir = t.TempDir()
			}
			e, err := open(fsys, dir, testConfig())
			if err != nil {
				t.Fatal(err)
			}
			_, err = open(fsys, dir, testConfig())
			if !errors.Is(err, ErrLocked) || !errors.Is(err, vfs.ErrLocked) {
				t.Fatalf("second open: %v", err)
			}
			mustOK(t, e.Close())
			e, err = open(fsys, dir, testConfig())
			if err != nil {
				t.Fatalf("open after close: %v", err)
			}
			mustOK(t, e.Close())
		})
	}
}

// TestDropStore: un store borrado desaparece con sus datos, su *Store deja de servir, y con el mismo nombre nace
// uno vacío; todo sigue igual tras reabrir.
func TestDropStore(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	mustOK(t, a.Put("x", bytes.Repeat([]byte("a"), 5000)))
	mustOK(t, b.Put("x", []byte("b")))
	mustOK(t, e.DropStore("a"))
	if err := a.Put("y", nil); !errors.Is(err, ErrStoreDropped) {
		t.Fatalf("put on dropped: %v", err)
	}
	if err := e.DropStore("a"); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("drop again: %v", err)
	}
	if names := e.Stores(); len(names) != 1 || names[0] != "b" {
		t.Fatalf("stores = %v", names)
	}
	a2 := mustStore(t, e, "a")
	if _, ok, _ := a2.Read("x"); ok || a2.Count() != 0 {
		t.Fatal("recreated store is not empty")
	}
	mustOK(t, a2.Put("z", []byte("new")))
	mustOK(t, e.Close())

	e = openTest(t, fsys, testConfig())
	defer e.Close()
	checkContent(t, mustStore(t, e, "a"), map[string][]byte{"z": []byte("new")}, []string{"x"})
	checkContent(t, mustStore(t, e, "b"), map[string][]byte{"x": []byte("b")}, nil)
}

// TestCleanReopenReplaysNothing (regla 2): tras un cierre normal, Open no rehace ningún registro.
func TestCleanReopenReplaysNothing(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	s := mustStore(t, e, "s")
	for i := range 2000 {
		mustOK(t, s.Put(fmt.Sprintf("k%d", i), bytes.Repeat([]byte("v"), i%3000)))
	}
	mustOK(t, e.Close())
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	if e.replayed != 0 {
		t.Fatalf("clean reopen replayed %d records", e.replayed)
	}
	if n := mustStore(t, e, "s").Count(); n != 2000 {
		t.Fatalf("count = %d", n)
	}
}

// TestClosed: tras Close todo devuelve ErrClosed y cerrar otra vez no hace nada.
func TestClosed(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	s := mustStore(t, e, "s")
	mustOK(t, e.Close())
	mustOK(t, e.Close())
	if err := s.Put("a", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("put after close: %v", err)
	}
	if _, err := e.Store("t"); !errors.Is(err, ErrClosed) {
		t.Fatalf("store after close: %v", err)
	}
	if err := e.Checkpoint(); !errors.Is(err, ErrClosed) {
		t.Fatalf("checkpoint after close: %v", err)
	}
}

/**
* mustOK falla la prueba si err no es nil.
* @param t testing.TB, err error
**/
func mustOK(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

/**
* checkBool comprueba el resultado de Insert, Update o Delete.
* @param t testing.TB, what string, want bool
* @return func(bool, error)
**/
func checkBool(t testing.TB, what string, want bool) func(bool, error) {
	return func(got bool, err error) {
		t.Helper()
		if err != nil || got != want {
			t.Fatalf("%s = %v, %v; want %v", what, got, err, want)
		}
	}
}

/**
* checkContent comprueba que el store tenga exactamente want (y no tenga missing), y que Count coincida.
* @param t testing.TB, s *Store, want map[string][]byte, missing []string
**/
func checkContent(t testing.TB, s *Store, want map[string][]byte, missing []string) {
	t.Helper()
	for id, v := range want {
		got, ok, err := s.Read(id)
		if err != nil || !ok || !bytes.Equal(got, v) {
			t.Fatalf("read %q = %d bytes %v %v; want %d bytes", id, len(got), ok, err, len(v))
		}
	}
	for _, id := range missing {
		if _, ok, err := s.Read(id); err != nil || ok {
			t.Fatalf("read %q = %v %v; want missing", id, ok, err)
		}
	}
	if n := s.Count(); n != int64(len(want)) {
		t.Fatalf("count = %d; want %d", n, len(want))
	}
}
