package fsm

import (
	"sync"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Manager lleva la lista de páginas libres y el mapa de visibilidad de todos los stores de un motor. No guarda
// páginas en RAM: todo vive en las páginas de la caché. Es seguro usarlo desde varias goroutines.
type Manager struct {
	pool buffer.Pool
	sm   smgr.Manager
	log  wal.Log

	// vmExtendMu serializa la extensión del fork del mapa (rara: una página cubre VMBlocksPerPage bloques), para
	// que dos goroutines no agreguen la misma página dos veces. Es una hoja: no se toma otro lock con él, salvo los
	// de la caché al extender.
	vmExtendMu sync.Mutex
}

// Comprobación en compilación de que Manager cumple las interfaces.
var (
	_ FreeList      = (*Manager)(nil)
	_ VisibilityMap = (*Manager)(nil)
)

/**
* New crea el gestor sobre la caché, los archivos de datos y el WAL del motor.
* @param pool buffer.Pool, sm smgr.Manager, log wal.Log
* @return *Manager
**/
func New(pool buffer.Pool, sm smgr.Manager, log wal.Log) *Manager {
	return &Manager{pool: pool, sm: sm, log: log}
}

/**
* Allocate saca la primera página de la lista de libres de file (KindFreePop) o, si la lista está vacía, extiende
* el fork principal. Devuelve la página fijada y bloqueada en exclusivo: una sacada de la lista viene vacía (con
* espacio especial en cero), una nueva en ceros; quien la pide la inicia con BlockWillInit en su registro, y la suelta
* con Unlock y Release. Bloquea la meta y después la página libre; quien llama puede tener páginas del árbol
* bloqueadas, nunca la meta.
* @param file smgr.FileNo, xid page.XID: el store y la transacción que pide la página
* @return buffer.Handle, error: ErrNoMeta, ErrNotFree (lista dañada) o el error de la caché o del WAL
**/
func (m *Manager) Allocate(file smgr.FileNo, xid page.XID) (buffer.Handle, error) {
	meta, err := m.pinMeta(file)
	if err != nil {
		return nil, err
	}
	head := freeHead(meta.Page())
	if head == 0 {
		release(meta)
		return m.pool.PinNew(file, smgr.ForkMain)
	}

	taken, err := pinLocked(m.pool, file, smgr.ForkMain, head)
	if err != nil {
		release(meta)
		return nil, err
	}
	o, err := taken.Page().Opaque()
	if err != nil || o.Flags&page.OpaqueFree == 0 || taken.Page().LayoutVersion() != page.LayoutVersion {
		release(taken)
		release(meta)
		return nil, blockErr(ErrNotFree, file, head)
	}

	blocks := []wal.BlockRef{
		{File: file, Fork: smgr.ForkMain, Block: metaBlock, Delta: encodeBlock(o.Next)},
		{File: file, Fork: smgr.ForkMain, Block: head, Flags: wal.BlockWillInit},
	}
	res, err := m.logChange(wal.KindFreePop, xid, blocks, []buffer.Handle{meta, taken})
	release(meta)
	if res != nil {
		if werr := res.Write(); err == nil {
			err = werr
		}
	}
	if err != nil {
		release(taken)
		return nil, err
	}
	return taken, nil
}

/**
* Free devuelve la página block de file a la lista de libres (KindFreePush): la página queda vacía con
* page.OpaqueFree y next = la cabeza anterior, y la meta apunta a ella. Quien llama no puede tener bloqueada ni la
* página ni la meta (Free las bloquea en ese orden: meta, después la página).
* @param file smgr.FileNo, block page.BlockNumber, xid page.XID
* @return error: ErrFreeMeta, ErrDoubleFree, ErrNoMeta o el error de la caché o del WAL
**/
func (m *Manager) Free(file smgr.FileNo, block page.BlockNumber, xid page.XID) error {
	if block == metaBlock {
		return blockErr(ErrFreeMeta, file, block)
	}
	meta, err := m.pinMeta(file)
	if err != nil {
		return err
	}
	freed, err := pinLocked(m.pool, file, smgr.ForkMain, block)
	if err != nil {
		release(meta)
		return err
	}
	if o, err := freed.Page().Opaque(); err == nil && o.Flags&page.OpaqueFree != 0 {
		release(freed)
		release(meta)
		return blockErr(ErrDoubleFree, file, block)
	}

	return m.push(file, block, xid, meta, freed)
}

/**
* push anota block como la cabeza nueva de la lista (KindFreePush) y suelta las dos páginas.
* @param file smgr.FileNo, block page.BlockNumber, xid page.XID
* @param meta, freed buffer.Handle: la meta y la página, bloqueadas en exclusivo
* @return error
**/
func (m *Manager) push(file smgr.FileNo, block page.BlockNumber, xid page.XID, meta, freed buffer.Handle) error {
	blocks := []wal.BlockRef{
		{File: file, Fork: smgr.ForkMain, Block: metaBlock, Delta: encodeBlock(block)},
		{File: file, Fork: smgr.ForkMain, Block: block, Flags: wal.BlockWillInit, Delta: encodeBlock(freeHead(meta.Page()))},
	}
	res, err := m.logChange(wal.KindFreePush, xid, blocks, []buffer.Handle{meta, freed})
	release(freed)
	release(meta)
	if res != nil {
		if werr := res.Write(); err == nil {
			err = werr
		}
	}
	return err
}

// growMax es lo más que extiende Grow de una vez: 16 páginas (128 KB); growPart, la parte del archivo que agrega
// (1/64: a lo sumo un 1,6 % de páginas libres de más).
const (
	growMax  = 16
	growPart = 64
)

/**
* Grow deja páginas en la lista de libres de file si está vacía: extiende el archivo de una vez con 1/growPart de lo
* que mide (entre 1 y growMax páginas, como la extensión en grupo de PostgreSQL; un store chico crece de a una) sin
* páginas bloqueadas, y pone cada una en la lista (KindFreePush), de la última a la primera para que se usen en
* orden. Un corte entre la extensión y los registros deja páginas en ceros fuera del árbol: Verify las cuenta como
* perdidas y Repair las recupera, como tras un corte en medio de una partición. Dos llamadas a la vez pueden extender
* las dos; todo queda en la lista.
* @param file smgr.FileNo, xid page.XID: el de quien la va a usar
* @return error
**/
func (m *Manager) Grow(file smgr.FileNo, xid page.XID) error {
	meta, err := m.pinMeta(file)
	if err != nil {
		return err
	}
	head := freeHead(meta.Page())
	release(meta)
	if head != 0 {
		return nil
	}
	first, n, err := m.extend(file)
	if err != nil {
		return err
	}
	for i := n - 1; i >= 0; i-- {
		if err := m.pushNew(file, first+page.BlockNumber(i), xid); err != nil {
			return err
		}
	}
	return nil
}

/**
* extend agrega al fork principal de file 1/growPart de su largo (entre 1 y growMax páginas) en ceros, con una sola
* escritura si la caché sabe (buffer.Extender).
* @param file smgr.FileNo
* @return page.BlockNumber: la primera página nueva, int: cuántas, error
**/
func (m *Manager) extend(file smgr.FileNo) (page.BlockNumber, int, error) {
	nblocks, err := m.sm.NBlocks(file, smgr.ForkMain)
	if err != nil {
		return 0, 0, err
	}
	n := min(max(int(nblocks)/growPart, 1), growMax)
	if ext, ok := m.pool.(buffer.Extender); ok && n > 1 {
		first, err := ext.ExtendBy(file, smgr.ForkMain, n)
		return first, n, err
	}
	h, err := m.pool.PinNew(file, smgr.ForkMain)
	if err != nil {
		return 0, 0, err
	}
	block := h.Tag().Block
	release(h)
	return block, 1, nil
}

/**
* pushNew pone en la lista una página recién extendida (en ceros, que nadie más ve): la fija sin leerla y la anota
* como libre (KindFreePush), con la meta bloqueada antes que la página, como Free.
* @param file smgr.FileNo, block page.BlockNumber, xid page.XID
* @return error
**/
func (m *Manager) pushNew(file smgr.FileNo, block page.BlockNumber, xid page.XID) error {
	meta, err := m.pinMeta(file)
	if err != nil {
		return err
	}
	freed, err := m.pool.PinZero(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: block})
	if err != nil {
		release(meta)
		return err
	}
	return m.push(file, block, xid, meta, freed)
}

