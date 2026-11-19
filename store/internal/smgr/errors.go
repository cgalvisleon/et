package smgr

import "errors"

// Errores de smgr. Se devuelven envueltos con el archivo y el bloque; se comparan con errors.Is.
var (
	// ErrClosed: el gestor ya se cerró.
	ErrClosed = errors.New("smgr: cerrado")

	// ErrNoFork: el fork no tiene archivos (falta Create, o el store se borró).
	ErrNoFork = errors.New("smgr: el fork no existe")

	// ErrBadFork: el número de fork no es ForkMain ni ForkVM.
	ErrBadFork = errors.New("smgr: fork desconocido")

	// ErrBadPage: el buffer no mide page.Size.
	ErrBadPage = errors.New("smgr: el buffer no mide una página")

	// ErrBlockOutOfRange: el bloque está más allá del final del fork (Read, Write) o es el número inválido.
	ErrBlockOutOfRange = errors.New("smgr: bloque fuera de rango")

	// ErrBlockExists: Extend sobre un bloque que ya existe.
	ErrBlockExists = errors.New("smgr: el bloque ya existe")

	// ErrShortFile: un archivo es más corto de lo que debe (uno que no es el último no mide 1 GB, o falta un bloque
	// que el fork dice tener). Es corrupción o un cambio desde fuera del motor.
	ErrShortFile = errors.New("smgr: archivo corto")

	// ErrLongFile: un archivo mide más de BlocksPerFile bloques.
	ErrLongFile = errors.New("smgr: archivo más largo que un segmento")
)
