package buffer

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// arenaFreed dice si Close (o el último Release) ya liberó el arena.
func (p *pool) arenaFreed() bool {
	p.arenaMu.Lock()
	defer p.arenaMu.Unlock()
	return p.arena == nil
}

// TestArenaReopen: abrir, usar (cambiar, vaciar y leer del disco, con escritor de fondo) y cerrar varias cachés
// seguidas sobre los mismos archivos: cada Close libera su arena y la siguiente lee lo que la anterior escribió.
func TestArenaReopen(t *testing.T) {
	e := newOSEnv(t, 16)
	e.pool.Close()
	for round := range 5 {
		p := e.openPool(t, 16, time.Millisecond)
		e.pool = p
		block := e.newBlock(t, 1, page.XID(100+round))
		if err := p.FlushAll(nil); err != nil {
			t.Fatal(err)
		}
		for b := range block + 1 {
			if got := marker(t, p, tag1(b)); got != page.XID(100+int(b)) {
				t.Fatalf("ronda %d, bloque %d: %d", round, b, got)
			}
		}
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
		if !p.arenaFreed() {
			t.Fatalf("ronda %d: Close no liberó el arena", round)
		}
		if _, err := p.Pin(tag1(0), Normal); !errors.Is(err, ErrClosed) {
			t.Fatalf("Pin tras Close: %v", err)
		}
	}
}

// TestCloseWithPinned: Close con un búfer fijado no libera el arena (la página sigue legible por quien la tiene);
// nadie más fija ni vacía; el Release que deja todo sin pins lo libera.
func TestCloseWithPinned(t *testing.T) {
	e := newOSEnv(t, 4)
	e.newBlock(t, 1, 7)
	h, err := e.pool.Pin(tag1(0), Normal)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.pool.Close(); err != nil {
		t.Fatal(err)
	}
	if e.pool.arenaFreed() {
		t.Fatal("Close liberó el arena con un búfer fijado")
	}
	h.Lock(Share)
	if got := h.Page().PruneXID(); got != 7 {
		t.Fatalf("página fijada tras Close: %d", got)
	}
	if _, err := e.pool.Pin(tag1(0), Normal); !errors.Is(err, ErrClosed) {
		t.Fatalf("Pin tras Close: %v", err)
	}
	if err := e.pool.FlushAll(nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("FlushAll tras Close: %v", err)
	}
	h.Release()
	if !e.pool.arenaFreed() {
		t.Fatal("el último Release no liberó el arena")
	}
}

// TestCloseDuringUse: varias goroutines fijan, leen y cambian páginas (con reemplazo y escritor de fondo) mientras
// otra cierra la caché: ninguna toca el arena liberado (sería un SIGSEGV), los errores son ErrClosed y al terminar
// todas el arena quedó libre.
func TestCloseDuringUse(t *testing.T) {
	for round := range 10 {
		e := newOSEnv(t, 8)
		e.writeDisk(t, 32)
		p := e.openPool(t, 8, time.Millisecond)
		var wg sync.WaitGroup
		for g := range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				useUntilClosed(t, p, g)
			}()
		}
		time.Sleep(time.Duration(round) * time.Millisecond)
		if err := p.Close(); err != nil {
			t.Fatal(err)
		}
		wg.Wait()
		if !p.arenaFreed() {
			t.Fatalf("ronda %d: el arena sigue reservado", round)
		}
	}
}

/**
* useUntilClosed fija bloques al azar de los 32 del archivo 1, lee su marca y a veces la cambia, hasta que la caché
* devuelve ErrClosed.
* @param t *testing.T, p *pool, g int: la goroutine (siembra el recorrido)
**/
func useUntilClosed(t *testing.T, p *pool, g int) {
	for i := g; ; i += 7 {
		h, err := p.Pin(tag1(page.BlockNumber(i%32)), Normal)
		if errors.Is(err, ErrClosed) {
			return
		}
		if err != nil && !errors.Is(err, ErrNoBuffers) {
			t.Errorf("Pin: %v", err)
			return
		}
		if err != nil {
			continue
		}
		h.Lock(Exclusive)
		pg := h.Page()
		pg.SetPruneXID(pg.PruneXID())
		h.MarkDirty(pg.LSN())
		h.Release()
	}
}

// TestPinNewAfterClose: PinNew tras Close no extiende el archivo.
func TestPinNewAfterClose(t *testing.T) {
	e := newOSEnv(t, 4)
	e.pool.Close()
	if _, err := e.pool.PinNew(1, smgr.ForkMain); !errors.Is(err, ErrClosed) {
		t.Fatalf("PinNew: %v", err)
	}
	if n, _ := e.smgr.NBlocks(1, smgr.ForkMain); n != 0 {
		t.Fatalf("NBlocks = %d", n)
	}
}
