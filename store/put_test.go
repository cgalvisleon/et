package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// testOptions: segmentos de segmentSize bytes y fsync en cada escritura según sync.
func testOptions(segmentSize int64, sync bool) options {
	return options{segmentSize: segmentSize, sync: sync}
}

// openTest abre un store en un directorio temporal y lo cierra al terminar la prueba.
func openTest(t *testing.T, dir string, opts options) *Store {
	t.Helper()
	s, err := open(dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// value lee el último valor de id con Read y falla la prueba si hay error.
func value(t *testing.T, s *Store, id string) ([]byte, bool) {
	t.Helper()
	data, ok, err := s.Read(id)
	if err != nil {
		t.Fatal(err)
	}
	return data, ok
}

// lastSegment: la ruta del último segmento de dir.
func lastSegment(t *testing.T, dir string) string {
	t.Helper()
	nums, err := listSegments(dir)
	if err != nil || len(nums) == 0 {
		t.Fatalf("segmentos: %v %v", nums, err)
	}
	return filepath.Join(dir, segmentName(nums[len(nums)-1]))
}

// TestPutAndReopen: lo guardado con Put sigue ahí al reabrir, con el índice rehecho desde los segmentos.
func TestPutAndReopen(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, false))
	for i := range 1000 {
		if err := s.Put(fmt.Sprintf("k%04d", i), fmt.Appendf(nil, "v%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s = openTest(t, dir, testOptions(1<<20, false))
	if n := s.index.count(); n != 1000 {
		t.Fatalf("ids al reabrir: %d", n)
	}
	for _, i := range []int{0, 499, 999} {
		got, ok := value(t, s, fmt.Sprintf("k%04d", i))
		if !ok || string(got) != fmt.Sprintf("v%d", i) {
			t.Fatalf("k%04d = %q, %v", i, got, ok)
		}
	}
	if s.lastLSN != 1000 {
		t.Fatalf("lsn al reabrir: %d", s.lastLSN)
	}
}

// TestPutReplaces: Put sobre un id existente lo reemplaza sin mirar lo que había; vacío y nil se guardan vacíos.
func TestPutReplaces(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(1<<20, false))
	for _, v := range []string{"uno", "dos", "tres"} {
		if err := s.Put("k", []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := value(t, s, "k"); string(got) != "tres" {
		t.Fatalf("k = %q", got)
	}
	if err := s.Put("vacío", nil); err != nil {
		t.Fatal(err)
	}
	if got, ok := value(t, s, "vacío"); !ok || len(got) != 0 {
		t.Fatalf("vacío = %q, %v", got, ok)
	}
	if s.index.count() != 2 {
		t.Fatalf("ids: %d", s.index.count())
	}
}

// TestPutConcurrent: 16 goroutines escriben a la vez (cada una sus ids y todas uno común): no se pierde ninguno, el
// id común queda con el valor de mayor lsn, y al reabrir todo coincide.
func TestPutConcurrent(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(64<<10, false)) // segmentos de 64 KB: también rota en paralelo
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Go(func() {
			for i := range 500 {
				if err := s.Put(fmt.Sprintf("g%02d-%03d", g, i), []byte(strings.Repeat("x", i%50))); err != nil {
					t.Error(err)
					return
				}
				if err := s.Put("común", fmt.Appendf(nil, "g%02d-%03d", g, i)); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	wg.Wait()
	if n := s.index.count(); n != 16*500+1 {
		t.Fatalf("ids: %d", n)
	}
	if len(s.segmentList()) < 2 {
		t.Fatalf("no rotó: %d segmentos", len(s.segmentList()))
	}
	want, _ := value(t, s, "común")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s = openTest(t, dir, testOptions(64<<10, false))
	if n := s.index.count(); n != 16*500+1 {
		t.Fatalf("ids al reabrir: %d", n)
	}
	if got, _ := value(t, s, "común"); !bytes.Equal(got, want) {
		t.Fatalf("común al reabrir = %q, antes %q", got, want)
	}
}

// TestPutRecordLargerThanSegment: un registro más grande que un segmento va solo en uno nuevo.
func TestPutRecordLargerThanSegment(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(1024, false))
	big := bytes.Repeat([]byte("b"), 5000)
	for _, id := range []string{"a", "grande", "c"} {
		data := []byte(id)
		if id == "grande" {
			data = big
		}
		if err := s.Put(id, data); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := value(t, s, "grande"); !bytes.Equal(got, big) {
		t.Fatalf("grande: %d bytes", len(got))
	}
	if r, _ := s.index.get("grande"); r.offset != 0 {
		t.Fatalf("grande no empezó un segmento: offset %d", r.offset)
	}
}

// TestRecoverTruncatesTail: un registro a medio escribir al final del último segmento (un corte durante la
// escritura) se descarta al abrir; lo anterior sigue y se puede seguir escribiendo.
func TestRecoverTruncatesTail(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, false))
	s.Put("a", []byte("1"))
	s.Put("b", []byte("2"))
	s.Close()

	path := lastSegment(t, dir)
	info, _ := os.Stat(path)
	valid := info.Size()
	partial := encodeRecord(kindPut, "c", []byte("33333"))
	sealRecord(partial, 3)
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	f.Write(partial[:len(partial)-2]) // le faltan dos bytes
	f.Close()

	s = openTest(t, dir, testOptions(1<<20, false))
	if _, ok := s.index.get("c"); ok {
		t.Fatal("el registro incompleto entró al índice")
	}
	if info, _ := os.Stat(path); info.Size() != valid {
		t.Fatalf("no se cortó la cola: %d, válido %d", info.Size(), valid)
	}
	if err := s.Put("c", []byte("3")); err != nil {
		t.Fatal(err)
	}
	if got, _ := value(t, s, "c"); string(got) != "3" {
		t.Fatalf("c = %q", got)
	}
	if r, _ := s.index.get("c"); r.lsn != 3 {
		t.Fatalf("lsn de c: %d", r.lsn)
	}
}

// TestRecoverStopsAtHole: con escrituras en paralelo, un corte puede dejar un hueco (una escritura reservada que no
// llegó al disco) seguido de registros completos. Lo que sigue al hueco nunca se confirmó: se descarta.
func TestRecoverStopsAtHole(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, false))
	s.Put("a", []byte("1"))
	s.Put("b", []byte("2"))
	s.Put("c", []byte("3"))
	rb, _ := s.index.get("b")
	s.Close()

	path := lastSegment(t, dir)
	f, _ := os.OpenFile(path, os.O_WRONLY, 0o644)
	f.WriteAt(make([]byte, rb.size), rb.offset) // b nunca llegó: su lugar quedó en ceros
	f.Close()

	s = openTest(t, dir, testOptions(1<<20, false))
	if _, ok := s.index.get("a"); !ok {
		t.Fatal("se perdió a")
	}
	for _, id := range []string{"b", "c"} {
		if _, ok := s.index.get(id); ok {
			t.Fatalf("%s sobrevivió al hueco", id)
		}
	}
	if info, _ := os.Stat(path); info.Size() != rb.offset {
		t.Fatalf("no se cortó en el hueco: %d, hueco en %d", info.Size(), rb.offset)
	}
}

// TestRecoverCorruptSealedFails: un registro dañado en un segmento sellado es corrupción real: Open falla en vez de
// perder datos en silencio.
func TestRecoverCorruptSealedFails(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(256, false))
	for i := range 20 {
		s.Put(fmt.Sprintf("k%02d", i), bytes.Repeat([]byte("v"), 40))
	}
	s.Close()

	first := filepath.Join(dir, segmentName(1))
	f, _ := os.OpenFile(first, os.O_WRONLY, 0o644)
	f.WriteAt([]byte{0xFF}, headerSize+1) // un byte cambiado dentro del primer registro
	f.Close()

	if _, err := open(dir, testOptions(256, false)); err == nil {
		t.Fatal("abrió un segmento sellado dañado")
	}
}

// TestOpenLocked: un segundo Open del mismo directorio falla mientras el primero está abierto, y funciona al cerrarlo.
func TestOpenLocked(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, false))
	if _, err := open(dir, testOptions(1<<20, false)); err == nil {
		t.Fatal("abrió dos veces el mismo store")
	}
	s.Close()
	again, err := open(dir, testOptions(1<<20, false))
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
}

