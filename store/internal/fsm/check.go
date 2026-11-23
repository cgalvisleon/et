package fsm

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Verificación y reparación (Verify y Repair del motor, fase 3): recorrer la lista de libres y el mapa de
// visibilidad sin cambiar nada, y devolver a la lista una página que nadie usa.

/**
* WalkFree recorre la lista de libres de file desde la cabeza de la meta, sin cambiar nada: cada página tiene que ser
* libre (page.OpaqueFree) y menor que limit. Fija una página a la vez con buffer.Ring, en compartido. Nunca da más de
* limit pasos: un bucle termina en visit (que ve una página repetida) o en el tope.
* @param file smgr.FileNo, limit page.BlockNumber: los bloques del fork principal
* @param visit func(block page.BlockNumber) error: cada página de la lista, en orden; un error corta el recorrido
* @return error: ErrNoMeta, ErrNotFree (con el bloque), el de visit o el de la caché
**/
func (m *Manager) WalkFree(file smgr.FileNo, limit page.BlockNumber, visit func(block page.BlockNumber) error) error {
	head, err := m.readFreeHead(file)
	if err != nil {
		return err
	}
	for steps, block := page.BlockNumber(0), head; block != 0; steps++ {
		if block >= limit || block == metaBlock || steps >= limit {
			return blockErr(ErrNotFree, file, block)
		}
		if err := visit(block); err != nil {
			return err
		}
		next, err := m.freeNext(file, block)
		if err != nil {
			return err
		}
		block = next
	}
	return nil
}

/**
* readFreeHead lee la cabeza de la lista de libres de la meta, con bloqueo compartido.
* @param file smgr.FileNo
* @return page.BlockNumber, error: ErrNoMeta
**/
func (m *Manager) readFreeHead(file smgr.FileNo) (page.BlockNumber, error) {
	h, err := m.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: metaBlock}, buffer.Normal)
	if err != nil {
		return 0, err
	}
	h.Lock(buffer.Share)
	defer release(h)
	if err := checkMeta(h.Page()); err != nil {
		return 0, blockErr(err, file, metaBlock)
	}
	return freeHead(h.Page()), nil
}

/**
* freeNext lee la siguiente de una página de la lista, comprobando que sea libre.
* @param file smgr.FileNo, block page.BlockNumber
* @return page.BlockNumber, error: ErrNotFree
**/
func (m *Manager) freeNext(file smgr.FileNo, block page.BlockNumber) (page.BlockNumber, error) {
	h, err := m.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: block}, buffer.Ring)
	if err != nil {
		return 0, err
	}
	h.Lock(buffer.Share)
	defer release(h)
	o, err := h.Page().Opaque()
	if h.Page().IsNew() || err != nil || o.Flags&page.OpaqueFree == 0 || h.Page().LayoutVersion() != page.LayoutVersion {
		return 0, blockErr(ErrNotFree, file, block)
	}
	return o.Next, nil
}

/**
* ForEachVM llama a visit por cada bloque con bits en el mapa de visibilidad de file, en orden, sin cambiar nada.
* Lee las páginas del mapa de a una con buffer.Ring; un store sin mapa no tiene bits.
* @param file smgr.FileNo
* @param visit func(block page.BlockNumber, bits VMBits) error: un error corta el recorrido
* @return error
**/
func (m *Manager) ForEachVM(file smgr.FileNo, visit func(block page.BlockNumber, bits VMBits) error) error {
	n, err := m.sm.NBlocks(file, smgr.ForkVM)
	if errors.Is(err, smgr.ErrNoFork) {
		return nil
	}
	if err != nil {
		return err
	}
	for vmBlock := page.BlockNumber(0); vmBlock < n; vmBlock++ {
		if err := m.visitVMPage(file, vmBlock, visit); err != nil {
			return err
		}
	}
	return nil
}

/**
* visitVMPage llama a visit por cada bloque con bits de una página del mapa. Copia los bits de a uno con la página
* bloqueada y llama a visit ya soltada.
* @param file smgr.FileNo, vmBlock page.BlockNumber
* @param visit func(block page.BlockNumber, bits VMBits) error
* @return error
**/
func (m *Manager) visitVMPage(file smgr.FileNo, vmBlock page.BlockNumber, visit func(page.BlockNumber, VMBits) error) error {
	h, err := m.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkVM, Block: vmBlock}, buffer.Ring)
	if err != nil {
		return err
	}
	h.Lock(buffer.Share)
	var set []vmSet
	if p := h.Page(); !p.IsNew() {
		first := vmBlock * VMBlocksPerPage
		for i := page.BlockNumber(0); i < VMBlocksPerPage; i++ {
			if bits := vmGet(p, first+i); bits != 0 {
				set = append(set, vmSet{block: first + i, bits: bits})
			}
		}
	}
	release(h)
	for _, s := range set {
		if err := visit(s.block, s.bits); err != nil {
			return err
		}
	}
	return nil
}

// vmSet son los bits de un bloque leídos del mapa.
type vmSet struct {
	block page.BlockNumber
	bits  VMBits
}

/**
* Reclaim devuelve a la lista de libres una página que nadie usa, sea lo que sea su contenido (en ceros, sacada de la
* lista sin usar, de una cadena huérfana, libre fuera de la lista, o ilegible): la fija sin leerla (PinZero) y la
* inicia entera con KindFreePush (BlockWillInit), así una página rota no lo impide. Es para Repair, que ya comprobó
* que la página no es del árbol, de una cadena viva ni de la lista; Free es la de uso normal. Quien llama no puede
* tener bloqueadas ni la meta ni la página.
* @param file smgr.FileNo, block page.BlockNumber, xid page.XID
* @return error: ErrFreeMeta, ErrNoMeta o el error de la caché o del WAL
**/
func (m *Manager) Reclaim(file smgr.FileNo, block page.BlockNumber, xid page.XID) error {
	if block == metaBlock {
		return blockErr(ErrFreeMeta, file, block)
	}
	meta, err := m.pinMeta(file)
	if err != nil {
		return err
	}
	freed, err := m.pool.PinZero(buffer.Tag{File: file, Fork: smgr.ForkMain, Block: block})
	if err != nil {
		release(meta)
		return err
	}
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
