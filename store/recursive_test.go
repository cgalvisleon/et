package store_test

// Prueba del diseño recursivo (tarea 41 de F-23): encima de la API pública del store se construyen las piezas de una
// base de datos —un índice secundario y un catálogo de colecciones— y se mide cuánto cuesta cada nivel. Todo usa solo
// lo exportado por store/catalog.go; nada de esto es API del paquete.
//
// El patrón:
//   - una colección es un store de datos (id → JSON) más un store por índice, cuyas claves son valor\x00id con valor
//     vacío: el orden del B+tree agrupa por valor y, dentro de él, por id;
//   - el documento y sus entradas de índice se escriben en el mismo lote, así nunca divergen (todo o nada, también
//     tras un corte), y dos lotes que cambian el mismo documento chocan en su id (ErrSerialization) y reintentan;
//   - una consulta por valor es ForEachPrefixAt(valor\x00) sobre un snapshot más ReadAt del documento con ese mismo
//     snapshot: índice y datos se leen en la misma foto;
//   - el catálogo es otro store (nombre → definición): se escribe en el mismo lote que los primeros documentos.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store"
)

// ---------------------------------------------------------------------------------------------------------------
// Colecciones con índices secundarios
// ---------------------------------------------------------------------------------------------------------------

// doc es un documento: campos de texto. Los valores indexados no pueden tener NUL (es el separador de la clave).
type doc map[string]string

// indexDef describe un índice secundario: su nombre y el campo que indexa.
type indexDef struct {
	Name  string `json:"name"`
	Field string `json:"field"`
}

// collectionDef describe una colección: lo que guarda el catálogo.
type collectionDef struct {
	Name    string     `json:"name"`
	Indexes []indexDef `json:"indexes"`
}

// collection es una colección abierta: el store de datos y un store por índice, en el orden de def.Indexes.
type collection struct {
	def     collectionDef
	data    *store.Store
	indexes []*store.Store
}

// errDiverged señala que el índice y los datos no coinciden.
var errDiverged = errors.New("test: index and data diverged")

/**
* indexStoreName da el nombre del store de un índice: <colección>_by_<índice>.
* @param coll, index string
* @return string
**/
func indexStoreName(coll, index string) string {
	return coll + "_by_" + index
}

/**
* indexKey da la clave de una entrada de índice: valor\x00id.
* @param value, id string
* @return string
**/
func indexKey(value, id string) string {
	return value + "\x00" + id
}

/**
* openCollection abre (o crea) los stores de una colección. Es DDL: cada Engine.Store confirma por su cuenta, fuera
* de cualquier lote.
* @param e *store.Engine, def collectionDef
* @return *collection, error
**/
func openCollection(e *store.Engine, def collectionDef) (*collection, error) {
	data, err := e.Store(def.Name)
	if err != nil {
		return nil, err
	}
	c := &collection{def: def, data: data}
	for _, ix := range def.Indexes {
		s, err := e.Store(indexStoreName(def.Name, ix.Name))
		if err != nil {
			return nil, err
		}
		c.indexes = append(c.indexes, s)
	}
	return c, nil
}

/**
* storeNames da los stores que forman la colección: datos e índices.
* @return []string
**/
func (c *collection) storeNames() []string {
	names := []string{c.def.Name}
	for _, ix := range c.def.Indexes {
		names = append(names, indexStoreName(c.def.Name, ix.Name))
	}
	return names
}

/**
* readIn lee el documento id como lo ve el lote tx.
* @param tx *store.Batch, id string
* @return doc, bool: si existe, error
**/
func (c *collection) readIn(tx *store.Batch, id string) (doc, bool, error) {
	raw, found, err := c.data.ReadIn(tx, id)
	if err != nil || !found {
		return nil, found, err
	}
	var d doc
	return d, true, json.Unmarshal(raw, &d)
}

