package page

import (
	"errors"
	"fmt"
)

// Errores del paquete. Los de página dañada envuelven ErrCorrupt con el detalle (errors.Is sirve).
var (
	ErrSize          = errors.New("page: buffer is not 8192 bytes")
	ErrCorrupt       = errors.New("page: corrupted page")
	ErrChecksum      = errors.New("page: checksum mismatch")
	ErrNoSpace       = errors.New("page: not enough free space")
	ErrInvalidOffset = errors.New("page: invalid item offset")
	ErrNoItem        = errors.New("page: item has no storage")
	ErrEmptyItem     = errors.New("page: empty item")
	ErrSpecialSize   = errors.New("page: invalid special space size")
	ErrNoOpaque      = errors.New("page: page has no opaque special space")
	ErrUnsorted      = errors.New("page: offsets must be strictly ascending")
)

/**
* corrupt devuelve ErrCorrupt con el detalle del problema.
* @param format string, args ...any: el detalle, como fmt.Sprintf
* @return error
**/
func corrupt(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrCorrupt}, args...)...)
}
