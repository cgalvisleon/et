package buffer

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// pool implementa Pool.
type pool struct {
	smgr    smgr.Manager
	flusher wal.Flusher

	descs []desc
	parts [NumPartitions]partition
	hand  atomic.Uint64 // manecilla del clock-sweep: el próximo búfer a mirar es hand % len(descs)
	ring  ring

	// extendMu serializa PinNew: NBlocks + Extend de un fork tienen que ir juntos (a confirmar: uno por motor, no
	// por fork, por simple).
	extendMu sync.Mutex

	scratch sync.Pool // copias de páginas para escribir; acotadas por las escrituras concurrentes

	failed atomic.Pointer[error] // primer fallo de escritura o de WaitDurable
	closed atomic.Bool           // tras Close nadie fija un búfer más (pinLocked)

	// arena es la memoria de todos los búferes (mapArena, fuera del heap de Go). arenaMu serializa su liberación:
	// freeArena la libera una sola vez, cuando la caché está cerrada y ningún búfer está fijado.
	arenaMu sync.Mutex
	arena   page.Page

	stop chan struct{}
	wg   sync.WaitGroup

	hits, misses, dirty, written atomic.Int64
	used                         atomic.Int64 // búferes no vacíos: sube al dar uno vacío a una página, baja al vaciarlo
}

/**
* Open crea la caché con cfg.Buffers búferes en un solo arena fuera del heap de Go (toda la memoria se reserva acá y
* no crece) y arranca el escritor de fondo.
* @param cfg Config
* @return Pool, error: ErrConfig si falta Smgr o Flusher o Buffers < 0; el de mapArena si no hay memoria
**/
func Open(cfg Config) (Pool, error) {
	if cfg.Smgr == nil || cfg.Flusher == nil || cfg.Buffers < 0 {
		return nil, ErrConfig
	}
	n := cfg.Buffers
	if n == 0 {
		n = EnvBuffers()
	}

	arena, err := mapArena(n)
	if err != nil {
		return nil, err
	}
	p := &pool{smgr: cfg.Smgr, flusher: cfg.Flusher, descs: make([]desc, n), arena: arena, stop: make(chan struct{})}
	for i := range p.descs {
		d := &p.descs[i]
		d.id = int32(i)
		d.data = arena[i*page.Size : (i+1)*page.Size : (i+1)*page.Size]
	}
	for i := range p.parts {
		p.parts[i].m = map[Tag]int32{}
	}
	p.ring.init(ringLen(n))
	p.scratch.New = func() any { return heapPages(1) }

	delay := cfg.BgWriterDelay
	if delay == 0 {
		delay = DefaultBgWriterDelay
	}
	if delay > 0 {
		p.wg.Add(1)
		go p.bgWriter(delay)
	}
	return p, nil
}

/**
* heapPages reserva n páginas contiguas en el heap de Go alineadas a 8 (se piden como uint64): las marcas de
* visibilidad se ponen con un OR atómico de 32 bits y la copia para escribir se lee con cargas atómicas. Son las
* copias para escribir (scratch) y, donde no hay mmap, el arena.
* @param n int: cantidad de páginas
* @return page.Page: n×page.Size bytes
**/
func heapPages(n int) page.Page {
	words := make([]uint64, n*page.Size/8)
	return unsafe.Slice((*byte)(unsafe.Pointer(&words[0])), n*page.Size)
}

/**
* Pin fija la página tag; si no está en la caché la lee del disco y la verifica (ver Pool.Pin).
* @param tag Tag, strategy Strategy
* @return Handle, error: el error de lectura o de suma lleva el tag; ErrNoBuffers, ErrClosed, ErrFailed
**/
func (p *pool) Pin(tag Tag, strategy Strategy) (Handle, error) {
	d, err := p.pin(tag, strategy, false)
	if err != nil {
		return nil, err
	}
	return &handle{pool: p, d: d}, nil
}

