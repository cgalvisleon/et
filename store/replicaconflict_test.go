package store

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// Pruebas de los conflictos de snapshot en la réplica acotados por lo que cada snapshot ve (tarea 125).

/**
* timedCatchUp copia a la réplica lo durable del primario y devuelve cuánto tardó (la espera a los snapshots que
* notan un registro entra ahí).
* @param t testing.TB, p, r *Engine
* @return time.Duration
**/
func timedCatchUp(t testing.TB, p, r *Engine) time.Duration {
	t.Helper()
	start := time.Now()
	catchUp(t, p, r, "r1")
	return time.Since(start)
}

/**
* updateAll escribe cada id de ids rounds veces con un valor que dice la vuelta.
* @param t testing.TB, s *Store, ids []string, rounds, from int
**/
func updateAll(t testing.TB, s *Store, ids []string, from, rounds int) {
	t.Helper()
	for r := from; r < from+rounds; r++ {
		for _, id := range ids {
			mustOK(t, s.Put(id, fmt.Appendf(nil, "%s-v%d-%0200d", id, r, r)))
		}
	}
}

/**
* readsRound comprueba que el snapshot v lea en s la vuelta round de cada id.
* @param t testing.TB, s *Store, v *Snapshot, ids []string, round int
**/
func readsRound(t testing.TB, s *Store, v *Snapshot, ids []string, round int) {
	t.Helper()
	for _, id := range ids {
		got, ok, err := s.ReadAt(v, id)
		mustOK(t, err)
		want := fmt.Sprintf("%s-v%d-%0200d", id, round, round)
		if !ok || string(got) != want {
			t.Fatalf("%s: leído %q (%v), se esperaba la vuelta %d", id, got, ok, round)
		}
	}
}

// TestReplicaConflictBySnapshot: en la réplica, un snapshot largo que no ve nada de lo que el primario poda (ids
// nacidos después de tomarlo) sigue abierto y el aplicador no lo espera, aunque lleguen muchas confirmaciones y podas
// antes de partir; las confirmaciones de xid que no ve tampoco lo afectan; la limpieza que quita versiones que ve (o
// congela versiones que no veía) espera el plazo y lo cancela, y en la misma tanda uno tomado después de las
// actualizaciones (ve todo lo que queda) sigue abierto.
func TestReplicaConflictBySnapshot(t *testing.T) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	_, err := p.CreateSlot("r1")
	mustOK(t, err)
	cold, hot := mustStore(t, p, "cold"), mustStore(t, p, "hot")
	var coldIDs, hotIDs []string
	for i := range 4 { // dos vueltas caben en una hoja: actualizar no poda
		coldIDs = append(coldIDs, fmt.Sprintf("c%03d", i))
	}
	for i := range 5 { // pocos ids: la poda antes de partir los deja en una hoja (sin borrar ni reciclar páginas)
		hotIDs = append(hotIDs, fmt.Sprintf("h%03d", i))
	}
	updateAll(t, cold, coldIDs, 0, 1)
	r, _ := newReplica(t, p)
	defer r.Close()
	catchUp(t, p, r, "r1")
	rcold, rhot := mustStore(t, r, "cold"), mustStore(t, r, "hot")

	long := r.Snapshot() // no ve nada de hot
	defer long.Close()
	old := r.Snapshot() // va a ver la vuelta 0 de cold, que la limpieza quita
	defer old.Close()

	// Muchas versiones de ids nuevos: confirmaciones y podas antes de partir (tarea 108) en el primario. (Una limpieza
	// también congelaría las más nuevas, que los dos no ven: eso sí los afecta, ver abajo.)
	pruned := hot.tree.Pruned()
	updateAll(t, hot, hotIDs, 0, 60)
	if hot.tree.Pruned() == pruned {
		t.Fatal("el primario no podó antes de partir")
	}
	if d := timedCatchUp(t, p, r); d >= time.Second {
		t.Fatalf("el aplicador esperó %v con snapshots que no ven nada de lo podado", d)
	}
	readsRound(t, rcold, long, coldIDs, 0)
	if _, ok, err := rhot.ReadAt(long, hotIDs[0]); err != nil || ok {
		t.Fatalf("el snapshot largo ve un id nacido después: %v %v", ok, err)
	}

	// Actualizar cold no cancela a nadie (las confirmaciones son de xid que ninguno ve, y no hay poda)...
	updateAll(t, cold, coldIDs, 1, 1)
	if d := timedCatchUp(t, p, r); d >= time.Second {
		t.Fatalf("las confirmaciones esperaron %v", d)
	}
	readsRound(t, rcold, old, coldIDs, 0)
	fresh := r.Snapshot() // ve la vuelta 1
	defer fresh.Close()

	// ...pero la limpieza sí afecta a los que ven la vuelta 0 de cold, que quita, o no ven lo que congela (long y old):
	// esperan el plazo y se cancelan. fresh ve todo lo que queda: sigue abierto.
	mustOK(t, p.Vacuum())
	mustOK(t, p.Checkpoint()) // WalSince entrega lo durable: el checkpoint hace durable la limpieza
	if d := timedCatchUp(t, p, r); d < replicaConfig().conflictDelay {
		t.Fatalf("la limpieza de versiones visibles no esperó el plazo: %v", d)
	}
	for _, v := range []*Snapshot{long, old} {
		if _, _, err := rcold.ReadAt(v, coldIDs[0]); !errors.Is(err, ErrRecoveryConflict) {
			t.Fatalf("un snapshot que veía lo limpiado no se canceló: %v", err)
		}
	}
	readsRound(t, rcold, fresh, coldIDs, 1)
	if n := r.Stats().RecoveryConflicts; n != 2 {
		t.Fatalf("conflictos: %d, se esperaban 2", n)
	}
}

