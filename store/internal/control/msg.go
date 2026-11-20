package control

import "errors"

// Errores del paquete. Los detalles van envueltos con %w: se distinguen con errors.Is.
var (
	// ErrBadControl: una copia del control no es válida (magia, versión, CRC, tamaños o campos fuera de rango).
	ErrBadControl = errors.New("control: invalid control copy")

	// ErrFormatVersion: un control (con su CRC válido) o un catálogo de otra versión del formato en disco: lo escribió
	// un binario más nuevo, o uno anterior sin migración (ver «Versiones del formato» en store/doc.go).
	ErrFormatVersion = errors.New("control: unsupported on-disk format version")

	// ErrNoValidCopy: ninguna de las dos copias del control es válida; el motor no se puede abrir.
	ErrNoValidCopy = errors.New("control: no valid copy of the control file")

	// ErrExists: Create encontró un control válido; no se pisa un motor existente.
	ErrExists = errors.New("control: a valid control file already exists")

	// ErrBadCatalog: el catálogo no es válido (magia, versión, CRC, largos, nombres o números repetidos).
	ErrBadCatalog = errors.New("control: invalid catalog")

	// ErrFailed: una escritura o un fsync anterior falló; no se reintenta (ver store/doc.go, «Fallos de escritura»).
	ErrFailed = errors.New("control: a previous write failed, reopen the engine")

	// ErrBadName: el nombre de un store está vacío, pasa de MaxNameLen bytes, no es UTF-8 válido o tiene un nulo.
	ErrBadName = errors.New("control: store name must be valid UTF-8 without NUL, 1 to 255 bytes")

	// ErrBadFileNo: número de archivo 0, ya usado por otro store o agotado.
	ErrBadFileNo = errors.New("control: invalid file number")

	// ErrNegativeCount: un cambio dejaría el contador de un store por debajo de cero (o pasaría de MaxInt64).
	ErrNegativeCount = errors.New("control: store count would be negative or overflow")
)