/**
* PinNew agrega al fork una página en ceros y la devuelve fijada y con bloqueo exclusivo. Un fallo de Extend se
* devuelve sin dejar la caché inutilizable: no se perdió nada durable (a confirmar).
* @param file smgr.FileNo, fork smgr.Fork
* @return Handle, error
**/
func (p *pool) PinNew(file smgr.FileNo, fork smgr.Fork) (Handle, error) {
	if err := p.usable(); err != nil {
		return nil, err
	}
	d, err := p.extend(file, fork)
	if err != nil {
		return nil, err
	}
	h := &handle{pool: p, d: d}
	h.Lock(Exclusive)
	return h, nil
}

/**
* PinZero fija la página sin leerla, en ceros y con bloqueo exclusivo (ver Pool.PinZero). Si ya estaba en la caché,
* la pone en ceros igual: quien llama la pisa entera.
* @param tag Tag: un bloque que existe en el fork
* @return Handle, error
**/
func (p *pool) PinZero(tag Tag) (Handle, error) {
	d, err := p.pin(tag, Normal, true)
	if err != nil {
		return nil, err
	}
	h := &handle{pool: p, d: d}
	h.Lock(Exclusive)
	clear(d.data)
	return h, nil
}

/**
* extend agrega el bloque NBlocks al fork y lo fija en un búfer en ceros, sin leerlo.
* @param file smgr.FileNo, fork smgr.Fork
* @return *desc: fijado, error
**/
func (p *pool) extend(file smgr.FileNo, fork smgr.Fork) (*desc, error) {
	p.extendMu.Lock()
	defer p.extendMu.Unlock()
	block, err := p.smgr.NBlocks(file, fork)
	if err != nil {
		return nil, fmt.Errorf("buffer: extender %d/%d: %w", file, fork, err)
	}
	if err := p.smgr.Extend(file, fork, block, zeroPage); err != nil {
		return nil, fmt.Errorf("buffer: extender %d/%d: %w", file, fork, err)
	}
	return p.pin(Tag{File: file, Fork: fork, Block: block}, Normal, true)
}

// zeroPage es la página nueva que se escribe al extender. Nadie la cambia.
var zeroPage = make(page.Page, page.Size)

/**
* pin busca la página en la caché y la fija; si no está, le da un búfer y la carga. Repite mientras pierda una
* carrera con otra goroutine que carga o expulsa la misma página.
* @param tag Tag, strategy Strategy, fresh bool: la página es nueva (en ceros), no se lee del disco
* @return *desc: fijado y válido, error
**/
func (p *pool) pin(tag Tag, strategy Strategy, fresh bool) (*desc, error) {
	for {
		if err := p.usable(); err != nil {
			return nil, err
		}
		if d, found := p.pinExisting(tag, strategy); found {
			if d.waitLoaded() {
				p.hits.Add(1)
				return d, nil
			}
			p.unpin(d) // la carga falló: se intenta de nuevo (y quien la repita verá el error)
			continue
		}
		d, retry, err := p.load(tag, strategy, fresh)
		if retry {
			runtime.Gosched()
			continue
		}
		return d, err
	}
}

/**
* pinExisting fija la página si el mapa la tiene. Una entrada del mapa que ya no coincide con su búfer (un reemplazo
* a medio hacer) se espera: dura lo que tarda en actualizar dos entradas del mapa.
* @param tag Tag, strategy Strategy
* @return *desc: fijado (puede estar cargándose), bool: si estaba
**/
func (p *pool) pinExisting(tag Tag, strategy Strategy) (*desc, bool) {
	for {
		id, ok := p.lookup(tag)
		if !ok {
			return nil, false
		}
		d := &p.descs[id]
		d.mu.Lock()
		if d.tag == tag && d.state != stateEmpty {
			if !p.pinLocked(d) {
				d.mu.Unlock()
				return nil, false // cerrada: pin lo ve en usable
			}
			d.bumpUsage(strategy)
			d.mu.Unlock()
			return d, true
		}
		d.mu.Unlock()
		runtime.Gosched()
	}
}

