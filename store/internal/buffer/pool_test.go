package buffer

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// TestPinNewFlushRead: una página nueva cambiada y vaciada se lee igual desde otra caché; Stats cuenta.
func TestPinNewFlushRead(t *testing.T) {
	e := newOSEnv(t, 8)
	for i := range 3 {
		e.newBlock(t, 1, page.XID(100+i))
	}
	if s := e.pool.Stats(); s.Dirty != 3 || s.Buffers != 8 || s.Misses != 3 {
		t.Fatalf("Stats = %+v", s)
	}
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if s := e.pool.Stats(); s.Dirty != 0 || s.Written != 3 {
		t.Fatalf("Stats tras FlushAll = %+v", s)
	}
	if e.flusher.Durable() != 3 {
		t.Fatalf("durable = %d, se esperaba 3", e.flusher.Durable())
	}

	other := e.openPool(t, 8, -1)
	for i := range 3 {
		if got := marker(t, other, tag1(page.BlockNumber(i))); got != page.XID(100+i) {
			t.Fatalf("bloque %d: %d", i, got)
		}
	}
	touch(t, other, tag1(0), Normal)
	if s := other.Stats(); s.Hits != 1 || s.Misses != 3 {
		t.Fatalf("Stats de la otra caché = %+v", s)
	}
}

// TestPinnedNeverEvicted: con todos los búferes fijados no hay reemplazo (ErrNoBuffers) y las páginas fijadas no se
// tocan; al soltar una, se reutiliza esa.
func TestPinnedNeverEvicted(t *testing.T) {
	e := newOSEnv(t, 4)
	e.writeDisk(t, 6)
	var held []Handle
	for b := range 4 {
		h, err := e.pool.Pin(tag1(page.BlockNumber(b)), Normal)
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, h)
	}
	if _, err := e.pool.Pin(tag1(4), Normal); !errors.Is(err, ErrNoBuffers) {
		t.Fatalf("Pin con todo fijado: %v", err)
	}

	held[0].Release()
	touch(t, e.pool, tag1(4), Normal)
	if e.pool.cached(tag1(0)) || !e.pool.cached(tag1(4)) {
		t.Fatal("se esperaba que el bloque 4 reemplace al 0, el único sin pin")
	}
	for b, h := range held[1:] {
		h.Lock(Share)
		if got := h.Page().PruneXID(); got != page.XID(b+2) || h.Tag() != tag1(page.BlockNumber(b+1)) {
			t.Fatalf("bloque fijado %d cambió: %d %s", b+1, got, h.Tag())
		}
		h.Release() // suelta también el bloqueo
		h.Release() // idempotente
	}
}

// TestClockSweep: una página muy usada sobrevive a un recorrido que trae más páginas que búferes; las usadas una vez
// se van.
func TestClockSweep(t *testing.T) {
	e := newOSEnv(t, 4)
	e.writeDisk(t, 8)
	for range MaxUsageCount {
		touch(t, e.pool, tag1(0), Normal)
	}
	for b := 1; b < 8; b++ {
		touch(t, e.pool, tag1(page.BlockNumber(b)), Normal)
	}
	if !e.pool.cached(tag1(0)) {
		t.Fatal("el bloque 0 (uso 5) fue expulsado")
	}
	if e.pool.cached(tag1(1)) {
		t.Fatal("el bloque 1 (uso 1) sigue en la caché")
	}
	misses := e.pool.Stats().Misses
	touch(t, e.pool, tag1(0), Normal)
	if e.pool.Stats().Misses != misses {
		t.Fatal("el bloque 0 no fue un acierto")
	}
}

