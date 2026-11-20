package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// group son las versiones de un id dentro de una hoja: el id y la primera visible (la más nueva que ve la lectura).
type group struct {
	id    []byte
	found bool
	v     Version
}

// ScanOptions dice cómo recorre un Cursor.
type ScanOptions struct {
	// From es el primer id (incluido) hacia adelante, o el último (incluido) hacia atrás; "" = desde el extremo.
	From string

	// Snapshot es la foto que decide la versión visible de cada id (nil = la más nueva confirmada).
	Snapshot *xact.Snapshot

	// Backward recorre en orden de id descendente.
	Backward bool

	// Ring fija las hojas siguientes a la primera con buffer.Ring: un recorrido grande no vacía la caché.
	Ring bool

	// KeysOnly no copia los valores en línea (Version.Value queda nil): contar, listar ids, saber si existe.
	KeysOnly bool

	// Borrow copia los valores en línea de cada hoja a un búfer del cursor que se reutiliza: valen hasta la próxima
	// llamada a Next. Solo hacia adelante (hacia atrás, el primer id de cada hoja pasa a la siguiente y se copia).
	Borrow bool
}

// Cursor recorre el árbol de a una hoja por llamada a Next, sin páginas fijadas ni bloqueadas entre llamadas: quien
// llama puede hacer lo que quiera entre una hoja y la siguiente (leer el store, soltar sus locks). Lo que guarda no
// crece con el store: la hoja leída, sus versiones y, hacia atrás, el primer id pendiente.
type Cursor struct {
	t       *BTree
	opts    ScanOptions
	started bool
	done    bool
	leaf    node // la última hoja leída (solo sus campos: su página ya se soltó)

	cur     []byte // hacia adelante: el id que se está mirando (sus versiones pueden seguir en la hoja siguiente)
	decided bool   // hacia adelante: ya se encontró la versión visible de cur
	pending *group // hacia atrás: el primer id de la hoja leída, que se decide con la anterior

	batch []Version
	arena []byte // los valores de la hoja con Borrow
}

/**
* NewCursor prepara un recorrido; no lee nada hasta Next.
* @param opts ScanOptions
* @return *Cursor
**/
func (t *BTree) NewCursor(opts ScanOptions) *Cursor {
	return &Cursor{t: t, opts: opts}
}

/**
* Next lee la hoja siguiente y devuelve, en el orden del recorrido, la versión visible de cada id ya decidido, sin
* las lápidas. Puede devolver una lista vacía sin haber terminado. El slice y sus valores son del cursor: valen hasta
* la próxima llamada.
* @return []Version, bool: false cuando ya no quedan hojas (la lista de esa llamada puede no estar vacía), error
**/
func (c *Cursor) Next() ([]Version, bool, error) {
	if c.done {
		return nil, false, nil
	}
	var err error
	if c.opts.Backward {
		err = c.nextBackward()
	} else {
		err = c.nextForward()
	}
	if err != nil {
		c.done = true
		return nil, false, err
	}
	return c.batch, !c.done, nil
}

/**
* strategy es la estrategia de la caché para las hojas siguientes a la primera.
* @return buffer.Strategy
**/
func (c *Cursor) strategy() buffer.Strategy {
	if c.opts.Ring {
		return buffer.Ring
	}
	return buffer.Normal
}

/**
* version copia una entrada según las opciones: sin valor (KeysOnly), con el valor en el búfer del cursor (Borrow) o
* con su propia copia.
* @param e entry
* @return Version
**/
func (c *Cursor) version(e entry) Version {
	if !c.opts.KeysOnly && !c.opts.Borrow {
		return e.version()
	}
	v := e.header()
	if c.opts.KeysOnly || v.Flags&(EntryTombstone|EntryOverflow) != 0 {
		return v
	}
	start := len(c.arena)
	c.arena = append(c.arena, e.inlineValue()...)
	v.Value = c.arena[start:len(c.arena):len(c.arena)]
	return v
}

/**
* nextForward lee la hoja siguiente hacia adelante: la primera vez baja hasta From; después sigue el enlace derecho
* anotado al leer la anterior (una partición posterior solo movió a la derecha lo ya leído).
* @return error
**/
func (c *Cursor) nextForward() error {
	h, n, off, err := c.firstOrRight()
	if err != nil {
		return err
	}
	c.batch, c.arena = c.batch[:0], c.arena[:0]
	var hs hints
	hs.at(h)
	for ; off <= n.maxOff() && err == nil; off++ {
		var e entry
		if e, err = n.entryAt(off); err != nil {
			break
		}
		if c.cur == nil || !bytes.Equal(e.id(), c.cur) {
			c.cur, c.decided = append(c.cur[:0], e.id()...), false
		}
		if c.decided {
			continue
		}
		var ok bool
		if ok, err = c.t.visible(e, c.opts.Snapshot, &hs); ok {
			c.decided = true
			if !e.tombstone() {
				c.batch = append(c.batch, c.version(e))
			}
		}
	}
	c.leaf, c.done = n, n.rightmost()
	h.Release()
	hs.at(nil)
	if err != nil {
		return err
	}
	c.t.flushHints(&hs)
	return nil
}