// TestReplicaConflictCommitSeen: la confirmación de un lote cuyas escrituras ya llegaron antes de tomar el snapshot sí
// lo afecta (vería el xid): el aplicador lo espera y lo cancela; a otro tomado antes de esas escrituras no.
func TestReplicaConflictCommitSeen(t *testing.T) {
	p := openTest(t, crashfs.New(), smallWAL())
	defer p.Close()
	_, err := p.CreateSlot("r1")
	mustOK(t, err)
	s := mustStore(t, p, "docs")
	ids := []string{"a", "b"}
	updateAll(t, s, ids, 0, 1)
	r, _ := newReplica(t, p)
	defer r.Close()
	catchUp(t, p, r, "r1")
	rs := mustStore(t, r, "docs")

	before := r.Snapshot() // tomado antes de las escrituras de los lotes
	defer before.Close()
	first, second := p.NewBatch(), p.NewBatch()
	mustOK(t, s.PutIn(first, "a", []byte("a-v1-"+fmt.Sprintf("%0200d", 1))))
	mustOK(t, s.PutIn(second, "b", fmt.Appendf(nil, "%s-v%d-%0200d", "b", 1, 1)))
	mustOK(t, second.Commit()) // hace durable también la escritura de first, todavía sin confirmar
	if d := timedCatchUp(t, p, r); d >= time.Second {
		t.Fatalf("la confirmación de un xid que nadie ve esperó %v", d)
	}
	after := r.Snapshot() // su Xmax ya pasa el xid de first
	defer after.Close()
	readsRound(t, rs, after, []string{"a"}, 0)
	readsRound(t, rs, after, []string{"b"}, 1)

	mustOK(t, first.Commit())
	if d := timedCatchUp(t, p, r); d < replicaConfig().conflictDelay {
		t.Fatalf("la confirmación no esperó al snapshot que la vería: %v", d)
	}
	if _, _, err := rs.ReadAt(after, "a"); !errors.Is(err, ErrRecoveryConflict) {
		t.Fatalf("el snapshot que vería el lote no se canceló: %v", err)
	}
	readsRound(t, rs, before, ids, 0)
}
