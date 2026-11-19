package overflow

import (
	"encoding/binary"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// le es el orden de bytes de todo el formato.
var le = binary.LittleEndian

// mainSize es el largo de los datos principales de KindOverflowWrite: next(4) seq(4).
const mainSize = 8

// encodeMain arma los datos principales de KindOverflowWrite: la siguiente página y la posición en la cadena.
func encodeMain(next page.BlockNumber, seq int) []byte {
	return le.AppendUint32(le.AppendUint32(nil, uint32(next)), uint32(seq))
}

// decodeMain lee los datos principales de KindOverflowWrite (ya validados por checkShape).
func decodeMain(main []byte) (page.BlockNumber, uint32) {
	return page.BlockNumber(le.Uint32(main)), le.Uint32(main[4:])
}

/**
* badRecord arma el error de un registro con forma inválida.
* @param kind wal.Kind, why string
* @return error: envuelve ErrBadRecord
**/
func badRecord(kind wal.Kind, why string) error {
	return fmt.Errorf(MSG_BAD_RECORD, ErrBadRecord, kind, why)
}

/**
* checkShape valida la forma de un KindOverflowWrite antes de tocar páginas: un bloque del fork principal (nunca la
* meta) con BlockWillInit, un trozo de 1 a ChunkSize bytes y los datos principales de mainSize bytes.
* @param rec *wal.Record
* @return error: ErrBadRecord con el detalle
**/
func checkShape(rec *wal.Record) error {
	if rec.Kind != wal.KindOverflowWrite {
		return badRecord(rec.Kind, "not an overflow kind")
	}
	if len(rec.Blocks) != 1 || len(rec.Main) != mainSize {
		return badRecord(rec.Kind, "needs one block and next(4) seq(4)")
	}
	b := rec.Blocks[0]
	if b.Fork != smgr.ForkMain || b.Block == 0 || b.Flags&wal.BlockWillInit == 0 {
		return badRecord(rec.Kind, "block must be a main-fork page (not the meta) with BlockWillInit")
	}
	if len(b.Delta) == 0 || len(b.Delta) > ChunkSize {
		return badRecord(rec.Kind, "chunk must be 1 to ChunkSize bytes")
	}
	if next, _ := decodeMain(rec.Main); next == b.Block {
		return badRecord(rec.Kind, "page points to itself")
	}
	return nil
}

/**
* apply deja p como la página de desborde número seq de su cadena, con el trozo chunk y la siguiente next, sin tocar
* pd_lsn. Es la misma función en el camino normal y al rehacer: así las dos llegan a la misma página.
* @param p page.Page: bloqueada en exclusivo; se pisa entera
* @param next page.BlockNumber, seq uint32, chunk []byte: de 1 a ChunkSize bytes
* @return error: solo si p no mide page.Size
**/
func apply(p page.Page, next page.BlockNumber, seq uint32, chunk []byte) error {
	if err := p.Init(page.OpaqueSize); err != nil {
		return err
	}
	copy(p[page.HeaderSize:], chunk)
	if err := p.SetLower(page.HeaderSize + len(chunk)); err != nil {
		return err
	}
	return p.SetOpaque(page.Opaque{Next: next, Level: seq, Flags: page.OpaqueOverflow})
}
