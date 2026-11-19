package store

import (
	"bytes"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// state es el contenido de un motor: store → id → valor. Un store ausente y uno vacío valen lo mismo.
type state map[string]map[string]string

/**
* clone copia el estado (es chico: pocos stores e ids).
* @return state
**/
func (s state) clone() state {
	cp := state{}
	for name, ids := range s {
		cp[name] = maps.Clone(ids)
	}
	return cp
}

/**
* equal compara dos estados, con los stores vacíos como ausentes.
* @param o state
* @return bool
**/
func (s state) equal(o state) bool {
	for _, name := range unionNames(s, o) {
		if !maps.Equal(s[name], o[name]) {
			return false
		}
	}
	return true
}

/**
* unionNames son los nombres de stores de los dos estados, en orden.
* @param a, b state
* @return []string
**/
func unionNames(a, b state) []string {
	names := slices.Collect(maps.Keys(a))
	for name := range b {
		if _, ok := a[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// op es una operación al azar de la prueba de modelo.
type op struct {
	kind  string // put, remove, insert, update, delete, read, drop, checkpoint, vacuum, batch, snapshot
	store string
	id    string
	value string

	// Lote (kind batch): sus operaciones, si termina con Rollback y lo que debe devolver cada una. wants se arma en
	// el generador con su largo y lo llena apply: las copias de op comparten el arreglo.
	sub      []op
	rollback bool
	wants    []bool
}

func (o op) String() string {
	return fmt.Sprintf("%s %s/%.20q (%d B)", o.kind, o.store, o.id, len(o.value))
}

/**
* apply aplica op al estado del modelo y dice qué deberían devolver Insert, Update y Delete.
* @param s state: se cambia
* @return bool
**/
func (o op) apply(s state) bool {
	ids := s[o.store]
	cur, exists := ids[o.id]
	switch o.kind {
	case "put", "insert", "update":
		if o.kind == "insert" && exists || o.kind == "update" && (!exists || cur == o.value) {
			return false
		}
		if ids == nil {
			ids = map[string]string{}
			s[o.store] = ids
		}
		ids[o.id] = o.value
		return true
	case "remove", "delete":
		delete(ids, o.id)
		return exists || o.kind == "remove"
	case "read":
		return exists
	case "drop":
		delete(s, o.store)
	case "batch":
		local := s.clone()
		for i, sub := range o.sub {
			o.wants[i] = sub.apply(local)
		}
		if !o.rollback {
			for name, ids := range local {
				s[name] = ids
			}
		}
	}
	return true
}

// generator arma operaciones al azar sobre pocos stores e ids, para que choquen: valores chicos y grandes (desborde),
// ids cortos y algunos largos (particiones), checkpoints, limpiezas y bajas de stores de vez en cuando.
type generator struct {
	rng *rand.Rand
	ids []string
}

/**
* newGenerator crea el generador con la semilla.
* @param seed uint64
* @return *generator
**/
func newGenerator(seed uint64) *generator {
	g := &generator{rng: rand.New(rand.NewPCG(seed, 23))}
	for i := range 80 {
		id := fmt.Sprintf("id-%03d", i)
		if i%10 == 0 {
			id = strings.Repeat(id, 50+g.rng.IntN(240)) // de 300 a 1.900 bytes
		}
		g.ids = append(g.ids, id)
	}
	return g
}

/**
* next devuelve la operación siguiente.
* @return op
**/
func (g *generator) next() op {
	o := op{store: []string{"a", "b", "c"}[g.rng.IntN(3)], id: g.ids[g.rng.IntN(len(g.ids))]}
	switch r := g.rng.IntN(1000); {
	case r < 5:
		o.kind = "drop"
	case r < 10:
		o.kind = "checkpoint"
	case r < 18:
		o.kind = "vacuum"
	case r < 70:
		return g.batch()
	case r < 85:
		o.kind = "snapshot"
	case r < 400:
		o.kind = "put"
	case r < 550:
		o.kind = "insert"
	case r < 700:
		o.kind = "update"
	case r < 820:
		o.kind = "delete"
	default:
		o.kind = "remove"
	}
	o.value = g.value()
	return o
}

/**
* batch arma un lote de 2 a 12 operaciones (también lecturas) sobre los tres stores; uno de cada cuatro termina con
* Rollback.
* @return op
**/
func (g *generator) batch() op {
	o := op{kind: "batch", rollback: g.rng.IntN(4) == 0}
	for len(o.sub) < 2+g.rng.IntN(11) {
		sub := g.next()
		switch sub.kind {
		case "drop", "checkpoint", "vacuum", "batch", "snapshot":
			continue
		}
		if g.rng.IntN(5) == 0 {
			sub.kind = "read"
		}
		o.sub = append(o.sub, sub)
	}
	o.wants = make([]bool, len(o.sub))
	return o
}

/**
* value da un valor al azar: casi siempre chico, a veces grande (va a desborde).
* @return string
**/
func (g *generator) value() string {
	n := g.rng.IntN(120)
	if g.rng.IntN(8) == 0 {
		n = 2000 + g.rng.IntN(40000)
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + g.rng.IntN(26))
	}
	return string(b)
}

/**
* run hace op sobre el motor.
* @param e *Engine
* @return bool: lo que devolvió Insert, Update o Delete (true en las demás), error
**/
func (o op) run(e *Engine) (bool, error) {
	switch o.kind {
	case "checkpoint":
		return true, e.Checkpoint()
	case "vacuum":
		return true, e.Vacuum()
	case "drop":
		if !slices.Contains(e.Stores(), o.store) {
			return true, nil
		}
		return true, e.DropStore(o.store)
	}
	switch o.kind {
	case "batch":
		return true, o.runBatch(e)
	case "snapshot":
		return true, nil // los snapshots los maneja crashDuring; para los demás no hacen nada
	}
	s, err := e.Store(o.store)
	if err != nil {
		return false, err
	}
	switch o.kind {
	case "put":
		return true, s.Put(o.id, []byte(o.value))
	case "remove":
		return true, s.Remove(o.id)
	case "insert":
		return s.Insert(o.id, []byte(o.value))
	case "update":
		return s.Update(o.id, []byte(o.value))
	}
	return s.Delete(o.id)
}

/**
* runBatch hace el lote o: cada operación dentro de un Batch, comparando lo que devuelve con o.wants (una lectura
* compara si encontró el id), y Commit o Rollback.
* @param e *Engine
* @return error: el del motor, o la primera diferencia con el modelo
**/
func (o op) runBatch(e *Engine) error {
	tx := e.NewBatch()
	for i, sub := range o.sub {
		s, err := e.Store(sub.store)
		if err != nil {
			return err
		}
		got := true
		switch sub.kind {
		case "put":
			err = s.PutIn(tx, sub.id, []byte(sub.value))
		case "remove":
			err = s.RemoveIn(tx, sub.id)
		case "insert":
			got, err = s.InsertIn(tx, sub.id, []byte(sub.value))
		case "update":
			got, err = s.UpdateIn(tx, sub.id, []byte(sub.value))
		case "delete":
			got, err = s.DeleteIn(tx, sub.id)
		case "read":
			_, got, err = s.ReadIn(tx, sub.id)
		}
		if err != nil {
			return err
		}
		if got != o.wants[i] {
			return fmt.Errorf("in a batch, %v returned %v, model says %v", sub, got, o.wants[i])
		}
	}
	if o.rollback {
		return tx.Rollback()
	}
	return tx.Commit()
}

// openSnap es un snapshot abierto de la prueba de modelo con el estado que tiene que ver; dropped son los stores
// borrados después de abrirlo (DropStore borra sus archivos: el snapshot no los conserva).
type openSnap struct {
	v       *Snapshot
	want    state
	dropped map[string]bool
}

/**
* check lee con el snapshot todos los ids del generador en los stores a, b y c y los compara con su estado.
* @param e *Engine, ids []string
* @return error: el del motor o la primera diferencia
**/
func (o openSnap) check(e *Engine, ids []string) error {
	defer o.v.Close()
	for _, name := range []string{"a", "b", "c"} {
		if o.dropped[name] {
			continue
		}
		s, err := e.Store(name)
		if err != nil {
			return err
		}
		for _, id := range ids {
			got, found, err := s.ReadAt(o.v, id)
			if err != nil {
				return err
			}
			want, exists := o.want[name][id]
			if found != exists || string(got) != want {
				return fmt.Errorf("snapshot: store %s id %.20q = %v (%d B), want %v (%d B)",
					name, id, found, len(got), exists, len(want))
			}
		}
	}
	return nil
}

// TestModelCrash es la prueba de modelo con cortes del motor entero: operaciones al azar contra un mapa en memoria
// (sueltas, lotes de varias operaciones en varios stores que a veces terminan con Rollback, y snapshots que comparan
// su vista unas operaciones después), sobre crashfs, con checkpoints y limpiezas (Vacuum) a mano y de fondo (la
// limpieza de fondo, en la mitad de las semillas); un corte en un momento al azar (entre operaciones o en medio de
// una) en los tres modos y varias semillas. Al reabrir, el contenido es exactamente el estado tras la última
// operación confirmada o tras la que estaba en curso (un lote entero o nada; nunca se pierde una confirmada:
// SYNC_ON_WRITE), Count coincide, btree.Verify pasa en cada store y, tras una limpieza, ninguna página está repetida
// en la lista de libres ni es a la vez libre y de una cadena viva (las huérfanas de cadenas escritas antes de un corte
// sin su hoja se cuentan, no se rechazan); el motor acepta escrituras y un segundo y un tercer corte también se
// recuperan.
func TestModelCrash(t *testing.T) {
	seeds := 8
	if testing.Short() {
		seeds = 2
	}
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for seed := range seeds {
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				t.Parallel()
				runModel(t, mode, uint64(seed)*31+uint64(mode))
			})
		}
	}
}

