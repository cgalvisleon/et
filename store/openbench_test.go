package store

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* TestOpenBench mide Open tras un Close limpio con muchas claves (tarea 69 de F-23) sobre el disco real: carga
* STORE_BENCH_KEYS claves (10 millones por omisión) de 100 B en un store, cierra y abre y cierra STORE_BENCH_OPENS
* veces (5), con la configuración de Open (variables de entorno). Muestra cada Open con el desglose por paso
* (config.openStep) y la mediana de cada paso. STORE_BENCH_DIR elige el directorio; con STORE_BENCH_KEEP=1 no se
* borra y, si ya tiene un motor, no se vuelve a cargar. STORE_BENCH_CPUPROFILE perfila solo los Open. Solo corre con
* STORE_BENCH=1.
* @param t *testing.T
**/
func TestOpenBench(t *testing.T) {
	if os.Getenv("STORE_BENCH") == "" {
		t.Skip("STORE_BENCH=1 para medir")
	}
	n := benchEnvInt("STORE_BENCH_KEYS", 10_000_000)
	opens := benchEnvInt("STORE_BENCH_OPENS", 5)
	root := os.Getenv("STORE_BENCH_DIR")
	if root == "" {
		root = t.TempDir()
	}
	dir := filepath.Join(root, "openbench")
	if os.Getenv("STORE_BENCH_KEEP") == "" {
		defer os.RemoveAll(dir)
	}
	if _, err := os.Stat(filepath.Join(dir, control.FileName)); err != nil {
		e, err := open(vfs.OS(), dir, envConfig())
		mustOK(t, err)
		start := time.Now()
		loadKeys(t, e, mustStore(t, e, "bench"), n)
		mustOK(t, e.Close())
		t.Logf("%d claves cargadas en %v", n, time.Since(start).Round(time.Millisecond))
	}

	steps := map[string][]time.Duration{}
	var order []string
	stopProfiles := startProfiles(t)
	for i := range opens {
		cfg := envConfig()
		var line strings.Builder
		last, opened := time.Now(), false
		cfg.openStep = func(step string) {
			if opened { // el checkpoint de Close no cuenta
				return
			}
			now := time.Now()
			if _, seen := steps[step]; !seen {
				order = append(order, step)
			}
			steps[step] = append(steps[step], now.Sub(last))
			line.WriteString(" " + step + "=" + now.Sub(last).Round(10*time.Microsecond).String())
			last = now
		}
		io := &ioCounter{FS: vfs.OS(), root: dir, files: map[string]*ioCount{}}
		start := time.Now()
		e, err := open(io, dir, cfg)
		mustOK(t, err)
		took := time.Since(start)
		opened = true
		if i == 0 {
			t.Logf("E/S de Open: %s", io.report())
		}
		steps["total"] = append(steps["total"], took)
		t0 := time.Now()
		mustOK(t, e.Close())
		t.Logf("Open %d: %v (Close %v):%s", i+1, took.Round(10*time.Microsecond),
			time.Since(t0).Round(10*time.Microsecond), line.String())
	}
	stopProfiles()
	for _, step := range append(order, "total") {
		times := slices.Clone(steps[step])
		slices.Sort(times)
		t.Logf("mediana %-22s %v", step, times[len(times)/2].Round(10*time.Microsecond))
	}
}

// ioCount es la E/S de Open en un archivo (o directorio).
type ioCount struct{ opens, reads, readBytes, writes, writeBytes, syncs, sizes, readDirs int }

// ioCounter es un vfs.FS que cuenta la E/S por archivo, con el nombre relativo a root.
type ioCounter struct {
	vfs.FS
	root  string
	mu    sync.Mutex
	files map[string]*ioCount
}

// count suma a la cuenta del archivo name.
func (c *ioCounter) count(name string, add func(*ioCount)) {
	rel, _ := filepath.Rel(c.root, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.files[rel] == nil {
		c.files[rel] = &ioCount{}
	}
	add(c.files[rel])
}

func (c *ioCounter) OpenFile(name string, flag int) (vfs.File, error) {
	c.count(name, func(n *ioCount) { n.opens++ })
	f, err := c.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return &ioFile{File: f, c: c, name: name}, nil
}

func (c *ioCounter) SyncDir(dir string) error {
	c.count(dir, func(n *ioCount) { n.syncs++ })
	return c.FS.SyncDir(dir)
}

func (c *ioCounter) ReadDir(dir string) ([]string, error) {
	c.count(dir, func(n *ioCount) { n.readDirs++ })
	return c.FS.ReadDir(dir)
}

// report da la cuenta de cada archivo, en orden.
func (c *ioCounter) report() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var b strings.Builder
	for _, name := range slices.Sorted(maps.Keys(c.files)) {
		n := c.files[name]
		fmt.Fprintf(&b, "\n  %-28s open %d, read %d (%d B), write %d (%d B), sync %d, size %d, readdir %d", name,
			n.opens, n.reads, n.readBytes, n.writes, n.writeBytes, n.syncs, n.sizes, n.readDirs)
	}
	return b.String()
}

// ioFile es un vfs.File que cuenta su E/S en un ioCounter.
type ioFile struct {
	vfs.File
	c    *ioCounter
	name string
}

func (f *ioFile) ReadAt(p []byte, off int64) (int, error) {
	f.c.count(f.name, func(n *ioCount) { n.reads++; n.readBytes += len(p) })
	return f.File.ReadAt(p, off)
}

func (f *ioFile) WriteAt(p []byte, off int64) (int, error) {
	f.c.count(f.name, func(n *ioCount) { n.writes++; n.writeBytes += len(p) })
	return f.File.WriteAt(p, off)
}

func (f *ioFile) Sync() error {
	f.c.count(f.name, func(n *ioCount) { n.syncs++ })
	return f.File.Sync()
}

func (f *ioFile) Size() (int64, error) {
	f.c.count(f.name, func(n *ioCount) { n.sizes++ })
	return f.File.Size()
}

// benchEnvInt lee un entero positivo de la variable name, o def.
func benchEnvInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}
