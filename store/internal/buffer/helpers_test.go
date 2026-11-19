package buffer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// fakeFlusher es un WAL de mentira: lo durable solo avanza con WaitDurable, así toda página con pd_lsn mayor obliga
// a esperar. next da lsn crecientes para las páginas.
type fakeFlusher struct {
	mu      sync.Mutex
	durable page.LSN
	err     error // si no es nil, WaitDurable falla con él
	next    atomic.Uint64
	waits   atomic.Int64
}

func (f *fakeFlusher) Durable() page.LSN {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.durable
}

func (f *fakeFlusher) WaitDurable(lsn page.LSN) error {
	f.waits.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.durable = max(f.durable, lsn)
	return nil
}

func (f *fakeFlusher) fail(err error) {
	f.mu.Lock()
	f.err = err
	f.mu.Unlock()
}

// nextLSN da el lsn del próximo «registro».
func (f *fakeFlusher) nextLSN() page.LSN { return page.LSN(f.next.Add(1)) }

// checkSmgr envuelve un smgr.Manager y revisa cada página que llega al disco: pd_lsn ≤ lo durable del WAL («WAL
// antes que la página») y suma válida. Cuenta escrituras por archivo.
type checkSmgr struct {
	smgr.Manager
	flusher *fakeFlusher

	mu         sync.Mutex
	violations []string
	writes     map[smgr.FileNo]int
}

func (m *checkSmgr) Write(file smgr.FileNo, fork smgr.Fork, block page.BlockNumber, buf page.Page) error {
	m.mu.Lock()
	if lsn, durable := buf.LSN(), m.flusher.Durable(); lsn > durable {
		m.violations = append(m.violations, fmt.Sprintf("%d/%d: pd_lsn %d > durable %d", file, block, lsn, durable))
	}
	if err := page.Verify(buf, block); err != nil {
		m.violations = append(m.violations, fmt.Sprintf("%d/%d: %v", file, block, err))
	}
	m.writes[file]++
	m.mu.Unlock()
	return m.Manager.Write(file, fork, block, buf)
}

func (m *checkSmgr) writesOf(file smgr.FileNo) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writes[file]
}

// check falla la prueba si alguna página llegó mal al disco.
func (m *checkSmgr) check(t *testing.T) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.violations {
		t.Error(v)
	}
}

// env es una caché de prueba sobre un smgr real, con el fork principal del archivo 1 creado y sin escritor de fondo.
type env struct {
	fs      vfs.FS
	dir     string
	smgr    *checkSmgr
	flusher *fakeFlusher
	pool    *pool
}

// newEnv abre smgr y la caché en dir sobre fs.
func newEnv(t *testing.T, fs vfs.FS, dir string, buffers int) *env {
	t.Helper()
	m, err := smgr.Open(fs, dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Create(1, smgr.ForkMain); err != nil {
		t.Fatal(err)
	}
	f := &fakeFlusher{}
	cm := &checkSmgr{Manager: m, flusher: f, writes: map[smgr.FileNo]int{}}
	e := &env{fs: fs, dir: dir, smgr: cm, flusher: f}
	e.pool = e.openPool(t, buffers, -1)
	t.Cleanup(func() {
		cm.check(t)
		m.Close()
	})
	return e
}

// newOSEnv es newEnv sobre el disco real en un directorio temporal.
func newOSEnv(t *testing.T, buffers int) *env {
	return newEnv(t, vfs.OS(), t.TempDir(), buffers)
}

// openPool abre otra caché (vacía) sobre el mismo smgr: lo que lee viene del disco.
func (e *env) openPool(t *testing.T, buffers int, delay time.Duration) *pool {
	t.Helper()
	p, err := Open(Config{Smgr: e.smgr, Flusher: e.flusher, Buffers: buffers, BgWriterDelay: delay})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p.(*pool)
}

// tag1 es el bloque b del fork principal del archivo 1.
func tag1(b page.BlockNumber) Tag { return Tag{File: 1, Fork: smgr.ForkMain, Block: b} }

// newBlock agrega una página iniciada al archivo file con marker en pd_prune_xid y un lsn nuevo, sucia.
func (e *env) newBlock(t *testing.T, file smgr.FileNo, marker page.XID) page.BlockNumber {
	t.Helper()
	h, err := e.pool.PinNew(file, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	p := h.Page()
	if err := p.Init(page.OpaqueSize); err != nil {
		t.Fatal(err)
	}
	p.SetPruneXID(marker)
	lsn := e.flusher.nextLSN()
	p.SetLSN(lsn)
	h.MarkDirty(lsn)
	return h.Tag().Block
}

// writeDisk agrega n páginas iniciadas, con su suma y marker = bloque + 1, directo con smgr (sin la caché).
func (e *env) writeDisk(t *testing.T, n int) {
	t.Helper()
	buf := make(page.Page, page.Size)
	for range n {
		block, err := e.smgr.NBlocks(1, smgr.ForkMain)
		if err != nil {
			t.Fatal(err)
		}
		buf.Init(page.OpaqueSize)
		buf.SetPruneXID(page.XID(block) + 1)
		buf.SetChecksum(block)
		if err := e.smgr.Extend(1, smgr.ForkMain, block, buf); err != nil {
			t.Fatal(err)
		}
	}
}

// marker lee pd_prune_xid de un bloque con bloqueo compartido.
func marker(t *testing.T, p Pool, tag Tag) page.XID {
	t.Helper()
	h, err := p.Pin(tag, Normal)
	if err != nil {
		t.Fatalf("Pin(%s): %v", tag, err)
	}
	defer h.Release()
	h.Lock(Share)
	return h.Page().PruneXID()
}

// touch fija y suelta un bloque.
func touch(t *testing.T, p Pool, tag Tag, strategy Strategy) {
	t.Helper()
	h, err := p.Pin(tag, strategy)
	if err != nil {
		t.Fatalf("Pin(%s): %v", tag, err)
	}
	h.Release()
}

// cached dice si un bloque está en la caché, sin fijarlo.
func (p *pool) cached(tag Tag) bool {
	_, ok := p.lookup(tag)
	return ok
}
