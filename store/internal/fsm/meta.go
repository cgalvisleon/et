package fsm

import (
	"encoding/binary"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// le es el orden de bytes de todo el formato.
var le = binary.LittleEndian

// Campos de la página meta que usa fsm: los define page (page/meta.go), compartidos con btree.
const (
	metaBlock   = page.MetaBlock
	metaMagic   = page.MetaMagic
	offMagic    = page.MetaOffMagic
	offFreeHead = page.MetaOffFreeHead
)

/**
* checkMeta comprueba que p sea una página meta: iniciada, con su magia y la marca page.OpaqueMeta.
* @param p page.Page
* @return error: ErrNoMeta si no lo es
**/
func checkMeta(p page.Page) error {
	if p.IsNew() || le.Uint32(p[offMagic:]) != metaMagic {
		return ErrNoMeta
	}
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueMeta == 0 {
		return ErrNoMeta
	}
	return nil
}

// freeHead devuelve la primera página de la lista de libres; 0 = vacía.
func freeHead(meta page.Page) page.BlockNumber {
	return page.BlockNumber(le.Uint32(meta[offFreeHead:]))
}

// setFreeHead pone la primera página de la lista de libres.
func setFreeHead(meta page.Page, block page.BlockNumber) {
	le.PutUint32(meta[offFreeHead:], uint32(block))
}

// blockErr envuelve err con el store y el bloque.
func blockErr(err error, file smgr.FileNo, block page.BlockNumber) error {
	return fmt.Errorf(MSG_BLOCK, err, file, block)
}