/**
* pinMeta fija y bloquea en exclusivo la página meta de file y comprueba que lo sea.
* @param file smgr.FileNo
* @return buffer.Handle, error: ErrNoMeta si el bloque 0 no es una meta
**/
func (m *Manager) pinMeta(file smgr.FileNo) (buffer.Handle, error) {
	meta, err := pinLocked(m.pool, file, smgr.ForkMain, metaBlock)
	if err != nil {
		return nil, err
	}
	if err := checkMeta(meta.Page()); err != nil {
		release(meta)
		return nil, blockErr(err, file, metaBlock)
	}
	return meta, nil
}

/**
* pinLocked fija una página y la bloquea en exclusivo.
* @param pool buffer.Pool, file smgr.FileNo, fork smgr.Fork, block page.BlockNumber
* @return buffer.Handle, error
**/
func pinLocked(pool buffer.Pool, file smgr.FileNo, fork smgr.Fork, block page.BlockNumber) (buffer.Handle, error) {
	h, err := pool.Pin(buffer.Tag{File: file, Fork: fork, Block: block}, buffer.Normal)
	if err != nil {
		return nil, err
	}
	h.Lock(buffer.Exclusive)
	return h, nil
}

// release suelta el bloqueo y el pin de h.
func release(h buffer.Handle) {
	h.Unlock()
	h.Release()
}