// TestRing: un recorrido grande con Ring usa solo los búferes del anillo y no expulsa las páginas calientes; el
// mismo recorrido con Normal sí.
func TestRing(t *testing.T) {
	const buffers = 64
	ringN := ringLen(buffers)
	hot := buffers - ringN
	e := newOSEnv(t, buffers)
	e.writeDisk(t, hot+100)
	for range 2 {
		for b := range hot {
			touch(t, e.pool, tag1(page.BlockNumber(b)), Normal)
		}
	}
	for b := hot; b < hot+100; b++ {
		touch(t, e.pool, tag1(page.BlockNumber(b)), Ring)
	}

	inRing := 0
	for b := hot; b < hot+100; b++ {
		if e.pool.cached(tag1(page.BlockNumber(b))) {
			inRing++
		}
	}
	if inRing > ringN {
		t.Fatalf("el recorrido ocupa %d búferes, el anillo tiene %d", inRing, ringN)
	}
	misses := e.pool.Stats().Misses
	for b := range hot {
		touch(t, e.pool, tag1(page.BlockNumber(b)), Normal)
	}
	if got := e.pool.Stats().Misses - misses; got != 0 {
		t.Fatalf("%d páginas calientes expulsadas por el anillo", got)
	}

	// Con Normal el mismo recorrido vacía la caché.
	for b := hot; b < hot+100; b++ {
		touch(t, e.pool, tag1(page.BlockNumber(b)), Normal)
	}
	evicted := 0
	for b := range hot {
		if !e.pool.cached(tag1(page.BlockNumber(b))) {
			evicted++
		}
	}
	if evicted == 0 {
		t.Fatal("con Normal se esperaba expulsar páginas calientes")
	}
}

// TestConcurrentSameBlockLoad: muchas goroutines fijan a la vez una página que no está: se lee una sola vez.
func TestConcurrentSameBlockLoad(t *testing.T) {
	e := newOSEnv(t, 8)
	e.writeDisk(t, 1)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if got := marker(t, e.pool, tag1(0)); got != 1 {
				t.Errorf("marker = %d", got)
			}
		})
	}
	wg.Wait()
	if s := e.pool.Stats(); s.Misses != 1 || s.Hits != 15 {
		t.Fatalf("Stats = %+v", s)
	}
}

// TestConcurrentReadWrite: goroutines que cambian (bloqueo exclusivo) y leen (compartido) el mismo bloque y otros,
// con menos búferes que bloques (reemplazo con páginas sucias), mientras otra vacía y el escritor de fondo escribe.
// Al final cada bloque tiene tantos incrementos como se hicieron, leído del disco, y ninguna página llegó antes que
// su WAL.
func TestConcurrentReadWrite(t *testing.T) {
	const blocks, workers, ops = 32, 8, 300
	e := newOSEnv(t, 8)
	for range blocks {
		e.newBlock(t, 1, 0)
	}
	var counts [blocks]atomic.Int64

	stop := make(chan struct{})
	var bg sync.WaitGroup
	bg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if err := e.pool.FlushAll(nil); err != nil {
				t.Error(err)
				return
			}
			if err := e.pool.bgWriteRound(); err != nil {
				t.Error(err)
				return
			}
		}
	})

	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			rng := rand.New(rand.NewPCG(uint64(w), 1))
			for range ops {
				b := page.BlockNumber(rng.IntN(blocks))
				if rng.IntN(2) == 0 {
					b = 0 // la mitad sobre el mismo bloque
				}
				h, err := e.pool.Pin(tag1(b), Normal)
				if errors.Is(err, ErrNoBuffers) {
					continue // 8 trabajadores + el vaciado pueden fijar los 8 búferes
				}
				if err != nil {
					t.Error(err)
					return
				}
				if rng.IntN(2) == 0 {
					h.Lock(Exclusive)
					p := h.Page()
					p.SetPruneXID(p.PruneXID() + 1)
					lsn := e.flusher.nextLSN()
					p.SetLSN(lsn)
					h.MarkDirty(lsn)
					counts[b].Add(1)
				} else {
					h.Lock(Share)
					if h.Page().IsNew() {
						t.Errorf("bloque %d en ceros", b)
					}
				}
				h.Release()
			}
		})
	}
	wg.Wait()
	close(stop)
	bg.Wait()

	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	other := e.openPool(t, blocks, -1)
	for b := range blocks {
		if got, want := marker(t, other, tag1(page.BlockNumber(b))), counts[b].Load(); int64(got) != want {
			t.Fatalf("bloque %d: %d incrementos en disco, se hicieron %d", b, got, want)
		}
	}
}

