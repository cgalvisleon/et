package store

import (
	"bytes"
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestPressureWAL: el WAL sin checkpoint sube la presión (causa wal) hasta 1 al llegar al tope de throttle, y un
// Checkpoint la baja.
func TestPressureWAL(t *testing.T) {
	cfg := testConfig()
	cfg.maxWALSize = 1 << 20
	cfg.checkEvery = time.Hour // sin checkpoints de fondo
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	s := mustStore(t, e, "s")
	start := e.Pressure()
	limit := cfg.maxWALSize * walBacklogFactor
	value := bytes.Repeat([]byte("x"), 1000)
	half := 0.0
	for i := 0; e.walBacklog() <= limit; i++ {
		mustOK(t, s.Put(fmt.Sprintf("id-%03d", i%300), value))
		if half == 0 && e.walBacklog() > limit/2 {
			half = e.Pressure()
		}
	}
	st := e.Stats()
	if half < 0.5 || st.Pressure < 1 || st.PressureCause != PressureWAL || st.Pressure != e.Pressure() {
		t.Fatalf("presión al empezar %.3f, a mitad %.3f, al tope %.3f (%q)", start, half, st.Pressure, st.PressureCause)
	}
	mustOK(t, e.Checkpoint())
	if p := e.Pressure(); p > 0.05 {
		t.Fatalf("presión tras Checkpoint = %.3f (%q)", p, e.Stats().PressureCause)
	}
}

// TestPressureDeferred: las escrituras diferidas sin fsync suben la presión (causa deferred) y WaitDurable la baja.
func TestPressureDeferred(t *testing.T) {
	cfg := testConfig()
	cfg.buffers = 4096 // las páginas sucias no pesan
	e := openTest(t, crashfs.New(), cfg)
	defer e.Close()
	s := mustStore(t, e, "s")
	for i := range 3000 {
		mustOK(t, s.Deferred().Put(fmt.Sprintf("d-%05d", i), []byte("v")))
	}
	st := e.Stats()
	want := float64(e.log.Published()-e.log.Durable()) / maxDeferredLag
	if st.PressureCause != PressureDeferred || st.Pressure < 0.25 || st.Pressure != want {
		t.Fatalf("presión con diferidas = %.3f (%q), quiero %.3f", st.Pressure, st.PressureCause, want)
	}
	mustOK(t, e.WaitDurable())
	if st = e.Stats(); st.PressureCause == PressureDeferred || st.Pressure > 0.05 {
		t.Fatalf("presión tras WaitDurable = %.3f (%q)", st.Pressure, st.PressureCause)
	}
}

// TestPressureDirty: con una caché chica las páginas sucias pesan (causa dirty) y el checkpoint las limpia.
func TestPressureDirty(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	defer e.Close()
	s := mustStore(t, e, "s")
	for i := range 400 {
		mustOK(t, s.Put(fmt.Sprintf("k-%04d", i), bytes.Repeat([]byte("y"), 200)))
	}
	st := e.Stats()
	want := float64(st.CacheDirty) / float64(st.CacheSize)
	if st.PressureCause != PressureDirty || st.Pressure != want || want <= 0 {
		t.Fatalf("presión = %.3f (%q); sucias %d de %d", st.Pressure, st.PressureCause, st.CacheDirty, st.CacheSize)
	}
	mustOK(t, e.Checkpoint())
	if p := e.Pressure(); p > 0.05 {
		t.Fatalf("presión tras Checkpoint = %.3f", p)
	}
}

// TestPressureReadOnly: un fsync que falla por disco lleno deja el motor de solo lectura y la presión en 1.
func TestPressureReadOnly(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	s := mustStore(t, e, "a")
	for i := range 50 {
		mustOK(t, s.Put(fmt.Sprintf("k-%03d", i), []byte("v")))
	}
	mustOK(t, e.Checkpoint())
	mustOK(t, e.log.Room()) // el WAL ya tiene su lugar: la falta de espacio la encuentra el fsync
	if p := e.Pressure(); p >= 1 {
		t.Fatalf("presión antes de llenar el disco = %.3f", p)
	}
	fsys.SetCapacity(fsys.Used() - 1)
	for i := 0; !e.Stats().ReadOnly; i++ {
		if i == 50 {
			t.Fatal("ningún Put llegó al fsync que falla")
		}
		if err := s.Put(fmt.Sprintf("after-%d", i), []byte("v")); !errors.Is(err, ErrDiskFull) {
			t.Fatalf("put con el disco lleno: %v", err)
		}
	}
	if st := e.Stats(); st.Pressure < 1 || st.PressureCause != PressureReadOnly || e.Pressure() < 1 {
		t.Fatalf("presión de solo lectura = %.3f (%q)", st.Pressure, st.PressureCause)
	}
	closeWithin(t, e, 10*time.Second)
}

/**
* checkUsage pide DirUsage y comprueba que cuadre: Total = la suma de las partes = lo que usa el disco simulado, y
* Data = la suma de los stores (sin huérfanos).
* @param t *testing.T, e *Engine, fsys crashfs.FS
* @return DiskUsage
**/
func checkUsage(t *testing.T, e *Engine, fsys crashfs.FS) DiskUsage {
	t.Helper()
	u, err := e.DirUsage()
	mustOK(t, err)
	var stores int64
	for _, size := range u.Stores {
		stores += size
	}
	if u.Total != u.Data+u.WAL+u.Xact+u.Control || u.Total != fsys.Used() || stores != u.Data {
		t.Fatalf("uso %+v: stores %d, disco %d", u, stores, fsys.Used())
	}
	if u.WAL < testConfig().wal.SegmentSize || u.Control == 0 {
		t.Fatalf("uso %+v: el WAL cuenta lo preasignado y la raíz el control", u)
	}
	return u
}

// TestDirUsage: el uso crece al cargar, cada store da lo mismo por Store.DirUsage, y un store borrado o vaciado deja
// de contar; en todo momento las partes suman el total, que es lo que ocupa el disco simulado.
func TestDirUsage(t *testing.T) {
	fsys := crashfs.New()
	e := openTest(t, fsys, testConfig())
	defer e.Close()
	before := checkUsage(t, e, fsys)
	a, b := mustStore(t, e, "a"), mustStore(t, e, "b")
	value := bytes.Repeat([]byte("z"), 300)
	for i := range 600 {
		mustOK(t, a.Put(fmt.Sprintf("k-%04d", i), value))
		mustOK(t, b.Put(fmt.Sprintf("k-%04d", i), value))
	}
	mustOK(t, e.Checkpoint())
	loaded := checkUsage(t, e, fsys)
	if loaded.Total <= before.Total || loaded.Data <= before.Data || loaded.Stores["a"] < 10*8192 {
		t.Fatalf("uso antes %+v, después de cargar %+v", before, loaded)
	}
	for _, s := range []*Store{a, b} {
		size, err := s.DirUsage()
		mustOK(t, err)
		if size != loaded.Stores[s.Name()] {
			t.Fatalf("store %s: DirUsage %d, en el motor %d", s.Name(), size, loaded.Stores[s.Name()])
		}
	}

	mustOK(t, e.DropStore("b"))
	mustOK(t, e.Checkpoint())
	dropped := checkUsage(t, e, fsys)
	if _, ok := dropped.Stores["b"]; ok || dropped.Data != loaded.Data-loaded.Stores["b"] {
		t.Fatalf("tras DropStore: %+v (antes %+v)", dropped, loaded)
	}
	if _, err := b.DirUsage(); !errors.Is(err, ErrStoreDropped) {
		t.Fatalf("DirUsage de un store borrado: %v", err)
	}

	mustOK(t, a.Empty())
	mustOK(t, e.Checkpoint())
	emptied := checkUsage(t, e, fsys)
	if emptied.Stores["a"] >= dropped.Stores["a"]/4 {
		t.Fatalf("tras Empty: %+v (antes %+v)", emptied, dropped)
	}

	mustOK(t, e.Close())
	if _, err := e.DirUsage(); !errors.Is(err, ErrClosed) {
		t.Fatalf("DirUsage del motor cerrado: %v", err)
	}
}

// TestDirUsageOS: sobre el disco real, Total es la suma de los largos de los archivos del directorio.
func TestDirUsageOS(t *testing.T) {
	dir := t.TempDir()
	e, err := open(vfs.OS(), dir, testConfig())
	mustOK(t, err)
	defer e.Close()
	s := mustStore(t, e, "a")
	for i := range 200 {
		mustOK(t, s.Put(fmt.Sprintf("k-%04d", i), bytes.Repeat([]byte("w"), 300)))
	}
	mustOK(t, e.Checkpoint())
	u, err := e.DirUsage()
	mustOK(t, err)
	var walked int64
	mustOK(t, filepath.WalkDir(dir, func(_ string, d iofs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		walked += info.Size()
		return err
	}))
	if u.Total != walked || u.Data != u.Stores["a"] || u.Data == 0 {
		t.Fatalf("uso %+v; el directorio mide %d", u, walked)
	}
}

// TestPressureUsageConcurrent: Pressure, Stats y DirUsage (del motor y de un store) en concurrencia con escrituras,
// diferidas, checkpoints y stores que se crean y se borran (para -race).
func TestPressureUsageConcurrent(t *testing.T) {
	e := openTest(t, crashfs.New(), testConfig())
	defer e.Close()
	s := mustStore(t, e, "s")
	stop := make(chan struct{})
	var wg sync.WaitGroup
	check := func(err error) { // desde las goroutines: t.Error, no t.Fatal
		if err != nil {
			t.Error(err)
		}
	}
	run := func(fn func(i int)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				fn(i)
			}
		}()
	}
	run(func(i int) { check(s.Put(fmt.Sprintf("k-%03d", i%500), []byte("valor"))) })
	run(func(i int) { check(s.Deferred().Put(fmt.Sprintf("d-%03d", i%500), []byte("v"))) })
	run(func(i int) {
		name := fmt.Sprintf("t-%d", i%3)
		x, err := e.Store(name)
		if err != nil {
			t.Error(err)
			return
		}
		check(x.Put("k", []byte("v")))
		check(e.DropStore(name))
	})
	run(func(i int) {
		if i%20 == 0 {
			check(e.Checkpoint())
		}
	})
	run(func(int) {
		if p := e.Pressure(); p < 0 {
			t.Errorf("presión negativa: %f", p)
		}
		_ = e.Stats()
		if _, err := e.DirUsage(); err != nil {
			t.Errorf("DirUsage: %v", err)
		}
		if _, err := s.DirUsage(); err != nil {
			t.Errorf("Store.DirUsage: %v", err)
		}
	})
	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
	mustOK(t, e.WaitDurable())
	mustOK(t, e.Checkpoint())
	u, err := e.DirUsage()
	mustOK(t, err)
	if len(u.Stores) != 1 || u.Stores["s"] == 0 {
		t.Fatalf("uso al final: %+v", u)
	}
}