/**
* saveIn guarda el documento id dentro del lote tx y ajusta sus índices: quita la entrada del valor viejo y pone la
* del nuevo solo si el campo cambió.
* @param tx *store.Batch, id string, d doc
* @return error: los del lote (ErrSerialization, ErrConflict…)
**/
func (c *collection) saveIn(tx *store.Batch, id string, d doc) error {
	old, found, err := c.readIn(tx, id)
	if err != nil {
		return err
	}
	for i, ix := range c.def.Indexes {
		value := d[ix.Field]
		if found {
			if old[ix.Field] == value {
				continue
			}
			if err := c.indexes[i].RemoveIn(tx, indexKey(old[ix.Field], id)); err != nil {
				return err
			}
		}
		if err := c.indexes[i].PutIn(tx, indexKey(value, id), nil); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return c.data.PutIn(tx, id, raw)
}

/**
* deleteIn borra el documento id y sus entradas de índice dentro del lote tx.
* @param tx *store.Batch, id string
* @return bool: si existía, error
**/
func (c *collection) deleteIn(tx *store.Batch, id string) (bool, error) {
	old, found, err := c.readIn(tx, id)
	if err != nil || !found {
		return false, err
	}
	for i, ix := range c.def.Indexes {
		if err := c.indexes[i].RemoveIn(tx, indexKey(old[ix.Field], id)); err != nil {
			return false, err
		}
	}
	return c.data.DeleteIn(tx, id)
}

/**
* indexOf da la posición del índice sobre field.
* @param field string
* @return int: -1 si no hay
**/
func (c *collection) indexOf(field string) int {
	return slices.IndexFunc(c.def.Indexes, func(ix indexDef) bool { return ix.Field == field })
}

/**
* query devuelve los ids de los documentos con field = value usando el índice: recorre valor\x00 con el snapshot v y
* lee cada documento con el mismo snapshot. Si el índice apunta a un documento que no existe o tiene otro valor,
* devuelve errDiverged.
* @param v *store.Snapshot, field, value string
* @return []string: en orden, error
**/
func (c *collection) query(v *store.Snapshot, field, value string) ([]string, error) {
	i := c.indexOf(field)
	if i < 0 {
		return nil, fmt.Errorf("test: no index on %q", field)
	}
	prefix := indexKey(value, "")
	var ids []string
	err := c.indexes[i].ForEachPrefixAt(v, prefix, func(key string, _ []byte) (bool, error) {
		id := key[len(prefix):]
		raw, found, err := c.data.ReadAt(v, id)
		if err != nil {
			return false, err
		}
		if !found {
			return false, fmt.Errorf("%w: %q points to a missing document", errDiverged, key)
		}
		var d doc
		if err := json.Unmarshal(raw, &d); err != nil {
			return false, err
		}
		if d[field] != value {
			return false, fmt.Errorf("%w: %q points to %s = %q", errDiverged, key, field, d[field])
		}
		ids = append(ids, id)
		return true, nil
	}, true, 0, 0)
	return ids, err
}

/**
* scanFilter devuelve los ids con field = value recorriendo todos los documentos del snapshot v: la consulta sin
* índice.
* @param v *store.Snapshot, field, value string
* @return []string: en orden, error
**/
func (c *collection) scanFilter(v *store.Snapshot, field, value string) ([]string, error) {
	var ids []string
	err := c.data.ForEachAt(v, func(id string, raw []byte) (bool, error) {
		var d doc
		if err := json.Unmarshal(raw, &d); err != nil {
			return false, err
		}
		if d[field] == value {
			ids = append(ids, id)
		}
		return true, nil
	}, true, 0, 0)
	return ids, err
}

/**
* checkConsistent compara, en un mismo snapshot, cada índice con las entradas que salen de los documentos.
* @param e *store.Engine
* @return error: errDiverged con la primera diferencia
**/
func (c *collection) checkConsistent(e *store.Engine) error {
	v := e.Snapshot()
	defer v.Close()
	want := make([][]string, len(c.indexes))
	err := c.data.ForEachAt(v, func(id string, raw []byte) (bool, error) {
		var d doc
		if err := json.Unmarshal(raw, &d); err != nil {
			return false, err
		}
		for i, ix := range c.def.Indexes {
			want[i] = append(want[i], indexKey(d[ix.Field], id))
		}
		return true, nil
	}, true, 0, 0)
	if err != nil {
		return err
	}
	for i, ix := range c.def.Indexes {
		var got []string
		err := c.indexes[i].ForEachAt(v, func(key string, _ []byte) (bool, error) {
			got = append(got, key)
			return true, nil
		}, true, 0, 0)
		if err != nil {
			return err
		}
		slices.Sort(want[i])
		if !slices.Equal(got, want[i]) {
			return fmt.Errorf("%w: index %s has %d entries, documents give %d", errDiverged, ix.Name, len(got), len(want[i]))
		}
		if n := c.indexes[i].Count(); n != int64(len(got)) {
			return fmt.Errorf("%w: index %s Count %d, scan %d", errDiverged, ix.Name, n, len(got))
		}
	}
	return nil
}

/**
* runTx corre fn en un lote y lo confirma; si el lote choca con otro (ErrSerialization o ErrConflict) lo deshace y
* vuelve a empezar con una foto nueva.
* @param e *store.Engine, deferred bool: CommitDeferred en vez de Commit, fn func(tx *store.Batch) error
* @return int: reintentos, error: el de fn o el del lote si no es un choque
**/
func runTx(e *store.Engine, deferred bool, fn func(tx *store.Batch) error) (int, error) {
	for retries := 0; ; retries++ {
		tx := e.NewBatch()
		err := fn(tx)
		if err == nil && deferred {
			err = tx.CommitDeferred()
		} else if err == nil {
			err = tx.Commit()
		}
		if err == nil {
			return retries, nil
		}
		_ = tx.Rollback()
		if !errors.Is(err, store.ErrSerialization) && !errors.Is(err, store.ErrConflict) {
			return retries, err
		}
	}
}

// usersDef es la colección de la prueba: usuarios indexados por ciudad.
var usersDef = collectionDef{Name: "users", Indexes: []indexDef{{Name: "city", Field: "city"}}}

// cities son los valores del campo indexado.
var cities = []string{"bogota", "medellin", "cali", "barranquilla", "pasto", "leticia"}

/**
* openEngine abre un motor en dir o termina la prueba.
* @param tb testing.TB, dir string
* @return *store.Engine
**/
func openEngine(tb testing.TB, dir string) *store.Engine {
	tb.Helper()
	e, err := store.Open(dir)
	if err != nil {
		tb.Fatalf("Open: %v", err)
	}
	return e
}

/**
* mustCollection abre la colección o termina la prueba.
* @param tb testing.TB, e *store.Engine, def collectionDef
* @return *collection
**/
func mustCollection(tb testing.TB, e *store.Engine, def collectionDef) *collection {
	tb.Helper()
	c, err := openCollection(e, def)
	if err != nil {
		tb.Fatalf("openCollection: %v", err)
	}
	return c
}

// ---------------------------------------------------------------------------------------------------------------
// 1. Índice secundario
// ---------------------------------------------------------------------------------------------------------------

// TestRecursiveIndexBasic inserta, cambia de ciudad y borra, y consulta por el índice en cada paso.
func TestRecursiveIndexBasic(t *testing.T) {
	e := openEngine(t, t.TempDir())
	defer e.Close()
	users := mustCollection(t, e, usersDef)

	save := func(id, city string) {
		t.Helper()
		if _, err := runTx(e, false, func(tx *store.Batch) error {
			return users.saveIn(tx, id, doc{"email": id + "@x.co", "city": city})
		}); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
	}
	expect := func(city string, want ...string) {
		t.Helper()
		v := e.Snapshot()
		defer v.Close()
		got, err := users.query(v, "city", city)
		if err != nil {
			t.Fatalf("query %s: %v", city, err)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("query %s = %v, want %v", city, got, want)
		}
	}

	save("u1", "bogota")
	save("u2", "bogota")
	save("u3", "cali")
	expect("bogota", "u1", "u2")
	expect("cali", "u3")

	// Un snapshot tomado antes del cambio sigue viendo índice y datos de antes.
	old := e.Snapshot()
	save("u2", "cali")
	expect("bogota", "u1")
	expect("cali", "u2", "u3")
	if got, err := users.query(old, "city", "bogota"); err != nil || !slices.Equal(got, []string{"u1", "u2"}) {
		t.Fatalf("old snapshot query = %v, %v", got, err)
	}
	old.Close()

	// Mismo valor indexado: el índice no se toca.
	save("u3", "cali")
	if _, err := runTx(e, false, func(tx *store.Batch) error {
		_, err := users.deleteIn(tx, "u1")
		return err
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	expect("bogota")
	expect("cali", "u2", "u3")
	// "bogota\x00" no debe encontrar "bogota2\x00…".
	save("u4", "bogota2")
	expect("bogota")

	if err := users.checkConsistent(e); err != nil {
		t.Fatal(err)
	}
	if n := users.indexes[0].Count(); n != 3 {
		t.Fatalf("index Count = %d, want 3", n)
	}

	// La comprobación sí detecta una divergencia: una entrada escrita fuera del lote del documento.
	if err := users.indexes[0].Put(indexKey("cali", "u9"), nil); err != nil {
		t.Fatal(err)
	}
	if err := users.checkConsistent(e); !errors.Is(err, errDiverged) {
		t.Fatalf("checkConsistent with a stray entry = %v, want errDiverged", err)
	}
	v := e.Snapshot()
	defer v.Close()
	if _, err := users.query(v, "city", "cali"); !errors.Is(err, errDiverged) {
		t.Fatalf("query with a stray entry = %v, want errDiverged", err)
	}
}

// TestRecursiveIndexConcurrent pone escritores concurrentes sobre pocos ids (para que choquen) y lectores que en cada
// snapshot comparan la consulta por índice con el recorrido completo; al final compara índice y documentos.
func TestRecursiveIndexConcurrent(t *testing.T) {
	const (
		writers = 8
		ops     = 250
		ids     = 48
	)
	e := openEngine(t, t.TempDir())
	defer e.Close()
	users := mustCollection(t, e, usersDef)

	var retries atomic.Int64
	var done atomic.Bool
	errs := make(chan error, writers+2)
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(w)))
			deferred := w%2 == 1 // la mitad confirma sin esperar el fsync
			for op := range ops {
				id := "u" + strconv.Itoa(rng.Intn(ids))
				city := cities[rng.Intn(len(cities))]
				del := rng.Intn(5) == 0
				n, err := runTx(e, deferred, func(tx *store.Batch) error {
					if del {
						_, err := users.deleteIn(tx, id)
						return err
					}
					return users.saveIn(tx, id, doc{"email": fmt.Sprintf("%s.%d.%d@x.co", id, w, op), "city": city})
				})
				retries.Add(int64(n))
				if err != nil {
					errs <- fmt.Errorf("writer %d: %w", w, err)
					return
				}
			}
		}()
	}
	var readers sync.WaitGroup
	var queries atomic.Int64
	for r := range 2 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			rng := rand.New(rand.NewSource(int64(100 + r)))
			for !done.Load() {
				city := cities[rng.Intn(len(cities))]
				v := e.Snapshot()
				byIndex, err := users.query(v, "city", city)
				var byScan []string
				if err == nil {
					byScan, err = users.scanFilter(v, "city", city)
				}
				v.Close()
				if err == nil && !slices.Equal(byIndex, byScan) {
					err = fmt.Errorf("%w: city %s index %v scan %v", errDiverged, city, byIndex, byScan)
				}
				if err != nil {
					errs <- fmt.Errorf("reader %d: %w", r, err)
					return
				}
				queries.Add(1)
			}
		}()
	}
	wg.Wait()
	done.Store(true)
	readers.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if err := e.WaitDurable(); err != nil {
		t.Fatal(err)
	}
	if err := users.checkConsistent(e); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d lotes, %d reintentos por choque, %d consultas comparadas, %d usuarios", writers*ops, retries.Load(),
		queries.Load(), users.data.Count())
}

