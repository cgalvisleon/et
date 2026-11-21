package crashfs

import "maps"

// fileOp es una escritura (data en off) o, si truncate, un cambio de largo a off.
type fileOp struct {
	off      int64
	data     []byte
	truncate bool
}

// apply devuelve buf con op aplicada: una escritura más allá del final rellena el hueco con ceros.
func (op fileOp) apply(buf []byte) []byte {
	if op.truncate {
		return resize(buf, op.off)
	}
	end := op.off + int64(len(op.data))
	if end > int64(len(buf)) {
		buf = resize(buf, end)
	}
	copy(buf[op.off:], op.data)
	return buf
}

// place devuelve el lugar asignado tras op: una escritura ocupa los bytes que escribe (también dentro de un hueco);
// un truncado libera lo que corta y, si agranda, no ocupa nada (deja un hueco).
func (op fileOp) place(alloc extents) extents {
	if op.truncate {
		return alloc.cut(op.off)
	}
	return alloc.add(op.off, op.off+int64(len(op.data)))
}

// resize corta buf o lo extiende con ceros hasta size.
func resize(buf []byte, size int64) []byte {
	if size <= int64(len(buf)) {
		return buf[:size]
	}
	return append(buf, make([]byte, size-int64(len(buf)))...)
}

// sectors parte una escritura en trozos que no cruzan un límite de sector del archivo.
func (op fileOp) sectors(size int64) []fileOp {
	var pieces []fileOp
	for start := int64(0); start < int64(len(op.data)); {
		off := op.off + start
		end := min((off/size+1)*size-op.off, int64(len(op.data)))
		pieces = append(pieces, fileOp{off: off, data: op.data[start:end]})
		start = end
	}
	return pieces
}

// inode es el contenido de un archivo, independiente de sus nombres.
type inode struct {
	data    []byte   // lo que ven las lecturas
	durable []byte   // el contenido en el último Sync
	pending []fileOp // escrituras y truncados desde el último Sync, en orden

	alloc        extents // los bytes con lugar asignado (escritos): un hueco no ocupa disco
	durableAlloc extents // alloc en el último Sync
}

// change aplica op al contenido actual y a su lugar asignado, y la anota como pendiente de Sync.
func (n *inode) change(op fileOp) {
	n.data = op.apply(n.data)
	n.alloc = op.place(n.alloc)
	n.pending = append(n.pending, op)
}

// sync vuelve durables las operaciones pendientes. No copia data: tras un Sync fallido pueden diferir.
func (n *inode) sync() {
	for _, op := range n.pending {
		n.durable = op.apply(n.durable)
	}
	n.durableAlloc = n.alloc.clone()
	n.pending = nil
}

// dropPending olvida las operaciones pendientes sin volverlas durables (un Sync que falló).
func (n *inode) dropPending() {
	n.pending = nil
}

// reset deja el nodo en data con el lugar asignado alloc, todo durable (el estado tras un corte).
func (n *inode) reset(data []byte, alloc extents) {
	n.data = data
	n.durable = append([]byte(nil), data...)
	n.alloc, n.durableAlloc = alloc, alloc.clone()
	n.pending = nil
}

// entryChange da de alta (node != nil) o de baja (node == nil) un nombre de un directorio.
type entryChange struct {
	name string
	node *inode
}

// dirOp es un cambio atómico de un directorio: un alta, una baja o un renombre dentro de él.
type dirOp []entryChange

// apply aplica el cambio a las entradas.
func (op dirOp) apply(entries map[string]*inode) {
	for _, c := range op {
		if c.node == nil {
			delete(entries, c.name)
		} else {
			entries[c.name] = c.node
		}
	}
}

// directory son las entradas de un directorio, actuales y durables.
type directory struct {
	live    map[string]*inode // lo que ven OpenFile y ReadDir
	durable map[string]*inode // las entradas en el último SyncDir
	pending []dirOp           // cambios desde el último SyncDir, en orden
}

// newDirectory crea un directorio vacío.
func newDirectory() *directory {
	return &directory{live: map[string]*inode{}, durable: map[string]*inode{}}
}

// change aplica op a las entradas actuales y lo anota como pendiente de SyncDir.
func (d *directory) change(op ...entryChange) {
	dirOp(op).apply(d.live)
	d.pending = append(d.pending, op)
}

// sync vuelve durables las entradas actuales.
func (d *directory) sync() {
	d.durable = maps.Clone(d.live)
	d.pending = nil
}

// crash deja el directorio en sus entradas durables más los primeros keep cambios pendientes.
func (d *directory) crash(keep int) {
	for _, op := range d.pending[:keep] {
		op.apply(d.durable)
	}
	d.live = maps.Clone(d.durable)
	d.pending = nil
}
