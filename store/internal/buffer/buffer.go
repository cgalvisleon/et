package buffer

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

const (
	// DefaultCacheSize es el tamaño por omisión de la caché en bytes (STORE_CACHE_SIZE se da en MB).
	DefaultCacheSize = 128 << 20

	// MaxUsageCount es el tope del contador de uso del clock-sweep.
	MaxUsageCount = 5

	// RingSize es el tamaño del anillo de un recorrido grande.
	RingSize = 256 << 10
)

// Tag identifica una página: store, fork y bloque.
type Tag struct {
	File  smgr.FileNo
	Fork  smgr.Fork
	Block page.BlockNumber
}

// LockMode es el bloqueo de contenido.
type LockMode int

const (
	Share     LockMode = 1
	Exclusive LockMode = 2
)

// Strategy dice cómo reemplaza búferes una lectura.
type Strategy int

const (
	Normal Strategy = 0 // clock-sweep sobre toda la caché
	Ring   Strategy = 1 // recorrido grande: reutiliza un anillo de RingSize
)

// Handle es una página fijada. Se suelta con Release; el bloqueo de contenido se toma y se suelta aparte.
type Handle interface {
	Tag() Tag

	// Page es el contenido; solo se lee con el bloqueo tomado y solo se cambia con el exclusivo.
	Page() page.Page

	Lock(mode LockMode)
	Unlock()

	// MarkDirty marca la página sucia tras un cambio con el lsn de su registro (ya puesto en pd_lsn). Requiere el
	// bloqueo exclusivo.
	MarkDirty(lsn page.LSN)

	// MarkDirtyHint marca sucia tras poner marcas de visibilidad (con bloqueo compartido o exclusivo).
	MarkDirtyHint()

	// Release suelta el pin.
	Release()
}

// Pool es la caché de páginas.
type Pool interface {
	// Pin fija la página tag, leyéndola del disco si no está (y verificando su suma).
	Pin(tag Tag, strategy Strategy) (Handle, error)

	// PinNew extiende el fork con una página en ceros y la devuelve fijada y bloqueada en exclusivo.
	PinNew(file smgr.FileNo, fork smgr.Fork) (Handle, error)

	// FlushAll escribe todas las páginas que estaban sucias al llamarlo, al ritmo que da pace (nil = sin pausa), y
	// devuelve; el fsync de los archivos lo hace el checkpoint con smgr.SyncAll.
	FlushAll(pace func(written, total int)) error

	// DropFile quita de la caché, sin escribirlas, las páginas de un store que se borra o se vacía.
	DropFile(file smgr.FileNo)

	// Stats: aciertos, fallos, sucias y escritas desde Open.
	Stats() Stats

	// Close detiene el escritor de fondo. No escribe nada: el motor hace antes su checkpoint final.
	Close() error
}

// Stats son los contadores de la caché.
type Stats struct {
	Hits    int64
	Misses  int64
	Dirty   int64
	Written int64
	Buffers int
}