// Corte simulado: la API pública no da el disco simulado (crashfs es interno), así que el corte es el de un proceso:
// un proceso hijo escribe lotes y se lo mata con SIGKILL a mitad de camino; el padre reabre el directorio. Prueba la
// recuperación desde el WAL (lo que el proceso no llegó a escribir en las páginas), no la pérdida de lo no sincronizado
// en un corte de luz: eso lo cubre TestModelCrash dentro del paquete.

// crashChildEnv lleva al proceso hijo el directorio del motor y la secuencia con que empieza.
const crashChildEnv = "STORE_RECURSIVE_CRASH_CHILD"

/**
* crashChild es el proceso hijo: un solo escritor que guarda o borra usuarios al azar en lotes durables y anuncia por
* stdout «TRY seq id ciudad» antes de confirmar y «ACK seq» cuando Commit volvió. No termina: lo matan.
* @param dir string, seq int
**/
func crashChild(dir string, seq int) {
	e, err := store.Open(dir)
	if err != nil {
		fmt.Printf("ERR %v\n", err)
		os.Exit(2)
	}
	users, err := openCollection(e, usersDef)
	if err != nil {
		fmt.Printf("ERR %v\n", err)
		os.Exit(2)
	}
	rng := rand.New(rand.NewSource(int64(seq)))
	for ; ; seq++ {
		id := "u" + strconv.Itoa(rng.Intn(32))
		city := cities[rng.Intn(len(cities))]
		if rng.Intn(5) == 0 {
			city = "-"
		}
		fmt.Printf("TRY %d %s %s\n", seq, id, city)
		_, err := runTx(e, false, func(tx *store.Batch) error {
			if city == "-" {
				_, err := users.deleteIn(tx, id)
				return err
			}
			return users.saveIn(tx, id, doc{"email": fmt.Sprintf("%s.%d@x.co", id, seq), "city": city})
		})
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("ACK %d\n", seq)
	}
}