/**
* bumpUsage sube el contador de uso al fijar: hasta MaxUsageCount con Normal; un recorrido con anillo solo lo lleva
* a 1, para no quedarse con la caché (como PostgreSQL). Se llama con d.mu tomado.
* @param strategy Strategy
**/
func (d *desc) bumpUsage(strategy Strategy) {
	if strategy == Ring {
		d.usage = max(d.usage, 1)
		return
	}
	d.usage = min(d.usage+1, MaxUsageCount)
}

/**
* waitLoaded espera a que termine la carga del búfer, si estaba cargándose. Se llama con el búfer fijado.
* @return bool: si quedó válido (false: la carga falló)
**/
func (d *desc) waitLoaded() bool {
	d.mu.Lock()
	done, state := d.done, d.state
	d.mu.Unlock()
	if state == stateLoading {
		<-done
		d.mu.Lock()
		state = d.state
		d.mu.Unlock()
	}
	return state == stateValid
}

/**
* load le da a tag un búfer (víctima del reemplazo ya limpia), lo publica en el mapa y lo carga del disco (o en
* ceros si fresh), fuera de todo lock. Sigue el orden de BufferAlloc: primero la entrada nueva del mapa, después el
* cambio de tag bajo d.mu, al final se quita la entrada vieja.
* @param tag Tag, strategy Strategy, fresh bool
* @return *desc: fijado y válido, bool: hay que repetir (otro ganó la carrera), error
**/
func (p *pool) load(tag Tag, strategy Strategy, fresh bool) (*desc, bool, error) {
	d, err := p.victim(strategy)
	if err != nil {
		return nil, false, err
	}
	if !p.insert(tag, d.id) {
		p.unpin(d) // otra goroutine la está cargando: se la fija en la próxima vuelta
		return nil, true, nil
	}

	d.mu.Lock()
	if d.pins != 1 || d.dirty {
		// Alguien la fijó por su tag viejo mientras tanto: se busca otra.
		d.mu.Unlock()
		p.remove(tag, d.id)
		p.unpin(d)
		return nil, true, nil
	}
	oldTag, hadOld := d.tag, d.state != stateEmpty
	d.tag, d.state, d.usage, d.done = tag, stateLoading, 1, make(chan struct{})
	d.mu.Unlock()
	if hadOld {
		p.remove(oldTag, d.id)
	} else {
		p.used.Add(1)
	}

	p.misses.Add(1)
	err = p.read(d, tag, fresh)
	if err != nil {
		p.remove(tag, d.id)
	}
	d.mu.Lock()
	if err != nil {
		d.tag, d.state, d.usage = Tag{}, stateEmpty, 0
	} else {
		d.state = stateValid
	}
	close(d.done)
	d.mu.Unlock()
	if err != nil {
		p.used.Add(-1)
		p.unpin(d)
		return nil, false, err
	}
	return d, false, nil
}

/**
* read llena el búfer con la página: en ceros si es nueva, si no la lee de smgr y la verifica con page.Verify (una
* página en ceros es válida; una suma mala es un error, nunca un pánico).
* @param d *desc: en stateLoading, solo lo toca quien carga; tag Tag, fresh bool
* @return error: con el tag
**/
func (p *pool) read(d *desc, tag Tag, fresh bool) error {
	if fresh {
		clear(d.data)
		return nil
	}
	if err := p.smgr.Read(tag.File, tag.Fork, tag.Block, d.data); err != nil {
		return fmt.Errorf("buffer: leer %s: %w", tag, err)
	}
	if err := page.Verify(d.data, tag.Block); err != nil {
		return fmt.Errorf("buffer: página %s: %w", tag, err)
	}
	return nil
}

/**
* String da el tag como archivo/fork/bloque, para los errores.
* @return string
**/
func (t Tag) String() string {
	return fmt.Sprintf("%d/%d/%d", t.File, t.Fork, t.Block)
}

