package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Hojas de la forma 2 (formato 2, tarea 134 de F-23; doc.go «Hojas de la forma 2»): LayoutVersion 2, un ítem de
// prefijo después de la clave alta y entradas con la cabecera corta que guardan el id sin el prefijo. Este archivo
// arma esas hojas: el ítem de prefijo, la elección del prefijo, la reconstrucción de una hoja (migración desde la
// forma 1, prefijo que se acorta, izquierda de una partición) y KindLeafRewrite.

// MaxPrefixLen es el largo máximo del prefijo de una hoja: con él acotado, partir una hoja siempre encuentra un corte
// contando el ítem de prefijo de cada mitad con los largos de antes (las entradas solo se achican al recortarlas).
const MaxPrefixLen = 255

// rewriteDeltaSize es el delta de KindLeafRewrite: prefixLen(2).
const rewriteDeltaSize = 2

/**
* encodePrefixItem arma el ítem de prefijo de una hoja de la forma 2: un pivote sin enlace ni lsn cuya clave es el
* prefijo (puede ser vacío).
* @param prefix []byte: hasta MaxPrefixLen bytes
* @return []byte
**/
func encodePrefixItem(prefix []byte) []byte {
	return encodePivot(key{id: prefix, lsn: MaxLSN}, 0)
}

/**
* readPrefixItem lee y valida el ítem de prefijo de una hoja de la forma 2.
* @param p page.Page, off page.OffsetNumber
* @return []byte: el prefijo (vista de la página), error: ErrCorrupt
**/
func readPrefixItem(p page.Page, off page.OffsetNumber) ([]byte, error) {
	b, err := p.Item(off)
	if err != nil {
		return nil, corrupt("prefix item %d: %v", off, err)
	}
	v, err := parsePivot(b)
	if err != nil || v.downlink() != 0 || le.Uint16(v[offPivotFlags:]) != 0 || len(v)-PivotHeaderSize > MaxPrefixLen {
		return nil, corrupt("bad prefix item %d", off)
	}
	return v[PivotHeaderSize:], nil
}

/**
* initLeaf deja p como una hoja vacía con el espacio especial o: de la forma 2 (con su ítem de prefijo vacío) si v2,
* si no de la forma 1 (un KindInitStore del formato 1).
* @param p page.Page, o page.Opaque: de una hoja de más a la derecha, v2 bool
* @return error
**/
func initLeaf(p page.Page, o page.Opaque, v2 bool) error {
	if err := initTreePage(p, o); err != nil || !v2 {
		return err
	}
	if err := p.SetLayoutVersion(page.LayoutVersion2); err != nil {
		return err
	}
	_, err := p.AddItem(encodePrefixItem(nil))
	return err
}

/**
* hasIDPrefix dice si el id entero de k (pre + id) empieza con prefix.
* @param k key, prefix []byte
* @return bool
**/
func hasIDPrefix(k key, prefix []byte) bool {
	if len(prefix) <= len(k.pre) {
		return bytes.HasPrefix(k.pre, prefix)
	}
	return bytes.Equal(prefix[:len(k.pre)], k.pre) && bytes.HasPrefix(k.id, prefix[len(k.pre):])
}

/**
* choosePrefix elige el prefijo de una hoja que se arma: el común de su primera y su última entrada (las demás quedan
* entre esas dos, así que lo tienen), acotado a MaxPrefixLen; vacío sin entradas. Sale de las entradas y no de las
* cotas de la hoja: así una hoja que solo recibe ids crecientes (la de más a la derecha, la cola de un grupo de un
* índice) ya está recortada cuando se parte, y su izquierda queda llena. Una escritura posterior con un id sin el
* prefijo lo acorta antes (needsRewrite, rewriteLeaf): todo id de la hoja tiene su prefijo, nada más.
* @param first, last *key: la primera y la última entrada (nil = ninguna)
* @return int: el largo del prefijo (sus bytes son el comienzo del id de la primera entrada)
**/
func choosePrefix(first, last *key) int {
	if first == nil || last == nil {
		return 0
	}
	return min(commonPrefixKeys(*first, *last), MaxPrefixLen)
}

// leafContent es lo que se vuelve a armar de una hoja: su espacio especial, sus marcas de página, pd_prune_xid, su
// clave alta (nil en la de más a la derecha) y sus entradas (vistas de una copia de la página).
type leafContent struct {
	o       page.Opaque
	flags   uint16
	prune   page.XID
	highKey []byte
	entries []entry
}