// crashOp es una operación anunciada por el hijo.
type crashOp struct {
	seq      int
	id, city string
}

/**
* runCrashChild lanza el hijo sobre dir, lo mata tras acks confirmaciones y devuelve lo anunciado.
* @param t *testing.T, dir string, seq, acks int
* @return []crashOp: las confirmadas en orden, *crashOp: la que quedó sin ack (nil si ninguna)
**/
func runCrashChild(t *testing.T, dir string, seq, acks int) ([]crashOp, *crashOp) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRecursiveIndexCrash$")
	cmd.Env = append(os.Environ(),
		crashChildEnv+"="+dir+"|"+strconv.Itoa(seq),
		"STORE_CACHE_SIZE=8", "WAL_SEGMENT_SIZE=4", "CHECKPOINT_TIMEOUT=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 64)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(out)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()

	var acked []crashOp
	var pending *crashOp
	killed := false
	deadline := time.After(60 * time.Second)
	for {
		select {
		case <-deadline:
			_ = cmd.Process.Kill()
			t.Fatal("crash child: timeout waiting for commits")
		case line, ok := <-lines:
			if !ok {
				_ = cmd.Wait()
				if !killed {
					t.Fatal("crash child exited by itself")
				}
				return acked, pending
			}
			f := strings.Fields(line)
			switch {
			case len(f) == 4 && f[0] == "TRY":
				n, _ := strconv.Atoi(f[1])
				pending = &crashOp{seq: n, id: f[2], city: f[3]}
			case len(f) == 2 && f[0] == "ACK" && pending != nil && f[1] == strconv.Itoa(pending.seq):
				acked = append(acked, *pending)
				pending = nil
				if len(acked) == acks && !killed {
					_ = cmd.Process.Kill() // lo ya escrito en la tubería se sigue leyendo
					killed = true
				}
			case f[0] == "ERR":
				t.Fatalf("crash child: %s", line)
			}
		}
	}
}

