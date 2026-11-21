package store

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Medida de la pregunta 75 de F-23: cuánto del disco son los 28 B fijos del encabezado de cada versión en la hoja,
// con la forma en que josefina guarda un modelo sobre el motor (internal/etstore y internal/jdb de josefina):
//
//   - el store principal: clave = jid (UUID v7, 36 B), valor = el documento JSON;
//   - por cada índice BTree, dos stores: «btidx_<campo>» (igualdad: uvarint(len(v)) + v + jid, v = encodeKey: "s/…",
//     "i/…", "d/<unix>"; valor vacío) y «btord_<campo>» (ordenado: escape(clase + valor) + 0x00 0x01 + jid; valor
//     vacío);
//   - por cada campo único, «btuniq_<campo>#u»: clave = encodeKey(valor), valor = el jid.
//
// Solo corre con STORE_BENCH=1 (disco de verdad, ~1 GB de pico y unos segundos):
//
//	STORE_BENCH=1 go test -p 1 ./store -run TestIndexHeaderShare -v

// indexField es un campo indexado del modelo de prueba: cómo sale su valor para el documento i.
type indexField struct {
	name   string
	unique bool
	value  func(i int, rng *rand.Rand, at time.Time) any
}

// headerFields son los 7 índices típicos de un modelo de josefina (pregunta 75: «unos 7 por modelo»).
var headerFields = []indexField{
	{name: "created_at", value: func(_ int, _ *rand.Rand, at time.Time) any { return at.Format(time.RFC3339Nano) }},
	{name: "updated_at", value: func(_ int, _ *rand.Rand, at time.Time) any { return at.Format(time.RFC3339Nano) }},
	{name: "status", value: func(_ int, rng *rand.Rand, _ time.Time) any {
		return []string{"active", "archived", "pending", "canceled", "draft"}[rng.Intn(5)]
	}},
	{name: "project_id", value: func(_ int, rng *rand.Rand, _ time.Time) any { return fmt.Sprintf("01JD%022d", rng.Intn(50)) }},
	{name: "kind", value: func(_ int, rng *rand.Rand, _ time.Time) any { return float64(rng.Intn(10)) }},
	{name: "name", value: func(i int, rng *rand.Rand, _ time.Time) any {
		return fmt.Sprintf("%s %s %d", []string{"Ana", "Luis", "Marta", "Pedro", "Sofía"}[rng.Intn(5)],
			[]string{"Gómez", "Rodríguez", "Pérez", "Martínez"}[rng.Intn(4)], i%997)
	}},
	{name: "code", unique: true, value: func(i int, _ *rand.Rand, _ time.Time) any { return fmt.Sprintf("C-%07d", i) }},
}

// equalityKey es la clave de «btidx_»: uvarint(len(v)) + v + jid, con v = encodeKey de jdb.
func equalityKey(v any, jid string) string {
	enc := encodeIndexValue(v)
	b := binary.AppendUvarint(nil, uint64(len(enc)))
	return string(append(append(b, enc...), jid...))
}

// encodeIndexValue es encodeKey de jdb (KeyFromAny): fecha RFC3339 → "d/<unix>", entero → "i/…", texto → "s/…".
func encodeIndexValue(v any) string {
	switch x := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, x); err == nil {
			return "d/" + fmt.Sprint(t.Unix())
		}
		return "s/" + x
	case float64:
		return "i/" + fmt.Sprint(int64(x))
	}
	return "s/"
}

// orderedIndexKey es la clave de «btord_»: escape(clase + valor) + 0x00 0x01 + jid (orderedKey y entryOf).
func orderedIndexKey(v any, jid string) string {
	var key []byte
	switch x := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339Nano, x); err == nil {
			key = binary.BigEndian.AppendUint64([]byte{4}, uint64(t.UnixNano())^1<<63)
			key = append(key, x...)
		} else {
			key = append([]byte{3}, x...)
		}
	case float64:
		key = binary.BigEndian.AppendUint64([]byte{2}, uint64(int64(x))^1<<63)
	}
	esc := make([]byte, 0, len(key)+8)
	for _, c := range key {
		if c == 0 {
			esc = append(esc, 0, 0xff)
			continue
		}
		esc = append(esc, c)
	}
	return string(append(append(esc, 0, 1), jid...))
}

