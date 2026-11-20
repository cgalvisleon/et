package fsm

import (
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// testPool es un doble simple de buffer.Pool sobre smgr para las pruebas: guarda en un mapa cada página que se fijó
// (sin expulsar nunca), verifica la suma al leer y cumple «WAL antes que la página» al escribir.
type testPool struct {
	sm smgr.Manager
	fl wal.Flusher // nil: no se puede escribir al disco (mientras se rehace dentro de wal.Open)

	mu   sync.Mutex
	bufs map[buffer.Tag]*testBuf
}

// testBuf es una página de la caché de prueba.
type testBuf struct {
	tag   buffer.Tag
	data  page.Page
	lock  sync.RWMutex
	dirty atomic.Bool
}

// testHandle es un pin sobre un testBuf; recuerda el modo de su bloqueo para soltarlo.
type testHandle struct {
	buf  *testBuf
	mode buffer.LockMode
}

var _ buffer.Pool = (*testPool)(nil)

func newTestPool(sm smgr.Manager, fl wal.Flusher) *testPool {
	return &testPool{sm: sm, fl: fl, bufs: map[buffer.Tag]*testBuf{}}
}

func (p *testPool) Pin(tag buffer.Tag, _ buffer.Strategy) (buffer.Handle, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if b, ok := p.bufs[tag]; ok {
		return &testHandle{buf: b}, nil
	}
	data := make(page.Page, page.Size)
	if err := p.sm.Read(tag.File, tag.Fork, tag.Block, data); err != nil {
		return nil, err
	}
	if err := page.Verify(data, tag.Block); err != nil {
		return nil, err
	}
	b := &testBuf{tag: tag, data: data}
	p.bufs[tag] = b
	return &testHandle{buf: b}, nil
}

// PinZero fija la página sin leerla, en ceros y bloqueada en exclusivo, como buffer.Pool.PinZero.
func (p *testPool) PinZero(tag buffer.Tag) (buffer.Handle, error) {
	p.mu.Lock()
	b, ok := p.bufs[tag]
	if !ok {
		b = &testBuf{tag: tag, data: make(page.Page, page.Size)}
		p.bufs[tag] = b
	}
	p.mu.Unlock()
	h := &testHandle{buf: b}
	h.Lock(buffer.Exclusive)
	clear(b.data)
	return h, nil
}

func (p *testPool) PinNew(file smgr.FileNo, fork smgr.Fork) (buffer.Handle, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n, err := p.sm.NBlocks(file, fork)
	if err != nil {
		return nil, err
	}
	data := make(page.Page, page.Size)
	if err := p.sm.Extend(file, fork, n, data); err != nil {
		return nil, err
	}
	b := &testBuf{tag: buffer.Tag{File: file, Fork: fork, Block: n}, data: data}
	p.bufs[b.tag] = b
	h := &testHandle{buf: b}
	h.Lock(buffer.Exclusive)
	return h, nil
}

// FlushAll escribe las páginas sucias, cada una tras el WAL durable hasta su pd_lsn, sobre una copia con su suma.
func (p *testPool) FlushAll(func(written, total int)) error {
	return p.flush(func(buffer.Tag) bool { return true })
}

// flush escribe las páginas sucias que elige keep.
func (p *testPool) flush(keep func(buffer.Tag) bool) error {
	p.mu.Lock()
	bufs := make([]*testBuf, 0, len(p.bufs))
	for _, b := range p.bufs {
		bufs = append(bufs, b)
	}
	p.mu.Unlock()
	for _, b := range bufs {
		if !keep(b.tag) || !b.dirty.Load() {
			continue
		}
		b.lock.RLock()
		b.dirty.Store(false)
		cp := append(page.Page(nil), b.data...)
		b.lock.RUnlock()
		if err := p.fl.WaitDurable(cp.LSN()); err != nil {
			return err
		}
		cp.SetChecksum(b.tag.Block)
		if err := p.sm.Write(b.tag.File, b.tag.Fork, b.tag.Block, cp); err != nil {
			return err
		}
	}
	return nil
}

func (p *testPool) DropFile(smgr.FileNo) {}
func (p *testPool) Stats() buffer.Stats  { return buffer.Stats{} }
func (p *testPool) Failed() error        { return nil }
func (p *testPool) Close() error         { return nil }

func (h *testHandle) Tag() buffer.Tag    { return h.buf.tag }
func (h *testHandle) Page() page.Page    { return h.buf.data }
func (h *testHandle) MarkDirty(page.LSN) { h.buf.dirty.Store(true) }
func (h *testHandle) MarkDirtyHint()     { h.buf.dirty.Store(true) }
func (h *testHandle) Release()           {}

func (h *testHandle) Lock(mode buffer.LockMode) {
	if mode == buffer.Exclusive {
		h.buf.lock.Lock()
	} else {
		h.buf.lock.RLock()
	}
	h.mode = mode
}

func (h *testHandle) Unlock() {
	if h.mode == buffer.Exclusive {
		h.buf.lock.Unlock()
	} else {
		h.buf.lock.RUnlock()
	}
	h.mode = 0
}