// TestConcurrentPinNew: PinNew desde varias goroutines da bloques distintos y seguidos.
func TestConcurrentPinNew(t *testing.T) {
	e := newOSEnv(t, 16)
	var wg sync.WaitGroup
	var seen sync.Map
	for w := range 8 {
		wg.Go(func() {
			for i := range 10 {
				h, err := e.pool.PinNew(1, smgr.ForkMain)
				if err != nil {
					t.Error(err)
					return
				}
				if !h.Page().IsNew() {
					t.Error("PinNew no dio una página en ceros")
				}
				if _, dup := seen.LoadOrStore(h.Tag().Block, w*100+i); dup {
					t.Errorf("bloque %d repetido", h.Tag().Block)
				}
				h.Release()
			}
		})
	}
	wg.Wait()
	if n, _ := e.smgr.NBlocks(1, smgr.ForkMain); n != 80 {
		t.Fatalf("NBlocks = %d", n)
	}
}

// TestWALBeforePage: al expulsar páginas sucias con pd_lsn por delante de lo durable, la caché espera al WAL antes
// de escribir (checkSmgr revisa cada escritura), y una página limpia o ya durable no espera.
func TestWALBeforePage(t *testing.T) {
	e := newOSEnv(t, 4)
	for range 12 {
		e.newBlock(t, 1, 7) // con 4 búferes, 8 de estas se expulsan sucias
	}
	if e.smgr.writesOf(1) < 8 {
		t.Fatalf("se esperaban al menos 8 escrituras por reemplazo, hubo %d", e.smgr.writesOf(1))
	}
	if e.flusher.waits.Load() == 0 {
		t.Fatal("ninguna escritura esperó al WAL")
	}
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if e.flusher.Durable() != 12 {
		t.Fatalf("durable = %d", e.flusher.Durable())
	}

	// Ya todo durable: sucias de nuevo con el mismo lsn, no hace falta esperar.
	waits := e.flusher.waits.Load()
	h, err := e.pool.Pin(tag1(11), Normal)
	if err != nil {
		t.Fatal(err)
	}
	h.Lock(Exclusive)
	h.MarkDirty(h.Page().LSN())
	h.Release()
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if e.flusher.waits.Load() != waits {
		t.Fatal("esperó al WAL con la página ya durable")
	}
}

// TestChecksumOnRead: una página con la suma mala da un error claro (page.ErrChecksum, con el bloque), no un pánico,
// y no deja el búfer ocupado; las demás se siguen leyendo. Un bloque en ceros es una página nueva válida.
func TestChecksumOnRead(t *testing.T) {
	e := newOSEnv(t, 4)
	e.writeDisk(t, 2)
	buf := make(page.Page, page.Size)
	if err := e.smgr.Manager.Read(1, smgr.ForkMain, 1, buf); err != nil {
		t.Fatal(err)
	}
	buf[100] ^= 0xFF
	if err := e.smgr.Manager.Write(1, smgr.ForkMain, 1, buf); err != nil {
		t.Fatal(err)
	}
	if err := e.smgr.Extend(1, smgr.ForkMain, 2, zeroPage); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		_, err := e.pool.Pin(tag1(1), Normal)
		if !errors.Is(err, page.ErrChecksum) {
			t.Fatalf("Pin de la página corrupta: %v", err)
		}
		t.Log(err)
	}
	if e.pool.cached(tag1(1)) {
		t.Fatal("la página corrupta quedó en el mapa")
	}
	if got := marker(t, e.pool, tag1(0)); got != 1 {
		t.Fatalf("bloque 0: %d", got)
	}
	h, err := e.pool.Pin(tag1(2), Normal)
	if err != nil {
		t.Fatal(err)
	}
	if !h.Page().IsNew() {
		t.Fatal("el bloque en ceros no es una página nueva")
	}
	h.Release()

	// Un bloque más allá del final es un error de smgr, también sin pánico.
	if _, err := e.pool.Pin(tag1(99), Normal); !errors.Is(err, smgr.ErrBlockOutOfRange) {
		t.Fatalf("Pin fuera de rango: %v", err)
	}
	// Una página nueva sucia sin iniciar se guarda en ceros (sin suma) y se vuelve a leer.
	h, err = e.pool.PinNew(1, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}
	h.MarkDirty(0)
	h.Release()
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	touch(t, e.openPool(t, 4, -1), tag1(3), Normal)
}