// headerStat es lo medido de un tipo de store.
type headerStat struct {
	stores int
	keys   int64
	live   int64 // ids + valores vivos
	disk   int64 // DirUsage
	packed map[int]int64
}

// headerSizes son los encabezados que se comparan: el de hoy (28) y candidatos compactos.
var headerSizes = []int{28, 20, 16, 12}

// entrySize es lo que ocupa una versión en línea en la hoja con un encabezado de h bytes: alineada a 8, más su
// puntero de 4 B.
func entrySize(h, id, value int) int64 {
	return int64((h+id+value+7)&^7 + 4)
}

// TestIndexHeaderShare mide, en un modelo de josefina de 100.000 documentos con 7 índices (14 stores) y un campo único,
// qué parte del disco de cada clase de store son los 28 B fijos de cada versión, el llenado de las páginas y cuánto
// ahorraría un encabezado más chico (estimado: el disco por el cociente entre lo empaquetado con cada encabezado).
// Mide tras cargar y otra vez tras actualizar una cuarta parte de los documentos (updated_at y status cambian, como
// hace josefina) y limpiar.
func TestIndexHeaderShare(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	const docs, updates = 100_000, 25_000
	cfg := testConfig()
	cfg.syncOnWrite = false
	cfg.buffers = 4096 // 32 MB de caché
	cfg.wal = wal.Options{SegmentSize: 64 << 20, Spares: 2, Preallocate: true, Buffer: 1 << 20}
	cfg.maxWALSize = 256 << 20
	cfg.checkpointTimeout = 30 * time.Second
	dir := t.TempDir()
	e, err := open(vfs.OS(), dir+"/"+testDir, cfg)
	mustOK(t, err)
	defer e.Close()

	main := mustStore(t, e, "model")
	eq, ord, uniq := map[string]*Store{}, map[string]*Store{}, map[string]*Store{}
	for _, f := range headerFields {
		eq[f.name] = mustStore(t, e, "btidx_"+f.name)
		ord[f.name] = mustStore(t, e, "btord_"+f.name)
		if f.unique {
			uniq[f.name] = mustStore(t, e, "btuniq_"+f.name+"#u")
		}
	}

	rng := rand.New(rand.NewSource(75))
	jids := make([]string, docs)
	values := make([]map[string]any, docs)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	write := func(i int, old map[string]any, at time.Time) {
		b := e.NewBatch()
		doc := map[string]any{}
		for _, f := range headerFields {
			v := f.value(i, rng, at)
			if f.name == "created_at" && old != nil {
				v = old["created_at"]
			}
			if (f.name == "kind" || f.name == "project_id" || f.name == "name" || f.unique) && old != nil {
				v = old[f.name]
			}
			doc[f.name] = v
			if old != nil && old[f.name] == v {
				continue
			}
			if old != nil {
				mustOK(t, eq[f.name].RemoveIn(b, equalityKey(old[f.name], jids[i])))
				mustOK(t, ord[f.name].RemoveIn(b, orderedIndexKey(old[f.name], jids[i])))
			}
			mustOK(t, eq[f.name].PutIn(b, equalityKey(v, jids[i]), nil))
			mustOK(t, ord[f.name].PutIn(b, orderedIndexKey(v, jids[i]), nil))
			if f.unique {
				_, err := uniq[f.name].InsertIn(b, encodeIndexValue(v), []byte(jids[i]))
				mustOK(t, err)
			}
		}
		mustOK(t, main.PutIn(b, jids[i], docJSON(doc, rng)))
		mustOK(t, b.CommitDeferred())
		values[i] = doc
	}
	clean := func() {
		for range 3 {
			mustOK(t, e.Vacuum())
		}
		mustOK(t, e.Checkpoint())
	}
	began := time.Now()
	for i := range docs {
		jids[i] = reg.UUID()
		write(i, nil, start.Add(time.Duration(i)*time.Second))
	}
	clean()
	t.Logf("tras cargar %d documentos (%s):", docs, time.Since(began).Round(time.Millisecond))
	report(t, measureHeaders(t, e))
	for k := range updates {
		i := rng.Intn(docs)
		write(i, values[i], start.Add(time.Duration(docs+k)*time.Second))
	}
	clean()
	t.Logf("tras actualizar %d documentos al azar y limpiar:", updates)
	report(t, measureHeaders(t, e))
}

