package store

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// consumer es un consumidor lógico de prueba: aplica los cambios a un modelo y anota el orden de las confirmaciones.
type consumer struct {
	model   contents
	commits []uint64 // CommitLSN de cada transacción entregada, en orden
	last    TxInfo
	changes int
}

/**
* newConsumer arma un consumidor con el modelo vacío.
* @return *consumer
**/
func newConsumer() *consumer {
	return &consumer{model: contents{}}
}

/**
* apply es la fn de ChangeReader.Next: aplica el cambio al modelo y exige confirmaciones en orden creciente.
* @param tx TxInfo, c Change
* @return error
**/
func (k *consumer) apply(tx TxInfo, c Change) error {
	if tx != k.last {
		if tx.CommitLSN <= k.last.CommitLSN {
			return fmt.Errorf("commit lsn %d after %d", tx.CommitLSN, k.last.CommitLSN)
		}
		k.commits = append(k.commits, tx.CommitLSN)
		k.last = tx
	}
	k.changes++
	switch c.Op {
	case ChangePut:
		if strings.HasPrefix(c.ID, "rb-") {
			return fmt.Errorf("a rolled back write came out: %s/%s", c.Store, c.ID)
		}
		k.store(c.Store)[c.ID] = string(c.Data)
	case ChangeDelete:
		delete(k.store(c.Store), c.ID)
	case ChangeCreate:
		k.store(c.Store)
	case ChangeEmpty:
		k.model[c.Store] = map[string]string{}
	case ChangeDrop:
		delete(k.model, c.Store)
	}
	return nil
}

/**
* store es el mapa de un store del modelo (lo crea si falta).
* @param name string
* @return map[string]string
**/
func (k *consumer) store(name string) map[string]string {
	m := k.model[name]
	if m == nil {
		m = map[string]string{}
		k.model[name] = m
	}
	return m
}