/**
* runModel hace tres rondas de operaciones y cortes sobre el mismo disco, comprobando cada recuperación.
* @param t *testing.T, mode crashfs.Mode, seed uint64
**/
func runModel(t *testing.T, mode crashfs.Mode, seed uint64) {
	base := crashfs.New()
	gen := newGenerator(seed)
	confirmed := state{}
	var pending *state // el estado si la operación en curso al cortar llegó a ser durable
	cfg := testConfig()
	cfg.checkpointTimeout = []time.Duration{time.Hour, 40 * time.Millisecond, 5 * time.Millisecond}[seed%3]
	cfg.checkEvery = 5 * time.Millisecond
	cfg.buffers = 32
	if seed%2 == 1 {
		cfg.vacuumEvery, cfg.vacuumWrites = 3*time.Millisecond, 20
	}

	for round := range 3 {
		proc := newProc(base)
		e := openTest(t, proc, cfg)
		confirmed = checkRecovered(t, e, confirmed, pending)
		verifyTrees(t, e)
		vacuumAndAudit(t, e, confirmed)
		t.Logf("round %d: replayed %d records, %d stores", round, e.replayed, len(e.Stores()))
		target := 20 + gen.rng.IntN(1500)
		confirmed, pending = crashDuring(t, e, proc, gen, confirmed, target, mode, seed+uint64(round))
	}
	e := openTest(t, newProc(base), cfg)
	confirmed = checkRecovered(t, e, confirmed, pending)
	verifyTrees(t, e)
	vacuumAndAudit(t, e, confirmed)
	mustOK(t, e.Close())
}

