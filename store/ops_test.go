package store

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// mustBool: falla la prueba si err no es nil o si got no es want.
func mustBool(t *testing.T, what string, got bool, err error, want bool) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if got != want {
		t.Fatalf("%s = %v, quería %v", what, got, want)
	}
}

// snapshot: el valor de cada id de ids según Read (los ausentes no aparecen).
func snapshot(t *testing.T, s *Store, ids []string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, id := range ids {
		if data, ok := value(t, s, id); ok {
			out[id] = string(data)
		}
	}
	return out
}

// sameSnapshot: falla si a y b no tienen los mismos ids con los mismos valores.
func sameSnapshot(t *testing.T, a, b map[string]string) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("ids: %d y %d", len(a), len(b))
	}
	for id, v := range a {
		if b[id] != v {
			t.Fatalf("%s = %q y %q", id, v, b[id])
		}
	}
}

// TestReadAndReopen: Read devuelve lo último escrito, también tras reabrir; el slice devuelto es del llamador.
func TestReadAndReopen(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(4096, false)) // segmentos chicos: lee de varios
	for i := range 300 {
		s.Put(fmt.Sprintf("k%03d", i), fmt.Appendf(nil, "v%d", i))
	}
	s.Put("k007", []byte("nuevo"))

	got, ok := value(t, s, "k007")
	if !ok || string(got) != "nuevo" {
		t.Fatalf("k007 = %q, %v", got, ok)
	}
	got[0] = 'X' // modificar lo leído no cambia el store
	if again, _ := value(t, s, "k007"); string(again) != "nuevo" {
		t.Fatalf("k007 tras modificar lo leído = %q", again)
	}
	s.Close()

	s = openTest(t, dir, testOptions(4096, false))
	for _, i := range []int{0, 150, 299} {
		if got, ok := value(t, s, fmt.Sprintf("k%03d", i)); !ok || string(got) != fmt.Sprintf("v%d", i) {
			t.Fatalf("k%03d al reabrir = %q, %v", i, got, ok)
		}
	}
	if got, _ := value(t, s, "k007"); string(got) != "nuevo" {
		t.Fatalf("k007 al reabrir = %q", got)
	}
}

// TestReadMissingAndRejects: un id que no existe da false sin error; un id inválido y leer tras Close dan error.
func TestReadMissingAndRejects(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(1<<20, false))
	s.Put("a", []byte("1"))
	if data, ok, err := s.Read("b"); data != nil || ok || err != nil {
		t.Fatalf("Read de inexistente: %q %v %v", data, ok, err)
	}
	if _, _, err := s.Read(""); err == nil {
		t.Fatal("Read aceptó id vacío")
	}
	s.Close()
	if _, _, err := s.Read("a"); err == nil || err.Error() != MSG_CLOSED {
		t.Fatalf("Read después de Close: %v", err)
	}
}

// TestReadDetectsCorruption: un byte cambiado en el disco bajo un registro hace fallar Read (verifica el crc).
func TestReadDetectsCorruption(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, false))
	s.Put("a", []byte("valor"))
	r, _ := s.index.get("a")
	f, _ := os.OpenFile(lastSegment(t, dir), os.O_WRONLY, 0o644)
	f.WriteAt([]byte("V"), r.offset+headerSize+1) // primer byte del valor
	f.Close()
	if _, _, err := s.Read("a"); err == nil {
		t.Fatal("Read no detectó el registro dañado")
	}
}

