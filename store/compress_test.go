package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Pruebas y mediciones de la compresión de valores (formato 2, tarea 132; «Formato 2» en doc.go).

// jsonWords es el vocabulario de los textos de los documentos de prueba.
var jsonWords = strings.Fields(`activo pendiente cancelado transformador poste circuito subestación medidor
	acometida luminaria revisión mantenimiento preventivo correctivo orden trabajo cuadrilla técnico zona norte sur
	oriente occidente Medellín Bogotá Cali Barranquilla calle carrera avenida número barrio vereda municipio
	departamento Antioquia Valle Atlántico observación instalado retirado reemplazo daño falla operación consumo
	material cable conector aislador cruceta herraje tornillo arandela cinta pintura sello`)

// jsonKinds son los tipos de componente de los documentos de prueba.
var jsonKinds = []string{"cable", "conector", "aislador", "cruceta", "herraje", "medidor", "transformador"}

/**
* ulidLike es un id al azar de 26 caracteres en base 32 (incompresible, como los _id de josefina).
* @param r *rand.Rand
* @return string
**/
func ulidLike(r *rand.Rand) string {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	b := make([]byte, 26)
	for i := range b {
		b[i] = alphabet[r.IntN(len(alphabet))]
	}
	return string(b)
}

/**
* sentence arma un texto de n palabras del vocabulario.
* @param r *rand.Rand, n int
* @return string
**/
func sentence(r *rand.Rand, n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = jsonWords[r.IntN(len(jsonWords))]
	}
	return strings.Join(words, " ")
}

/**
* jsonDoc arma un documento JSON parecido a los de josefina (ids al azar, fechas, estados, textos y una lista de
* componentes con las mismas claves) de unos size bytes.
* @param r *rand.Rand, size int
* @return []byte
**/
func jsonDoc(r *rand.Rand, size int) []byte {
	doc := map[string]any{
		"_id":        ulidLike(r),
		"_idx":       fmt.Sprint(1732000000000 + r.IntN(1e9)),
		"status":     []string{"ACTIVE", "PENDING", "ARCHIVED"}[r.IntN(3)],
		"created_at": fmt.Sprintf("2026-%02d-%02dT%02d:%02d:%02d.%03dZ", 1+r.IntN(12), 1+r.IntN(28), r.IntN(24), r.IntN(60), r.IntN(60), r.IntN(1000)),
		"tenant_id":  "01JA7Z3K9Q2M4X8V6B0N5C1D3F",
		"code":       fmt.Sprintf("OP-%07d", r.IntN(1e7)),
		"name":       sentence(r, 3),
	}
	var components []map[string]any
	for {
		data, _ := json.Marshal(doc)
		if len(data) >= size {
			return data[:min(len(data), size+64)] // corto por bytes: el formato no importa, el contenido sí
		}
		if len(data) < 300 {
			doc["description"] = sentence(r, 8)
			doc["address"] = map[string]any{"city": jsonWords[r.IntN(len(jsonWords))], "line": sentence(r, 4), "lat": 6 + r.Float64(), "lng": -75 - r.Float64()}
			continue
		}
		components = append(components, map[string]any{
			"id":       ulidLike(r),
			"kind":     jsonKinds[r.IntN(len(jsonKinds))],
			"quantity": r.IntN(20),
			"price":    float64(r.IntN(1e7)) / 100,
			"serial":   fmt.Sprintf("SN%010d", r.IntN(1e9)),
			"status":   []string{"instalado", "retirado", "pendiente"}[r.IntN(3)],
			"note":     sentence(r, 1+r.IntN(5)),
		})
		doc["components"] = components
	}
}

// compressSizes son los tamaños de documento medidos (doc-sizes.md: p50 de los modelos grandes de 256 B a 16 KB).
var compressSizes = []int{256, 512, 1024, 2048, 4096, 8192, 16384}