// TestHintsUnderShare: lectores con bloqueo compartido ponen marcas con un OR atómico y MarkDirtyHint mientras otra
// goroutine vacía la caché; -race no ve carreras y ninguna marca se pierde.
func TestHintsUnderShare(t *testing.T) {
	e := newOSEnv(t, 4)
	e.newBlock(t, 1, 1)
	const word = 4096 // en el hueco libre de la página: ni encabezado ni ítems
	var wg sync.WaitGroup
	stop := make(chan struct{})
	var flusherWG sync.WaitGroup
	flusherWG.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if err := e.pool.FlushAll(nil); err != nil {
				t.Error(err)
				return
			}
		}
	})
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				h, err := e.pool.Pin(tag1(0), Normal)
				if err != nil {
					t.Error(err)
					return
				}
				h.Lock(Share)
				p := h.Page()
				atomic.OrUint32((*uint32)(unsafe.Pointer(&p[word])), 1<<((g*4+i)%32))
				h.MarkDirtyHint()
				h.Release()
			}
		})
	}
	wg.Wait()
	close(stop)
	flusherWG.Wait()
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}

	other := e.openPool(t, 4, -1)
	h, err := other.Pin(tag1(0), Normal)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	h.Lock(Share)
	if got := le32(h.Page()[word:]); got != 0xFFFFFFFF {
		t.Fatalf("marcas en disco = %#x, se esperaban todas", got)
	}
}

// le32 lee un uint32 little-endian.
func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// TestCheckpointSurvivesCrash: lo que escribe FlushAll, seguido de smgr.SyncAll, sobrevive a un corte en los tres
// modos de crashfs; pace se llama una vez por página sucia.
func TestCheckpointSurvivesCrash(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"LoseUnsynced": crashfs.LoseUnsynced, "ReorderUnsynced": crashfs.ReorderUnsynced, "TornWrites": crashfs.TornWrites,
	}
	for name, mode := range modes {
		for seed := range int64(3) {
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				fs := crashfs.New()
				e := newEnv(t, fs, "/db", 8)
				for i := range 20 {
					e.newBlock(t, 1, page.XID(1000+i))
				}
				dirty := int(e.pool.Stats().Dirty)
				calls, last := 0, 0
				err := e.pool.FlushAll(func(written, total int) {
					calls++
					if total != dirty || written != last+1 {
						t.Errorf("pace(%d, %d) tras %d, con %d sucias", written, total, last, dirty)
					}
					last = written
				})
				if err != nil {
					t.Fatal(err)
				}
				if calls != dirty {
					t.Fatalf("pace se llamó %d veces, había %d sucias", calls, dirty)
				}
				if err := e.smgr.SyncAll(); err != nil {
					t.Fatal(err)
				}
				// Un cambio posterior sin checkpoint puede perderse, pero no rompe lo anterior.
				e.newBlock(t, 1, 5000)

				fs.Crash(mode, seed)
				after := newEnv(t, fs, "/db", 8)
				for i := range 20 {
					if got := marker(t, after.pool, tag1(page.BlockNumber(i))); got != page.XID(1000+i) {
						t.Fatalf("bloque %d tras el corte: %d", i, got)
					}
				}
			})
		}
	}
}