// TestRecursiveIndexCrash mata tres veces al proceso que escribe y comprueba al reabrir que índice y datos coinciden
// y que cada lote confirmado está (el único sin confirmar puede estar o no, entero).
func TestRecursiveIndexCrash(t *testing.T) {
	if v := os.Getenv(crashChildEnv); v != "" {
		dir, seq, _ := strings.Cut(v, "|")
		n, _ := strconv.Atoi(seq)
		crashChild(dir, n)
		return
	}
	if testing.Short() {
		t.Skip("lanza procesos hijos")
	}
	dir := t.TempDir()
	state := map[string]string{} // id → ciudad de lo que debe haber
	seq := 1
	rng := rand.New(rand.NewSource(41))
	for round := range 3 {
		acked, pending := runCrashChild(t, dir, seq, 30+rng.Intn(60))
		for _, op := range acked {
			state[op.id] = op.city
		}

		e := openEngine(t, dir)
		users := mustCollection(t, e, usersDef)
		if err := users.checkConsistent(e); err != nil {
			e.Close()
			t.Fatalf("round %d: %v", round, err)
		}
		got := map[string]string{}
		err := users.data.ForEach(func(id string, raw []byte) (bool, error) {
			var d doc
			err := json.Unmarshal(raw, &d)
			got[id] = d["city"]
			return true, err
		}, true, 0, 0)
		if err != nil {
			e.Close()
			t.Fatal(err)
		}
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}
		for _, id := range unionKeys(state, got) {
			want, have := cityOrDash(state, id), cityOrDash(got, id)
			if want == have {
				continue
			}
			if pending != nil && pending.id == id && pending.city == have {
				continue // el lote en vuelo llegó entero al WAL antes del corte
			}
			t.Fatalf("round %d: %s is %q, want %q (pending %+v)", round, id, have, want, pending)
		}
		state = got
		seq += len(acked) + 1
		t.Logf("round %d: %d lotes confirmados antes del corte, %d usuarios al reabrir", round, len(acked), len(got))
	}
}

/**
* cityOrDash da la ciudad de id en m, o "-" si no está.
* @param m map[string]string, id string
* @return string
**/
func cityOrDash(m map[string]string, id string) string {
	if c, ok := m[id]; ok {
		return c
	}
	return "-"
}

