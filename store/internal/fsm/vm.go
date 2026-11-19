package fsm

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

const (
	// vmAllBits son todos los bits que existen.
	vmAllBits = VMAllVisible | VMAllFrozen

	// vmBlocksPerByte son los bloques que cubre un byte del mapa.
	vmBlocksPerByte = 8 / VMBitsPerBlock

	// vmMask son los bits de un bloque, ya bajados a la derecha.
	vmMask = 1<<VMBitsPerBlock - 1
)

// vmBlockOf es la página del mapa que tiene los bits de block.
func vmBlockOf(block page.BlockNumber) page.BlockNumber {
	return block / VMBlocksPerPage
}

/**
* vmPos es dónde están los bits de block dentro de su página del mapa: el byte y el desplazamiento en bits.
* @param block page.BlockNumber: bloque del fork principal
* @return int, uint: posición del byte en la página y desplazamiento dentro de él
**/
func vmPos(block page.BlockNumber) (int, uint) {
	i := int(block % VMBlocksPerPage)
	return page.HeaderSize + i/vmBlocksPerByte, uint(i%vmBlocksPerByte) * VMBitsPerBlock
}

// vmGet lee los bits de block en su página del mapa (una página nueva no tiene bits).
func vmGet(p page.Page, block page.BlockNumber) VMBits {
	if p.IsNew() {
		return 0
	}
	off, shift := vmPos(block)
	return VMBits(p[off]>>shift) & vmMask
}

// vmPut reemplaza los bits de block en su página del mapa, ya iniciada.
func vmPut(p page.Page, block page.BlockNumber, bits VMBits) {
	off, shift := vmPos(block)
	p[off] = p[off]&^(vmMask<<shift) | byte(bits)<<shift
}

/**
* initVMPage inicia una página del mapa: encabezado sin espacio especial y pd_lower = page.Size, para que la imagen
* de página entera del WAL no tome el mapa por el hueco entre pd_lower y pd_upper.
* @param p page.Page
**/
func initVMPage(p page.Page) {
	_ = p.Init(0)
	_ = p.SetLower(page.Size)
}

/**
* Get devuelve los bits de block en el mapa de visibilidad de file. Un bloque que el mapa todavía no cubre (fork sin
* crear o más corto) no tiene bits. Bloquea la página del mapa en compartido.
* @param file smgr.FileNo, block page.BlockNumber: bloque del fork principal
* @return VMBits, error
**/
func (m *Manager) Get(file smgr.FileNo, block page.BlockNumber) (VMBits, error) {
	vmBlock := vmBlockOf(block)
	n, err := m.sm.NBlocks(file, smgr.ForkVM)
	if errors.Is(err, smgr.ErrNoFork) || (err == nil && vmBlock >= n) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	h, err := m.pool.Pin(buffer.Tag{File: file, Fork: smgr.ForkVM, Block: vmBlock}, buffer.Normal)
	if err != nil {
		return 0, err
	}
	h.Lock(buffer.Share)
	bits := vmGet(h.Page(), block)
	h.Unlock()
	h.Release()
	return bits, nil
}

/**
* Set pone bits de block (KindVMSet). Como en PostgreSQL, congelado implica visible: VMAllFrozen pone también
* VMAllVisible. Si ya estaban, no escribe nada. Extiende el fork del mapa si no cubre block.
* @param file smgr.FileNo, block page.BlockNumber, bits VMBits
* @return error: ErrBits si bits está vacío o fuera de VMAllVisible | VMAllFrozen
**/
func (m *Manager) Set(file smgr.FileNo, block page.BlockNumber, bits VMBits) error {
	if bits == 0 || bits&^vmAllBits != 0 {
		return ErrBits
	}
	if bits&VMAllFrozen != 0 {
		bits |= VMAllVisible
	}
	return m.changeVM(wal.KindVMSet, file, block, bits)
}

/**
* Clear borra todos los bits de block (KindVMClear). Si no tenía, no escribe nada.
* @param file smgr.FileNo, block page.BlockNumber
* @return error
**/
func (m *Manager) Clear(file smgr.FileNo, block page.BlockNumber) error {
	if bits, err := m.Get(file, block); err != nil || bits == 0 {
		return err
	}
	return m.changeVM(wal.KindVMClear, file, block, vmAllBits)
}

/**
* changeVM pone o borra bits de block siguiendo el orden de un cambio: bloquea la página del mapa en exclusivo,
* arma y reserva el registro, aplica, suelta y escribe. Si los bits no cambian, no escribe registro.
* @param kind wal.Kind: KindVMSet o KindVMClear
* @param file smgr.FileNo, block page.BlockNumber, bits VMBits
* @return error
**/
func (m *Manager) changeVM(kind wal.Kind, file smgr.FileNo, block page.BlockNumber, bits VMBits) error {
	vmBlock := vmBlockOf(block)
	if err := m.extendVM(file, vmBlock); err != nil {
		return err
	}
	h, err := pinLocked(m.pool, file, smgr.ForkVM, vmBlock)
	if err != nil {
		return err
	}
	cur := vmGet(h.Page(), block)
	if (kind == wal.KindVMSet && cur|bits == cur) || (kind == wal.KindVMClear && cur&^bits == cur) {
		release(h)
		return nil
	}
	blocks := []wal.BlockRef{{File: file, Fork: smgr.ForkVM, Block: vmBlock, Delta: encodeVM(block, bits)}}
	res, err := m.logChange(kind, 0, blocks, []buffer.Handle{h})
	release(h)
	if res != nil {
		if werr := res.Write(); err == nil {
			err = werr
		}
	}
	return err
}

/**
* extendVM hace que el fork del mapa de file llegue hasta vmBlock, creándolo si no existe. Las páginas nuevas quedan
* en ceros (sin bits); la primera escritura en cada una la inicia en su propio registro.
* @param file smgr.FileNo, vmBlock page.BlockNumber
* @return error
**/
func (m *Manager) extendVM(file smgr.FileNo, vmBlock page.BlockNumber) error {
	if n, err := m.sm.NBlocks(file, smgr.ForkVM); err == nil && vmBlock < n {
		return nil
	}
	m.vmExtendMu.Lock()
	defer m.vmExtendMu.Unlock()
	return ensureBlock(m.pool, m.sm, file, smgr.ForkVM, vmBlock)
}

/**
* ensureBlock extiende el fork hasta que block exista (creando el fork si falta), con páginas en ceros pedidas a la
* caché. Lo usan la extensión del mapa y el rehacer, que puede encontrar el archivo más corto que el WAL.
* @param pool buffer.Pool, sm smgr.Manager, file smgr.FileNo, fork smgr.Fork, block page.BlockNumber
* @return error
**/
func ensureBlock(pool buffer.Pool, sm smgr.Manager, file smgr.FileNo, fork smgr.Fork, block page.BlockNumber) error {
	n, err := sm.NBlocks(file, fork)
	if errors.Is(err, smgr.ErrNoFork) {
		err = sm.Create(file, fork)
	}
	if err != nil {
		return err
	}
	for ; n <= block; n++ {
		h, err := pool.PinNew(file, fork)
		if err != nil {
			return err
		}
		release(h)
	}
	return nil
}
