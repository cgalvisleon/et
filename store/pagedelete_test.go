package store

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
)

// Borrado de páginas del árbol en el motor (tarea 70): réplica, Verify/Repair tras un corte y el disco en régimen.

// slideID es el id número i de una ventana deslizante: crece con i, así los nuevos van al final y los viejos se borran
// del principio.
func slideID(i int) string { return fmt.Sprintf("k%012d", i) }

/**
* slideWindow mueve la ventana de n ids desde from: escribe los ids [from, from+n), borra los de n posiciones antes y
* reemplaza uno de cada cuatro de los que quedan. Diferidas: el disco no es lo que se mide.
* @param s *Store, from, n, window, size int, rng *rand.Rand
* @return error
**/
func slideWindow(s *Store, from, n, window, size int, rng *rand.Rand) error {
	d := s.Deferred()
	value := make([]byte, size)
	for i := from; i < from+n; i++ {
		rng.Read(value)
		if err := d.Put(slideID(i), value); err != nil {
			return err
		}
		if i >= window {
			if err := d.Remove(slideID(i - window)); err != nil {
				return err
			}
		}
		if i >= window && rng.Intn(4) == 0 {
			rng.Read(value)
			if err := d.Put(slideID(i-rng.Intn(window)), value); err != nil {
				return err
			}
		}
	}
	return nil
}

// TestPageDeleteReplica: una réplica física aplica los borrados de páginas del primario (medio muertas, desenganches y
// reciclados) mientras se lee con snapshots en los dos sentidos: los recorridos salen en orden y estables, y al final
// la réplica coincide con el primario (contenido, Count, Verify limpio y las mismas páginas borradas en espera).
func TestPageDeleteReplica(t *testing.T) {
	cfg := smallWAL()
	cfg.checkpointTimeout = 200 * time.Millisecond // los checkpoints por tamaño se reparten en este intervalo
	p := openTest(t, crashfs.New(), cfg)
	defer p.Close()
	_, err := p.CreateSlot("r1")
	mustOK(t, err)
	s := mustStore(t, p, "w")
	rng := rand.New(rand.NewSource(1))
	const window = 2000
	mustOK(t, slideWindow(s, 0, window, window, 200, rng))
	r, _ := newReplica(t, p)
	defer r.Close()

	var stop atomic.Bool
	errs := make(chan error, 4)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); orderedReads(r, "w", &stop, errs) }()
	go func() {
		defer wg.Done()
		for !stop.Load() {
			if n, err := streamOnce(p, r, "r1"); err != nil {
				errs <- err
				return
			} else if n == 0 {
				time.Sleep(time.Millisecond)
			}
		}
	}()
	for round := range 8 {
		mustOK(t, slideWindow(s, window*(round+1), window, window, 200, rng))
		mustOK(t, p.Vacuum())
	}
	stop.Store(true)
	wg.Wait()
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
	if st := p.Stats(); st.PagesDeleted == 0 || st.PagesRecycled == 0 {
		t.Fatalf("the primary did not delete or recycle pages: %+v", st)
	}
	catchUp(t, p, r, "r1")
	checkContents(t, r, dumpEngine(t, p))
	pr, rr := checkClean(t, p), checkClean(t, r)
	if pr.Deleted != rr.Deleted || pr.TreePages != rr.TreePages {
		t.Fatalf("primary %d tree and %d deleted pages, replica %d and %d", pr.TreePages, pr.Deleted, rr.TreePages, rr.Deleted)
	}
	t.Logf("primary: %s", strings.ReplaceAll(pr.String(), "\n", "; "))
}

/**
* orderedReads lee el store name de la réplica con snapshots hasta stop: hacia adelante y hacia atrás, los ids salen
* en orden estricto y la cantidad coincide con CountAt. Un conflicto con el aplicador (ErrRecoveryConflict) se tolera.
* @param r *Engine, name string, stop *atomic.Bool, errs chan error
**/
func orderedReads(r *Engine, name string, stop *atomic.Bool, errs chan error) {
	s, err := r.Store(name)
	if err != nil {
		errs <- err
		return
	}
	for asc := true; !stop.Load(); asc = !asc {
		if err := orderedOnce(r, s, asc); err != nil && !errors.Is(err, ErrRecoveryConflict) {
			errs <- err
			return
		}
	}
}

/**
* orderedOnce recorre el store con un snapshot propio y comprueba el orden y la cantidad.
* @param r *Engine, s *Store, asc bool
* @return error
**/
func orderedOnce(r *Engine, s *Store, asc bool) error {
	v := r.Snapshot()
	defer v.Close()
	want, err := s.CountAt(v)
	if err != nil {
		return err
	}
	var ids []string
	err = s.ForEachAt(v, func(id string, _ []byte) (bool, error) {
		ids = append(ids, id)
		return true, nil
	}, asc, 0, 0)
	if err != nil {
		return err
	}
	if !asc {
		slices.Reverse(ids)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			return fmt.Errorf("scan (asc %v): %q is not before %q", asc, ids[i-1], ids[i])
		}
	}
	if int64(len(ids)) != want {
		return fmt.Errorf("scan (asc %v): %d ids, CountAt %d", asc, len(ids), want)
	}
	return nil
}