// TestRemoveIdempotentAndReopen: Remove borra, repetirlo o borrar lo que no existe no falla, y al reabrir sigue
// borrado; un Put después de Remove lo vuelve a crear.
func TestRemoveIdempotentAndReopen(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, false))
	s.Put("a", []byte("1"))
	s.Put("b", []byte("2"))
	for range 2 {
		if err := s.Remove("a"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Remove("nunca"); err != nil {
		t.Fatal(err)
	}
	if _, ok := value(t, s, "a"); ok {
		t.Fatal("a sigue después de Remove")
	}
	if s.index.count() != 1 || s.index.entries() != 1 {
		t.Fatalf("ids %d, elementos %d: quedaron lápidas", s.index.count(), s.index.entries())
	}
	s.Close()

	s = openTest(t, dir, testOptions(1<<20, false))
	if _, ok := value(t, s, "a"); ok {
		t.Fatal("a volvió al reabrir")
	}
	if got, _ := value(t, s, "b"); string(got) != "2" {
		t.Fatalf("b = %q", got)
	}
	s.Put("a", []byte("otra vez"))
	if got, _ := value(t, s, "a"); string(got) != "otra vez" {
		t.Fatalf("a = %q", got)
	}
	if s.lastLSN != 6 {
		t.Fatalf("lsn: %d (los borrados también cuentan)", s.lastLSN)
	}
}

// TestInsertUpdateDelete: cada condicional devuelve si hizo algo; Update con el mismo valor no escribe.
func TestInsertUpdateDelete(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(1<<20, false))

	ok, err := s.Update("k", []byte("x"))
	mustBool(t, "Update de inexistente", ok, err, false)
	ok, err = s.Delete("k")
	mustBool(t, "Delete de inexistente", ok, err, false)
	ok, err = s.Insert("k", []byte("1"))
	mustBool(t, "Insert", ok, err, true)
	ok, err = s.Insert("k", []byte("2"))
	mustBool(t, "Insert de existente", ok, err, false)
	if got, _ := value(t, s, "k"); string(got) != "1" {
		t.Fatalf("k = %q", got)
	}

	lsn := s.lastLSN
	ok, err = s.Update("k", []byte("1"))
	mustBool(t, "Update con el mismo valor", ok, err, false)
	if s.lastLSN != lsn {
		t.Fatal("Update con el mismo valor escribió")
	}
	ok, err = s.Update("k", []byte("2"))
	mustBool(t, "Update", ok, err, true)
	if got, _ := value(t, s, "k"); string(got) != "2" {
		t.Fatalf("k = %q", got)
	}

	ok, err = s.Delete("k")
	mustBool(t, "Delete", ok, err, true)
	ok, err = s.Delete("k")
	mustBool(t, "Delete repetido", ok, err, false)
	ok, err = s.Update("k", []byte("3"))
	mustBool(t, "Update de borrado", ok, err, false)
	ok, err = s.Insert("k", []byte("4"))
	mustBool(t, "Insert tras Delete", ok, err, true)

	if _, err := s.Insert("", nil); err == nil {
		t.Fatal("Insert aceptó id vacío")
	}
	s.Close()
	if _, err := s.Update("k", []byte("5")); err == nil || err.Error() != MSG_CLOSED {
		t.Fatalf("Update después de Close: %v", err)
	}
}

// TestInsertRace: 64 escritores hacen Insert del mismo id a la vez, mientras otros hacen Put de otros ids:
// exactamente uno gana, y el valor guardado es el suyo.
func TestInsertRace(t *testing.T) {
	for round := range 20 {
		s := openTest(t, t.TempDir(), testOptions(4096, false))
		var wins atomic.Int32
		var winner atomic.Value
		var wg sync.WaitGroup
		for g := range 64 {
			wg.Go(func() {
				s.Put(fmt.Sprintf("otro-%d", g), []byte("x"))
				mine := fmt.Sprintf("g%02d", g)
				ok, err := s.Insert("único", []byte(mine))
				if err != nil {
					t.Error(err)
				}
				if ok {
					wins.Add(1)
					winner.Store(mine)
				}
			})
		}
		wg.Wait()
		if wins.Load() != 1 {
			t.Fatalf("ronda %d: %d Insert ganaron", round, wins.Load())
		}
		if got, _ := value(t, s, "único"); string(got) != winner.Load() {
			t.Fatalf("ronda %d: único = %q, ganó %v", round, got, winner.Load())
		}
		s.Close()
	}
}

// TestInsertDeleteRace: muchos escritores alternan Insert y Delete del mismo id. Como cada condicional es atómica,
// Insert ganados − Delete ganados es 1 si al final el id existe y 0 si no.
func TestInsertDeleteRace(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(8192, false))
	var inserts, deletes atomic.Int64
	var wg sync.WaitGroup
	for g := range 32 {
		wg.Go(func() {
			for i := range 200 {
				if (g+i)%2 == 0 {
					if ok, _ := s.Insert("k", fmt.Appendf(nil, "%d-%d", g, i)); ok {
						inserts.Add(1)
					}
				} else if ok, _ := s.Delete("k"); ok {
					deletes.Add(1)
				}
			}
		})
	}
	wg.Wait()
	_, exists := value(t, s, "k")
	want := int64(0)
	if exists {
		want = 1
	}
	if inserts.Load()-deletes.Load() != want {
		t.Fatalf("insert %d, delete %d, existe %v", inserts.Load(), deletes.Load(), exists)
	}
	if s.pending.size() != 0 {
		t.Fatalf("quedaron %d ids pendientes", s.pending.size())
	}
}

// TestUpdateRace: muchos escritores hacen Update del mismo id con valores propios mientras otros hacen Put: ningún
// Update se pierde ni falla, y el valor final es el de alguna escritura.
func TestUpdateRace(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(8192, false))
	s.Put("k", []byte("inicial"))
	var updates atomic.Int64
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Go(func() {
			for i := range 100 {
				v := fmt.Appendf(nil, "g%02d-%03d", g, i)
				if g%4 == 0 {
					s.Put("k", v)
					continue
				}
				ok, err := s.Update("k", v)
				if err != nil {
					t.Error(err)
				}
				if ok {
					updates.Add(1)
				}
			}
		})
	}
	wg.Wait()
	if updates.Load() != 12*100 { // cada valor es único: todo Update cambia algo
		t.Fatalf("Update ganados: %d de %d", updates.Load(), 12*100)
	}
	if got, _ := value(t, s, "k"); !strings.HasPrefix(string(got), "g") {
		t.Fatalf("k = %q", got)
	}
}