/**
* vacuumAndAudit limpia el motor recién recuperado y comprueba que el contenido no cambió, que Verify pasa y el
* inventario de páginas de cada store (sin repetidas; las huérfanas se cuentan en el log).
* @param t *testing.T, e *Engine, want state
**/
func vacuumAndAudit(t *testing.T, e *Engine, want state) {
	t.Helper()
	mustOK(t, e.Vacuum())
	if got := engineState(t, e); !got.equal(want) {
		t.Fatalf("Vacuum changed the content:\n%s", diffState(got, want))
	}
	verifyTrees(t, e)
	orphans := 0
	for _, name := range e.Stores() {
		s := mustStore(t, e, name)
		s.vacuumMu.Lock() // la limpieza de fondo puede estar a medias
		orphans += auditStore(t, s).orphans()
		s.vacuumMu.Unlock()
	}
	t.Logf("after Vacuum: %d orphan blocks", orphans)
}

/**
* crashDuring corre operaciones en una goroutine y corta el disco cuando van target confirmadas (a veces en medio de
* la siguiente). Comprueba que cada operación confirmada devolvió lo que dice el modelo y que Close del motor viejo
* no se cuelga.
* @return state: el estado tras la última confirmada, *state: el estado si la que estaba en curso llegó a durable
**/
func crashDuring(t *testing.T, e *Engine, proc *procFS, gen *generator, start state, target int,
	mode crashfs.Mode, seed uint64) (state, *state) {
	var (
		mu       sync.Mutex
		done     = start.clone()
		inFlight *state
		killed   atomic.Bool
		reached  = make(chan struct{})
		finished = make(chan struct{})
	)
	delay := time.Duration(gen.rng.IntN(300)) * time.Microsecond
	var snaps []openSnap // snapshots abiertos: se comparan unas operaciones después

	go func() {
		defer close(finished)
		for n := 1; ; n++ {
			o := gen.next()
			if o.kind == "snapshot" {
				if err := snapshotStep(e, gen, &snaps, done); err != nil && !killed.Load() {
					t.Errorf("%v", err)
					return
				}
				o = gen.next() // el snapshot no cuenta como operación: la siguiente sí
				for o.kind == "snapshot" {
					o = gen.next()
				}
			}
			if o.kind == "drop" {
				for _, open := range snaps {
					open.dropped[o.store] = true
				}
			}
			mu.Lock()
			after := done.clone()
			want := o.apply(after)
			inFlight = &after
			mu.Unlock()

			got, err := o.run(e)
			if err != nil {
				if !killed.Load() {
					t.Errorf("%v before the crash: %v", o, err)
				}
				return
			}
			if got != want {
				t.Errorf("%v returned %v, model says %v", o, got, want)
			}
			mu.Lock()
			done, inFlight = after, nil
			mu.Unlock()
			if n == target {
				close(reached)
			}
		}
	}()

	select {
	case <-reached:
	case <-finished:
		t.Fatal("the operations stopped before the crash")
	}
	time.Sleep(delay)
	killed.Store(true)
	proc.kill(mode, int64(seed))
	<-finished
	closeWithin(t, e, 10*time.Second)

	mu.Lock()
	defer mu.Unlock()
	return done, inFlight
}

