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

// ItemsChecker es lo opcional de un Handle que recuerda si quien usa la página ya validó sus punteros a ítems desde que
// el búfer la cargó (la caché del motor lo cumple; btree lo usa para no repetir page.CheckItems en cada acceso: los
// cambios en memoria los hacen funciones que validan lo que tocan, y lo que llega del disco se valida una vez).
type ItemsChecker interface {
	ItemsChecked() bool
	SetItemsChecked()
}

// Extender es lo opcional de una Pool que extiende un fork con varias páginas de una vez (la caché del motor lo
// cumple; fsm.Manager.Grow lo usa).
type Extender interface {
	// ExtendBy agrega n páginas en ceros al final del fork con una sola llamada a smgr.Extend (que escribe los ceros
	// de todas, sin huecos) y devuelve la primera. No fija ninguna: quien llama las fija con PinZero.
	ExtendBy(file smgr.FileNo, fork smgr.Fork, n int) (page.BlockNumber, error)
}

// Pool es la caché de páginas.
type Pool interface {
	// Pin fija la página tag, leyéndola del disco si no está (y verificando su suma). Tras un fallo de disco lleno
	// sigue andando, de solo lectura (ver Failed).
	Pin(tag Tag, strategy Strategy) (Handle, error)

	// PinNew extiende el fork con una página en ceros y la devuelve fijada y bloqueada en exclusivo.
	PinNew(file smgr.FileNo, fork smgr.Fork) (Handle, error)

	// PinZero fija la página tag sin leerla del disco, en ceros y bloqueada en exclusivo (como RBM_ZERO_AND_LOCK de
	// PostgreSQL). Es para rehacer un registro que la pisa entera (imagen o BlockWillInit): la página en disco puede
	// estar rota por un corte y leerla fallaría por la suma. El bloque tiene que existir en el fork.
	PinZero(tag Tag) (Handle, error)

	// FlushAll escribe todas las páginas que estaban sucias al llamarlo, al ritmo que da pace (nil = sin pausa), y
	// devuelve; el fsync de los archivos lo hace el checkpoint con smgr.SyncAll.
	FlushAll(pace func(written, total int)) error

	// DropFile quita de la caché, sin escribirlas, las páginas de un store que se borra o se vacía.
	DropFile(file smgr.FileNo)

	// Stats: aciertos, fallos, sucias y escritas desde Open.
	Stats() Stats

	// Failed es el primer fallo de escritura (ErrFailed con la causa), o nil. Desde un fallo la caché no escribe nada
	// más; si fue de disco lleno queda de solo lectura: Pin sigue (reusa solo búferes limpios) y PinNew y PinZero
	// fallan.
	Failed() error

	// Close detiene el escritor de fondo. No escribe nada: el motor hace antes su checkpoint final.
	Close() error
}

// Stats son los contadores de la caché.
type Stats struct {
	Hits    int64
	Misses  int64
	Dirty   int64
	Written int64
	Used    int64 // búferes con una página (cargada o cargándose); el resto está vacío
	Buffers int
}