// TestPageDeleteVerifyRepair: una limpieza que borra páginas, cortada en distintos puntos de sus operaciones de disco
// (con el escritor del WAL de fondo, así lo hecho llega a ser durable), deja como mucho borrados a medias (Verify los
// informa como reparables, FindHalfDead), páginas sin dueño o cadenas huérfanas; Repair los termina y Verify da
// limpio, con el contenido intacto. Al menos un corte tiene que dejar una hoja medio muerta.
func TestPageDeleteVerifyRepair(t *testing.T) {
	base := crashfs.New()
	cfg := testConfig()
	cfg.buffers = 24
	cfg.walWriterDelay = time.Millisecond
	cfg.wal.Buffer = 0 // cada registro a su escritura: los cortes caen entre los pasos del borrado
	e := openTest(t, base, cfg)
	s := mustStore(t, e, "w")
	want := map[string]string{}
	for i := range 3000 {
		id := strings.Repeat("p", 150) + slideID(i)
		mustOK(t, s.Put(id, []byte(id[150:])))
		if i >= 1500 {
			want[id] = id[150:]
		}
	}
	for i := range 1500 {
		mustOK(t, s.Remove(strings.Repeat("p", 150)+slideID(i)))
	}
	mustOK(t, e.Checkpoint())
	mustOK(t, e.Close())
	img := readImage(t, base, testDir)
	total := vacuumOps(t, img, cfg)
	t.Logf("Vacuum does %d disk operations", total)
	halfDead := 0
	for cut := int64(1); cut < total; cut += max(1, total/40) {
		for _, mode := range []crashfs.Mode{crashfs.ReorderUnsynced, crashfs.TornWrites} {
			halfDead += crashVacuum(t, img, cfg, want, mode, cut)
		}
	}
	if halfDead == 0 {
		t.Fatal("no cut left a half-dead leaf: the test does not check FindHalfDead")
	}
	t.Logf("%d cuts left half-dead leaves", halfDead)
}

/**
* vacuumOps cuenta las operaciones de disco de un Vacuum entero sobre una copia de img.
* @param t testing.TB, img diskImage, cfg config
* @return int64
**/
func vacuumOps(t testing.TB, img diskImage, cfg config) int64 {
	t.Helper()
	proc := newProc(writeImage(t, img))
	e := openTest(t, proc, cfg)
	proc.killAfter(1<<40, crashfs.LoseUnsynced, 0)
	mustOK(t, e.Vacuum())
	if e.Stats().PagesDeleted == 0 {
		t.Fatal("the vacuum deleted no pages")
	}
	n := 1<<40 - proc.budget.Load()
	mustOK(t, e.Close())
	return n
}

/**
* crashVacuum corta un Vacuum tras cut operaciones de disco, reabre, comprueba Verify (solo hallazgos que explica un
* corte), Repair, Verify limpio y el contenido.
* @param t *testing.T, img diskImage, cfg config, want map[string]string, mode crashfs.Mode, cut int64
* @return int: 1 si quedó una hoja medio muerta
**/
func crashVacuum(t *testing.T, img diskImage, cfg config, want map[string]string, mode crashfs.Mode, cut int64) int {
	base := writeImage(t, img)
	proc := newProc(base)
	e := openTest(t, proc, cfg)
	proc.killAfter(cut, mode, cut)
	_ = e.Vacuum()
	if proc.budget.Load() > 0 {
		proc.kill(mode, cut)
	}
	_ = e.Close()

	e = openTest(t, newProc(base), testConfig())
	defer e.Close()
	r, err := e.Verify()
	mustOK(t, err)
	found := 0
	for _, f := range r.Findings {
		switch f.Kind {
		case FindHalfDead:
			found = 1
		case FindLostPage, FindOrphanChain:
		default:
			t.Fatalf("cut %d: unexpected finding %s\n%s", cut, f, r)
		}
	}
	_, err = e.Repair()
	mustOK(t, err)
	checkClean(t, e)
	checkStoreContent(t, mustStore(t, e, "w"), want)
	mustOK(t, e.Vacuum())
	checkClean(t, e)
	return found
}

// TestDiskSteadyState mide la meta de espacio (F-23): con una ventana deslizante que escribe al final, borra al
// principio y reemplaza en el medio, y la limpieza de fondo corriendo, el disco del store en régimen estable no pasa
// de 1,5 × los datos vivos (ids + valores). Mide después de cada vuelta completa de la ventana; el régimen es desde
// la tercera.
func TestDiskSteadyState(t *testing.T) {
	cfg := smallWAL()
	// la limpieza de fondo va por tiempo: 2 ms la mantiene al ritmo de las escrituras (con 5 ms quedaba atrás desde
	// que el WAL junta sus escrituras, tarea 100, y el pico de la ventana pasaba de 1,5)
	cfg.vacuumEvery, cfg.vacuumWrites = 2*time.Millisecond, 0
	cfg.checkpointTimeout = 50 * time.Millisecond
	cfg.buffers = 512
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	s := mustStore(t, e, "w")
	rng := rand.New(rand.NewSource(7))
	const window, size, rounds = 20000, 256, 10
	live := float64(window * (len(slideID(0)) + size))
	var ratios []float64
	for round := range rounds {
		mustOK(t, slideWindow(s, round*window, window, window, size, rng))
		time.Sleep(20 * time.Millisecond) // una limpieza de fondo más
		u, err := e.DirUsage()
		mustOK(t, err)
		ratios = append(ratios, float64(u.Stores["w"])/live)
	}
	st := e.Stats()
	t.Logf("disk / live data per round: %.2f; %d vacuums, %d pages deleted, %d recycled", ratios, st.Vacuums,
		st.PagesDeleted, st.PagesRecycled)
	r, err := e.Verify()
	mustOK(t, err)
	t.Logf("pages: %s", strings.ReplaceAll(r.String(), "\n", "; "))
	if steady := slices.Max(ratios[2:]); steady > 1.5 {
		t.Fatalf("steady disk is %.2f × the live data (goal ≤ 1.5)", steady)
	}
	if st.PagesDeleted == 0 || st.PagesRecycled == 0 {
		t.Fatalf("no pages deleted or recycled: %+v", st)
	}
}