// TestCompressRoundTrip: cada método comprime y descomprime documentos de todos los tamaños, deja sin comprimir lo
// corto o lo que no achica, y descomprimir un valor dañado da ErrCorrupt sin pánico.
func TestCompressRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	random := make([]byte, 4096)
	for i := range random {
		random[i] = byte(r.Uint32())
	}
	for _, method := range []byte{compressZstd, compressS2} {
		c := compression{method: method, min: defaultCompressMin}
		for _, size := range compressSizes {
			doc := jsonDoc(r, size)
			stored, ok := c.encode(doc)
			if !ok && size < 1024 {
				continue // un documento chico con ids al azar puede no ahorrar un octavo
			}
			if !ok || len(stored) >= len(doc) {
				t.Fatalf("method %d size %d: not compressed (%d → %d)", method, size, len(doc), len(stored))
			}
			back, err := decompress(nil, stored)
			if err != nil || !bytes.Equal(back, doc) {
				t.Fatalf("method %d size %d: round trip %v", method, size, err)
			}
			for _, cut := range []int{0, 1, compressHeaderSize, len(stored) / 2, len(stored) - 1} {
				if _, err := decompress(nil, stored[:cut]); err == nil {
					t.Fatalf("method %d: truncated to %d decoded", method, cut)
				}
			}
			bad := bytes.Clone(stored)
			bad[1]++ // el largo anotado miente
			if _, err := decompress(nil, bad); err == nil {
				t.Fatalf("method %d: wrong length decoded", method)
			}
		}
		if _, ok := c.encode(bytes.Repeat([]byte{'x'}, defaultCompressMin-1)); ok {
			t.Fatal("short value compressed")
		}
		if _, ok := c.encode(random); ok {
			t.Fatal("random value compressed")
		}
	}
	if _, ok := (compression{}).encode(bytes.Repeat([]byte{'x'}, 4096)); ok {
		t.Fatal("compressed with STORE_COMPRESSION=off")
	}
	stored, _ := compression{method: compressZstd, min: 1}.encode(bytes.Repeat([]byte{'x'}, 4096))
	stored[0] = 9
	if _, err := decompress(nil, stored); err == nil {
		t.Fatal("unknown method decoded")
	}
}

// FuzzDecompress: decompress nunca entra en pánico ni devuelve un largo distinto del anotado.
func FuzzDecompress(f *testing.F) {
	r := rand.New(rand.NewPCG(3, 4))
	for _, method := range []byte{compressZstd, compressS2} {
		stored, _ := compression{method: method, min: 1}.encode(jsonDoc(r, 1024))
		f.Add(stored)
	}
	f.Fuzz(func(t *testing.T, stored []byte) {
		out, err := decompress(nil, stored)
		if err == nil && len(out) != int(le.Uint32(stored[1:])) {
			t.Fatalf("length %d, header %d", len(out), le.Uint32(stored[1:]))
		}
	})
}