/**
* drain lee el lector hasta alcanzar el WAL durable.
* @param t testing.TB, r *ChangeReader, k *consumer
**/
func drain(t testing.TB, r *ChangeReader, k *consumer) {
	t.Helper()
	for {
		n, err := r.Next(10, k.apply)
		mustOK(t, err)
		if n == 0 {
			// sin transacciones nuevas: puede quedar WAL sin entregar que no confirma nada
			if _, err := r.Next(0, k.apply); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
}

/**
* sameModel compara el modelo del consumidor con el contenido del motor.
* @param t testing.TB, k *consumer, want contents
**/
func sameModel(t testing.TB, k *consumer, want contents) {
	t.Helper()
	if len(k.model) != len(want) {
		t.Fatalf("consumer has %d stores, engine %d", len(k.model), len(want))
	}
	for name, w := range want {
		g, ok := k.model[name]
		if !ok || len(g) != len(w) {
			t.Fatalf("store %q: consumer %d ids (found %v), engine %d", name, len(g), ok, len(w))
		}
		for id, v := range w {
			if g[id] != v {
				t.Fatalf("store %q id %q: consumer %.20q, engine %.20q", name, id, g[id], v)
			}
		}
	}
}

/**
* rolledBack escribe un lote que se deshace (sus ids empiezan con rb-): nunca debe salir en el flujo.
* @param t testing.TB, e *Engine, s *Store, i int
**/
func rolledBack(t testing.TB, e *Engine, s *Store, i int) {
	t.Helper()
	tx := e.NewBatch()
	mustOK(t, s.PutIn(tx, fmt.Sprintf("rb-%d", i), bytes.Repeat([]byte{'r'}, 100+i%3*9000)))
	mustOK(t, tx.Rollback())
}

// TestChangesOrder: con escritores concurrentes (sueltos, diferidos, lotes con y sin desborde, lotes deshechos) y DDL
// suelto y en lotes, el flujo lógico entrega cada transacción confirmada una vez, en orden de confirmación, nunca una
// deshecha; aplicado a un modelo da exactamente el contenido del motor.
func TestChangesOrder(t *testing.T) {
	e := openTest(t, crashfs.New(), smallWAL())
	defer e.Close()
	_, err := e.CreateLogicalSlot("c")
	mustOK(t, err)
	if _, err := e.CreateLogicalSlot("c"); !errors.Is(err, ErrSlotExists) {
		t.Fatalf("duplicated logical slot: %v", err)
	}
	stores := fillStores(t, e, []string{"a", "b", "empty"}, 60)
	r, err := e.Changes("c")
	mustOK(t, err)
	defer r.Close()
	k := newConsumer()
	for round := range 5 {
		mustOK(t, writeN(stores[:2], 3, 150, int64(round)))
		rolledBack(t, e, stores[0], round)
		mustOK(t, ddlOnce(e, round))
		mustOK(t, e.WaitDurable())
		drain(t, r, k)
		mustOK(t, r.Confirm())
	}
	sameModel(t, k, dumpEngine(t, e))
	if len(k.commits) < 100 {
		t.Fatalf("only %d transactions", len(k.commits))
	}
	if err := e.AdvanceSlot("c", 1); !errors.Is(err, ErrSlotNotFound) {
		t.Fatalf("AdvanceSlot on a logical slot: %v", err)
	}
	if _, err := e.Changes("nope"); !errors.Is(err, ErrSlotNotFound) {
		t.Fatalf("Changes of an unknown slot: %v", err)
	}
}

// TestChangesSpill: una transacción grande (miles de cambios y valores en desborde) pasa a disco sin que la memoria
// del búfer pase de su tope, mientras otras chicas confirman en el medio; sale entera y en su lugar, y su archivo de
// derrame se borra al entregarla.
func TestChangesSpill(t *testing.T) {
	cfg := testConfig()
	cfg.decodeWorkMem = 64 << 10
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	_, err := e.CreateLogicalSlot("big")
	mustOK(t, err)
	s := mustStore(t, e, "a")
	small := mustStore(t, e, "small")
	r, err := e.Changes("big")
	mustOK(t, err)
	defer r.Close()

	tx := e.NewBatch()
	rng := rand.New(rand.NewSource(3))
	for i := range 1500 {
		v := bytes.Repeat([]byte{byte('a' + i%26)}, 500+rng.Intn(800))
		if i%100 == 0 {
			v = bytes.Repeat([]byte{'z'}, 20000+i) // a desborde
		}
		mustOK(t, s.PutIn(tx, fmt.Sprintf("k%05d", i), v))
		if i%300 == 0 {
			mustOK(t, small.Put(fmt.Sprint(i), []byte("between")))
			mustOK(t, e.WaitDurable())
			k := newConsumer() // entrega las chicas mientras la grande sigue abierta
			_, err := r.Next(0, k.apply)
			mustOK(t, err)
		}
	}
	mustOK(t, tx.Commit())
	k := newConsumer()
	drain(t, r, k)
	if r.spilled == 0 || r.peakMem > r.workMem+64<<10 {
		t.Fatalf("spilled %d bytes, peak memory %d (limit %d)", r.spilled, r.peakMem, r.workMem)
	}
	if len(k.commits) != 1 || len(k.model["a"]) != 1500 {
		t.Fatalf("%d transactions, %d ids", len(k.commits), len(k.model["a"]))
	}
	for id, v := range dumpEngine(t, e)["a"] {
		if k.model["a"][id] != v {
			t.Fatalf("id %s differs", id)
		}
	}
	if names, err := e.fs.ReadDir(r.dir); err == nil && len(names) != 0 {
		t.Fatalf("spill files left: %v", names)
	}
}

// TestChangesRestart: un consumidor que confirma, cierra y vuelve a abrir el lector (también con el motor cerrado y
// abierto en el medio, y con un lote abierto que cruza el punto donde confirmó) recibe exactamente la misma secuencia
// de transacciones que uno que nunca se detuvo: sin repetidos ni huecos.
func TestChangesRestart(t *testing.T) {
	base := crashfs.New()
	cfg := smallWAL()
	e := openTest(t, base, cfg)
	_, err := e.CreateLogicalSlot("steady")
	mustOK(t, err)
	_, err = e.CreateLogicalSlot("restart")
	mustOK(t, err)
	stores := fillStores(t, e, []string{"a", "b"}, 40)
	steady, restart := newConsumer(), newConsumer()

	open := func() *ChangeReader {
		r, err := e.Changes("restart")
		mustOK(t, err)
		return r
	}
	r := open()
	for round := range 6 {
		long := e.NewBatch() // abierto mientras el consumidor confirma y se reinicia
		mustOK(t, stores[0].PutIn(long, fmt.Sprintf("long-%d", round), bytes.Repeat([]byte{'L'}, 9000)))
		mustOK(t, writeN(stores, 2, 80, int64(round)))
		mustOK(t, e.WaitDurable())
		_, err := r.Next(7, restart.apply) // una parte
		mustOK(t, err)
		mustOK(t, r.Confirm())
		mustOK(t, r.Close())
		mustOK(t, stores[1].PutIn(long, fmt.Sprintf("long2-%d", round), []byte("tail")))
		mustOK(t, long.Commit())
		if round == 3 {
			mustOK(t, e.Close())
			e = openTest(t, base, cfg)
			stores = []*Store{mustStore(t, e, "a"), mustStore(t, e, "b")}
		}
		mustOK(t, e.WaitDurable())
		r = open()
		drain(t, r, restart)
		mustOK(t, r.Confirm())
	}
	mustOK(t, r.Close())
	whole, err := e.Changes("steady")
	mustOK(t, err)
	drain(t, whole, steady)
	mustOK(t, whole.Close())
	if fmt.Sprint(steady.commits) != fmt.Sprint(restart.commits) || steady.changes != restart.changes {
		t.Fatalf("restarted consumer: %d transactions, %d changes; steady: %d, %d", len(restart.commits),
			restart.changes, len(steady.commits), steady.changes)
	}
	sameModel(t, restart, dumpEngine(t, e))
	mustOK(t, e.Close())
}

// TestChangesBatchDDLUndo (tarea 94): el KindDropStore de xid 0 que deshace un store creado dentro de un lote (el
// mismo lote lo borra con DropStoreIn, o el lote se deshace) no sale como DDL suelto, y los cambios del lote en ese
// store tampoco: el consumidor termina igual que el motor. Antes el flujo entregaba [drop:x, create:x, put:/k,
// drop:y] y el consumidor quedaba con un store fantasma "".
func TestChangesBatchDDLUndo(t *testing.T) {
	e := openTest(t, crashfs.New(), smallWAL())
	defer e.Close()
	_, err := e.CreateLogicalSlot("c")
	mustOK(t, err)
	a := mustStore(t, e, "a")
	r, err := e.Changes("c")
	mustOK(t, err)
	defer r.Close()

	tx := e.NewBatch()
	x, err := e.StoreIn(tx, "x")
	mustOK(t, err)
	mustOK(t, x.PutIn(tx, "k", []byte("v")))
	mustOK(t, x.PutIn(tx, "big", bytes.Repeat([]byte("b"), 20000)))
	mustOK(t, a.PutIn(tx, "kept", []byte("1")))
	mustOK(t, e.DropStoreIn(tx, "x"))
	mustOK(t, tx.Commit())

	tx = e.NewBatch()
	y, err := e.StoreIn(tx, "y")
	mustOK(t, err)
	mustOK(t, y.PutIn(tx, "k", []byte("v")))
	mustOK(t, tx.Rollback())

	tx = e.NewBatch()
	z, err := e.StoreIn(tx, "z")
	mustOK(t, err)
	mustOK(t, z.PutIn(tx, "k", []byte("z")))
	mustOK(t, tx.Commit())

	mustOK(t, e.WaitDurable())
	k := newConsumer()
	var ops []string
	_, err = r.Next(0, func(tx TxInfo, c Change) error {
		ops = append(ops, fmt.Sprintf("%s:%s/%s", c.Op, c.Store, c.ID))
		return k.apply(tx, c)
	})
	mustOK(t, err)
	sameModel(t, k, dumpEngine(t, e))
	for _, op := range ops {
		if strings.Contains(op, ":x/") || strings.Contains(op, ":y/") || strings.Contains(op, ":/") {
			t.Fatalf("the stream has %s: %v", op, ops)
		}
	}
}