/**
* unpin suelta un pin.
* @param d *desc
**/
func (p *pool) unpin(d *desc) {
	d.mu.Lock()
	d.pins--
	last := d.pins == 0
	d.mu.Unlock()
	if last && p.closed.Load() {
		p.freeArena() // el último pin de una caché cerrada libera la memoria que Close no pudo
	}
}

/**
* pinLocked fija el búfer salvo que la caché esté cerrada. Toda fijada pasa por acá, con d.mu tomado: como Close
* marca closed antes de mirar los pins (también bajo d.mu), tras Close los pins solo bajan, y freeArena puede liberar
* el arena en cuanto los ve todos en 0.
* @param d *desc: con d.mu tomado
* @return bool: false si la caché está cerrada
**/
func (p *pool) pinLocked(d *desc) bool {
	if p.closed.Load() {
		return false
	}
	d.pins++
	return true
}

/**
* setDirty marca el búfer sucio (si sigue teniendo su página) y anota justDirtied: una escritura en curso no lo
* marcará limpio al terminar, porque su copia puede no tener este cambio.
* @param d *desc: fijado
**/
func (p *pool) setDirty(d *desc) {
	d.mu.Lock()
	if d.state == stateValid {
		if !d.dirty {
			d.dirty = true
			p.dirty.Add(1)
		}
		d.justDirtied = true
	}
	d.mu.Unlock()
}

/**
* usable dice si la caché todavía se puede usar.
* @return error: nil, ErrClosed o ErrFailed con la causa
**/
func (p *pool) usable() error {
	if err := p.failed.Load(); err != nil {
		return *err
	}
	if p.closed.Load() {
		return ErrClosed
	}
	return nil
}

/**
* fail deja la caché inutilizable con el primer error (sin reintentar nada, ver store/doc.go «Fallos de
* escritura») y lo devuelve.
* @param cause error
* @return error: ErrFailed envolviendo el primer fallo
**/
func (p *pool) fail(cause error) error {
	err := fmt.Errorf("%w: %w", ErrFailed, cause)
	p.failed.CompareAndSwap(nil, &err)
	return *p.failed.Load()
}

/**
* Stats devuelve los contadores.
* @return Stats
**/
func (p *pool) Stats() Stats {
	return Stats{
		Hits:    p.hits.Load(),
		Misses:  p.misses.Load(),
		Dirty:   p.dirty.Load(),
		Written: p.written.Load(),
		Used:    p.used.Load(),
		Buffers: len(p.descs),
	}
}

/**
* Close detiene el escritor de fondo y libera el arena; no escribe nada. Desde acá nadie fija un búfer más; si
* alguno sigue fijado (un Handle sin Release), el arena se libera con el último Release y no antes: tocarlo
* liberado sería un SIGSEGV. Llamarlo otra vez no hace nada.
* @return error: el de unmapArena
**/
func (p *pool) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(p.stop)
	p.wg.Wait()
	return p.freeArena()
}

/**
* freeArena libera el arena si la caché está cerrada y ningún búfer está fijado; si no, no hace nada. Después deja
* los búferes sin memoria (data nil): un acceso tardío es un pánico de índice, no memoria ajena, y el GC no ve
* punteros a un rango que el runtime podría reutilizar.
* @return error: el de unmapArena
**/
func (p *pool) freeArena() error {
	p.arenaMu.Lock()
	defer p.arenaMu.Unlock()
	if p.arena == nil || p.pinned() {
		return nil
	}
	for i := range p.descs {
		d := &p.descs[i]
		d.mu.Lock()
		d.data = nil
		d.mu.Unlock()
	}
	mem := p.arena
	p.arena = nil
	return unmapArena(mem)
}

/**
* pinned dice si algún búfer está fijado. Se llama con la caché cerrada: los pins ya no suben, así que si los ve
* todos en 0 siguen en 0.
* @return bool
**/
func (p *pool) pinned() bool {
	for i := range p.descs {
		d := &p.descs[i]
		d.mu.Lock()
		pins := d.pins
		d.mu.Unlock()
		if pins != 0 {
			return true
		}
	}
	return false
}