// BenchmarkValueCodec mide la CPU de comprimir y descomprimir documentos tipo josefina con zstd (nivel más rápido) y
// S2, y la razón (comprimido ÷ original, con el encabezado de 5 B). Se corre con STORE_BENCH=1 o -bench.
func BenchmarkValueCodec(b *testing.B) {
	r := rand.New(rand.NewPCG(5, 6))
	for _, size := range compressSizes {
		docs := make([][]byte, 64)
		for i := range docs {
			docs[i] = jsonDoc(r, size)
		}
		for _, m := range []struct {
			name   string
			method byte
		}{{"zstd", compressZstd}, {"s2", compressS2}} {
			c := compression{method: m.method, min: 1}
			stored := make([][]byte, len(docs))
			in, out := 0, 0
			for i, d := range docs {
				stored[i] = c.force(d)
				in, out = in+len(d), out+len(stored[i])
			}
			ratio := float64(out) / float64(in)
			b.Run(fmt.Sprintf("%s/%d/encode", m.name, size), func(b *testing.B) {
				b.SetBytes(int64(in / len(docs)))
				for i := 0; b.Loop(); i++ {
					c.force(docs[i%len(docs)])
				}
				b.ReportMetric(ratio, "ratio")
			})
			b.Run(fmt.Sprintf("%s/%d/decode", m.name, size), func(b *testing.B) {
				b.SetBytes(int64(in / len(docs)))
				buf := make([]byte, 0, 2*size)
				for i := 0; b.Loop(); i++ {
					if _, err := decompress(buf, stored[i%len(stored)]); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(ratio, "ratio")
			})
		}
	}
}

// josefinaMix es la distribución de tamaños de los documentos de la base josephine (doc-sizes.md, tarea 128): un
// tamaño representativo por tramo y su porcentaje de documentos.
var josefinaMix = []struct {
	size    int
	percent float64
}{{200, 13.4}, {400, 26.4}, {800, 22.1}, {1500, 28.1}, {3000, 6.3}, {6000, 1.3}, {12000, 1.6}, {30000, 0.7}}

/**
* mixSize elige un tamaño de documento con la distribución de josefinaMix.
* @param r *rand.Rand
* @return int
**/
func mixSize(r *rand.Rand) int {
	p := r.Float64() * 100
	for _, m := range josefinaMix {
		if p -= m.percent; p < 0 {
			return m.size
		}
	}
	return josefinaMix[len(josefinaMix)-1].size
}

// TestCompressDisk mide el efecto de la compresión en el motor con documentos tipo josefina (la mezcla de tamaños de
// la base josephine): disco del store ÷ bytes de los documentos tras Checkpoint y Vacuum, CPU de Put diferido y de
// Read, sin comprimir, con S2 y con zstd. Solo con STORE_BENCH=1 (disco de verdad):
//
//	STORE_BENCH=1 go test -p 1 ./store -run TestCompressDisk -v
func TestCompressDisk(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	const docs = 40_000
	r := rand.New(rand.NewPCG(11, 12))
	values := make([][]byte, docs)
	raw := int64(0)
	for i := range values {
		values[i] = jsonDoc(r, mixSize(r))
		raw += int64(len(values[i]))
	}
	for _, m := range []struct {
		name   string
		method byte
	}{{"off", compressNone}, {"s2", compressS2}, {"zstd", compressZstd}, {"auto", compressAuto}} {
		cfg := testConfig()
		cfg.syncOnWrite = false
		cfg.buffers = 4096
		cfg.wal = wal.Options{SegmentSize: 64 << 20, Spares: 2, Preallocate: true, Buffer: 1 << 20}
		cfg.maxWALSize = 256 << 20
		cfg.checkpointTimeout = 30 * time.Second
		cfg.compress = compression{method: m.method, min: defaultCompressMin}
		e, err := open(vfs.OS(), t.TempDir()+"/"+testDir, cfg)
		mustOK(t, err)
		s := mustStore(t, e, "docs")
		ids := make([]string, docs)
		began := time.Now()
		for i := range values {
			ids[i] = fmt.Sprintf("%026d", r.Uint64())
			mustOK(t, s.Deferred().Put(ids[i], values[i]))
		}
		put := time.Since(began) / docs
		mustOK(t, e.WaitDurable())
		walBytes := e.log.Written()
		for range 2 {
			mustOK(t, e.Vacuum())
		}
		mustOK(t, e.Checkpoint())
		u, err := e.DirUsage()
		mustOK(t, err)
		began = time.Now()
		for i, id := range ids {
			got, found, err := s.Read(id)
			if err != nil || !found || len(got) != len(values[i]) {
				t.Fatalf("read %s: %v %v %d", id, err, found, len(got))
			}
		}
		read := time.Since(began) / docs
		began = time.Now()
		mustOK(t, s.ForEach(func(string, []byte) (bool, error) { return true, nil }, true, 0, 0))
		scan := time.Since(began) / docs
		rep, err := e.Verify()
		mustOK(t, err)
		if !rep.Clean() {
			t.Fatalf("verify: %s", rep)
		}
		t.Logf("%-4s disco %6.1f MB = %.2f × los %.1f MB de JSON; WAL %6.1f MB; Put diferido %v/doc, Read %v/doc, ForEach %v/doc",
			m.name, float64(u.Stores["docs"])/1e6, float64(u.Stores["docs"])/float64(raw), float64(raw)/1e6,
			float64(walBytes)/1e6, put, read, scan)
		mustOK(t, e.Close())
	}
}

/**
* force comprime data aunque no ahorre (para medir la CPU y la razón de cada tamaño).
* @param data []byte
* @return []byte: el valor comprimido con su encabezado
**/
func (c compression) force(data []byte) []byte {
	saved := c
	saved.min = 0
	out, ok := saved.encodeWith(data, len(data)+compressHeaderSize+64)
	if !ok {
		panic("force: not compressed")
	}
	return out
}

// zstdConfig es testConfig con la compresión de valores de STORE_COMPRESSION=zstd.
func zstdConfig() config {
	cfg := testConfig()
	cfg.compress = compression{method: compressZstd, min: defaultCompressMin}
	return cfg
}

// compressCases son valores de prueba: en línea sin comprimir (chico o que no achica), comprimidos en línea y
// comprimidos en desborde.
func compressCases(r *rand.Rand) map[string][]byte {
	random := make([]byte, 3000)
	for i := range random {
		random[i] = byte(r.Uint32())
	}
	return map[string][]byte{
		"small":     jsonDoc(r, 100),
		"random":    random, // no achica: en desborde sin comprimir
		"inline2k":  jsonDoc(r, 2048),
		"inline4k":  jsonDoc(r, 4096),
		"over16k":   jsonDoc(r, 16384),
		"over64k":   jsonDoc(r, 65536),
		"repeated":  bytes.Repeat([]byte("abc"), 100_000),
		"empty":     {},
		"threshold": bytes.Repeat([]byte{'x'}, defaultCompressMin),
	}
}

/**
* checkValues comprueba que s devuelve want por Read, ReadAt, ForEach y ForEachBorrowed.
* @param t testing.TB, e *Engine, s *Store, want map[string][]byte
**/
func checkValues(t testing.TB, e *Engine, s *Store, want map[string][]byte) {
	t.Helper()
	v := e.Snapshot()
	defer v.Close()
	for id, data := range want {
		got, found, err := s.Read(id)
		mustOK(t, err)
		if !found || !bytes.Equal(got, data) {
			t.Fatalf("Read %s: found %v, %d bytes, want %d", id, found, len(got), len(data))
		}
		got, found, err = s.ReadAt(v, id)
		mustOK(t, err)
		if !found || !bytes.Equal(got, data) {
			t.Fatalf("ReadAt %s: found %v, %d bytes", id, found, len(got))
		}
	}
	for _, borrowed := range []bool{false, true} {
		seen := 0
		fn := func(id string, data []byte) (bool, error) {
			if !bytes.Equal(data, want[id]) {
				t.Fatalf("scan (borrowed %v) %s: %d bytes, want %d", borrowed, id, len(data), len(want[id]))
			}
			seen++
			return true, nil
		}
		if borrowed {
			mustOK(t, s.ForEachBorrowed(fn))
		} else {
			mustOK(t, s.ForEach(fn, true, 0, 0))
		}
		if seen != len(want) {
			t.Fatalf("scan (borrowed %v) saw %d of %d", borrowed, seen, len(want))
		}
	}
}

// TestCompressedValues: con STORE_COMPRESSION=zstd el motor guarda comprimido lo que achica (en línea y en
// desborde) y devuelve siempre el valor original por todas las lecturas, en lotes, tras reabrir (rehaciendo el WAL),
// en una copia de Backup, en una réplica y por el flujo lógico (también derramado a disco); Update con el mismo valor
// no escribe; Verify limpio. Un valor comprimido dañado da ErrCorrupt al leerlo y un hallazgo «value» en Verify.
func TestCompressedValues(t *testing.T) {
	r := rand.New(rand.NewPCG(13, 14))
	want := compressCases(r)
	fsys := crashfs.New()
	cfg := zstdConfig()
	cfg.decodeWorkMem = 4096 // el flujo lógico derrama a disco
	e := openTest(t, fsys, cfg)
	_, err := e.CreateLogicalSlot("feed")
	mustOK(t, err)
	s := mustStore(t, e, "docs")
	b := e.NewBatch()
	for id, data := range want {
		if id == "inline2k" {
			continue
		}
		mustOK(t, s.PutIn(b, id, data))
		got, found, err := s.ReadIn(b, id)
		mustOK(t, err)
		if !found || !bytes.Equal(got, data) {
			t.Fatalf("ReadIn %s", id)
		}
	}
	mustOK(t, b.Commit())
	mustOK(t, s.Deferred().Put("inline2k", want["inline2k"]))
	mustOK(t, e.WaitDurable())

	compressed := 0
	mustOK(t, s.withSnapshot(func(v *Snapshot) error {
		cur := s.tree.NewCursor(btree.ScanOptions{Snapshot: &v.snap})
		for {
			batch, more, err := cur.Next()
			if err != nil {
				return err
			}
			for _, ver := range batch {
				if ver.Flags&btree.EntryCompressed != 0 {
					compressed++
				}
				if ver.ID == "inline4k" && ver.Flags&btree.EntryOverflow != 0 {
					t.Fatal("a 4 KB JSON document went to overflow")
				}
				if (ver.ID == "random" || ver.ID == "small" || ver.ID == "empty") && ver.Flags&btree.EntryCompressed != 0 {
					t.Fatalf("%s is compressed", ver.ID)
				}
			}
			if !more {
				return nil
			}
		}
	}))
	if compressed != 6 {
		t.Fatalf("%d compressed versions, want 6", compressed)
	}
	checkValues(t, e, s, want)
	if applied, err := s.Update("over16k", want["over16k"]); err != nil || applied {
		t.Fatalf("Update with the same value: %v %v", applied, err)
	}
	want["over16k"] = jsonDoc(r, 20000)
	if applied, err := s.Update("over16k", want["over16k"]); err != nil || !applied {
		t.Fatalf("Update: %v %v", applied, err)
	}
	if inserted, err := s.Insert("over64k", jsonDoc(r, 9000)); err != nil || inserted {
		t.Fatalf("Insert over an existing id: %v %v", inserted, err)
	}

	got := map[string][]byte{}
	rd, err := e.Changes("feed")
	mustOK(t, err)
	_, err = rd.Next(100, func(_ TxInfo, c Change) error {
		if c.Op == ChangePut {
			got[c.ID] = c.Data
		}
		return nil
	})
	mustOK(t, err)
	mustOK(t, rd.Close())
	for id, data := range want {
		if !bytes.Equal(got[id], data) {
			t.Fatalf("logical feed %s: %d bytes, want %d", id, len(got[id]), len(data))
		}
	}

	bfs := crashfs.New()
	_, err = e.backup(bfs, testDir)
	mustOK(t, err)
	checkClean(t, e)
	mustOK(t, e.Close())

	e = openTest(t, fsys, testConfig()) // sin compresión: lee lo comprimido igual
	checkValues(t, e, mustStore(t, e, "docs"), want)
	checkClean(t, e)
	mustOK(t, e.Close())

	rcfg := testConfig()
	rcfg.replica = true
	rep, err := open(bfs, testDir, rcfg)
	mustOK(t, err)
	checkValues(t, rep, mustStore(t, rep, "docs"), want)
	mustOK(t, rep.Close())

	// Un valor comprimido dañado: el motor lo escribe tal cual porque ya viene marcado como comprimido.
	e = openTest(t, fsys, zstdConfig())
	s = mustStore(t, e, "docs")
	bad := bytes.Clone(e.cfg.compress.force(want["inline4k"]))
	le.PutUint32(bad[1:], le.Uint32(bad[1:])+1) // el largo anotado miente (sin suma propia, un byte cambiado dentro
	// de los datos no siempre se nota: lo cubre la suma de la página)
	_, err = s.write("bad", change{data: bad, compressed: true})
	mustOK(t, err)
	if _, _, err := s.Read("bad"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Read of a damaged value: %v", err)
	}
	report, err := e.Verify()
	mustOK(t, err)
	if len(report.Findings) != 1 || report.Findings[0].Kind != FindValue || report.Findings[0].Repairable {
		t.Fatalf("Verify of a damaged value:\n%s", report)
	}
	mustOK(t, e.Close())
}

// TestCompressCrash: escrituras durables de documentos comprimidos (en línea y en desborde) cortadas en los tres
// modos de crashfs a mitad de camino: al recuperar, cada id tiene un valor que se intentó escribir y no anterior al
// último confirmado, y Verify solo encuentra lo que explica un corte.
func TestCompressCrash(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for seed := range 3 {
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				runCompressCrash(t, mode, int64(seed))
			})
		}
	}
}