/**
* unionKeys da las claves de a y b, sin repetir y en orden.
* @param a, b map[string]string
* @return []string
**/
func unionKeys(a, b map[string]string) []string {
	var keys []string
	for k := range a {
		keys = append(keys, k)
	}
	for k := range b {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}

// ---------------------------------------------------------------------------------------------------------------
// 2. Catálogo
// ---------------------------------------------------------------------------------------------------------------

// catalogName es el store que describe las colecciones.
const catalogName = "catalog"

// errExists señala una colección que ya está en el catálogo.
var errExists = errors.New("test: collection already exists")

/**
* createCollection crea los stores de def (DDL, cada uno confirma solo) y, en un mismo lote, la inserta en el catálogo
* junto con sus primeros documentos: o queda descrita y con datos, o nada de eso es visible.
* @param e *store.Engine, cat *store.Store, def collectionDef, seed map[string]doc
* @return *collection, error: errExists si ya estaba
**/
func createCollection(e *store.Engine, cat *store.Store, def collectionDef, seed map[string]doc) (*collection, error) {
	c, err := openCollection(e, def)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}
	_, err = runTx(e, false, func(tx *store.Batch) error {
		ok, err := cat.InsertIn(tx, def.Name, raw)
		if err != nil {
			return err
		}
		if !ok {
			return errExists
		}
		for id, d := range seed {
			if err := c.saveIn(tx, id, d); err != nil {
				return err
			}
		}
		return nil
	})
	return c, err
}

/**
* loadCollections lee el catálogo con un snapshot y abre cada colección descrita (sus stores ya deben existir).
* @param e *store.Engine, cat *store.Store
* @return map[string]*collection, error
**/
func loadCollections(e *store.Engine, cat *store.Store) (map[string]*collection, error) {
	v := e.Snapshot()
	defer v.Close()
	existing := e.Stores()
	colls := map[string]*collection{}
	err := cat.ForEachAt(v, func(name string, raw []byte) (bool, error) {
		var def collectionDef
		if err := json.Unmarshal(raw, &def); err != nil {
			return false, err
		}
		c := &collection{def: def}
		for _, s := range c.storeNames() {
			if _, ok := slices.BinarySearch(existing, s); !ok {
				return false, fmt.Errorf("test: catalog names %s but store %s is missing", name, s)
			}
		}
		c, err := openCollection(e, def)
		colls[name] = c
		return err == nil, err
	}, true, 0, 0)
	return colls, err
}

/**
* dropOrphans borra los stores que el catálogo no describe: los que dejó un createCollection cuyo lote no confirmó o
* un dropCollection cortado entre su lote y los DropStore.
* @param e *store.Engine, colls map[string]*collection
* @return []string: los borrados, error
**/
func dropOrphans(e *store.Engine, colls map[string]*collection) ([]string, error) {
	described := map[string]bool{catalogName: true}
	for _, c := range colls {
		for _, s := range c.storeNames() {
			described[s] = true
		}
	}
	var dropped []string
	for _, s := range e.Stores() {
		if described[s] {
			continue
		}
		if err := e.DropStore(s); err != nil {
			return dropped, err
		}
		dropped = append(dropped, s)
	}
	return dropped, nil
}

