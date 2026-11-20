package store

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// testDir es el directorio del motor en el disco simulado.
const testDir = "/db"

// errKilled es el error de toda operación de un motor cuyo proceso «murió» en un corte.
var errKilled = errors.New("test: process killed by a crash")

/**
* testConfig es una configuración chica para pruebas: segmentos de 1 MB, caché de 64 páginas, checkpoints solo a mano
* salvo que la prueba los pida.
* @return config
**/
func testConfig() config {
	return config{
		syncOnWrite:       true,
		checkpointTimeout: time.Hour,
		maxWALSize:        1 << 40,
		lockTimeout:       5 * time.Second,
		txTimeout:         10 * time.Minute,
		wal:               wal.Options{SegmentSize: 1 << 20, Spares: 2, Preallocate: true, Buffer: 64 << 10},
		buffers:           64,
		checkEvery:        10 * time.Millisecond,
	}
}

// procFS envuelve el disco simulado con la vida de un «proceso»: kill corta el disco y desde ese momento toda
// operación de ese proceso falla, incluidas las que abren archivos nuevos. Así un motor viejo que sigue corriendo
// tras el corte (checkpointer, Close) no puede tocar el disco que verá el motor nuevo. mu evita que una operación
// quede a caballo del corte.
type procFS struct {
	base crashfs.FS
	mu   sync.RWMutex
	dead bool

	// budget, si es mayor que 0, son las operaciones que quedan antes de un corte en killMode (killAfter).
	budget   atomic.Int64
	killMode crashfs.Mode
	killSeed int64
}

/**
* killAfter arma un corte en mode cuando el proceso haga n operaciones más sobre el disco (en medio de lo que esté
* haciendo). Se llama antes de empezar lo que se quiere cortar, sin operaciones en curso.
* @param n int64, mode crashfs.Mode, seed int64
**/
func (p *procFS) killAfter(n int64, mode crashfs.Mode, seed int64) {
	p.killMode, p.killSeed = mode, seed
	p.budget.Store(n)
}

/**
* newProc da un proceso nuevo sobre el disco base.
* @param base crashfs.FS
* @return *procFS
**/
func newProc(base crashfs.FS) *procFS {
	return &procFS{base: base}
}

/**
* kill simula el corte: el disco pierde lo no durable según mode y el proceso muere.
* @param mode crashfs.Mode, seed int64
**/
func (p *procFS) kill(mode crashfs.Mode, seed int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dead = true
	p.base.Crash(mode, seed)
}

/**
* do corre op si el proceso vive.
* @param op func() error
* @return error
**/
func (p *procFS) do(op func() error) error {
	if p.budget.Load() > 0 && p.budget.Add(-1) == 0 {
		p.kill(p.killMode, p.killSeed)
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.dead {
		return errKilled
	}
	return op()
}

func (p *procFS) OpenFile(name string, flag int) (vfs.File, error) {
	var f vfs.File
	err := p.do(func() (err error) { f, err = p.base.OpenFile(name, flag); return })
	if err != nil {
		return nil, err
	}
	return &procFile{p: p, f: f}, nil
}

func (p *procFS) Remove(name string) error { return p.do(func() error { return p.base.Remove(name) }) }
func (p *procFS) Rename(a, b string) error { return p.do(func() error { return p.base.Rename(a, b) }) }
func (p *procFS) MkdirAll(dir string) error {
	return p.do(func() error { return p.base.MkdirAll(dir) })
}
func (p *procFS) SyncDir(dir string) error { return p.do(func() error { return p.base.SyncDir(dir) }) }
func (p *procFS) ReadDir(dir string) ([]string, error) {
	var names []string
	err := p.do(func() (err error) { names, err = p.base.ReadDir(dir); return })
	return names, err
}
func (p *procFS) Lock(name string) (io.Closer, error) {
	var c io.Closer
	err := p.do(func() (err error) { c, err = p.base.Lock(name); return })
	if err != nil {
		return nil, err
	}
	return &procCloser{p: p, c: c}, nil
}

// procFile es un archivo de un proceso.
type procFile struct {
	p *procFS
	f vfs.File
}

func (f *procFile) ReadAt(b []byte, off int64) (n int, err error) {
	err = f.p.do(func() error { n, err = f.f.ReadAt(b, off); return err })
	return n, err
}
func (f *procFile) WriteAt(b []byte, off int64) (n int, err error) {
	err = f.p.do(func() error { n, err = f.f.WriteAt(b, off); return err })
	return n, err
}
func (f *procFile) Close() error           { return f.p.do(f.f.Close) }
func (f *procFile) Sync() error            { return f.p.do(f.f.Sync) }
func (f *procFile) Truncate(n int64) error { return f.p.do(func() error { return f.f.Truncate(n) }) }
func (f *procFile) Size() (n int64, err error) {
	err = f.p.do(func() error { n, err = f.f.Size(); return err })
	return n, err
}

// procCloser es el LOCK de un proceso: muerto, ya lo soltó el corte.
type procCloser struct {
	p *procFS
	c io.Closer
}

func (c *procCloser) Close() error { return c.p.do(c.c.Close) }

/**
* openTest abre un motor sobre fsys con cfg y falla la prueba si no puede.
* @param t testing.TB, fsys vfs.FS, cfg config
* @return *Engine
**/
func openTest(t testing.TB, fsys vfs.FS, cfg config) *Engine {
	t.Helper()
	e, err := open(fsys, testDir, cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return e
}

/**
* mustStore abre un store y falla la prueba si no puede.
* @param t testing.TB, e *Engine, name string
* @return *Store
**/
func mustStore(t testing.TB, e *Engine, name string) *Store {
	t.Helper()
	s, err := e.Store(name)
	if err != nil {
		t.Fatalf("store %q: %v", name, err)
	}
	return s
}

/**
* verifyTrees corre btree.Verify sobre cada store del motor.
* @param t testing.TB, e *Engine
**/
func verifyTrees(t testing.TB, e *Engine) {
	t.Helper()
	for _, name := range e.Stores() {
		s := mustStore(t, e, name)
		if _, err := s.tree.Verify(); err != nil {
			t.Fatalf("verify %q: %v", name, err)
		}
	}
}
