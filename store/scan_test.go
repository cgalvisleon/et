package store

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// collect recorre con each y devuelve los ids y valores en el orden entregado.
func collect(t testing.TB, each func(fn func(id string, data []byte) (bool, error)) error) ([]string, map[string]string) {
	t.Helper()
	var ids []string
	values := map[string]string{}
	err := each(func(id string, data []byte) (bool, error) {
		ids = append(ids, id)
		values[id] = string(data)
		return true, nil
	})
	if err != nil {
		t.Fatalf("recorrido: %v", err)
	}
	return ids, values
}

// fillStore escribe n ids "k0000"… con valores chicos y, cada 7, uno en desborde.
func fillStore(t testing.TB, s *Store, n int) map[string]string {
	t.Helper()
	m := map[string]string{}
	for i := range n {
		id, v := fmt.Sprintf("k%04d", i), fmt.Sprintf("v%d", i)
		if i%7 == 0 {
			v = strings.Repeat(v, 1500) // desborde
		}
		mustOK(t, s.Put(id, []byte(v)))
		m[id] = v
	}
	return m
}

// sortedKeys son las claves del mapa en orden.
func sortedKeys(m map[string]string) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// TestExists: suelto, dentro de un lote (ve sus propios cambios y lápidas) y con snapshot (ve la foto, también tras
// borrados y versiones nuevas sin limpiar), con valores en línea y en desborde.
func TestExists(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	mustOK(t, s.Put("a", []byte("1")))
	mustOK(t, s.Put("big", bytes.Repeat([]byte("x"), 20000)))
	mustOK(t, s.Put("gone", []byte("1")))
	checkBool(t, "exists a", true)(s.Exists("a"))
	checkBool(t, "exists big", true)(s.Exists("big"))
	checkBool(t, "exists zz", false)(s.Exists("zz"))
	if _, err := s.Exists(""); !errors.Is(err, ErrBadID) {
		t.Fatalf("exists empty id: %v", err)
	}

	v := e.Snapshot()
	defer v.Close()
	mustOK(t, s.Remove("gone"))
	mustOK(t, s.Put("new", []byte("n")))
	checkBool(t, "exists gone", false)(s.Exists("gone"))
	checkBool(t, "existsAt gone", true)(s.ExistsAt(v, "gone"))
	checkBool(t, "existsAt new", false)(s.ExistsAt(v, "new"))

	tx := e.NewBatch()
	mustOK(t, s.RemoveIn(tx, "a"))
	mustOK(t, s.PutIn(tx, "mine", bytes.Repeat([]byte("y"), 20000)))
	checkBool(t, "existsIn a", false)(s.ExistsIn(tx, "a"))
	checkBool(t, "existsIn mine", true)(s.ExistsIn(tx, "mine"))
	checkBool(t, "exists mine", false)(s.Exists("mine"))
	checkBool(t, "exists a before commit", true)(s.Exists("a"))
	mustOK(t, tx.Commit())
	checkBool(t, "exists mine after commit", true)(s.Exists("mine"))
	if _, err := s.ExistsIn(tx, "a"); !errors.Is(err, ErrTxDone) {
		t.Fatalf("existsIn done batch: %v", err)
	}
	mustOK(t, v.Close())
	if _, err := s.ExistsAt(v, "a"); !errors.Is(err, ErrTxDone) {
		t.Fatalf("existsAt closed snapshot: %v", err)
	}
}

