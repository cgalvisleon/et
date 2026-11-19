package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// castagnoli es la tabla del CRC-32C.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// Posiciones de los campos del encabezado (ver doc.go, «Registro»).
const (
	offCRC        = 0
	offLSN        = 4
	offTotalLen   = 12
	offXID        = 16
	offKind       = 24
	offFlags      = 25
	offBlockCount = 26
	offReserved   = 27
	offMainLen    = 28
)

// Marcas conocidas: un registro con otras es inválido.
const (
	knownFlags      = FlagCommit | FlagCountAdd | FlagCountSub
	knownBlockFlags = BlockHasImage | BlockImageCompressed | BlockWillInit
)

// badRecord arma el error de un registro que no se puede escribir.
func badRecord(why string) error {
	return fmt.Errorf(MSG_BAD_RECORD, why)
}

// recordSize valida rec para escribirlo y devuelve su largo en disco.
func recordSize(rec *Record) (int, error) {
	if rec.Kind == 0 {
		return 0, badRecord("kind 0")
	}
	if rec.Flags&^knownFlags != 0 {
		return 0, badRecord("unknown flags")
	}
	if len(rec.Blocks) > MaxBlocks {
		return 0, badRecord("too many blocks")
	}
	size := HeaderSize + len(rec.Main)
	for i := range rec.Blocks {
		b := &rec.Blocks[i]
		switch {
		case b.Flags&^knownBlockFlags != 0:
			return 0, badRecord("unknown block flags")
		case len(b.Image) > page.Size:
			return 0, badRecord("image larger than a page")
		case len(b.Delta) > math.MaxUint16:
			return 0, badRecord("delta too long")
		case b.Flags&BlockImageCompressed != 0 && len(b.Image) == 0:
			return 0, badRecord("compressed flag without image")
		}
		size += BlockHeaderSize + len(b.Image) + len(b.Delta)
	}
	if size > MaxRecordSize {
		return 0, badRecord("larger than MaxRecordSize")
	}
	return size, nil
}

// blockFlags son las marcas que se escriben de b: BlockHasImage sale de si trae imagen, no de lo que diga Flags.
func blockFlags(b *BlockRef) BlockFlags {
	flags := b.Flags &^ BlockHasImage
	if len(b.Image) > 0 {
		flags |= BlockHasImage
	}
	return flags
}

// encodeRecord escribe rec en buf (de recordSize(rec) bytes) sin lsn ni crc: los pone sealRecord al conocer el lsn.
func encodeRecord(rec *Record, buf []byte) {
	le := binary.LittleEndian
	le.PutUint32(buf[offTotalLen:], uint32(len(buf)))
	le.PutUint64(buf[offXID:], uint64(rec.XID))
	buf[offKind] = byte(rec.Kind)
	buf[offFlags] = byte(rec.Flags)
	buf[offBlockCount] = byte(len(rec.Blocks))
	buf[offReserved] = 0
	le.PutUint32(buf[offMainLen:], uint32(len(rec.Main)))

	off := HeaderSize
	for i := range rec.Blocks {
		b := &rec.Blocks[i]
		h := buf[off : off+BlockHeaderSize]
		le.PutUint32(h[0:], uint32(b.File))
		h[4] = byte(b.Fork)
		h[5] = byte(blockFlags(b))
		h[6], h[7] = 0, 0
		le.PutUint32(h[8:], uint32(b.Block))
		le.PutUint16(h[12:], uint16(len(b.Image)))
		le.PutUint16(h[14:], uint16(len(b.Delta)))
		off += BlockHeaderSize
		off += copy(buf[off:], b.Image)
		off += copy(buf[off:], b.Delta)
	}
	copy(buf[off:], rec.Main)
}

// sealRecord pone el lsn y el crc de todo lo que le sigue.
func sealRecord(buf []byte, lsn page.LSN) {
	binary.LittleEndian.PutUint64(buf[offLSN:], uint64(lsn))
	binary.LittleEndian.PutUint32(buf[offCRC:], crc32.Checksum(buf[offLSN:], castagnoli))
}

