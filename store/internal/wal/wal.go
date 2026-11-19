package wal

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

const (
	// Dir es el directorio de los segmentos dentro del motor.
	Dir = "wal"

	// HeaderSize es el largo del encabezado de un registro.
	HeaderSize = 32

	// BlockHeaderSize es el largo del encabezado de una referencia a bloque.
	BlockHeaderSize = 16

	// MaxBlocks es la cantidad máxima de bloques de un registro (una partición toca tres).
	MaxBlocks = 4

	// MaxRecordSize es el largo máximo de un registro: MaxBlocks imágenes y deltas de una página y sus encabezados
	// caben con holgura. Como WAL_SEGMENT_SIZE ≥ MinSegmentSize, un registro siempre cabe en un segmento.
	MaxRecordSize = 64 << 10

	// MinSegmentSize es el menor WAL_SEGMENT_SIZE aceptado.
	MinSegmentSize = 1 << 20
)

// ErrNeedImage: Reserve encontró un bloque que necesita su imagen entera (su página tiene un lsn anterior al punto
// de rehacer vigente) y el registro no la trae. Quien llama agrega la imagen —con la página todavía bloqueada— y
// vuelve a llamar.
var ErrNeedImage = errors.New("wal: block needs a full-page image")

// Position es un lugar del WAL: segmento, desplazamiento y lsn del registro que empieza ahí.
type Position struct {
	Segment uint64
	Offset  int64
	LSN     page.LSN
}

// BlockRef es un bloque que cambia un registro.
type BlockRef struct {
	File  smgr.FileNo
	Fork  smgr.Fork
	Block page.BlockNumber
	Flags BlockFlags

	// PageLSN es el pd_lsn de la página antes del cambio: con él Reserve decide si hace falta la imagen. No se
	// escribe en el registro.
	PageLSN page.LSN

	Image []byte // imagen previa sin el hueco, ya comprimida si BlockImageCompressed
	Delta []byte // el cambio, en el formato del dueño del Kind
}

// Record es un registro armado (para escribir) o leído (al recuperar o decodificar).
type Record struct {
	LSN    page.LSN // lo asigna Reserve; leído, el del registro
	XID    page.XID
	Kind   Kind
	Flags  Flags
	Blocks []BlockRef
	Main   []byte
}

// Reservation es el lugar reservado de un registro.
type Reservation interface {
	// LSN es el lsn asignado.
	LSN() page.LSN

	// Write completa el lsn y el crc, escribe el registro en su lugar (fuera de writeMu, en paralelo con otros) y lo
	// publica. Un error deja al WAL fallado (Failed).
	Write() error
}

// Flusher es lo que la caché y el archivo de estados necesitan del WAL para cumplir «WAL antes que la página».
type Flusher interface {
	// Durable es el mayor lsn tal que todos los anteriores tienen fsync.
	Durable() page.LSN

	// WaitDurable vuelve cuando Durable() ≥ lsn, haciendo el fsync si hace falta (group commit).
	WaitDurable(lsn page.LSN) error
}

// Log es el WAL abierto.
type Log interface {
	Flusher

	// Reserve, bajo writeMu: comprueba que cada bloque con PageLSN < RedoPoint().LSN traiga su imagen (si no,
	// ErrNeedImage y no reserva nada), asigna el lsn siguiente y el lugar (sellando el segmento activo si no cabe),
	// llama a underLock(lsn) si no es nil y suelta writeMu. underLock corre con writeMu tomado: corto, sin bloqueos
	// salvo hojas (lo usa el motor para sumar los contadores de Count en orden de lsn de confirmación).
	Reserve(rec *Record, underLock func(lsn page.LSN)) (Reservation, error)

	// Published es el mayor lsn tal que todos los anteriores están escritos.
	Published() page.LSN

	// RedoPoint es el punto de rehacer vigente (el del checkpoint en curso, o el último); lectura atómica.
	RedoPoint() Position

	// BeginCheckpoint, bajo writeMu, fija el punto de rehacer en la posición del próximo registro, llama a
	// underLock con esa posición (el motor toma ahí nextXID, el xid vivo más viejo y los contadores) y la devuelve.
	BeginCheckpoint(underLock func(redo Position)) (Position, error)

	// Recycle quita (o renombra para reutilizar) los segmentos que terminan antes de keep.
	Recycle(keep Position) error

	// SizeSince es cuántos bytes de WAL hay desde from hasta el final (para disparar el checkpoint por volumen).
	SizeSince(from Position) int64

	// Failed es el primer error de escritura o fsync; desde entonces Reserve, Write y WaitDurable fallan.
	Failed() error

	// Close espera las escrituras en curso, hace fsync y cierra.
	Close() error
}

// Reader recorre el WAL ya escrito (recuperación, Sync, WalSince).
type Reader interface {
	// Scan llama a fn con cada registro válido desde from, en orden de lsn. Un registro es válido si cabe en el
	// segmento, su crc es correcto y su lsn es el anterior + 1. En el primero que no lo es, mira el primer registro
	// del segmento siguiente: si es válido con lsn = anterior + 1, el segmento terminó ahí (el registro siguiente no
	// cabía) y sigue; si es válido con un lsn mayor, faltan registros confirmados y Scan falla (corrupción); en
	// cualquier otro caso (no existe, inválido o un lsn menor: un segmento reciclado) es el final del WAL. Devuelve
	// la posición siguiente al último registro válido. rec es válido solo durante fn.
	Scan(from Position, fn func(rec *Record) error) (end Position, err error)
}