// TestPutRejects: id vacío o demasiado largo, y escribir después de Close, devuelven error.
func TestPutRejects(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(1<<20, false))
	if err := s.Put("", []byte("x")); err == nil {
		t.Fatal("aceptó id vacío")
	}
	if err := s.Put(strings.Repeat("i", maxIDLen+1), nil); err == nil {
		t.Fatal("aceptó id demasiado largo")
	}
	s.Close()
	if err := s.Put("k", []byte("x")); err == nil || err.Error() != MSG_CLOSED {
		t.Fatalf("Put después de Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("segundo Close: %v", err)
	}
}

// TestPutGroupCommit: con fsync en cada escritura, 64 escritores a la vez comparten los fsync (group commit): hay
// muchos menos fsync que escrituras, y todo está al reabrir.
func TestPutGroupCommit(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(1<<20, true))
	const writers, each = 64, 20
	var wg sync.WaitGroup
	for g := range writers {
		wg.Go(func() {
			for i := range each {
				if err := s.Put(fmt.Sprintf("g%02d-%02d", g, i), []byte("v")); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	fsyncs := s.fsyncs.Load()
	if fsyncs >= writers*each/2 {
		t.Fatalf("%d fsync para %d escrituras: no se comparten", fsyncs, writers*each)
	}
	t.Logf("%d escrituras con %d fsync", writers*each, fsyncs)
	s.Close()

	s = openTest(t, dir, testOptions(1<<20, true))
	if n := s.index.count(); n != writers*each {
		t.Fatalf("ids al reabrir: %d", n)
	}
}

// TestPutAfterFailure: si una escritura falla, el store no acepta más hasta reabrirlo.
func TestPutAfterFailure(t *testing.T) {
	s := openTest(t, t.TempDir(), testOptions(1<<20, false))
	s.fail(errors.New("disco"))
	if err := s.Put("k", []byte("x")); err == nil || !strings.Contains(err.Error(), "disco") {
		t.Fatalf("Put tras un fallo: %v", err)
	}
}

// BenchmarkPut: escritura de 1 KB sin esperar el fsync, desde varias goroutines.
func BenchmarkPut(b *testing.B) {
	s, err := open(b.TempDir(), testOptions(128<<20, false))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	data := bytes.Repeat([]byte("v"), 1024)
	var n sync.Mutex
	next := 0
	b.SetBytes(int64(len(data)))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n.Lock()
			next++
			id := fmt.Sprintf("k%09d", next)
			n.Unlock()
			if err := s.Put(id, data); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPutSync: escritura de 100 B esperando el fsync, desde varias goroutines (mide el group commit).
func BenchmarkPutSync(b *testing.B) {
	s, err := open(b.TempDir(), testOptions(128<<20, true))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	data := bytes.Repeat([]byte("v"), 100)
	var n sync.Mutex
	next := 0
	b.SetParallelism(16)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n.Lock()
			next++
			id := fmt.Sprintf("k%09d", next)
			n.Unlock()
			if err := s.Put(id, data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.ReportMetric(float64(s.fsyncs.Load())/float64(b.N), "fsyncs/op")
}

// TestOpenCreatesMissingDirs: Open crea los directorios que faltan en la ruta, del más alto al más bajo (cada uno con
// el fsync de su padre), y abre igual un directorio que ya existe.
func TestOpenCreatesMissingDirs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a", "b", "store")
	missing, err := missingDirs(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "a"), filepath.Join(root, "a", "b"), dir}
	if strings.Join(missing, "|") != strings.Join(want, "|") {
		t.Fatalf("faltan %v, quería %v", missing, want)
	}

	s := openTest(t, dir, testOptions(1<<20, false))
	if err := s.Put("k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if missing, _ := missingDirs(dir); len(missing) != 0 {
		t.Fatalf("siguen faltando: %v", missing)
	}
	openTest(t, dir, testOptions(1<<20, false)).Close()

	file := filepath.Join(root, "archivo")
	os.WriteFile(file, []byte("x"), 0o644)
	if _, err := open(filepath.Join(file, "store"), testOptions(1<<20, false)); err == nil {
		t.Fatal("abrió un store debajo de un archivo")
	}
}

// TestRecoverRejectsOutOfOrderLSN: una copia de un registro viejo (crc válido, lsn que no es el siguiente) no entra al
// índice. Anexada al segmento activo, se corta ahí y el id no revive; anexada a un segmento sellado, Open falla.
func TestRecoverRejectsOutOfOrderLSN(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir, testOptions(256, false))
	s.Put("viejo", []byte("v1"))
	r, _ := s.index.get("viejo")
	for i := range 20 {
		s.Put(fmt.Sprintf("k%02d", i), bytes.Repeat([]byte("v"), 40))
	}
	s.Remove("viejo")
	s.Close()

	first := filepath.Join(dir, segmentName(1))
	data, _ := os.ReadFile(first)
	old := data[r.offset : r.offset+int64(r.size)] // el Put de «viejo», lsn 1

	path := lastSegment(t, dir)
	info, _ := os.Stat(path)
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	f.Write(old)
	f.Close()
	s = openTest(t, dir, testOptions(256, false))
	if _, ok := value(t, s, "viejo"); ok {
		t.Fatal("la copia del registro viejo revivió el id")
	}
	if after, _ := os.Stat(path); after.Size() != info.Size() {
		t.Fatalf("no se cortó la copia: %d, antes %d", after.Size(), info.Size())
	}
	if s.lastLSN != 22 {
		t.Fatalf("lsn al reabrir: %d", s.lastLSN)
	}
	s.Close()

	f, _ = os.OpenFile(first, os.O_WRONLY|os.O_APPEND, 0o644)
	f.Write(old)
	f.Close()
	if s, err := open(dir, testOptions(256, false)); err == nil {
		s.Close()
		t.Fatal("abrió un segmento sellado con un lsn fuera de orden")
	}
}
