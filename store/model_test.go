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

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/xact"
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
	kind     string // put, remove, insert, update, delete, read, drop, empty, checkpoint, vacuum, batch, snapshot, scan, waitdurable
	store    string
	id       string
	value    string
	deferred bool // una escritura suelta con Deferred, o un lote con CommitDeferred

	// Lote (kind batch): sus operaciones, si termina con Rollback y lo que debe devolver cada una. wants se arma en
	// el generador con su largo y lo llena apply: las copias de op comparten el arreglo.
	sub      []op
	rollback bool
	wants    []bool
}

func (o op) String() string {
	kind := o.kind
	if o.deferred {
		kind += " (deferred)"
	}
	return fmt.Sprintf("%s %s/%.20q (%d B)", kind, o.store, o.id, len(o.value))
}

/**
* durable dice si, al volver op, todo lo confirmado hasta ella es seguro durable (el WAL es durable hasta su lsn, y
* con él lo anterior): una escritura no diferida que cambió el estado (o un Put, que siempre escribe), un checkpoint o
* WaitDurable. Las que no escriben (un Insert que no aplica, un lote deshecho) no hacen fsync.
* @param before, after state: el estado antes y después de op
* @return bool
**/
func (o op) durable(before, after state) bool {
	switch o.kind {
	case "checkpoint", "waitdurable":
		return true
	case "put":
		return !o.deferred
	}
	return !o.deferred && !before.equal(after)
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
	case "drop", "empty":
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
	rng      *rand.Rand
	ids      []string
	deferred bool // también escrituras diferidas, lotes con CommitDeferred y WaitDurable (solo la prueba de modelo)
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
	case r < 22:
		o.kind = "empty"
	case r < 32:
		o.kind = "scan"
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
	if g.deferred {
		g.makeDeferred(&o)
	}
	return o
}

/**
* makeDeferred vuelve diferida una de cada cuatro escrituras sueltas y cambia una de cada 40 operaciones por
* WaitDurable.
* @param o *op
**/
func (g *generator) makeDeferred(o *op) {
	switch {
	case g.rng.IntN(40) == 0:
		o.kind = "waitdurable"
	case o.kind == "put", o.kind == "insert", o.kind == "update", o.kind == "delete", o.kind == "remove":
		o.deferred = g.rng.IntN(4) == 0
	}
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
		case "drop", "empty", "checkpoint", "vacuum", "batch", "snapshot", "scan", "waitdurable":
			continue
		}
		if g.rng.IntN(5) == 0 {
			sub.kind = "read"
		}
		o.sub = append(o.sub, sub)
	}
	o.wants = make([]bool, len(o.sub))
	o.deferred = g.deferred && !o.rollback && g.rng.IntN(4) == 0
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
	case "waitdurable":
		return true, e.WaitDurable()
	case "drop":
		if !slices.Contains(e.Stores(), o.store) {
			return true, nil
		}
		return true, e.DropStore(o.store)
	}
	switch o.kind {
	case "batch":
		return true, o.runBatch(e)
	case "snapshot", "scan":
		return true, nil // los snapshots y recorridos los maneja crashDuring; para los demás no hacen nada
	}
	s, err := e.Store(o.store)
	if err != nil {
		return false, err
	}
	if o.kind == "empty" {
		return true, s.Empty()
	}
	var w writer = s
	if o.deferred {
		w = s.Deferred()
	}
	switch o.kind {
	case "put":
		return true, w.Put(o.id, []byte(o.value))
	case "remove":
		return true, w.Remove(o.id)
	case "insert":
		return w.Insert(o.id, []byte(o.value))
	case "update":
		return w.Update(o.id, []byte(o.value))
	}
	return w.Delete(o.id)
}

// writer son las escrituras sueltas que tienen igual Store y Deferred.
type writer interface {
	Put(id string, data []byte) error
	Remove(id string) error
	Insert(id string, data []byte) (bool, error)
	Update(id string, data []byte) (bool, error)
	Delete(id string) (bool, error)
}