/**
* readLeafContent lee una hoja (de cualquier forma) de una copia de la página.
* @param old page.Page: copia; las vistas que devuelve son de ella
* @return leafContent, error: ErrCorrupt
**/
func readLeafContent(old page.Page) (leafContent, error) {
	o, err := old.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf == 0 || o.Flags&ignoredPage != 0 {
		return leafContent{}, corrupt("not a live leaf")
	}
	n, err := pageNode(old, o)
	if err != nil {
		return leafContent{}, err
	}
	c := leafContent{o: o, flags: old.Flags(), prune: old.PruneXID()}
	if !n.rightmost() {
		hk, err := n.highKey()
		if err != nil {
			return leafContent{}, err
		}
		c.highKey = hk
	}
	for off := n.firstData(); off <= n.maxOff(); off++ {
		e, err := n.entryAt(off)
		if err != nil {
			return leafContent{}, err
		}
		c.entries = append(c.entries, e)
	}
	return c, nil
}

/**
* firstKey es la clave de la primera entrada (nil = sin entradas).
* @return *key
**/
func (c leafContent) firstKey() *key {
	if len(c.entries) == 0 {
		return nil
	}
	k := c.entries[0].key()
	return &k
}

/**
* lastKey es la clave de la última entrada (nil = sin entradas).
* @return *key
**/
func (c leafContent) lastKey() *key {
	if len(c.entries) == 0 {
		return nil
	}
	k := c.entries[len(c.entries)-1].key()
	return &k
}

/**
* prefixBytes son los primeros prefixLen bytes del id de la primera entrada: el prefijo de la hoja que se arma.
* @param prefixLen int
* @return []byte, error: ErrCorrupt si no hay entradas para un prefijo no vacío o el id es más corto
**/
func (c leafContent) prefixBytes(prefixLen int) ([]byte, error) {
	if prefixLen == 0 {
		return nil, nil
	}
	if len(c.entries) == 0 || prefixLen > MaxPrefixLen {
		return nil, corrupt("prefix of %d bytes without entries", prefixLen)
	}
	id := c.entries[0].appendID(nil)
	if len(id) < prefixLen {
		return nil, corrupt("prefix of %d bytes longer than the first id", prefixLen)
	}
	return id[:prefixLen], nil
}

/**
* items2 arma los ítems de la hoja en la forma 2 con el prefijo dado: su clave alta, el ítem de prefijo y cada entrada
* recodificada; comprueba que toda entrada tenga el prefijo y que todo quepa en una página.
* @param prefix []byte
* @return [][]byte, error: ErrCorrupt
**/
func (c leafContent) items2(prefix []byte) ([][]byte, error) {
	items := make([][]byte, 0, len(c.entries)+2)
	if c.highKey != nil {
		items = append(items, c.highKey)
	}
	items = append(items, encodePrefixItem(prefix))
	size := 0
	for _, e := range c.entries {
		if !hasIDPrefix(e.key(), prefix) {
			return nil, corrupt("entry without the leaf prefix")
		}
		size += e.storedSize(len(prefix), false)
	}
	buf := make([]byte, 0, size) // un solo búfer para todas las entradas: cada ítem es un trozo suyo
	for _, e := range c.entries {
		start := len(buf)
		buf = e.appendReencode(buf, len(prefix), false)
		items = append(items, buf[start:len(buf):len(buf)])
	}
	if !fitInPage(items) {
		return nil, corrupt("leaf of form 2 does not fit in a page")
	}
	return items, nil
}

/**
* fitInPage dice si los ítems entran en una página del árbol vacía.
* @param items [][]byte
* @return bool
**/
func fitInPage(items [][]byte) bool {
	total := 0
	for _, it := range items {
		total += page.SpaceFor(len(it))
	}
	return total <= capacity
}

/**
* write2 arma p como la hoja de la forma 2 con el contenido c y un prefijo de prefixLen bytes. Valida todo antes de
* tocar p: si falla, p queda como estaba.
* @param p page.Page, prefixLen int
* @return error: ErrCorrupt
**/
func (c leafContent) write2(p page.Page, prefixLen int) error {
	prefix, err := c.prefixBytes(prefixLen)
	if err != nil {
		return err
	}
	items, err := c.items2(prefix)
	if err != nil {
		return err
	}
	if err := initTreePage(p, c.o); err != nil {
		return err
	}
	if err := p.SetLayoutVersion(page.LayoutVersion2); err != nil {
		return err
	}
	p.SetFlags(c.flags &^ page.FlagHasFreeLines)
	p.SetPruneXID(c.prune)
	for _, it := range items {
		if _, err := p.AddItem(it); err != nil {
			return err
		}
	}
	return nil
}