/**
* firstOrRight bloquea en compartido la hoja que sigue: la de From la primera vez, si no la hermana derecha de la
* última leída.
* @return buffer.Handle, node, page.OffsetNumber: el primer ítem a mirar, error
**/
func (c *Cursor) firstOrRight() (buffer.Handle, node, page.OffsetNumber, error) {
	if c.started {
		h, n, err := c.t.stepRightFrom(c.leaf, c.strategy())
		if err != nil {
			return nil, node{}, 0, err
		}
		return h, n, n.firstData(), nil
	}
	c.started = true
	k := searchKey([]byte(c.opts.From))
	h, n, _, err := c.t.descend(k, 0, buffer.Share, false)
	if err != nil {
		return nil, node{}, 0, err
	}
	off, err := n.lowerBound(k)
	if err != nil {
		h.Release()
		return nil, node{}, 0, err
	}
	return h, n, off, nil
}

/**
* nextBackward lee la hoja siguiente hacia atrás: la primera vez baja hasta From; después va a la hermana izquierda
* de la última leída (stepLeft). El primer id de cada hoja queda pendiente hasta leer la anterior.
* @return error
**/
func (c *Cursor) nextBackward() error {
	h, n, end, err := c.firstOrLeft()
	if err != nil {
		return err
	}
	var hs hints
	hs.at(h)
	groups, err := c.pageGroups(n, end-1, &hs)
	if err == nil {
		c.batch, c.pending = backwardBatch(c.batch[:0], groups, c.pending, n.o.Prev == 0)
	}
	c.leaf, c.done = n, n.o.Prev == 0
	h.Release()
	hs.at(nil)
	if err != nil {
		return err
	}
	c.t.flushHints(&hs)
	return nil
}

/**
* firstOrLeft bloquea en compartido la hoja que sigue hacia atrás: la de From la primera vez (o la de más a la
* derecha), si no la que hoy está a la izquierda de la última leída.
* @return buffer.Handle, node, page.OffsetNumber: uno más que el último ítem a mirar, error
**/
func (c *Cursor) firstOrLeft() (buffer.Handle, node, page.OffsetNumber, error) {
	if c.started {
		h, n, err := c.t.stepLeft(c.leaf, c.strategy())
		if err != nil {
			return nil, node{}, 0, err
		}
		return h, n, n.maxOff() + 1, nil
	}
	c.started = true
	k := key{top: true}
	if c.opts.From != "" {
		k = key{id: []byte(c.opts.From), lsn: 0} // después de todas las versiones de From
	}
	h, n, _, err := c.t.descend(k, 0, buffer.Share, false)
	if err != nil {
		return nil, node{}, 0, err
	}
	end, err := n.lowerBound(k)
	if err != nil {
		h.Release()
		return nil, node{}, 0, err
	}
	return h, n, end, nil
}

/**
* Scan recorre en orden de id ascendente, desde from (incluido; "" = desde el principio), la versión que ve snap de
* cada id (nil = la más nueva confirmada), sin las lápidas. Lee cada hoja entera con bloqueo compartido, la suelta y
* recién entonces llama a fn (fn puede leer el store; no puede escribir). La memoria es una hoja de versiones.
* @param from string, snap *xact.Snapshot
* @param fn func(v Version) (bool, error): false para parar
* @return error: el de fn o el del árbol
**/
func (t *BTree) Scan(from string, snap *xact.Snapshot, fn func(v Version) (bool, error)) error {
	return t.scanWith(ScanOptions{From: from, Snapshot: snap}, fn)
}

/**
* ScanBackward recorre en orden de id descendente, desde to (incluido; "" = desde el final), la versión que ve snap
* de cada id, sin las lápidas (ver Cursor: el primer id de cada hoja se decide con la anterior, y para ir a la
* izquierda se sigue a la derecha desde la hermana anotada hasta la página cuyo enlace derecho es la recién leída).
* @param to string, snap *xact.Snapshot, fn func(v Version) (bool, error): false para parar
* @return error
**/
func (t *BTree) ScanBackward(to string, snap *xact.Snapshot, fn func(v Version) (bool, error)) error {
	return t.scanWith(ScanOptions{From: to, Snapshot: snap, Backward: true}, fn)
}

/**
* scanWith recorre con un Cursor y llama a fn con cada versión, sin páginas bloqueadas.
* @param opts ScanOptions, fn func(v Version) (bool, error)
* @return error
**/
func (t *BTree) scanWith(opts ScanOptions, fn func(v Version) (bool, error)) error {
	c := t.NewCursor(opts)
	for {
		batch, more, err := c.Next()
		if err != nil {
			return err
		}
		if stop, ferr := emit(batch, fn); stop || ferr != nil {
			return ferr
		}
		if !more {
			return nil
		}
	}
}