/**
* runBatch hace el lote o: cada operación dentro de un Batch, comparando lo que devuelve con o.wants (una lectura
* compara si encontró el id), y Commit o Rollback. Si una operación falla, deshace el lote antes de volver.
* @param e *Engine
* @return error: el del motor, o la primera diferencia con el modelo
**/
func (o op) runBatch(e *Engine) (err error) {
	tx := e.NewBatch()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
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
			if len(sub.value)%2 == 0 {
				_, got, err = s.ReadIn(tx, sub.id)
			} else {
				got, err = s.ExistsIn(tx, sub.id)
			}
		}
		if err != nil {
			return err
		}
		if got != o.wants[i] {
			return fmt.Errorf("in a batch, %v returned %v, model says %v", sub, got, o.wants[i])
		}
	}
	switch {
	case o.rollback:
		return tx.Rollback()
	case o.deferred:
		return tx.CommitDeferred()
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
			if found, err = s.ExistsAt(o.v, id); err != nil || found != exists {
				return fmt.Errorf("snapshot: ExistsAt store %s id %.20q = %v, %v; want %v", name, id, found, err, exists)
			}
		}
		if err := checkScan(o.want[name], func(fn func(string, []byte) (bool, error)) error {
			return s.ForEachAt(o.v, fn, true, 0, 0)
		}); err != nil {
			return fmt.Errorf("snapshot: ForEachAt store %s: %w", name, err)
		}
		if n, err := s.CountAt(o.v); err != nil || n != int64(len(o.want[name])) {
			return fmt.Errorf("snapshot: CountAt store %s = %d, %v; want %d", name, n, err, len(o.want[name]))
		}
	}
	return nil
}

/**
* checkScan comprueba que un recorrido entregue exactamente los ids y valores de want, en orden ascendente.
* @param want map[string]string, each func(fn) error: el recorrido
* @return error: el del recorrido o la primera diferencia
**/
func checkScan(want map[string]string, each func(fn func(string, []byte) (bool, error)) error) error {
	var prev string
	n := 0
	err := each(func(id string, data []byte) (bool, error) {
		if n > 0 && id <= prev {
			return false, fmt.Errorf("id %.20q after %.20q", id, prev)
		}
		if v, ok := want[id]; !ok || v != string(data) {
			return false, fmt.Errorf("id %.20q = %d B, want %v (%d B)", id, len(data), ok, len(v))
		}
		prev, n = id, n+1
		return true, nil
	})
	if err == nil && n != len(want) {
		err = fmt.Errorf("%d ids, want %d", n, len(want))
	}
	return err
}

/**
* scanStep compara con el estado confirmado un recorrido al azar de un store: ForEach entero, ForEachBorrowed, una
* ventana de Keys (asc o desc, offset y limit) y KeysAfter paginado hasta el final. Solo lo llama la goroutine de
* las operaciones, así que done no cambia mientras tanto.
* @param e *Engine, gen *generator, done state
* @return error
**/
func scanStep(e *Engine, gen *generator, done state) error {
	name := []string{"a", "b", "c"}[gen.rng.IntN(3)]
	s, err := e.Store(name)
	if err != nil {
		return err
	}
	want := done[name]
	if err := checkScan(want, func(fn func(string, []byte) (bool, error)) error { return s.ForEach(fn, true, 0, 0) }); err != nil {
		return fmt.Errorf("ForEach store %s: %w", name, err)
	}
	if err := checkScan(want, s.ForEachBorrowed); err != nil {
		return fmt.Errorf("ForEachBorrowed store %s: %w", name, err)
	}
	ids := slices.Sorted(maps.Keys(want))
	asc, offset, limit := gen.rng.IntN(2) == 0, gen.rng.IntN(len(ids)+2), gen.rng.IntN(20)
	ref := slices.Clone(ids)
	if !asc {
		slices.Reverse(ref)
	}
	from := min(offset, len(ref))
	to := len(ref)
	if limit > 0 {
		to = min(from+limit, len(ref))
	}
	keys, err := s.Keys(asc, offset, limit)
	if err != nil || !slices.Equal(keys, ref[from:to]) {
		return fmt.Errorf("Keys(%v, %d, %d) store %s = %d ids, %v; want %d", asc, offset, limit, name, len(keys), err, to-from)
	}
	var paged []string
	for after := ""; ; {
		page, err := s.KeysAfter(after, 1+gen.rng.IntN(15))
		if err != nil {
			return err
		}
		if len(page) == 0 {
			break
		}
		paged, after = append(paged, page...), page[len(page)-1]
	}
	if !slices.Equal(paged, ids) {
		return fmt.Errorf("KeysAfter store %s: %d ids, want %d", name, len(paged), len(ids))
	}
	for range 4 {
		id := gen.ids[gen.rng.IntN(len(gen.ids))]
		_, exists := want[id]
		if found, err := s.Exists(id); err != nil || found != exists {
			return fmt.Errorf("Exists store %s id %.20q = %v, %v; want %v", name, id, found, err, exists)
		}
	}
	return nil
}

