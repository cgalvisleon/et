package buffer

import (
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
)

// descState dice qué tiene un búfer.
type descState uint8

const (
	stateEmpty   descState = iota // sin página: candidato inmediato del reemplazo
	stateLoading                  // cargándose del disco: quien lo fija espera a done
	stateValid                    // con la página tag
)

// desc es un búfer y su descriptor (BufferDesc de PostgreSQL).
//
// Bloqueos, en este orden: io → content → mu. mu es corto y nunca se espera con él tomado. Los locks de las
// particiones del mapa son hojas: no se toman con mu tomado ni al revés.
type desc struct {
	id   int32
	data page.Page // los page.Size bytes del búfer dentro del arena, alineados a 8

	// content es el bloqueo de contenido: compartido para leer, exclusivo para cambiar.
	content sync.RWMutex

	// io serializa las escrituras del búfer al disco: dos escritores (escritor de fondo, checkpoint, reemplazo)
	// nunca escriben el mismo búfer a la vez, así una copia vieja no puede llegar después que una nueva.
	io sync.Mutex

	// mu protege los campos de abajo (el encabezado del búfer).
	mu    sync.Mutex
	tag   Tag
	state descState
	done  chan struct{} // se cierra al terminar la carga (stateLoading → stateValid o stateEmpty)
	pins  int32
	usage uint8
	dirty bool
}

// handle es un búfer fijado por quien llamó a Pin o PinNew. Lo usa una sola goroutine.
type handle struct {
	pool     *pool
	d        *desc
	mode     LockMode // bloqueo de contenido tomado; 0 = ninguno
	released bool
}

/**
* Tag devuelve la página que tiene el búfer.
* @return Tag
**/
func (h *handle) Tag() Tag {
	h.d.mu.Lock()
	defer h.d.mu.Unlock()
	return h.d.tag
}

/**
* Page devuelve el contenido del búfer (ver Handle.Page).
* @return page.Page
**/
func (h *handle) Page() page.Page { return h.d.data }

/**
* Lock toma el bloqueo de contenido. Tomarlo dos veces con el mismo handle es un error de programación.
* @param mode LockMode: Share o Exclusive
**/
func (h *handle) Lock(mode LockMode) {
	if h.mode != 0 {
		panic("buffer: Lock con el bloqueo ya tomado")
	}
	switch mode {
	case Share:
		h.d.content.RLock()
	case Exclusive:
		h.d.content.Lock()
	default:
		panic("buffer: modo de bloqueo desconocido")
	}
	h.mode = mode
}

/**
* Unlock suelta el bloqueo de contenido.
**/
func (h *handle) Unlock() {
	switch h.mode {
	case Share:
		h.d.content.RUnlock()
	case Exclusive:
		h.d.content.Unlock()
	default:
		panic("buffer: Unlock sin bloqueo tomado")
	}
	h.mode = 0
}

/**
* MarkDirty marca la página sucia tras un cambio (ver Handle.MarkDirty). Exige el bloqueo exclusivo.
* @param lsn page.LSN: el lsn del registro, ya puesto en pd_lsn
**/
func (h *handle) MarkDirty(lsn page.LSN) {
	if h.mode != Exclusive {
		panic("buffer: MarkDirty sin bloqueo exclusivo")
	}
	h.pool.setDirty(h.d)
}

/**
* MarkDirtyHint marca la página sucia tras poner marcas de visibilidad. Exige algún bloqueo de contenido.
**/
func (h *handle) MarkDirtyHint() {
	if h.mode == 0 {
		panic("buffer: MarkDirtyHint sin bloqueo de contenido")
	}
	h.pool.setDirty(h.d)
}

/**
* Release suelta el bloqueo de contenido si sigue tomado y después el pin. Llamarlo otra vez no hace nada.
**/
func (h *handle) Release() {
	if h.released {
		return
	}
	if h.mode != 0 {
		h.Unlock()
	}
	h.released = true
	h.pool.unpin(h.d)
}
