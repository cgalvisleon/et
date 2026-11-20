package smgr

import (
	"container/list"
	"os"
	"sync"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// segKey identifica un archivo de datos: store, fork y número de segmento (desde 1).
type segKey struct {
	file FileNo
	fork Fork
	seg  uint32
}

// handle es un descriptor abierto de la caché. refs cuenta quienes lo usan ahora: con refs > 0 no se cierra.
// dropped: drop lo sacó de la caché mientras estaba en uso; el último release lo cierra.
type handle struct {
	key     segKey
	f       vfs.File
	refs    int
	elem    *list.Element
	dropped bool
}

// fileCache es la caché de descriptores de tamaño fijo. Cuando hay más de max abiertos, cierra los menos usados
// que nadie esté usando. Si todos están en uso, se pasa del tope un momento y lo recupera al soltarlos.
//
// Su mutex es una hoja: los archivos se abren, sincronizan y cierran fuera de él.
type fileCache struct {
	fs    vfs.FS
	max   int
	evict func(h *handle) // cierra un descriptor expulsado (antes hace fsync si tiene escrituras pendientes)

	mu   sync.Mutex
	open map[segKey]*handle
	lru  *list.List // frente = usado hace menos
}

/**
* newFileCache crea la caché de descriptores.
* @param fs vfs.FS, max int, evict func(*handle)
* @return *fileCache
**/
func newFileCache(fs vfs.FS, max int, evict func(h *handle)) *fileCache {
	return &fileCache{fs: fs, max: max, evict: evict, open: map[segKey]*handle{}, lru: list.New()}
}

/**
* acquire devuelve el descriptor del archivo, abriéndolo si hace falta; quien llama debe soltarlo con release.
* @param key segKey, name string (ruta completa), create bool (crea el archivo si no existe)
* @return *handle, error (envuelve fs.ErrNotExist si no existe y create es false)
**/
func (c *fileCache) acquire(key segKey, name string, create bool) (*handle, error) {
	c.mu.Lock()
	if h := c.open[key]; h != nil {
		c.use(h)
		c.mu.Unlock()
		return h, nil
	}
	c.mu.Unlock()

	flag := os.O_RDWR
	if create {
		flag |= os.O_CREATE
	}
	f, err := c.fs.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if h := c.open[key]; h != nil {
		// Otro lo abrió mientras tanto: se usa el suyo.
		c.use(h)
		c.mu.Unlock()
		f.Close()
		return h, nil
	}
	h := &handle{key: key, f: f, refs: 1}
	h.elem = c.lru.PushFront(h)
	c.open[key] = h
	victims := c.takeVictims()
	c.mu.Unlock()

	c.closeVictims(victims)
	return h, nil
}

/**
* release suelta un descriptor tomado con acquire y cierra los que sobren.
* @param h *handle
**/
func (c *fileCache) release(h *handle) {
	c.mu.Lock()
	h.refs--
	closeIt := h.dropped && h.refs == 0
	victims := c.takeVictims()
	c.mu.Unlock()

	if closeIt {
		h.f.Close() // sus archivos se borraron: sin fsync
	}
	c.closeVictims(victims)
}

/**
* drop saca de la caché y cierra, sin fsync, los descriptores de un fork (sus archivos se van a borrar). Quien llama
* tiene el fork bloqueado en exclusivo; un descriptor que todavía usa un Sync (tomado antes, con el fork suelto
* durante el fsync) se cierra en su último release.
* @param file FileNo, fork Fork
**/
func (c *fileCache) drop(file FileNo, fork Fork) {
	var closing []*handle
	c.mu.Lock()
	for key, h := range c.open {
		if key.file == file && key.fork == fork {
			c.remove(h)
			if h.refs > 0 {
				h.dropped = true
				continue
			}
			closing = append(closing, h)
		}
	}
	c.mu.Unlock()

	for _, h := range closing {
		h.f.Close()
	}
}

/**
* closeAll cierra todos los descriptores, sin fsync.
* @return error, el primero que falle
**/
func (c *fileCache) closeAll() error {
	c.mu.Lock()
	all := make([]*handle, 0, len(c.open))
	for _, h := range c.open {
		all = append(all, h)
		c.remove(h)
	}
	c.mu.Unlock()

	var first error
	for _, h := range all {
		if err := h.f.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

/**
* count dice cuántos descriptores hay abiertos (para las pruebas).
* @return int
**/
func (c *fileCache) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.open)
}

// use marca h como en uso y recién usado. Con el mutex tomado.
func (c *fileCache) use(h *handle) {
	h.refs++
	c.lru.MoveToFront(h.elem)
}

// remove saca h del mapa y de la lista. Con el mutex tomado.
func (c *fileCache) remove(h *handle) {
	delete(c.open, h.key)
	c.lru.Remove(h.elem)
}

// takeVictims saca los descriptores libres menos usados hasta volver al tope. Con el mutex tomado; quien llama los
// cierra después con closeVictims, fuera del mutex.
func (c *fileCache) takeVictims() []*handle {
	var victims []*handle
	e := c.lru.Back()
	for len(c.open) > c.max && e != nil {
		prev := e.Prev()
		h := e.Value.(*handle)
		if h.refs == 0 {
			c.remove(h)
			victims = append(victims, h)
		}
		e = prev
	}
	return victims
}

// closeVictims cierra los descriptores expulsados.
func (c *fileCache) closeVictims(victims []*handle) {
	for _, h := range victims {
		c.evict(h)
	}
}
