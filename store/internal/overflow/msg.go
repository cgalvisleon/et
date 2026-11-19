package overflow

import "errors"

// Errores del paquete. Los detalles van envueltos con %w: se distinguen con errors.Is.
var (
	// ErrBadLen: el valor a escribir está vacío o pasa MaxValueLen.
	ErrBadLen = errors.New("overflow: value length out of range")

	// ErrBadRef: la referencia no apunta a una cadena posible (First 0, o Len 0 o mayor que MaxValueLen).
	ErrBadRef = errors.New("overflow: invalid chain reference")

	// ErrBadChain: la cadena no coincide con su referencia (corte, bucle, orden o largo distintos, o una página que
	// no es de desborde).
	ErrBadChain = errors.New("overflow: damaged chain")

	// ErrBadRecord: un registro del WAL de overflow no tiene la forma esperada.
	ErrBadRecord = errors.New("overflow: invalid WAL record")
)

// Mensajes con detalle.
const (
	MSG_BAD_CHAIN  = "%w: store %d, block %d, page %d of the chain: %s"
	MSG_BAD_REF    = "%w: store %d, first block %d, length %d"
	MSG_BAD_LEN    = "%w: %d bytes (1 to %d)"
	MSG_BAD_RECORD = "%w: kind %d: %s"
)