/**
* snapshotStep abre un snapshot con el estado confirmado actual (hasta dos a la vez) o compara y cierra el más viejo.
* Solo lo llama la goroutine de las operaciones, así que done no cambia mientras tanto.
* @param e *Engine, gen *generator, snaps *[]openSnap, done state
* @return error
**/
func snapshotStep(e *Engine, gen *generator, snaps *[]openSnap, done state) error {
	if len(*snaps) < 2 && (len(*snaps) == 0 || gen.rng.IntN(2) == 0) {
		*snaps = append(*snaps, openSnap{v: e.Snapshot(), want: done.clone(), dropped: map[string]bool{}})
		return nil
	}
	oldest := (*snaps)[0]
	*snaps = (*snaps)[1:]
	return oldest.check(e, gen.ids)
}

/**
* closeWithin cierra el motor viejo y falla si Close no vuelve a tiempo.
* @param t *testing.T, e *Engine, limit time.Duration
**/
func closeWithin(t *testing.T, e *Engine, limit time.Duration) {
	closed := make(chan struct{})
	go func() {
		e.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(limit):
		t.Fatal("Close of the crashed engine hung")
	}
}

/**
* checkRecovered comprueba que el motor reabierto tenga exactamente confirmed o pending (si no es nil) y devuelve
* el que tiene.
* @param t *testing.T, e *Engine, confirmed state, pending *state
* @return state
**/
func checkRecovered(t *testing.T, e *Engine, confirmed state, pending *state) state {
	t.Helper()
	got := engineState(t, e)
	if got.equal(confirmed) {
		return confirmed
	}
	if pending != nil && got.equal(*pending) {
		return *pending
	}
	t.Fatalf("recovered state is neither the confirmed nor the pending one:\n%s", diffState(got, confirmed))
	return nil
}