/**
* runCompressCrash es una vuelta de TestCompressCrash.
* @param t *testing.T, mode crashfs.Mode, seed int64
**/
func runCompressCrash(t *testing.T, mode crashfs.Mode, seed int64) {
	r := rand.New(rand.NewPCG(uint64(seed), uint64(mode)))
	base := crashfs.New()
	cfg := zstdConfig()
	cfg.buffers = 32
	cfg.checkpointTimeout, cfg.checkEvery = 5*time.Millisecond, 2*time.Millisecond
	cfg.vacuumEvery, cfg.vacuumWrites = 3*time.Millisecond, 20
	proc := newProc(base)
	e := openTest(t, proc, cfg)
	s := mustStore(t, e, "docs")
	tried := map[string][][]byte{} // cada valor que se intentó escribir, en orden
	confirmed := map[string]int{}  // el índice en tried del último confirmado
	proc.killAfter(int64(300+r.IntN(3000)), mode, seed)
	for range 5000 {
		id := fmt.Sprintf("d%03d", r.IntN(120))
		v := jsonDoc(r, mixSize(r))
		tried[id] = append(tried[id], v)
		if err := s.Put(id, v); err != nil {
			break
		}
		confirmed[id] = len(tried[id]) - 1
	}
	proc.mu.RLock()
	dead := proc.dead
	proc.mu.RUnlock()
	if !dead {
		t.Fatal("no crash")
	}
	closeWithin(t, e, 10*time.Second)
	e = openTest(t, newProc(base), cfg)
	defer e.Close()
	s = mustStore(t, e, "docs")
	for id, values := range tried {
		got, found, err := s.Read(id)
		mustOK(t, err)
		last, ok := confirmed[id]
		if !found {
			if ok {
				t.Fatalf("%s: lost its confirmed value", id)
			}
			continue
		}
		at := slices.IndexFunc(values, func(v []byte) bool { return bytes.Equal(v, got) })
		if at < 0 || ok && at < last {
			t.Fatalf("%s: read value at %d, last confirmed %d of %d", id, at, last, len(values))
		}
	}
	checkAfterCrash(t, e)
}