/**
* stepRightFrom bloquea en compartido la hermana derecha de una hoja ya soltada.
* @param n node: la hoja leída (sus campos, no su página), strategy buffer.Strategy
* @return buffer.Handle, node, error
**/
func (t *BTree) stepRightFrom(n node, strategy buffer.Strategy) (buffer.Handle, node, error) {
	h, nn, err := t.lockNodeWith(n.o.Next, buffer.Share, strategy)
	if err == nil && nn.o.Level != n.o.Level {
		h.Release()
		return nil, node{}, corruptAt(nn.blk, "right sibling at level %d, expected %d", nn.o.Level, n.o.Level)
	}
	return h, nn, err
}

/**
* emit llama a fn con cada versión.
* @param batch []Version, fn func(v Version) (bool, error)
* @return bool: si fn pidió parar, error
**/
func emit(batch []Version, fn func(v Version) (bool, error)) (bool, error) {
	for _, v := range batch {
		cont, err := fn(v)
		if err != nil || !cont {
			return true, err
		}
	}
	return false, nil
}

/**
* pageGroups agrupa por id las entradas de datos de la hoja hasta last (incluido), en orden, con la primera versión
* visible de cada grupo (copiada según las opciones del cursor; hacia atrás no se usa Borrow).
* @param n node, last page.OffsetNumber, hs *hints: dónde marcar
* @return []group, error
**/
func (c *Cursor) pageGroups(n node, last page.OffsetNumber, hs *hints) ([]group, error) {
	var groups []group
	for off := n.firstData(); off <= last; off++ {
		e, err := n.entryAt(off)
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 || !bytes.Equal(groups[len(groups)-1].id, e.id()) {
			groups = append(groups, group{id: bytes.Clone(e.id())})
		}
		g := &groups[len(groups)-1]
		if g.found {
			continue
		}
		ok, err := c.t.visible(e, c.opts.Snapshot, hs)
		if err != nil {
			return nil, err
		}
		if ok {
			g.found, g.v = true, c.backwardVersion(e)
		}
	}
	return groups, nil
}

/**
* backwardVersion copia una entrada en un recorrido hacia atrás: sin valor con KeysOnly; si no, con su propia copia
* (el primer id de cada hoja sobrevive a la hoja siguiente, así que no puede vivir en el búfer reutilizado).
* @param e entry
* @return Version
**/
func (c *Cursor) backwardVersion(e entry) Version {
	if c.opts.KeysOnly {
		return e.header()
	}
	return e.version()
}

/**
* backwardBatch arma, de mayor a menor, los ids ya decididos de una hoja leída hacia atrás. El pendiente (el primer id
* de la hoja leída antes, a la derecha) se junta con el último grupo de esta si es el mismo id: gana la versión
* visible de esta hoja, que es más nueva. El primer grupo de esta hoja queda pendiente, salvo en la de más a la
* izquierda.
* @param batch []Version: se agrega acá, groups []group: en orden, pending *group, leftmost bool
* @return []Version, *group: el pendiente nuevo
**/
func backwardBatch(batch []Version, groups []group, pending *group, leftmost bool) ([]Version, *group) {
	if pending != nil && len(groups) > 0 && bytes.Equal(groups[len(groups)-1].id, pending.id) {
		g := &groups[len(groups)-1]
		if !g.found {
			g.found, g.v = pending.found, pending.v
		}
		pending = nil
	}
	if pending != nil && (len(groups) > 0 || leftmost) {
		batch = appendLive(batch, *pending)
		pending = nil
	}
	for i := len(groups) - 1; i >= 0; i-- {
		if i == 0 && !leftmost {
			pending = &groups[0]
			break
		}
		batch = appendLive(batch, groups[i])
	}
	return batch, pending
}

// appendLive agrega la versión del grupo si hay una visible y no es una lápida.
func appendLive(batch []Version, g group) []Version {
	if g.found && g.v.Flags&EntryTombstone == 0 {
		batch = append(batch, g.v)
	}
	return batch
}

/**
* stepLeft bloquea en compartido la hoja que hoy está a la izquierda de cur (ya soltada): la hermana anotada, o
* siguiendo a la derecha desde ella si se partió después de leer cur. Sin borrado de páginas (fase 1) siempre se llega.
* @param cur node: la hoja leída (sus campos), strategy buffer.Strategy
* @return buffer.Handle, node, error
**/
func (t *BTree) stepLeft(cur node, strategy buffer.Strategy) (buffer.Handle, node, error) {
	h, n, err := t.lockNodeWith(cur.o.Prev, buffer.Share, strategy)
	for err == nil && n.o.Next != cur.blk {
		if n.rightmost() || n.o.Level != cur.o.Level {
			h.Release()
			return nil, node{}, corruptAt(cur.blk, "left sibling chain does not reach the page")
		}
		next := n.o.Next
		h.Release()
		h, n, err = t.lockNodeWith(next, buffer.Share, strategy)
	}
	return h, n, err
}