/**
* measureHeaders recorre cada store del motor y suma, por clase, claves, datos vivos, disco y lo que ocuparían sus
* versiones empaquetadas en hojas con cada encabezado de headerSizes.
* @param t *testing.T, e *Engine
* @return map[string]*headerStat
**/
func measureHeaders(t *testing.T, e *Engine) map[string]*headerStat {
	u, err := e.DirUsage()
	mustOK(t, err)
	stats := map[string]*headerStat{}
	for _, name := range e.Stores() {
		class := storeClass(name)
		st := stats[class]
		if st == nil {
			st = &headerStat{packed: map[int]int64{}}
			stats[class] = st
		}
		st.stores++
		st.disk += u.Stores[name]
		s := mustStore(t, e, name)
		mustOK(t, s.ForEach(func(id string, value []byte) (bool, error) {
			st.keys++
			st.live += int64(len(id) + len(value))
			for _, h := range headerSizes {
				st.packed[h] += entrySize(h, len(id), len(value))
			}
			return true, nil
		}, true, 0, 0))
	}
	return stats
}

// docJSON es un documento de ~350 B con los campos indexados y otros.
func docJSON(doc map[string]any, rng *rand.Rand) []byte {
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("{")
	for _, k := range keys {
		fmt.Fprintf(&b, "%q:%q,", k, fmt.Sprint(doc[k]))
	}
	fmt.Fprintf(&b, "%q:%q}", "description", strings.Repeat("x", 120+rng.Intn(80)))
	return []byte(b.String())
}

// storeClass es la clase de un store para el informe: principal, igualdad, ordenado o unicidad.
func storeClass(name string) string {
	switch {
	case strings.HasPrefix(name, "btidx_"):
		return "igualdad (btidx)"
	case strings.HasPrefix(name, "btord_"):
		return "ordenado (btord)"
	case strings.HasSuffix(name, "#u"):
		return "unicidad (#u)"
	}
	return "principal"
}

// report escribe la tabla de la medida: por clase de store, disco ÷ datos vivos, parte del disco que son los 28 B
// y el disco estimado con cada encabezado candidato.
func report(t *testing.T, stats map[string]*headerStat) {
	classes := make([]string, 0, len(stats))
	for c := range stats {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	var total, totalIdx headerStat
	total.packed, totalIdx.packed = map[int]int64{}, map[int]int64{}
	t.Logf("%-18s %6s %9s %8s %9s %7s %7s %8s %7s %7s %7s", "clase", "stores", "claves", "B/clave", "disco MB",
		"disco÷", "28 B %", "llenado", "h=20", "h=16", "h=12")
	add := func(dst *headerStat, st *headerStat) {
		dst.stores += st.stores
		dst.keys += st.keys
		dst.live += st.live
		dst.disk += st.disk
		for h, v := range st.packed {
			dst.packed[h] += v
		}
	}
	line := func(name string, st *headerStat) {
		est := func(h int) string {
			return fmt.Sprintf("%.0f%%", 100*(1-float64(st.packed[h])/float64(st.packed[28])))
		}
		t.Logf("%-18s %6d %9d %8.1f %9.1f %7.2f %6.1f%% %7.0f%% %7s %7s %7s", name, st.stores, st.keys,
			float64(st.live)/float64(st.keys), float64(st.disk)/(1<<20), float64(st.disk)/float64(st.live),
			100*float64(28*st.keys)/float64(st.disk), 100*float64(st.packed[28])/float64(st.disk), est(20), est(16), est(12))
	}
	for _, c := range classes {
		line(c, stats[c])
		add(&total, stats[c])
		if c != "principal" {
			add(&totalIdx, stats[c])
		}
	}
	line("índices", &totalIdx)
	line("todo el modelo", &total)
	t.Logf("llenado: lo empaquetado con 28 B ÷ el disco; h=N: ahorro estimado del disco con un encabezado de N B (lo "+
		"empaquetado con N contra 28, con el mismo llenado; página %d B)", page.Size)
}