// TestPutRemoveRace: Put y Remove concurrentes sobre pocos ids: al terminar no quedan lápidas ni pendientes, y al
// reabrir el estado coincide con el que había en memoria (gana el mayor lsn, también al recuperar).
func TestPutRemoveRace(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(16<<10, false))
	ids := make([]string, 8)
	for i := range ids {
		ids[i] = fmt.Sprintf("id%d", i)
	}
	var wg sync.WaitGroup
	for g := range 32 {
		wg.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(g), 7))
			for i := range 400 {
				id := ids[rng.IntN(len(ids))]
				if rng.IntN(3) == 0 {
					s.Remove(id)
				} else {
					s.Put(id, fmt.Appendf(nil, "g%02d-%03d", g, i))
				}
			}
		})
	}
	wg.Wait()
	if s.pending.size() != 0 || s.index.entries() != s.index.count() {
		t.Fatalf("pendientes %d, elementos %d, ids %d", s.pending.size(), s.index.entries(), s.index.count())
	}
	before := snapshot(t, s, ids)
	s.Close()

	s = openTest(t, dir, testOptions(16<<10, false))
	sameSnapshot(t, before, snapshot(t, s, ids))
	if s.index.count() != len(before) {
		t.Fatalf("ids al reabrir: %d, antes %d", s.index.count(), len(before))
	}
}

// TestRecoverDeletes: los registros de borrado se aplican al recuperar, a través de varios segmentos y también cuando
// el borrado está en el segmento activo cortado por un registro incompleto.
func TestRecoverDeletes(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(512, false))
	for i := range 40 {
		s.Put(fmt.Sprintf("k%02d", i), bytes.Repeat([]byte("v"), 30))
	}
	for i := 0; i < 40; i += 2 {
		s.Remove(fmt.Sprintf("k%02d", i))
	}
	s.Put("k00", []byte("revive"))
	s.Remove("k01")
	s.Close()

	partial := encodeRecord(kindDelete, "k03", nil) // un borrado que no llegó entero: no se aplica
	sealRecord(partial, 1000)
	f, _ := os.OpenFile(lastSegment(t, dir), os.O_WRONLY|os.O_APPEND, 0o644)
	f.Write(partial[:len(partial)-1])
	f.Close()

	s = openTest(t, dir, testOptions(512, false))
	if len(s.segmentList()) < 3 {
		t.Fatalf("no rotó: %d segmentos", len(s.segmentList()))
	}
	for i := range 40 {
		id := fmt.Sprintf("k%02d", i)
		_, ok := value(t, s, id)
		want := i%2 == 1 && id != "k01" || id == "k00"
		if ok != want {
			t.Fatalf("%s existe %v, quería %v", id, ok, want)
		}
	}
	if s.index.count() != 20 || s.index.entries() != 20 {
		t.Fatalf("ids %d, elementos %d", s.index.count(), s.index.entries())
	}
}

// benchmarkRead: lee ids al azar de un store con n ids de valores de size bytes, desde varias goroutines.
func benchmarkRead(b *testing.B, n, size int) {
	s, err := open(b.TempDir(), testOptions(128<<20, false))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	data := bytes.Repeat([]byte("v"), size)
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("k%09d", i)
		if err := s.Put(ids[i], data); err != nil {
			b.Fatal(err)
		}
	}
	var seed atomic.Uint64
	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewPCG(seed.Add(1), 1))
		for pb.Next() {
			if _, ok, err := s.Read(ids[rng.IntN(n)]); err != nil || !ok {
				b.Fatal(ok, err)
			}
		}
	})
}

// BenchmarkRead: lectura por clave con los datos en la caché del sistema, 100 B y 1 KB, 100.000 ids.
func BenchmarkRead(b *testing.B) {
	for _, size := range []int{100, 1024} {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) { benchmarkRead(b, 100_000, size) })
	}
}

// BenchmarkReadOne: lectura por clave desde una sola goroutine (latencia sin competencia).
func BenchmarkReadOne(b *testing.B) {
	s, err := open(b.TempDir(), testOptions(128<<20, false))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	ids := make([]string, 100_000)
	for i := range ids {
		ids[i] = fmt.Sprintf("k%09d", i)
		s.Put(ids[i], bytes.Repeat([]byte("v"), 100))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if _, ok, err := s.Read(ids[i%len(ids)]); err != nil || !ok {
			b.Fatal(ok, err)
		}
	}
}
