package store

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

/**
* readChanges lee el lector hasta el WAL durable y devuelve cada cambio como «store/id:origen» (un DDL como «op:store»).
* @param t testing.TB, r *ChangeReader
* @return []string
**/
func readChanges(t testing.TB, r *ChangeReader) []string {
	t.Helper()
	var got []string
	_, err := r.Next(0, func(tx TxInfo, c Change) error {
		switch c.Op {
		case ChangePut, ChangeDelete:
			got = append(got, fmt.Sprintf("%s/%s:%d", c.Store, c.ID, tx.Origin))
		default:
			got = append(got, fmt.Sprintf("%s:%s", c.Op, c.Store))
		}
		return nil
	})
	mustOK(t, err)
	return got
}

// TestChangesFilter (tarea 75): ChangesWith deja pasar solo lo que el filtro elige, por store, por id y por origen
// (el modo multi de josefina descarta lo que aplicó por replicación); el filtro de id también sirve con valores en
// desborde, y un DDL suelto cuenta como origen 0.
func TestChangesFilter(t *testing.T) {
	e := openTest(t, crashfs.New(), smallWAL())
	defer e.Close()
	_, err := e.CreateLogicalSlot("f")
	mustOK(t, err)
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	mustOK(t, a.Put("a1", []byte("x")))
	mustOK(t, b.Put("b1", []byte("y")))
	tx := e.NewBatch()
	mustOK(t, tx.SetOrigin(3))
	mustOK(t, a.PutIn(tx, "a2", []byte("from 3")))
	mustOK(t, a.PutIn(tx, "skip-big", []byte(strings.Repeat("z", 30_000))))
	mustOK(t, tx.Commit())
	tx = e.NewBatch()
	mustOK(t, a.PutIn(tx, "a3", []byte(strings.Repeat("w", 30_000))))
	mustOK(t, b.PutIn(tx, "b2", nil))
	mustOK(t, tx.Commit())
	mustOK(t, e.WaitDurable())

	r, err := e.ChangesWith("f", ChangeFilter{
		Store:  func(s string) bool { return s == "a" },
		ID:     func(s, id string) bool { return !strings.HasPrefix(id, "skip") },
		Origin: func(o uint16) bool { return o == 0 },
	})
	mustOK(t, err)
	defer r.Close()
	if got, want := strings.Join(readChanges(t, r), " "), "create:a a/a1:0 a/a3:0"; got != want {
		t.Fatalf("filtered changes %q, want %q", got, want)
	}
}

// TestChangesFromPos (tarea 75): un lector sin ranura desde ChangeHead entrega lo que confirma después, también el
// lote que ya escribía al tomarla, y nada de antes; cortado en medio, ChangesFrom(Pos) sigue sin repetir ni saltar,
// también tras reabrir el motor; CommitLSN es el de su TxInfo; AdvanceLogicalSlot mueve una ranura a una posición;
// Confirm sin ranura da ErrSlotNotFound; y una posición reciclada da ErrWalRemoved.
func TestChangesFromPos(t *testing.T) {
	base := crashfs.New()
	cfg := smallWAL()
	e := openTest(t, base, cfg)
	_, err := e.CreateLogicalSlot("s")
	mustOK(t, err)
	a := mustStore(t, e, "a")
	mustOK(t, a.Put("before", []byte("1")))
	open := e.NewBatch()
	mustOK(t, a.PutIn(open, "open-1", []byte("2"))) // escribe antes de la posición, confirma después
	head, err := e.ChangeHead()
	mustOK(t, err)
	mustOK(t, a.PutIn(open, "open-2", []byte("3")))
	mustOK(t, open.Commit())
	var lsns []uint64
	for i := range 6 {
		tx := e.NewBatch()
		mustOK(t, a.PutIn(tx, fmt.Sprintf("after-%d", i), []byte("4")))
		mustOK(t, tx.Commit())
		lsns = append(lsns, tx.CommitLSN())
	}
	mustOK(t, e.WaitDurable())
	want := "a/open-1:0 a/open-2:0 a/after-0:0 a/after-1:0 a/after-2:0 a/after-3:0 a/after-4:0 a/after-5:0"

	whole, err := e.ChangesFrom(head, ChangeFilter{})
	mustOK(t, err)
	if got := strings.Join(readChanges(t, whole), " "); got != want {
		t.Fatalf("from head %q, want %q", got, want)
	}
	if err := whole.Confirm(); !errors.Is(err, ErrSlotNotFound) {
		t.Fatalf("Confirm without a slot: %v", err)
	}
	mustOK(t, whole.Close())

	// en partes: dos transacciones, la posición, y el resto con otro lector tras reabrir
	part, err := e.ChangesFrom(head, ChangeFilter{})
	mustOK(t, err)
	var got []string
	var commits []uint64
	_, err = part.Next(2, func(tx TxInfo, c Change) error {
		got = append(got, fmt.Sprintf("%s/%s:%d", c.Store, c.ID, tx.Origin))
		commits = append(commits, tx.CommitLSN)
		return nil
	})
	mustOK(t, err)
	pos := part.Pos()
	mustOK(t, part.Close())
	if commits[len(commits)-1] != lsns[0] {
		t.Fatalf("TxInfo.CommitLSN %d, Batch.CommitLSN %d", commits[len(commits)-1], lsns[0])
	}
	mustOK(t, e.Close())
	e = openTest(t, base, cfg)
	rest, err := e.ChangesFrom(pos, ChangeFilter{})
	mustOK(t, err)
	got = append(got, readChanges(t, rest)...)
	mustOK(t, rest.Close())
	if strings.Join(got, " ") != want {
		t.Fatalf("in two parts %q, want %q", strings.Join(got, " "), want)
	}

	// la ranura avanzada a pos entrega lo mismo que el segundo lector
	mustOK(t, e.AdvanceLogicalSlot("s", pos))
	slot, err := e.Changes("s")
	mustOK(t, err)
	if got, rest := strings.Join(readChanges(t, slot), " "), want[strings.Index(want, "a/after-1"):]; got != rest {
		t.Fatalf("advanced slot %q, want %q", got, rest)
	}
	mustOK(t, slot.Close())

	// una posición cuyo WAL ya se recicló
	mustOK(t, e.DropSlot("s"))
	for i := range 4 {
		mustOK(t, writeN([]*Store{mustStore(t, e, "a")}, 2, 400, int64(i)))
		mustOK(t, e.Checkpoint())
	}
	old, err := e.ChangesFrom(head, ChangeFilter{})
	mustOK(t, err)
	if _, err := old.Next(0, func(TxInfo, Change) error { return nil }); !errors.Is(err, ErrWalRemoved) {
		t.Fatalf("recycled position: %v", err)
	}
	mustOK(t, old.Close())
	mustOK(t, e.Close())
}