// TestWriteFailure: un fallo de escritura deja la caché inutilizable: todo devuelve ErrFailed con la causa y no se
// escribe nada más.
func TestWriteFailure(t *testing.T) {
	fs := crashfs.New()
	e := newEnv(t, fs, "/db", 4)
	e.newBlock(t, 1, 1)
	e.newBlock(t, 1, 2)
	fs.FailAfter(0)
	err := e.pool.FlushAll(nil)
	if !errors.Is(err, ErrFailed) || !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("FlushAll: %v", err)
	}
	fs.FailAfter(-1)
	if _, err := e.pool.Pin(tag1(0), Normal); !errors.Is(err, ErrFailed) {
		t.Fatalf("Pin tras el fallo: %v", err)
	}
	if _, err := e.pool.PinNew(1, smgr.ForkMain); !errors.Is(err, ErrFailed) {
		t.Fatalf("PinNew tras el fallo: %v", err)
	}
	writes := e.smgr.writesOf(1)
	if err := e.pool.FlushAll(nil); !errors.Is(err, ErrFailed) {
		t.Fatalf("FlushAll tras el fallo: %v", err)
	}
	if e.smgr.writesOf(1) != writes {
		t.Fatal("escribió tras el fallo")
	}
}

// TestWaitDurableFailure: si el WAL no puede hacer durable el pd_lsn, la página no se escribe y la caché falla.
func TestWaitDurableFailure(t *testing.T) {
	e := newOSEnv(t, 4)
	e.newBlock(t, 1, 1)
	walErr := errors.New("wal: fsync falló")
	e.flusher.fail(walErr)
	err := e.pool.FlushAll(nil)
	if !errors.Is(err, ErrFailed) || !errors.Is(err, walErr) {
		t.Fatalf("FlushAll: %v", err)
	}
	if e.smgr.writesOf(1) != 0 {
		t.Fatal("escribió la página sin el WAL durable")
	}
	if _, err := e.pool.Pin(tag1(0), Normal); !errors.Is(err, ErrFailed) {
		t.Fatalf("Pin tras el fallo: %v", err)
	}
}

// TestDropFile: las páginas de un store borrado salen de la caché sin escribirse.
func TestDropFile(t *testing.T) {
	e := newOSEnv(t, 8)
	if err := e.smgr.Create(2, smgr.ForkMain); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		e.newBlock(t, 2, 9)
	}
	e.newBlock(t, 1, 1)
	e.pool.DropFile(2)
	if s := e.pool.Stats(); s.Dirty != 1 {
		t.Fatalf("Stats tras DropFile = %+v", s)
	}
	if err := e.pool.FlushAll(nil); err != nil {
		t.Fatal(err)
	}
	if e.smgr.writesOf(2) != 0 || e.smgr.writesOf(1) != 1 {
		t.Fatalf("escrituras: archivo 2 = %d, archivo 1 = %d", e.smgr.writesOf(2), e.smgr.writesOf(1))
	}
	if err := e.smgr.Unlink(2); err != nil {
		t.Fatal(err)
	}
	if got := marker(t, e.pool, tag1(0)); got != 1 {
		t.Fatalf("bloque del archivo 1: %d", got)
	}
}

