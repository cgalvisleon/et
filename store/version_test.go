package store

import (
	"errors"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

/**
* mustVersion lee id con read y devuelve su valor y versión; falla la prueba con error.
* @param t testing.TB, read func(string) ([]byte, uint64, bool, error), id string
* @return string, uint64, bool
**/
func mustVersion(t testing.TB, read func(string) ([]byte, uint64, bool, error), id string) (string, uint64, bool) {
	t.Helper()
	data, version, found, err := read(id)
	if err != nil {
		t.Fatalf("GetVersion %q: %v", id, err)
	}
	if found != (version != 0) {
		t.Fatalf("GetVersion %q: found %v with version %d", id, found, version)
	}
	return string(data), version, found
}

// GetVersion: 0 si no existe; sube con cada escritura que aplica, no con un Update sin cambios ni con un Insert que
// no escribe; un borrado la vuelve 0; sobrevive a reabrir; es la de la versión visible.
func TestGetVersion(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "a")
	if _, v, found := mustVersion(t, s.GetVersion, "k"); found || v != 0 {
		t.Fatalf("missing id: version %d, %v", v, found)
	}
	mustOK(t, s.Put("k", []byte("1")))
	data, v1, found := mustVersion(t, s.GetVersion, "k")
	if !found || data != "1" || v1 == 0 {
		t.Fatalf("after Put: %q %d %v", data, v1, found)
	}
	checkBool(t, "update same", false)(s.Update("k", []byte("1")))
	checkBool(t, "insert existing", false)(s.Insert("k", []byte("x")))
	if _, v, _ := mustVersion(t, s.GetVersion, "k"); v != v1 {
		t.Fatalf("a write that did not apply changed the version: %d → %d", v1, v)
	}
	mustOK(t, s.Put("other", []byte("x")))
	if _, v, _ := mustVersion(t, s.GetVersion, "k"); v != v1 {
		t.Fatalf("a write to another id changed the version: %d → %d", v1, v)
	}
	checkBool(t, "update", true)(s.Deferred().Update("k", []byte("2")))
	data, v2, _ := mustVersion(t, s.GetVersion, "k")
	if data != "2" || v2 <= v1 {
		t.Fatalf("after Update: %q version %d, before %d", data, v2, v1)
	}
	checkBool(t, "delete", true)(s.Delete("k"))
	if _, v, found := mustVersion(t, s.GetVersion, "k"); found || v != 0 {
		t.Fatalf("after Delete: version %d, %v", v, found)
	}
	if _, _, _, err := s.GetVersion(""); !errors.Is(err, ErrBadID) {
		t.Fatalf("empty id: %v", err)
	}
}

// GetVersionAt ve la versión del snapshot aunque después cambie; GetVersionIn ve la del snapshot del lote y, tras un
// cambio propio, la que tendrá al confirmar.
func TestGetVersionAtIn(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("k", []byte("1")))
	_, v1, _ := mustVersion(t, s.GetVersion, "k")

	snap := e.Snapshot()
	tx := e.NewBatch()
	mustOK(t, s.Put("k", []byte("2")))
	_, v2, _ := mustVersion(t, s.GetVersion, "k")
	at := func(id string) ([]byte, uint64, bool, error) { return s.GetVersionAt(snap, id) }
	in := func(id string) ([]byte, uint64, bool, error) { return s.GetVersionIn(tx, id) }
	if data, v, _ := mustVersion(t, at, "k"); data != "1" || v != v1 {
		t.Fatalf("GetVersionAt = %q %d; want 1 %d", data, v, v1)
	}
	if data, v, _ := mustVersion(t, in, "k"); data != "1" || v != v1 {
		t.Fatalf("GetVersionIn = %q %d; want 1 %d", data, v, v1)
	}
	mustOK(t, tx.Rollback())

	tx = e.NewBatch()
	in = func(id string) ([]byte, uint64, bool, error) { return s.GetVersionIn(tx, id) }
	mustOK(t, s.PutIn(tx, "n", []byte("new")))
	_, own, found := mustVersion(t, in, "n")
	if !found || own <= v2 {
		t.Fatalf("own write: version %d, %v; want > %d", own, found, v2)
	}
	if _, v, found := mustVersion(t, s.GetVersion, "n"); found || v != 0 {
		t.Fatalf("an uncommitted write is visible outside: %d %v", v, found)
	}
	mustOK(t, tx.Commit())
	if _, v, _ := mustVersion(t, s.GetVersion, "n"); v != own {
		t.Fatalf("after Commit the version is %d; inside the batch it was %d", v, own)
	}

	mustOK(t, snap.Close())
	if _, _, _, err := s.GetVersionAt(snap, "k"); !errors.Is(err, ErrTxDone) {
		t.Fatalf("GetVersionAt on a closed snapshot: %v", err)
	}
	if _, _, _, err := s.GetVersionIn(tx, "k"); !errors.Is(err, ErrTxDone) {
		t.Fatalf("GetVersionIn on a finished batch: %v", err)
	}
	other := newTestEngine(t, nil)
	if _, _, _, err := mustStore(t, other, "a").GetVersionAt(e.Snapshot(), "k"); !errors.Is(err, ErrOtherEngine) {
		t.Fatalf("GetVersionAt with another engine's snapshot: %v", err)
	}
}

// La versión sobrevive a reabrir, a un checkpoint y a la limpieza (el lsn está en la clave de la entrada).
func TestGetVersionDurable(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("k", []byte("1")))
	mustOK(t, s.Put("k", []byte("2")))
	_, v, _ := mustVersion(t, s.GetVersion, "k")
	mustOK(t, e.Vacuum())
	mustOK(t, e.Checkpoint())
	if _, got, _ := mustVersion(t, s.GetVersion, "k"); got != v {
		t.Fatalf("after Vacuum and Checkpoint the version is %d, was %d", got, v)
	}
	mustOK(t, e.Close())
	e = openTest(t, fsys, testConfig())
	defer e.Close()
	if _, got, _ := mustVersion(t, mustStore(t, e, "a").GetVersion, "k"); got != v {
		t.Fatalf("after reopening the version is %d, was %d", got, v)
	}
}

// El patrón de concurrencia optimista con un lote: leer la versión con GetVersionIn y escribir en el mismo lote; si
// otro cambió el id después de que el lote empezó, la escritura da ErrSerialization (no hace falta UpdateIfVersion).
func TestOptimisticBatchPattern(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "a")
	mustOK(t, s.Put("k", []byte("1")))
	tx := e.NewBatch()
	if _, _, _, err := s.GetVersionIn(tx, "k"); err != nil {
		t.Fatal(err)
	}
	mustOK(t, s.Put("k", []byte("other")))
	if err := s.PutIn(tx, "k", []byte("mine")); !errors.Is(err, ErrSerialization) {
		t.Fatalf("PutIn after a concurrent change = %v; want ErrSerialization", err)
	}
	if err := tx.Commit(); !errors.Is(err, ErrSerialization) {
		t.Fatalf("Commit = %v", err)
	}
	if got, _ := readString(t, s.Read, "k"); got != "other" {
		t.Fatalf("k = %q", got)
	}
}