/**
* engineState lee el contenido entero del motor con un recorrido de cada store, y comprueba Count.
* @param t *testing.T, e *Engine
* @return state
**/
func engineState(t *testing.T, e *Engine) state {
	t.Helper()
	got := state{}
	for _, name := range e.Stores() {
		s := mustStore(t, e, name)
		ids := map[string]string{}
		err := s.tree.Scan("", nil, func(v btree.Version) (bool, error) {
			value, err := s.valueOf(v)
			ids[v.ID] = string(value)
			return true, err
		})
		mustOK(t, err)
		if s.Count() != int64(len(ids)) {
			t.Fatalf("store %q: Count = %d, has %d ids", name, s.Count(), len(ids))
		}
		got[name] = ids
	}
	return got
}

/**
* diffState describe la primera diferencia entre dos estados.
* @param got, want state
* @return string
**/
func diffState(got, want state) string {
	var b bytes.Buffer
	for _, name := range unionNames(got, want) {
		for id, v := range want[name] {
			if g, ok := got[name][id]; !ok || g != v {
				fmt.Fprintf(&b, "store %s id %.30q: got %v (%d B), want %d B\n", name, id, ok, len(g), len(v))
			}
		}
		for id := range got[name] {
			if _, ok := want[name][id]; !ok {
				fmt.Fprintf(&b, "store %s id %.30q: extra\n", name, id)
			}
		}
	}
	return b.String()
}

// TestCrashBetweenCheckpointSteps corta el disco al terminar cada paso del checkpoint (1 a 6), en los tres modos.
// Entre el 3 (catálogo con los contadores nuevos) y el 5 (control con el punto de rehacer nuevo), la recuperación
// rehace desde el punto viejo y no debe contar dos veces (countLSN). Todo lo confirmado sigue ahí y Count coincide.
func TestCrashBetweenCheckpointSteps(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for step := 1; step <= 6; step++ {
			t.Run(fmt.Sprintf("%s/step%d", name, step), func(t *testing.T) {
				t.Parallel()
				crashAtStep(t, mode, step)
			})
		}
	}
}

/**
* crashAtStep escribe, hace un checkpoint, escribe más y corta en el paso step del checkpoint siguiente.
* @param t *testing.T, mode crashfs.Mode, step int
**/
func crashAtStep(t *testing.T, mode crashfs.Mode, step int) {
	base := crashfs.New()
	proc := newProc(base)
	e := openTest(t, proc, testConfig())
	gen := newGenerator(uint64(step)*7 + uint64(mode))
	model := state{}
	runOps := func(n int) {
		for range n {
			o := gen.next()
			if o.kind == "checkpoint" || o.kind == "drop" {
				continue
			}
			want := o.apply(model)
			if got, err := o.run(e); err != nil || got != want {
				t.Fatalf("%v = %v, %v; model says %v", o, got, err, want)
			}
		}
	}
	runOps(300)
	mustOK(t, e.Checkpoint())
	runOps(300)
	e.afterStep = func(s int) {
		if s == step {
			proc.kill(mode, int64(step))
		}
	}
	e.Checkpoint()
	closeWithin(t, e, 10*time.Second)

	e = openTest(t, newProc(base), testConfig())
	checkRecovered(t, e, model, nil)
	verifyTrees(t, e)
	runOps(100)
	mustOK(t, e.Close())
	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	checkRecovered(t, e, model, nil)
}