// TestForEachSnapshot: ForEachAt y CountAt ven la foto del snapshot (versiones viejas que vacuum no puede quitar, sin
// lo borrado ni lo escrito después, sin lo que un lote abierto escribió); ForEach suelto ve lo último confirmado;
// los valores en desborde llegan enteros; asc y desc con offset y limit.
func TestForEachSnapshot(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	want := fillStore(t, s, 400)
	v := e.Snapshot()
	defer v.Close()

	latest := map[string]string{}
	for id, val := range want {
		latest[id] = val
	}
	for i := range 400 {
		id := fmt.Sprintf("k%04d", i)
		switch i % 5 {
		case 0:
			mustOK(t, s.Remove(id))
			delete(latest, id)
		case 1:
			mustOK(t, s.Put(id, []byte("changed")))
			latest[id] = "changed"
		}
	}
	mustOK(t, s.Put("zz-new", []byte("n")))
	latest["zz-new"] = "n"
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, "open-batch", []byte("x")))
	mustOK(t, e.Vacuum()) // no puede quitar lo que ve v

	ids, values := collect(t, func(fn func(string, []byte) (bool, error)) error {
		return s.ForEachAt(v, fn, true, 0, 0)
	})
	if !slices.Equal(ids, sortedKeys(want)) {
		t.Fatalf("ForEachAt: %d ids, se esperaban %d", len(ids), len(want))
	}
	for id, val := range want {
		if values[id] != val {
			t.Fatalf("ForEachAt %s: %d B, se esperaban %d B", id, len(values[id]), len(val))
		}
	}
	if n, err := s.CountAt(v); err != nil || n != int64(len(want)) {
		t.Fatalf("CountAt = %d, %v; se esperaba %d", n, err, len(want))
	}

	ids, values = collect(t, func(fn func(string, []byte) (bool, error)) error { return s.ForEach(fn, true, 0, 0) })
	if !slices.Equal(ids, sortedKeys(latest)) || values["k0001"] != "changed" {
		t.Fatalf("ForEach: %d ids, se esperaban %d", len(ids), len(latest))
	}
	mustOK(t, tx.Rollback())

	all := sortedKeys(want)
	back := slices.Clone(all)
	slices.Reverse(back)
	for _, c := range []struct {
		asc           bool
		offset, limit int
	}{{true, 0, 10}, {true, 37, 25}, {true, 390, 50}, {false, 0, 10}, {false, 13, 40}, {true, 500, 5}, {false, -3, 0}} {
		ref := all
		if !c.asc {
			ref = back
		}
		off := min(max(c.offset, 0), len(ref))
		end := len(ref)
		if c.limit > 0 {
			end = min(off+c.limit, len(ref))
		}
		ids, _ := collect(t, func(fn func(string, []byte) (bool, error)) error {
			return s.ForEachAt(v, fn, c.asc, c.offset, c.limit)
		})
		if !slices.Equal(ids, ref[off:end]) && !(len(ids) == 0 && off == end) {
			t.Fatalf("ForEachAt(%+v) = %v, se esperaba %v", c, ids, ref[off:end])
		}
	}
	keys, err := s.Keys(false, 2, 3)
	mustOK(t, err)
	lat := sortedKeys(latest)
	if !slices.Equal(keys, []string{lat[len(lat)-3], lat[len(lat)-4], lat[len(lat)-5]}) {
		t.Fatalf("Keys(desc, 2, 3) = %v", keys)
	}
}

// TestKeysAfter: paginar con KeysAfter recorre todos los ids una vez, en orden, hasta una página vacía; after no
// tiene que existir.
func TestKeysAfter(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	want := sortedKeys(fillStore(t, s, 333))
	var got []string
	after := ""
	for {
		page, err := s.KeysAfter(after, 50)
		mustOK(t, err)
		if len(page) == 0 {
			break
		}
		got = append(got, page...)
		after = page[len(page)-1]
	}
	if !slices.Equal(got, want) {
		t.Fatalf("KeysAfter paginado: %d ids, se esperaban %d", len(got), len(want))
	}
	page, err := s.KeysAfter("k0100x", 2) // no existe: empieza en el siguiente
	mustOK(t, err)
	if !slices.Equal(page, []string{"k0101", "k0102"}) {
		t.Fatalf("KeysAfter(k0100x) = %v", page)
	}
	all, err := s.KeysAfter("", 0)
	mustOK(t, err)
	if len(all) != len(want) {
		t.Fatalf("KeysAfter sin límite: %d", len(all))
	}
}

// TestForEachStops: fn que corta, que devuelve error o que entra en pánico; en todos los casos el snapshot propio
// se suelta (Empty no ve lotes ni snapshots abiertos).
func TestForEachStops(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	fillStore(t, s, 300)
	n := 0
	mustOK(t, s.ForEach(func(string, []byte) (bool, error) { n++; return n < 5, nil }, true, 0, 0))
	if n != 5 {
		t.Fatalf("fn llamada %d veces, se esperaban 5", n)
	}
	boom := errors.New("boom")
	if err := s.ForEach(func(string, []byte) (bool, error) { return true, boom }, false, 0, 0); !errors.Is(err, boom) {
		t.Fatalf("error de fn: %v", err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("el pánico de fn no llegó")
			}
		}()
		s.ForEachBorrowed(func(string, []byte) (bool, error) { panic("fn") })
	}()
	if n := e.openTx.Load(); n != 0 {
		t.Fatalf("quedaron %d snapshots abiertos", n)
	}
	mustOK(t, s.Empty())
}

// TestForEachBorrowed: entrega lo mismo que ForEach, en orden ascendente; data vale dentro de fn (con y sin
// snapshot).
func TestForEachBorrowed(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	want := fillStore(t, s, 500)
	mustOK(t, s.Put("empty", nil))
	want["empty"] = ""
	ids, values := collect(t, s.ForEachBorrowed)
	if !slices.Equal(ids, sortedKeys(want)) {
		t.Fatalf("ForEachBorrowed: %d ids", len(ids))
	}
	for id, v := range want {
		if values[id] != v {
			t.Fatalf("ForEachBorrowed %s: %d B, se esperaban %d B", id, len(values[id]), len(v))
		}
	}
	v := e.Snapshot()
	defer v.Close()
	mustOK(t, s.Put("later", []byte("x")))
	ids, _ = collect(t, func(fn func(string, []byte) (bool, error)) error { return s.ForEachBorrowedAt(v, fn) })
	if slices.Contains(ids, "later") || len(ids) != len(want) {
		t.Fatalf("ForEachBorrowedAt vio %d ids", len(ids))
	}
}