// TestModelCrash es la prueba de modelo con cortes del motor entero: operaciones al azar contra un mapa en memoria
// (sueltas, Empty, recorridos —ForEach, ForEachBorrowed, Keys, KeysAfter paginado, Exists— comparados con el modelo,
// lotes de varias operaciones en varios stores que a veces terminan con Rollback, y snapshots que comparan
// su vista unas operaciones después), sobre crashfs, con checkpoints y limpiezas (Vacuum) a mano y de fondo (la
// limpieza de fondo, en la mitad de las semillas); un corte en un momento al azar (entre operaciones o en medio de
// una) en los tres modos y varias semillas. Una de cada cuatro escrituras sueltas es diferida (Deferred), uno de cada
// cuatro lotes confirmados usa CommitDeferred y a veces se llama a WaitDurable. Al reabrir, el contenido es
// exactamente el estado tras alguna operación desde la última que dejó todo durable (una escritura no diferida que
// escribió, un checkpoint, WaitDurable) hasta la que estaba en curso: las diferidas sin esperar pueden perderse, pero
// solo enteras y en orden del WAL (un lote entero o nada; nunca se pierde una confirmada durable), Count coincide, btree.Verify pasa en cada store y, tras una limpieza, ninguna página está repetida
// en la lista de libres ni es a la vez libre y de una cadena viva (las huérfanas de cadenas escritas antes de un corte
// sin su hoja se cuentan, no se rechazan), y Engine.Verify solo encuentra lo que explica un corte (páginas sin dueño,
// cadenas y archivos huérfanos), que Repair deja limpio; el motor acepta escrituras y un segundo y un tercer corte también se
// recuperan. Los archivos de estados son de prueba (una página) y el primer xid está junto al final del primero:
// tras reabrir, toda versión por debajo del límite truncado está congelada, y al cerrar el primer archivo ya no está.
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
	gen.deferred = true
	candidates := []state{{}} // los estados que puede tener el motor tras el corte
	// Archivos de estados de una página y el primer xid junto al final del primero: se llena enseguida y se trunca.
	cfg := smallXact(xact.XidsPerPage - 10)
	cfg.checkpointTimeout = []time.Duration{time.Hour, 40 * time.Millisecond, 5 * time.Millisecond}[seed%3]
	cfg.checkEvery = 5 * time.Millisecond
	cfg.buffers = 32
	if seed%2 == 1 {
		cfg.vacuumEvery, cfg.vacuumWrites = 3*time.Millisecond, 20
	}
	if seed%4 >= 2 {
		cfg.walWriterDelay = 2 * time.Millisecond
	}

	for round := range 3 {
		proc := newProc(base)
		e := openTest(t, proc, cfg)
		confirmed := checkRecovered(t, e, candidates)
		verifyTrees(t, e)
		vacuumAndAudit(t, e, confirmed)
		checkAfterCrash(t, e)
		t.Logf("round %d: replayed %d records, %d stores", round, e.replayed, len(e.Stores()))
		target := 20 + gen.rng.IntN(1500)
		candidates = crashDuring(t, e, proc, gen, confirmed, target, mode, seed+uint64(round))
	}
	e := openTest(t, newProc(base), cfg)
	confirmed := checkRecovered(t, e, candidates)
	verifyTrees(t, e)
	vacuumAndAudit(t, e, confirmed)
	checkAfterCrash(t, e)
	mustOK(t, e.Close())
	if files := xactFiles(t, base); slices.Contains(files, "0001") {
		t.Fatalf("status files %v: 0001 not truncated", files)
	}
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
		checkLeaves(t, s, e.status.Oldest())
		s.vacuumMu.Unlock()
	}
	t.Logf("after Vacuum: %d orphan blocks", orphans)
}

