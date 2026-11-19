package fsm

import (
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// VMBits son los bits de un bloque en el mapa de visibilidad.
type VMBits uint8

const (
	VMAllVisible VMBits = 1 << 0
	VMAllFrozen  VMBits = 1 << 1

	// VMBitsPerBlock son los bits de cada bloque en el mapa.
	VMBitsPerBlock = 2

	// VMBlocksPerPage son los bloques que cubre una página del mapa.
	VMBlocksPerPage = (page.Size - page.HeaderSize) * 8 / VMBitsPerBlock
)

// FreeList es la lista de páginas libres de los stores.
type FreeList interface {
	// Allocate devuelve una página libre del fork principal de file, fijada y bloqueada en exclusivo, para que quien
	// la pide la inicie (con BlockWillInit en su registro). Anota en el WAL el cambio de la lista.
	Allocate(file smgr.FileNo, xid page.XID) (buffer.Handle, error)

	// Free devuelve la página block a la lista (la marca page.OpaqueFree) y lo anota en el WAL.
	Free(file smgr.FileNo, block page.BlockNumber, xid page.XID) error
}

// VisibilityMap es el mapa de visibilidad de los stores (fase 1B/2).
type VisibilityMap interface {
	// Get devuelve los bits de block; un bloque fuera del mapa no tiene bits.
	Get(file smgr.FileNo, block page.BlockNumber) (VMBits, error)

	// Set pone bits de block (KindVMSet).
	Set(file smgr.FileNo, block page.BlockNumber, bits VMBits) error

	// Clear borra todos los bits de block (KindVMClear).
	Clear(file smgr.FileNo, block page.BlockNumber) error
}
