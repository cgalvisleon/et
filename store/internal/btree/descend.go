package btree

import (
	"fmt"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
)

// finishError: una bajada para escribir encontró una página con OpaqueIncompleteSplit. Ya soltó todo lo suyo; quien
// llama suelta lo que tenga, termina esa partición (finishSplit) y repite. Así una escritura nunca trabaja sobre una
// página cuya hermana derecha no tiene enlace en el padre (como _bt_moveright con forupdate en nbtree), y el padre
// que busca una partición siempre existe.
type finishError struct {
	block page.BlockNumber
}

func (e *finishError) Error() string {
	return fmt.Sprintf("btree: block %d has an incomplete split", e.block)
}

/**
* descend baja desde la raíz (o desde la raíz rápida, si el nivel buscado está por debajo) hasta la página del nivel
* level donde va k, con Lehman-Yao: una sola página bloqueada a la vez (en compartido; la del nivel buscado en mode), y
* en cada una, si k pasa su clave alta (o la página se está borrando), se sigue a la derecha. No hace falta bloquear
* el camino: una página solo se parte hacia la derecha, y una que se borra pasa su espacio de claves a su hermana
* derecha y conserva su enlace a ella hasta reciclarse (doc.go, «Borrado de páginas»), así que lo que se movió de una
* página que alguien leyó está siempre en su cadena de hermanas derechas.
* @param k key, level uint32: 0 = hoja
* @param mode buffer.LockMode: el bloqueo de la página que se devuelve
* @param forWrite bool: una página con partición incompleta en el camino devuelve *finishError
* @return buffer.Handle, node: la página bloqueada; uint32: el nivel de la raíz leído de la meta; error
**/
func (t *BTree) descend(k key, level uint32, mode buffer.LockMode, forWrite bool) (buffer.Handle, node, uint32, error) {
	m, err := t.readMeta()
	if err != nil {
		return nil, node{}, 0, err
	}
	if m.level < level {
		return nil, node{}, 0, t.missingLevel(m, level)
	}
	blk, cur := m.root, m.level
	fast := m.fastRoot != m.root && m.fastRoot != 0 && level <= m.fastLevel && m.fastLevel < m.level
	if fast {
		blk, cur = m.fastRoot, m.fastLevel
	}
	for {
		lock := buffer.Share
		if cur == level {
			lock = mode
		}
		h, n, err := t.lockNode(blk, lock)
		if err != nil {
			return nil, node{}, 0, err
		}
		if fast && !validFastRoot(n, cur) {
			h.Release()
			blk, cur, fast = m.root, m.level, false // la raíz rápida cambió después de leer la meta
			continue
		}
		fast = false
		if n.o.Level != cur {
			h.Release()
			return nil, node{}, 0, corruptAt(blk, "level %d, expected %d", n.o.Level, cur)
		}
		if h, n, err = t.moveRight(h, n, k, lock, forWrite); err != nil {
			return nil, node{}, 0, err
		}
		if cur == level {
			return h, n, m.level, nil
		}
		child, err := n.childFor(k)
		h.Release()
		if err != nil {
			return nil, node{}, 0, err
		}
		blk, cur = child, cur-1
	}
}

/**
* validFastRoot dice si se puede bajar desde la raíz rápida leída de la meta: una página del árbol que no se está
* borrando, del nivel anotado y la de más a la izquierda de su nivel (desde ahí, siguiendo a la derecha, se llega a
* toda clave). Si no, se baja desde la raíz.
* @param n node: la raíz rápida, bloqueada; level uint32: su nivel según la meta
* @return bool
**/
func validFastRoot(n node, level uint32) bool {
	return !n.ignore() && n.o.Level == level && n.o.Prev == 0
}

/**
* missingLevel explica por qué el árbol no tiene el nivel pedido: solo pasa si la raíz se partió y la raíz nueva
* todavía no existe (partición incompleta de la raíz); entonces hay que terminarla.
* @param m meta, level uint32
* @return error: *finishError con la raíz, o ErrCorrupt
**/
func (t *BTree) missingLevel(m meta, level uint32) error {
	h, n, err := t.lockNode(m.root, buffer.Share)
	if err != nil {
		return err
	}
	defer h.Release()
	if n.incomplete() {
		return &finishError{block: m.root}
	}
	return corruptAt(m.root, "tree has no level %d (root level %d)", level, m.level)
}

/**
* moveRight sigue a la derecha mientras k pase la clave alta de la página (Lehman-Yao) o la página se esté borrando
* (medio muerta o borrada: nunca es la de más a la derecha). Suelta cada página antes de bloquear la siguiente: nunca
* tiene dos bloqueadas.
* @param h buffer.Handle, n node: la página bloqueada (se suelta si se avanza o hay error)
* @param k key, mode buffer.LockMode, forWrite bool: ver descend
* @return buffer.Handle, node, error
**/
func (t *BTree) moveRight(h buffer.Handle, n node, k key, mode buffer.LockMode, forWrite bool) (buffer.Handle, node, error) {
	for {
		if forWrite && n.incomplete() {
			h.Release()
			return nil, node{}, &finishError{block: n.blk}
		}
		var err error
		past := n.ignore() // una página que se borra pasó su espacio de claves a la derecha (P_IGNORE de nbtree)
		if !past {
			if past, err = n.pastHighKey(k); err != nil {
				h.Release()
				return nil, node{}, err
			}
		}
		if !past {
			return h, n, nil
		}
		next, level := n.o.Next, n.o.Level
		h.Release()
		if next == 0 {
			return nil, node{}, corruptAt(n.blk, "page being deleted without a right sibling")
		}
		if h, n, err = t.lockNode(next, mode); err != nil {
			return nil, node{}, err
		}
		if n.o.Level != level {
			h.Release()
			return nil, node{}, corruptAt(next, "right sibling at level %d, expected %d", n.o.Level, level)
		}
	}
}
