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
	defer c.t.guard.leave(c.t.guard.enter())
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
* anotado al leer la anterior (una partición posterior solo movió a la derecha lo ya leído). Una hoja medio muerta o
* borrada no tiene entradas: se pasa por ella a su hermana derecha. Las entradas con id menor que el que se está
* mirando se saltan: si la hoja que siguió a la leída se borró, su espacio de claves pasó a la derecha, y una versión
* escrita después puede caer ahí con un id ya recorrido (no es visible para un snapshot anterior, pero sí sin snapshot).
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
		if c.cur != nil && bytes.Compare(e.id(), c.cur) < 0 {
			continue // ya pasó: una hoja borrada le dejó su espacio de claves a esta (doc.go, «Borrado de páginas»)
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
	if h == nil { // las hojas de la izquierda se borraron después de leer la última
		c.batch, c.pending = backwardBatch(c.batch[:0], nil, c.pending, true)
		c.done = true
		return nil
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
* @return buffer.Handle, node: nil si ya no queda ninguna, page.OffsetNumber: uno más que el último ítem a mirar, error
**/
func (c *Cursor) firstOrLeft() (buffer.Handle, node, page.OffsetNumber, error) {
	if c.started {
		h, n, ok, err := c.t.stepLeft(c.leaf, c.strategy())
		if err != nil || !ok {
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

// walkLeftHops son los pasos a la derecha desde la hermana anotada antes de volver a mirar la página de partida (como
// _bt_walk_left): si no aparece, lo más probable es que la página de partida se haya borrado.
const walkLeftHops = 4

/**
* stepLeft bloquea en compartido la hoja que hoy está a la izquierda de cur (ya soltada), como _bt_walk_left: desde la
* hermana anotada sigue a la derecha (hasta walkLeftHops pasos) buscando la página no borrada cuyo enlace derecho es
* cur. Si no aparece, vuelve a cur: si se borró, sigue a su derecha hasta una no borrada (la que heredó su espacio de
* claves) y busca la izquierda de esa; si no, su enlace izquierdo cambió (la izquierda se partió o se borró) y repite
* con el nuevo. La página que devuelve puede estar medio muerta (sigue enganchada): no tiene entradas. cur no puede
* haberse reciclado: el recorrido retiene el horizonte (su snapshot) y cada Next la guarda (doc.go).
* @param cur node: la hoja leída (sus campos), strategy buffer.Strategy
* @return buffer.Handle, node, bool: false si ya no queda ninguna a la izquierda, error
**/
func (t *BTree) stepLeft(cur node, strategy buffer.Strategy) (buffer.Handle, node, bool, error) {
	obk, left := cur.blk, cur.o.Prev
	for left != 0 {
		h, n, found, err := t.leftFrom(left, obk, cur.o.Level, strategy)
		if err != nil || found {
			return h, n, found, err
		}
		if obk, left, err = t.reanchor(obk, left, strategy); err != nil {
			return nil, node{}, false, err
		}
	}
	return nil, node{}, false, nil // las de la izquierda se borraron: no queda nada antes
}

/**
* leftFrom sigue a la derecha desde left (hasta walkLeftHops pasos) buscando la página no borrada cuyo enlace derecho
* es obk; la devuelve bloqueada en compartido.
* @param left, obk page.BlockNumber, level uint32, strategy buffer.Strategy
* @return buffer.Handle, node, bool: si la encontró, error
**/
func (t *BTree) leftFrom(left, obk page.BlockNumber, level uint32, strategy buffer.Strategy) (buffer.Handle, node, bool, error) {
	blk := left
	for hops := 0; ; hops++ {
		h, n, err := t.lockNodeWith(blk, buffer.Share, strategy)
		if err != nil {
			return nil, node{}, false, err
		}
		if n.o.Level != level {
			h.Release()
			return nil, node{}, false, corruptAt(blk, "left sibling at level %d, expected %d", n.o.Level, level)
		}
		if !n.deleted() && n.o.Next == obk {
			return h, n, true, nil
		}
		next := n.o.Next
		h.Release()
		if next == 0 || hops >= walkLeftHops {
			return nil, node{}, false, nil
		}
		blk = next
	}
}

/**
* reanchor vuelve a la página de partida cuando su izquierda no apareció: si se borró, la de partida pasa a ser la
* primera no borrada a su derecha; si no, su enlace izquierdo tiene que haber cambiado.
* @param obk, left page.BlockNumber: la de partida y la izquierda que se probó, strategy buffer.Strategy
* @return page.BlockNumber, page.BlockNumber: la de partida y su izquierda nuevas, error
**/
func (t *BTree) reanchor(obk, left page.BlockNumber, strategy buffer.Strategy) (page.BlockNumber, page.BlockNumber, error) {
	h, n, err := t.lockNodeWith(obk, buffer.Share, strategy)
	if err != nil {
		return 0, 0, err
	}
	for n.deleted() {
		next := n.o.Next
		h.Release()
		if h, n, err = t.lockNodeWith(next, buffer.Share, strategy); err != nil {
			return 0, 0, err
		}
	}
	defer h.Release()
	if n.blk == obk && n.o.Prev == left {
		return 0, 0, corruptAt(obk, "could not find the left sibling %d", left)
	}
	return n.blk, n.o.Prev, nil
}
