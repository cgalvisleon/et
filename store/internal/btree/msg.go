package btree

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Errores del paquete. Los detalles van envueltos con %w: se distinguen con errors.Is.
var (
	// ErrCorrupt: una página del árbol no tiene la forma que exige el formato (o el árbol no es coherente).
	ErrCorrupt = errors.New("btree: corrupted tree")

	// ErrBadRecord: un registro del WAL de btree no tiene la forma esperada.
	ErrBadRecord = errors.New("btree: invalid WAL record")

	// ErrIDLength: el id no mide de 1 a MaxIDLen bytes.
	ErrIDLength = errors.New("btree: id must be 1 to 2000 bytes")

	// ErrEntryTooLarge: el valor no cabe en línea; quien llama lo escribe antes en desborde y pasa la overflow.Ref.
	ErrEntryTooLarge = errors.New("btree: value does not fit in line, write it to overflow first")

	// ErrSerialization: la última versión confirmada del id no es visible para el snapshot del lote (pregunta 34).
	ErrSerialization = errors.New("btree: could not serialize access due to a concurrent update")

	// ErrExists: Create encontró el store con páginas.
	ErrExists = errors.New("btree: the store already has pages")

	// ErrConfig: a Config le falta una pieza.
	ErrConfig = errors.New("btree: invalid configuration")
)

// Mensajes con detalle.
const (
	MSG_CORRUPT    = "%w: %s"
	MSG_CORRUPT_AT = "%w: block %d: %s"
	MSG_BAD_RECORD = "%w: kind %d: %s"
)

/**
* corrupt arma ErrCorrupt con el detalle.
* @param format string, args ...any: el detalle, como fmt.Sprintf
* @return error
**/
func corrupt(format string, args ...any) error {
	return fmt.Errorf(MSG_CORRUPT, ErrCorrupt, fmt.Sprintf(format, args...))
}

/**
* corruptAt arma ErrCorrupt con el bloque y el detalle.
* @param blk page.BlockNumber, format string, args ...any
* @return error
**/
func corruptAt(blk page.BlockNumber, format string, args ...any) error {
	return fmt.Errorf(MSG_CORRUPT_AT, ErrCorrupt, blk, fmt.Sprintf(format, args...))
}

/**
* badRecord arma ErrBadRecord con el tipo de registro y el detalle.
* @param kind wal.Kind, why string
* @return error
**/
func badRecord(kind wal.Kind, why string) error {
	return fmt.Errorf(MSG_BAD_RECORD, ErrBadRecord, kind, why)
}