// TestChangesFromReplica (tarea 75): ChangesFrom anda en una réplica física: entrega lo que el primario confirmó,
// con su origen, desde una posición del primario.
func TestChangesFromReplica(t *testing.T) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	_, err := p.CreateSlot("r")
	mustOK(t, err)
	a := mustStore(t, p, "a")
	r, _ := newReplica(t, p)
	defer r.Close()
	head, err := p.ChangeHead()
	mustOK(t, err)
	tx := p.NewBatch()
	mustOK(t, tx.SetOrigin(9))
	mustOK(t, a.PutIn(tx, "k", []byte("v")))
	mustOK(t, tx.Commit())
	mustOK(t, a.Put("m", []byte("w")))
	catchUp(t, p, r, "r")
	rd, err := r.ChangesFrom(head, ChangeFilter{})
	mustOK(t, err)
	defer rd.Close()
	if got, want := strings.Join(readChanges(t, rd), " "), "a/k:9 a/m:0"; got != want {
		t.Fatalf("on the replica %q, want %q", got, want)
	}
}

// TestSlotTopology (tarea 76, pregunta 4 de P-8): la topología de josefina cabe en las 16 ranuras. En el modo multi un
// nodo tiene una sola ranura lógica (la de sus cambios) que leen todos sus vecinos con ChangesFrom desde su propia
// posición; la ranura se avanza al mínimo de las posiciones con AdvanceLogicalSlot y retiene el WAL del vecino más
// atrasado (el checkpoint no lo recicla). Con 15 réplicas físicas del modo single más esa lógica se llega al tope, y
// la ranura 17 da ErrSlotLimit.
func TestSlotTopology(t *testing.T) {
	e := openTest(t, crashfs.New(), smallWAL())
	defer e.Close()
	_, err := e.CreateLogicalSlot("node")
	mustOK(t, err)
	a := mustStore(t, e, "a")
	head, err := e.ChangeHead()
	mustOK(t, err)
	neighbors := []ChangePos{head, head, head}
	for round := range 4 {
		mustOK(t, writeN([]*Store{a}, 2, 300, int64(round)))
		mustOK(t, e.WaitDurable())
		for i := range neighbors {
			if i == 2 && round > 0 {
				continue // el vecino 2 se atrasa
			}
			r, err := e.ChangesFrom(neighbors[i], ChangeFilter{})
			mustOK(t, err)
			_, err = r.Next(0, func(TxInfo, Change) error { return nil })
			mustOK(t, err)
			neighbors[i] = r.Pos()
			mustOK(t, r.Close())
		}
		low := neighbors[0]
		for _, p := range neighbors[1:] {
			if p.Restart < low.Restart {
				low = p
			}
		}
		mustOK(t, e.AdvanceLogicalSlot("node", low))
		mustOK(t, e.Checkpoint())
	}
	late, err := e.ChangesFrom(neighbors[2], ChangeFilter{})
	mustOK(t, err)
	n, err := late.Next(0, func(TxInfo, Change) error { return nil })
	mustOK(t, err)
	mustOK(t, late.Close())
	if n == 0 {
		t.Fatal("the late neighbor read nothing")
	}
	for i := 1; i < maxSlots; i++ {
		_, err := e.CreateSlot(fmt.Sprintf("replica-%02d", i))
		mustOK(t, err)
	}
	if _, err := e.CreateSlot("one-too-many"); !errors.Is(err, ErrSlotLimit) {
		t.Fatalf("slot %d: %v", maxSlots+1, err)
	}
}
