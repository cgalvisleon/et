package fsm

import "errors"

// Errores del paquete. Los detalles van envueltos con %w: se distinguen con errors.Is.
var (
	// ErrNoMeta: el bloque 0 del store no es una página meta (el store no se inició o la página está dañada).
	ErrNoMeta = errors.New("fsm: block 0 is not a meta page")

	// ErrFreeMeta: se pidió liberar la página meta.
	ErrFreeMeta = errors.New("fsm: the meta page cannot be freed")

	// ErrDoubleFree: la página ya está en la lista de libres.
	ErrDoubleFree = errors.New("fsm: page is already free")

	// ErrNotFree: la cabeza de la lista no es una página libre (lista dañada).
	ErrNotFree = errors.New("fsm: free list head is not a free page")

	// ErrBits: bits del mapa de visibilidad fuera de VMAllVisible | VMAllFrozen.
	ErrBits = errors.New("fsm: invalid visibility bits")

	// ErrBadRecord: un registro del WAL de fsm no tiene la forma esperada.
	ErrBadRecord = errors.New("fsm: invalid WAL record")
)

// Mensajes con detalle.
const (
	MSG_BAD_RECORD = "%w: kind %d: %s"
	MSG_BLOCK      = "%w: store %d, block %d"
)