// TestBgWriter: el escritor de fondo limpia las páginas sucias que el reemplazo usaría ya (sin pins, uso 0) y deja
// las fijadas.
func TestBgWriter(t *testing.T) {
	e := newOSEnv(t, 8)
	bg := e.openPool(t, 8, 5*time.Millisecond)
	e.pool = bg
	for i := range 4 {
		e.newBlock(t, 1, page.XID(i+1))
	}
	pinned, err := bg.Pin(tag1(0), Normal)
	if err != nil {
		t.Fatal(err)
	}
	for i := range bg.descs {
		d := &bg.descs[i]
		d.mu.Lock()
		d.usage = 0 // como tras una vuelta del clock-sweep
		d.mu.Unlock()
	}
	deadline := time.Now().Add(5 * time.Second)
	for bg.Stats().Dirty > 1 {
		if time.Now().After(deadline) {
			t.Fatalf("el escritor de fondo no limpió: %+v", bg.Stats())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if e.smgr.writesOf(1) != 3 {
		t.Fatalf("escrituras = %d, se esperaban 3 (la fijada no)", e.smgr.writesOf(1))
	}
	pinned.Release()
}

// TestClose: tras Close, Pin y PinNew devuelven ErrClosed; Close es idempotente.
func TestClose(t *testing.T) {
	e := newOSEnv(t, 4)
	e.writeDisk(t, 1)
	if err := e.pool.Close(); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Pin(tag1(0), Normal); !errors.Is(err, ErrClosed) {
		t.Fatalf("Pin: %v", err)
	}
	if _, err := e.pool.PinNew(1, smgr.ForkMain); !errors.Is(err, ErrClosed) {
		t.Fatalf("PinNew: %v", err)
	}
}

// TestOpenConfig: Open rechaza una configuración incompleta y EnvBuffers da 128 MB por omisión.
func TestOpenConfig(t *testing.T) {
	if _, err := Open(Config{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("Open vacío: %v", err)
	}
	if got := EnvBuffers(); got != DefaultCacheSize/page.Size {
		t.Fatalf("EnvBuffers = %d", got)
	}
	if ringLen(16384) != 32 || ringLen(64) != 8 || ringLen(4) != 1 {
		t.Fatal("ringLen")
	}
}

// stallSmgr detiene la primera escritura del bloque stall hasta que se cierre release, y avisa en started cuando
// empieza: así una escritura queda en curso (copiada, todavía sin llegar al disco) el tiempo que la prueba quiera.
type stallSmgr struct {
	smgr.Manager
	stall   page.BlockNumber
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (m *stallSmgr) Write(file smgr.FileNo, fork smgr.Fork, block page.BlockNumber, buf page.Page) error {
	if block == m.stall {
		m.once.Do(func() {
			close(m.started)
			<-m.release
		})
	}
	return m.Manager.Write(file, fork, block, buf)
}

// TestFlushAllWaitsWriteInFlight: una página sucia que otro (el reemplazo, el escritor de fondo) está escribiendo
// cuando empieza un checkpoint tiene que estar en el disco cuando FlushAll vuelve. Si se marcara limpia al empezar
// a escribirla, FlushAll no la vería, el fsync del checkpoint (SyncAll) llegaría antes que la escritura y, tras un
// corte, la página volvería a su versión vieja aunque el punto de rehacer ya pasó su cambio.
func TestFlushAllWaitsWriteInFlight(t *testing.T) {
	fs := crashfs.New()
	e := newEnv(t, fs, "/db", 8)
	p := e.openPool(t, 8, -1) // sin escritor de fondo: nadie más toca p.smgr
	e.pool = p
	blk := e.newBlock(t, 1, 777)
	stall := &stallSmgr{Manager: e.smgr, stall: blk, started: make(chan struct{}), release: make(chan struct{})}
	p.smgr = stall

	// Una escritura del bloque queda en curso, como la de un reemplazo o del escritor de fondo.
	d := &p.descs[mustLookup(t, p, tag1(blk))]
	if !p.pinIfTag(d, tag1(blk)) {
		t.Fatal("pinIfTag")
	}
	writing := make(chan error, 1)
	go func() {
		writing <- p.flush(d)
		p.unpin(d)
	}()
	<-stall.started

	// El checkpoint: FlushAll y fsync. La escritura en curso se suelta si FlushAll se queda esperándola.
	flushed := make(chan error, 1)
	go func() { flushed <- p.FlushAll(nil) }()
	select {
	case err := <-flushed:
		mustNil(t, err)
		mustNil(t, e.smgr.SyncAll())
		close(stall.release)
	case <-time.After(100 * time.Millisecond):
		close(stall.release)
		mustNil(t, <-flushed)
		mustNil(t, e.smgr.SyncAll())
	}
	mustNil(t, <-writing)

	fs.Crash(crashfs.LoseUnsynced, 1)
	after := newEnv(t, fs, "/db", 8)
	if got := marker(t, after.pool, tag1(blk)); got != 777 {
		t.Fatalf("tras el checkpoint y el corte, el bloque %d tiene %d: la escritura en curso no entró en el checkpoint", blk, got)
	}
}

// mustLookup devuelve el búfer que tiene tag en la caché.
func mustLookup(t *testing.T, p *pool, tag Tag) int32 {
	t.Helper()
	id, ok := p.lookup(tag)
	if !ok {
		t.Fatalf("%s no está en la caché", tag)
	}
	return id
}

// mustNil falla la prueba si err no es nil.
func mustNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