// TestRecursiveCatalog crea colecciones descritas en el catálogo en el mismo lote que sus datos, deshace una a mitad,
// reabre el motor leyendo el catálogo y borra la colección deshecha como huérfana.
func TestRecursiveCatalog(t *testing.T) {
	dir := t.TempDir()
	e := openEngine(t, dir)
	cat, err := e.Store(catalogName)
	if err != nil {
		t.Fatal(err)
	}

	users, err := createCollection(e, cat, usersDef, map[string]doc{
		"u1": {"email": "a@x.co", "city": "cali"},
		"u2": {"email": "b@x.co", "city": "pasto"},
	})
	if err != nil {
		t.Fatalf("create users: %v", err)
	}
	if _, err := createCollection(e, cat, usersDef, nil); !errors.Is(err, errExists) {
		t.Fatalf("create users twice: %v, want errExists", err)
	}

	// Una colección cuyo lote se deshace: ni catálogo ni datos visibles, solo los stores vacíos (DDL ya confirmado).
	ordersDef := collectionDef{Name: "orders", Indexes: []indexDef{{Name: "user", Field: "user"}, {Name: "state", Field: "state"}}}
	orders, err := openCollection(e, ordersDef)
	if err != nil {
		t.Fatal(err)
	}
	tx := e.NewBatch()
	raw, _ := json.Marshal(ordersDef)
	if err := cat.PutIn(tx, ordersDef.Name, raw); err != nil {
		t.Fatal(err)
	}
	if err := orders.saveIn(tx, "o1", doc{"user": "u1", "state": "open"}); err != nil {
		t.Fatal(err)
	}
	// Dentro del lote se ve la colección y su documento; fuera no.
	if ok, _ := cat.ExistsIn(tx, "orders"); !ok {
		t.Fatal("the batch does not see its own catalog entry")
	}
	if ok, _ := cat.Exists("orders"); ok {
		t.Fatal("an uncommitted catalog entry is visible")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if orders.data.Count() != 0 || orders.indexes[0].Count() != 0 || cat.Count() != 1 {
		t.Fatalf("after rollback: orders %d, index %d, catalog %d", orders.data.Count(), orders.indexes[0].Count(), cat.Count())
	}
	if err := users.checkConsistent(e); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}

	// Al reabrir, el catálogo dice qué hay; lo que no describe es huérfano.
	e = openEngine(t, dir)
	defer e.Close()
	cat, err = e.Store(catalogName)
	if err != nil {
		t.Fatal(err)
	}
	colls, err := loadCollections(e, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(colls) != 1 || colls["users"] == nil {
		t.Fatalf("collections = %v", colls)
	}
	dropped, err := dropOrphans(e, colls)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"orders", "orders_by_state", "orders_by_user"}
	if !slices.Equal(dropped, want) {
		t.Fatalf("orphans = %v, want %v", dropped, want)
	}
	v := e.Snapshot()
	got, err := colls["users"].query(v, "city", "cali")
	v.Close()
	if err != nil || !slices.Equal(got, []string{"u1"}) {
		t.Fatalf("query after reopen = %v, %v", got, err)
	}
	if want := []string{"catalog", "users", "users_by_city"}; !slices.Equal(e.Stores(), want) {
		t.Fatalf("stores = %v, want %v", e.Stores(), want)
	}
}

// ---------------------------------------------------------------------------------------------------------------
// 3. Costo por nivel
// ---------------------------------------------------------------------------------------------------------------

// benchDefs son los niveles medidos: sin índice, con uno y con dos.
var benchDefs = map[int]collectionDef{
	0: {Name: "users"},
	1: {Name: "users", Indexes: []indexDef{{Name: "city", Field: "city"}}},
	2: {Name: "users", Indexes: []indexDef{{Name: "city", Field: "city"}, {Name: "email", Field: "email"}}},
}