// totalLenOf lee totalLen de un encabezado (al menos HeaderSize bytes) sin validarlo.
func totalLenOf(header []byte) int {
	return int(binary.LittleEndian.Uint32(header[offTotalLen:]))
}

// decodeRecord lee el registro que empieza en buf (buf puede seguir después) y lo deja en rec, cuyos Image, Delta
// y Main son vistas de buf. No confía en ningún byte: largos, marcas, reservados y crc se comprueban antes de usarlos.
// Devuelve el largo del registro, o false si no es válido.
func decodeRecord(buf []byte, rec *Record) (int, bool) {
	if len(buf) < HeaderSize {
		return 0, false
	}
	total := totalLenOf(buf)
	if total < HeaderSize || total > MaxRecordSize || total > len(buf) {
		return 0, false
	}
	buf = buf[:total]
	le := binary.LittleEndian
	if le.Uint32(buf[offCRC:]) != crc32.Checksum(buf[offLSN:], castagnoli) {
		return 0, false
	}
	lsn := page.LSN(le.Uint64(buf[offLSN:]))
	kind := Kind(buf[offKind])
	flags := Flags(buf[offFlags])
	count := int(buf[offBlockCount])
	if lsn == 0 || kind == 0 || flags&^knownFlags != 0 || buf[offReserved] != 0 || count > MaxBlocks {
		return 0, false
	}

	*rec = Record{LSN: lsn, XID: page.XID(le.Uint64(buf[offXID:])), Kind: kind, Flags: flags, Blocks: rec.Blocks[:0]}
	off := HeaderSize
	for range count {
		b, n, ok := decodeBlock(buf[off:])
		if !ok {
			return 0, false
		}
		rec.Blocks = append(rec.Blocks, b)
		off += n
	}
	mainLen := int(le.Uint32(buf[offMainLen:]))
	if mainLen != total-off {
		return 0, false
	}
	rec.Main = buf[off:total:total]
	return total, true
}

// decodeBlock lee una referencia a bloque desde el principio de buf. Devuelve cuántos bytes ocupa, o false.
func decodeBlock(buf []byte) (BlockRef, int, bool) {
	if len(buf) < BlockHeaderSize {
		return BlockRef{}, 0, false
	}
	le := binary.LittleEndian
	flags := BlockFlags(buf[5])
	imageLen := int(le.Uint16(buf[12:]))
	deltaLen := int(le.Uint16(buf[14:]))
	hasImage := flags&BlockHasImage != 0
	switch {
	case flags&^knownBlockFlags != 0, buf[6] != 0, buf[7] != 0:
		return BlockRef{}, 0, false
	case hasImage != (imageLen > 0), imageLen > page.Size:
		return BlockRef{}, 0, false
	case flags&BlockImageCompressed != 0 && !hasImage:
		return BlockRef{}, 0, false
	}
	n := BlockHeaderSize + imageLen + deltaLen
	if n > len(buf) {
		return BlockRef{}, 0, false
	}
	image := buf[BlockHeaderSize : BlockHeaderSize+imageLen : BlockHeaderSize+imageLen]
	delta := buf[BlockHeaderSize+imageLen : n : n]
	if imageLen == 0 {
		image = nil
	}
	if deltaLen == 0 {
		delta = nil
	}
	return BlockRef{
		File:  smgr.FileNo(le.Uint32(buf[0:])),
		Fork:  smgr.Fork(buf[4]),
		Flags: flags,
		Block: page.BlockNumber(le.Uint32(buf[8:])),
		Image: image,
		Delta: delta,
	}, n, true
}

// needsImage dice si algún bloque de rec necesita su imagen entera y no la trae: su página tiene un lsn anterior al
// punto de rehacer y el delta no la inicia entera (BlockWillInit).
func needsImage(rec *Record, redo page.LSN) bool {
	for i := range rec.Blocks {
		b := &rec.Blocks[i]
		if b.PageLSN < redo && len(b.Image) == 0 && b.Flags&BlockWillInit == 0 {
			return true
		}
	}
	return false
}
