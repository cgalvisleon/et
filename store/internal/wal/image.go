package wal

import (
	"encoding/binary"
	"fmt"

	"github.com/klauspost/compress/s2"

	"github.com/cgalvisleon/et/store/internal/page"
)

// badImage arma el error de una imagen que no se puede restaurar.
func badImage(why string) error {
	return fmt.Errorf(MSG_BAD_IMAGE, why)
}

// holeOf devuelve el hueco [lower, upper) de una página (o de una imagen sin hueco, que empieza igual). Si el
// encabezado no describe un hueco válido (por ejemplo, una página nueva en ceros), devuelve false: la página va entera.
func holeOf(p []byte) (int, int, bool) {
	if len(p) < page.HeaderSize {
		return 0, 0, false
	}
	lower := int(binary.LittleEndian.Uint16(p[page.OffLower:]))
	upper := int(binary.LittleEndian.Uint16(p[page.OffUpper:]))
	if lower < page.HeaderSize || lower > upper || upper > page.Size {
		return 0, 0, false
	}
	return lower, upper, true
}

// MakeImage arma la imagen de página entera de p: la página sin el hueco [pd_lower, pd_upper), comprimida con S2 si
// así achica. Devuelve la imagen y sus marcas (BlockHasImage y, si se comprimió, BlockImageCompressed).
func MakeImage(p page.Page) ([]byte, BlockFlags, error) {
	if len(p) != page.Size {
		return nil, 0, badImage("page is not page.Size bytes")
	}
	raw := []byte(p)
	if lower, upper, ok := holeOf(p); ok && lower < upper {
		raw = make([]byte, 0, page.Size-(upper-lower))
		raw = append(raw, p[:lower]...)
		raw = append(raw, p[upper:]...)
	}
	compressed := s2.Encode(nil, raw)
	if len(compressed) < len(raw) {
		return compressed, BlockHasImage | BlockImageCompressed, nil
	}
	return append([]byte(nil), raw...), BlockHasImage, nil
}

// RestoreImage escribe en dst (page.Size bytes) la página que guarda img con sus marcas: descomprime si hace falta y
// rellena el hueco con ceros. Valida largos y hueco sin confiar en los bytes.
func RestoreImage(img []byte, flags BlockFlags, dst page.Page) error {
	if len(dst) != page.Size {
		return badImage("destination is not page.Size bytes")
	}
	if flags&BlockHasImage == 0 || len(img) == 0 {
		return badImage("no image")
	}
	raw := img
	if flags&BlockImageCompressed != 0 {
		n, err := s2.DecodedLen(img)
		if err != nil || n > page.Size {
			return badImage("bad compressed length")
		}
		var buf [page.Size]byte
		out, err := s2.Decode(buf[:n], img)
		if err != nil || len(out) != n {
			return badImage("bad compressed data")
		}
		raw = out
	}

	if len(raw) == page.Size {
		copy(dst, raw)
		return nil
	}
	lower, upper, ok := holeOf(raw)
	if !ok || len(raw) != lower+page.Size-upper {
		return badImage("length does not match its hole")
	}
	copy(dst[:lower], raw[:lower])
	clear(dst[lower:upper])
	copy(dst[upper:], raw[lower:])
	return nil
}

// SetImage pone en b la imagen de p (la página ANTES del cambio) con sus marcas.
func (b *BlockRef) SetImage(p page.Page) error {
	img, flags, err := MakeImage(p)
	if err != nil {
		return err
	}
	b.Image = img
	b.Flags = b.Flags&^(BlockHasImage|BlockImageCompressed) | flags
	return nil
}

// RestoreImage escribe en dst la página que guarda la imagen de b.
func (b *BlockRef) RestoreImage(dst page.Page) error {
	return RestoreImage(b.Image, b.Flags, dst)
}