/**
* crashDuring corre operaciones en una goroutine y corta el disco cuando van target confirmadas (a veces en medio de
* la siguiente). Comprueba que cada operación confirmada devolvió lo que dice el modelo y que Close del motor viejo
* no se cuelga.
* @return []state: los estados posibles tras el corte, desde el de la última operación que dejó todo durable hasta el
* de la que estaba en curso (las diferidas del medio pueden perderse, en orden)
**/
func crashDuring(t *testing.T, e *Engine, proc *procFS, gen *generator, start state, target int,
	mode crashfs.Mode, seed uint64) []state {
	var (
		mu       sync.Mutex
		done     = start.clone()
		chain    = []state{done} // estados desde la última operación que dejó todo durable
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
			for o.kind == "snapshot" || o.kind == "scan" { // no cuentan como operación: la siguiente sí
				step := func() error { return scanStep(e, gen, done) }
				if o.kind == "snapshot" {
					step = func() error { return snapshotStep(e, gen, &snaps, done) }
				}
				if err := step(); err != nil {
					if !killed.Load() {
						t.Errorf("%v", err)
					}
					return
				}
				o = gen.next()
			}
			if o.kind == "empty" { // Empty no corre con snapshots abiertos (ErrBusy): se comparan y cierran antes
				for _, open := range snaps {
					if err := open.check(e, gen.ids); err != nil {
						if !killed.Load() {
							t.Errorf("%v", err)
						}
						return
					}
				}
				snaps = nil
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
			if o.durable(done, after) {
				chain = chain[:0]
			}
			done, inFlight, chain = after, nil, append(chain, after)
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
	if inFlight != nil {
		chain = append(chain, *inFlight)
	}
	t.Logf("crash with %d possible states", len(chain))
	return chain
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
* checkRecovered comprueba que el motor reabierto tenga exactamente uno de los estados candidates y devuelve el que
* tiene.
* @param t *testing.T, e *Engine, candidates []state: el primero es el último seguro durable
* @return state
**/
func checkRecovered(t *testing.T, e *Engine, candidates []state) state {
	t.Helper()
	got := engineState(t, e)
	for i := len(candidates) - 1; i >= 0; i-- {
		if got.equal(candidates[i]) {
			if len(candidates) > 1 {
				t.Logf("recovered possible state %d of %d", i+1, len(candidates))
			}
			return candidates[i]
		}
	}
	t.Fatalf("recovered state is none of the %d possible ones; against the last durable one:\n%s",
		len(candidates), diffState(got, candidates[0]))
	return nil
}

/**
* recoveredOneOf es checkRecovered con el estado confirmado y, si no es nil, el de la operación en curso.
* @param t *testing.T, e *Engine, confirmed state, pending *state
* @return state
**/
func recoveredOneOf(t *testing.T, e *Engine, confirmed state, pending *state) state {
	t.Helper()
	candidates := []state{confirmed}
	if pending != nil {
		candidates = append(candidates, *pending)
	}
	return checkRecovered(t, e, candidates)
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
		mustOK(t, s.ForEach(func(id string, data []byte) (bool, error) {
			ids[id] = string(data)
			return true, nil
		}, true, 0, 0))
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
	recoveredOneOf(t, e, model, nil)
	verifyTrees(t, e)
	runOps(100)
	mustOK(t, e.Close())
	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	recoveredOneOf(t, e, model, nil)
}