/**
* applyLeafRewrite rehace KindLeafRewrite (la misma función al escribir y al rehacer): arma de nuevo la hoja, de
* cualquier forma, como de la forma 2 con un prefijo de prefixLen bytes (el comienzo del id de su primera entrada),
* conservando la clave alta, los enlaces, las marcas de la página (también page.FlagAllVisible: no cambia qué versiones
* hay ni sus marcas) y pd_prune_xid. Los números de los ítems de datos cambian solo en una hoja de la forma 1 (corren
* uno, por el ítem de prefijo).
* @param p page.Page, delta []byte: prefixLen(2)
* @return error: ErrBadRecord
**/
func applyLeafRewrite(p page.Page, delta []byte) error {
	old := append(page.Page(nil), p...)
	c, err := readLeafContent(old)
	if err == nil {
		err = c.write2(p, int(le.Uint16(delta)))
	}
	if err != nil {
		return badRecord(wal.KindLeafRewrite, err.Error())
	}
	return nil
}

/**
* rewritePrefix elige el prefijo con que se reescribe una hoja: el de choosePrefix, acotado para que id (la escritura
* que llega, si hay) lo tenga, si todo cabe con él; si no, ninguno (con prefijo vacío una hoja de la forma 1 siempre
* cabe: cada entrada se achica 8 B y el ítem de prefijo mide 16).
* @param c leafContent, id []byte: nil = ninguna escritura
* @return int, bool: el largo y si la hoja cabe con él (una de la forma 2 que pierde prefijo puede no caber: hay que
* partirla)
**/
func rewritePrefix(c leafContent, id []byte) (int, bool) {
	n := choosePrefix(c.firstKey(), c.lastKey())
	if id != nil && n > 0 {
		n = min(n, commonPrefix(c.entries[0].appendID(nil), id))
	}
	for _, try := range []int{n, 0} {
		prefix, err := c.prefixBytes(try)
		if err != nil {
			continue
		}
		if _, err := c.items2(prefix); err == nil {
			return try, true
		}
	}
	return 0, false
}

/**
* rewriteLeaf reescribe en la forma 2, con un KindLeafRewrite, la hoja bloqueada en exclusivo: una hoja del formato 1
* que se va a cambiar (la migración perezosa: una escritura o la limpieza la encuentra) o una de la forma 2 cuyo
* prefijo no tiene el id que llega (needsRewrite). Vuelve a leer la hoja después.
* @param h buffer.Handle, n node: la hoja, bloqueada en exclusivo; sigue bloqueada al volver
* @param id []byte: el id que se va a escribir (nil = la limpieza)
* @return node: la hoja reescrita, bool: false si no cabe (no escribió nada: hay que partirla), error
**/
func (t *BTree) rewriteLeaf(h buffer.Handle, n node, id []byte) (node, bool, error) {
	c, err := readLeafContent(n.p)
	if err != nil {
		return n, false, corruptAt(n.blk, "%v", err)
	}
	prefixLen, ok := rewritePrefix(c, id)
	if !ok {
		return n, false, nil
	}
	refs := []wal.BlockRef{{Delta: le.AppendUint16(nil, uint16(prefixLen))}}
	res, err := t.logChange(wal.KindLeafRewrite, 0, 0, refs, []buffer.Handle{h}, nil)
	if err != nil {
		return n, false, err
	}
	if err := res.Write(); err != nil {
		return n, false, err
	}
	t.rewritten.Add(1)
	nn, err := readNode(n.blk, h.Page())
	return nn, true, err
}

/**
* needsRewrite dice si una escritura de id tiene que reescribir antes la hoja: es de la forma 1 (migración perezosa)
* o id no tiene su prefijo.
* @param n node: una hoja, id []byte
* @return bool
**/
func needsRewrite(n node, id []byte) bool {
	return !n.v2 || !bytes.HasPrefix(id, n.pre)
}

/**
* Rewritten es cuántas hojas reescribió el árbol en la forma 2 desde que se abrió (KindLeafRewrite: migración desde el
* formato 1 y prefijos acortados).
* @return uint64
**/
func (t *BTree) Rewritten() uint64 {
	return t.rewritten.Load()
}

/**
* LeafVersions lee las versiones de una hoja de cualquiera de las dos formas, sin el valor en línea (para mirar una
* página leída aparte: pruebas y herramientas). Una versión congelada de la forma 2 sin desborde da XID 0.
* @param p page.Page: una hoja viva
* @return []Version, error: ErrCorrupt si no es una hoja que se pueda leer
**/
func LeafVersions(p page.Page) ([]Version, error) {
	c, err := readLeafContent(p)
	if err != nil {
		return nil, err
	}
	vs := make([]Version, len(c.entries))
	for i, e := range c.entries {
		vs[i] = e.header()
	}
	return vs, nil
}
