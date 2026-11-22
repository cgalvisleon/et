package store

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Pruebas de Open tras un cierre limpio (tarea 109): no corta el WAL, anota StateInProduction antes de escribir y
// un corte después no deja registros fantasma.

/**
* walSizes es el largo de cada segmento del WAL del motor.
* @param t testing.TB, fsys vfs.FS
* @return map[string]int64
**/
func walSizes(t testing.TB, fsys vfs.FS) map[string]int64 {
	t.Helper()
	dir := filepath.Join(testDir, wal.Dir)
	names, err := fsys.ReadDir(dir)
	mustOK(t, err)
	sizes := map[string]int64{}
	for _, name := range names {
		f, err := fsys.OpenFile(filepath.Join(dir, name), 0)
		mustOK(t, err)
		sizes[name], err = f.Size()
		mustOK(t, err)
		f.Close()
	}
	return sizes
}

/**
* putRange escribe los ids n0..n1-1 de s (durables, o diferidos con deferred) con un valor que sale del número.
* @param t testing.TB, s *Store, n0, n1 int, deferred bool
**/
func putRange(t testing.TB, s *Store, n0, n1 int, deferred bool) {
	t.Helper()
	for i := n0; i < n1; i++ {
		id, val := fmt.Sprintf("k%05d", i), fmt.Appendf(nil, "v%d-%0300d", i, i)
		if deferred {
			mustOK(t, s.Deferred().Put(id, val))
		} else {
			mustOK(t, s.Put(id, val))
		}
	}
}

/**
* checkRange comprueba los ids n0..n1-1 de s: presentes con su valor (must) o, si no, presentes con su valor o ausentes.
* @param t testing.TB, s *Store, n0, n1 int, must bool
* @return int: cuántos estaban
**/
func checkRange(t testing.TB, s *Store, n0, n1 int, must bool) int {
	t.Helper()
	found := 0
	for i := n0; i < n1; i++ {
		got, ok, err := s.Read(fmt.Sprintf("k%05d", i))
		mustOK(t, err)
		if !ok {
			if must {
				t.Fatalf("falta k%05d", i)
			}
			continue
		}
		if want := fmt.Sprintf("v%d-%0300d", i, i); string(got) != want {
			t.Fatalf("k%05d: valor %q", i, got)
		}
		found++
	}
	return found
}

// TestCleanOpenKeepsWAL: tras un Close limpio, Open no trunca ningún segmento (el activo y los repuestos siguen
// preasignados enteros) y deja el control en StateInProduction; tras un corte, Open corta como siempre.
func TestCleanOpenKeepsWAL(t *testing.T) {
	cfg := smallWAL()
	base := crashfs.New()
	e := openTest(t, base, cfg)
	s := mustStore(t, e, "docs")
	putRange(t, s, 0, 3000, false)
	mustOK(t, e.Checkpoint())
	putRange(t, s, 3000, 4000, false)
	mustOK(t, e.Close())
	before := walSizes(t, base)

	e = openTest(t, base, cfg)
	after := walSizes(t, base)
	for name, size := range after {
		if size != cfg.wal.SegmentSize || before[name] != 0 && before[name] != size {
			t.Fatalf("segmento %s: medía %d, tras Open %d", name, before[name], size)
		}
	}
	ctl, err := e.ctl.Read()
	mustOK(t, err)
	if ctl.State != control.StateInProduction {
		t.Fatalf("control en %d tras abrir", ctl.State)
	}
	checkRange(t, mustStore(t, e, "docs"), 0, 4000, true)
	mustOK(t, e.Close())

	p := newProc(base) // un corte: el próximo Open corta
	e = openTest(t, p, cfg)
	putRange(t, mustStore(t, e, "docs"), 4000, 4100, false)
	p.kill(crashfs.LoseUnsynced, 1)
	e.Close()
	e = openTest(t, base, cfg)
	defer e.Close()
	cut := false
	for _, size := range walSizes(t, base) {
		cut = cut || size < cfg.wal.SegmentSize
	}
	if !cut {
		t.Fatal("tras un corte Open no cortó el WAL")
	}
}

// TestCleanOpenCrash: cierre limpio → Open sin cortar → escrituras durables y diferidas (rotando a repuestos
// reciclados) → corte en los tres modos → Open: está todo lo durable, lo diferido es un prefijo con sus valores y
// Verify solo encuentra lo que explica un corte. También un corte en medio del Open limpio (antes, durante y después de anotar el control).
func TestCleanOpenCrash(t *testing.T) {
	for name, mode := range crashModes {
		for seed := int64(0); seed < 3; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				cleanOpenCrash(t, mode, seed)
			})
		}
	}
}

/**
* cleanOpenCrash es un caso de TestCleanOpenCrash.
* @param t *testing.T, mode crashfs.Mode, seed int64
**/
func cleanOpenCrash(t *testing.T, mode crashfs.Mode, seed int64) {
	cfg := smallWAL()
	base := crashfs.New()
	e := openTest(t, base, cfg)
	putRange(t, mustStore(t, e, "docs"), 0, 2000, false)
	mustOK(t, e.Checkpoint())
	putRange(t, mustStore(t, e, "docs"), 2000, 2500, false)
	mustOK(t, e.Close())

	// Un corte en medio del Open limpio: lo de antes sigue entero.
	for _, n := range []int64{1, 3, 8, 20, 60} {
		p := newProc(base)
		p.killAfter(n, mode, seed+n)
		if e, err := open(p, testDir, cfg); err == nil {
			e.Close()
		}
		p.kill(mode, seed+n)
		e = openTest(t, base, cfg)
		checkRange(t, mustStore(t, e, "docs"), 0, 2500, true)
		mustOK(t, e.Close())
	}

	p := newProc(base)
	e = openTest(t, p, cfg)
	s := mustStore(t, e, "docs")
	putRange(t, s, 2500, 3500, false)
	putRange(t, s, 3500, 4500, true)
	p.kill(mode, seed)
	e.Close()

	e = openTest(t, base, cfg)
	defer e.Close()
	s = mustStore(t, e, "docs")
	checkRange(t, s, 0, 3500, true)
	found := checkRange(t, s, 3500, 4500, false)
	if found > 0 {
		checkRange(t, s, 3500, 3500+found, true) // lo diferido sobrevive como un prefijo
	}
	checkAfterCrash(t, e)
}