/**
* benchPut mide guardar un documento nuevo por lote en una colección con indexes índices.
* @param b *testing.B, indexes int, deferred bool
**/
func benchPut(b *testing.B, indexes int, deferred bool) {
	e := openEngine(b, b.TempDir())
	defer e.Close()
	c := mustCollection(b, e, benchDefs[indexes])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("u%09d", i)
		if _, err := runTx(e, deferred, func(tx *store.Batch) error {
			return c.saveIn(tx, id, doc{"email": id + "@x.co", "city": cities[i%len(cities)]})
		}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if err := e.WaitDurable(); err != nil {
		b.Fatal(err)
	}
}

/**
* benchModes corre fn con Commit (fsync según SYNC_ON_WRITE) y con CommitDeferred.
* @param b *testing.B, fn func(b *testing.B, deferred bool)
**/
func benchModes(b *testing.B, fn func(b *testing.B, deferred bool)) {
	b.Run("commit", func(b *testing.B) { fn(b, false) })
	b.Run("deferred", func(b *testing.B) { fn(b, true) })
}

// BenchmarkPutPlain: un documento por lote, sin índices.
func BenchmarkPutPlain(b *testing.B) {
	benchModes(b, func(b *testing.B, deferred bool) { benchPut(b, 0, deferred) })
}

// BenchmarkPutIndexed: un documento por lote con 1 y 2 índices en el mismo lote.
func BenchmarkPutIndexed(b *testing.B) {
	for _, n := range []int{1, 2} {
		b.Run(fmt.Sprintf("idx=%d", n), func(b *testing.B) {
			benchModes(b, func(b *testing.B, deferred bool) { benchPut(b, n, deferred) })
		})
	}
}

// queryDocs y queryCities fijan la colección de las consultas: 20.000 usuarios en 100 ciudades (200 por ciudad).
const (
	queryDocs   = 20000
	queryCities = 100
)

/**
* loadQueryBench abre un motor con queryDocs usuarios indexados por ciudad, cargados en lotes diferidos de 500.
* @param b *testing.B
* @return *store.Engine, *collection
**/
func loadQueryBench(b *testing.B) (*store.Engine, *collection) {
	e := openEngine(b, filepath.Join(b.TempDir(), "db"))
	c := mustCollection(b, e, benchDefs[1])
	for start := 0; start < queryDocs; start += 500 {
		if _, err := runTx(e, true, func(tx *store.Batch) error {
			for i := start; i < start+500; i++ {
				id := fmt.Sprintf("u%09d", i)
				if err := c.saveIn(tx, id, doc{"email": id + "@x.co", "city": fmt.Sprintf("c%03d", i%queryCities)}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
	if err := e.WaitDurable(); err != nil {
		b.Fatal(err)
	}
	return e, c
}

/**
* benchQuery mide una consulta por ciudad (200 de 20.000) en un snapshot propio con find.
* @param b *testing.B, find func(c *collection, v *store.Snapshot, city string) ([]string, error)
**/
func benchQuery(b *testing.B, find func(c *collection, v *store.Snapshot, city string) ([]string, error)) {
	e, c := loadQueryBench(b)
	defer e.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := e.Snapshot()
		ids, err := find(c, v, fmt.Sprintf("c%03d", i%queryCities))
		v.Close()
		if err != nil {
			b.Fatal(err)
		}
		if len(ids) != queryDocs/queryCities {
			b.Fatalf("found %d, want %d", len(ids), queryDocs/queryCities)
		}
	}
}

// BenchmarkQueryByIndex: los 200 usuarios de una ciudad por el índice (prefijo + ReadAt de cada documento).
func BenchmarkQueryByIndex(b *testing.B) {
	benchQuery(b, (*collection).queryCity)
}

// BenchmarkScanFilter: los mismos 200 recorriendo y decodificando los 20.000 documentos.
func BenchmarkScanFilter(b *testing.B) {
	benchQuery(b, (*collection).scanCity)
}

/**
* queryCity es query sobre el campo city.
* @param v *store.Snapshot, city string
* @return []string, error
**/
func (c *collection) queryCity(v *store.Snapshot, city string) ([]string, error) {
	return c.query(v, "city", city)
}

/**
* scanCity es scanFilter sobre el campo city.
* @param v *store.Snapshot, city string
* @return []string, error
**/
func (c *collection) scanCity(v *store.Snapshot, city string) ([]string, error) {
	return c.scanFilter(v, "city", city)
}

// TestUniqueIndexKey: el patrón del índice único (pregunta 67) sin API nueva. La clave del índice es el valor solo;
// dos lotes que insertan el mismo valor chocan (el segundo espera al primero y, cuando confirma, da ErrSerialization),
// y un lote posterior lo ve con InsertIn = false.
func TestUniqueIndexKey(t *testing.T) {
	e := openEngine(t, t.TempDir())
	defer e.Close()
	users, err := e.Store("users")
	if err != nil {
		t.Fatal(err)
	}
	byEmail, err := e.Store("users_email")
	if err != nil {
		t.Fatal(err)
	}
	insertUser := func(tx *store.Batch, id, email string) (bool, error) {
		ok, err := byEmail.InsertIn(tx, email, []byte(id))
		if err != nil || !ok {
			return ok, err
		}
		return true, users.PutIn(tx, id, []byte(email))
	}

	first, second := e.NewBatch(), e.NewBatch()
	if ok, err := insertUser(first, "u1", "a@x.co"); err != nil || !ok {
		t.Fatalf("first insert: %v %v", ok, err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := insertUser(second, "u2", "a@x.co") // espera el bloqueo del primer lote sobre la clave
		done <- err
	}()
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, store.ErrSerialization) {
		t.Fatalf("second batch with the same email: %v, want ErrSerialization", err)
	}
	second.Rollback()

	third := e.NewBatch()
	if ok, err := insertUser(third, "u3", "a@x.co"); err != nil || ok {
		t.Fatalf("later batch with the same email: %v %v, want false (duplicate)", ok, err)
	}
	if err := third.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := users.Read("u2"); found {
		t.Fatal("u2 was written")
	}
}