// TestForEachPrefixRange: prefijos (también con 0xFF al final) y rangos [from, to), hacia los dos lados.
func TestForEachPrefixRange(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	ids := []string{"a", "ab", "abc", "abd", "ac", "b", "b\xff", "b\xff\xff", "b\xff\xffz", "c"}
	for _, id := range ids {
		mustOK(t, s.Put(id, []byte(id)))
	}
	cases := []struct {
		prefix string
		want   []string
	}{
		{"ab", []string{"ab", "abc", "abd"}}, {"b\xff", []string{"b\xff", "b\xff\xff", "b\xff\xffz"}},
		{"", ids}, {"zz", nil}, {"abc", []string{"abc"}},
	}
	for _, c := range cases {
		got, err := s.KeysPrefix(c.prefix, true, 0, 0)
		mustOK(t, err)
		if !slices.Equal(got, c.want) && !(len(got) == 0 && len(c.want) == 0) {
			t.Fatalf("KeysPrefix(%q) = %q, se esperaba %q", c.prefix, got, c.want)
		}
		back := slices.Clone(c.want)
		slices.Reverse(back)
		got, _ = collect(t, func(fn func(string, []byte) (bool, error)) error {
			return s.ForEachPrefix(c.prefix, fn, false, 0, 0)
		})
		if !slices.Equal(got, back) && !(len(got) == 0 && len(back) == 0) {
			t.Fatalf("ForEachPrefix(%q, desc) = %q, se esperaba %q", c.prefix, got, back)
		}
	}
	got, _ := collect(t, func(fn func(string, []byte) (bool, error)) error {
		return s.ForEachRange("abc", "b", fn, true, 1, 2)
	})
	if !slices.Equal(got, []string{"abd", "ac"}) {
		t.Fatalf("ForEachRange(abc, b) = %q", got)
	}
	got, _ = collect(t, func(fn func(string, []byte) (bool, error)) error {
		return s.ForEachRange("ab", "b", fn, false, 0, 0)
	})
	if !slices.Equal(got, []string{"ac", "abd", "abc", "ab"}) {
		t.Fatalf("ForEachRange(ab, b, desc) = %q", got)
	}
	v := e.Snapshot()
	defer v.Close()
	got, _ = collect(t, func(fn func(string, []byte) (bool, error)) error {
		return s.ForEachRangeAt(v, "", "ab", fn, true, 0, 0)
	})
	if !slices.Equal(got, []string{"a"}) {
		t.Fatalf("ForEachRangeAt('', ab) = %q", got)
	}
}

// TestScanCallbacks: fn no tiene nada tomado: puede leer con el mismo snapshot, escribir en el store (el recorrido
// no lo ve) y cerrar el snapshot (el recorrido termina con ErrTxDone en la hoja siguiente).
func TestScanCallbacks(t *testing.T) {
	e := newTestEngine(t, nil)
	s := mustStore(t, e, "s")
	want := fillStore(t, s, 600)
	v := e.Snapshot()
	n := 0
	err := s.ForEachAt(v, func(id string, data []byte) (bool, error) {
		if _, found, err := s.ReadAt(v, id); err != nil || !found {
			return false, fmt.Errorf("ReadAt(%s) = %v, %v", id, found, err)
		}
		n++
		return true, s.Put("w"+id, data)
	}, true, 0, 0)
	mustOK(t, err)
	if n != len(want) {
		t.Fatalf("el recorrido vio %d ids, se esperaban %d", n, len(want))
	}
	if s.Count() != int64(2*len(want)) {
		t.Fatalf("Count = %d", s.Count())
	}
	err = s.ForEachAt(v, func(string, []byte) (bool, error) { return true, v.Close() }, true, 0, 0)
	if !errors.Is(err, ErrTxDone) {
		t.Fatalf("recorrido con el snapshot cerrado en fn: %v", err)
	}
}

// TestScanRing: un recorrido entero de un store mucho más grande que la caché no expulsa las páginas calientes de
// otro store (estrategia de anillo).
func TestScanRing(t *testing.T) {
	e := newTestEngine(t, nil) // 64 páginas
	hot, big := mustStore(t, e, "hot"), mustStore(t, e, "big")
	for i := range 20 {
		mustOK(t, hot.Put(fmt.Sprintf("h%02d", i), []byte("x")))
	}
	for i := range 6000 {
		mustOK(t, big.Put(fmt.Sprintf("b%05d", i), bytes.Repeat([]byte("v"), 400)))
	}
	mustOK(t, big.ForEach(func(string, []byte) (bool, error) { return true, nil }, true, 0, 0)) // marcas
	mustOK(t, e.Checkpoint())
	for range 2 {
		for i := range 20 {
			checkBool(t, "hot", true)(hot.Exists(fmt.Sprintf("h%02d", i)))
		}
	}
	mustOK(t, big.ForEach(func(string, []byte) (bool, error) { return true, nil }, true, 0, 0))
	before := e.pool.Stats().Misses
	for i := range 20 {
		checkBool(t, "hot", true)(hot.Exists(fmt.Sprintf("h%02d", i)))
	}
	if miss := e.pool.Stats().Misses - before; miss != 0 {
		t.Fatalf("tras el recorrido, las lecturas calientes fallaron %d veces en la caché", miss)
	}
}
