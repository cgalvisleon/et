package store

import (
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// scanMode dice qué entrega un recorrido en data.
type scanMode int

const (
	scanValues   scanMode = iota // una copia del valor, del llamador
	scanBorrowed                 // el valor en un búfer del recorrido que se reutiliza: vale solo dentro de fn
	scanKeys                     // sin valor (data nil): contar y listar ids
)

// scanRange es lo que recorre un ForEach: los ids de [from, to) en un sentido, salteando offset ids visibles y
// entregando como mucho limit (≤ 0 = sin límite).
type scanRange struct {
	from, to      string // from incluido, to excluido; "" = sin límite de ese lado
	asc           bool
	offset, limit int
}

/**
* prefixRange es el rango de los ids que empiezan con prefix: [prefix, el siguiente prefijo).
* @param prefix string, asc bool, offset, limit int
* @return scanRange
**/
func prefixRange(prefix string, asc bool, offset, limit int) scanRange {
	return scanRange{from: prefix, to: prefixEnd(prefix), asc: asc, offset: offset, limit: limit}
}

/**
* prefixEnd es el menor id mayor que todos los que empiezan con prefix: el prefijo sin los 0xFF finales y con el
* último byte más uno.
* @param prefix string
* @return string: "" si no hay (prefijo vacío o todo 0xFF): sin límite
**/
func prefixEnd(prefix string) string {
	b := []byte(prefix)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] < 0xff {
			b[i]++
			return string(b[:i+1])
		}
	}
	return ""
}

/**
* cursorStart es desde dónde baja el cursor: from hacia adelante; hacia atrás, to (incluido en btree: el recorrido
* saltea ese id, que queda fuera del rango).
* @return string
**/
func (r scanRange) cursorStart() string {
	if r.asc {
		return r.from
	}
	return r.to
}

/**
* place dice dónde cae id respecto del rango en el sentido del recorrido.
* @param id string
* @return bool: si va, bool: si ya se pasó el rango (no queda nada más)
**/
func (r scanRange) place(id string) (bool, bool) {
	tooHigh := r.to != "" && id >= r.to
	tooLow := id < r.from
	if r.asc {
		return !tooHigh && !tooLow, tooHigh
	}
	return !tooHigh && !tooLow, tooLow
}

// scanner es un recorrido en curso: el store, el snapshot que decide la visibilidad y el cursor del árbol. Entre una
// hoja y la siguiente no tiene nada tomado: fn corre sin páginas fijadas ni locks del motor.
type scanner struct {
	s      *Store
	v      *Snapshot
	mode   scanMode
	r      scanRange
	cursor *btree.Cursor
}

/**
* scanAt recorre el store como lo ve el snapshot v y llama a fn con cada id visible del rango, en orden. Cada hoja
* se lee con v, el motor y el store tomados en lectura (Close de v, DropStore y Close del motor esperan a esa hoja),
* y se suelta antes de llamar a fn; los valores de desborde se leen igual, uno a la vez. fn puede leer y escribir en
* el motor: el recorrido no ve lo escrito después de su snapshot.
* @param v *Snapshot, r scanRange, mode scanMode
* @param fn func(id string, data []byte) (bool, error): false para parar
* @return error: el de fn, ErrOtherEngine, ErrTxDone o ErrTimeout si v terminó, ErrStoreDropped, ErrClosed…
**/
func (s *Store) scanAt(v *Snapshot, r scanRange, mode scanMode, fn func(id string, data []byte) (bool, error)) error {
	if v.e != s.e {
		return ErrOtherEngine
	}
	sc := &scanner{s: s, v: v, mode: mode, r: r}
	skip, sent := max(r.offset, 0), 0
	for {
		batch, more, err := sc.leaf()
		if err != nil {
			return err
		}
		for _, ver := range batch {
			in, past := r.place(ver.ID)
			if past {
				return nil
			}
			if !in {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			if cont, err := sc.emit(ver, fn); err != nil || !cont {
				return err
			}
			if sent++; r.limit > 0 && sent >= r.limit {
				return nil
			}
		}
		if !more {
			return nil
		}
	}
}

/**
* leaf lee la hoja siguiente con el snapshot, el motor y el store tomados en lectura; la primera vez crea el cursor.
* @return []btree.Version: del cursor, válidas hasta la próxima hoja, bool: si quedan hojas, error
**/
func (sc *scanner) leaf() ([]btree.Version, bool, error) {
	var batch []btree.Version
	var more bool
	err := sc.locked(func() error {
		if sc.cursor == nil {
			sc.cursor = sc.s.tree.NewCursor(btree.ScanOptions{
				From:     sc.r.cursorStart(),
				Snapshot: &sc.v.snap,
				Backward: !sc.r.asc,
				Ring:     true,
				KeysOnly: sc.mode == scanKeys,
				Borrow:   sc.mode == scanBorrowed,
			})
		}
		var err error
		batch, more, err = sc.cursor.Next()
		return sc.s.e.readError(err)
	})
	return batch, more, err
}

/**
* locked corre f con el snapshot vivo (en lectura: no se cierra mientras tanto) y el store abierto (begin).
* @param f func() error
* @return error
**/
func (sc *scanner) locked(f func() error) error {
	sc.v.mu.RLock()
	defer sc.v.mu.RUnlock()
	if sc.v.err != nil {
		return sc.v.err
	}
	if err := sc.s.begin(); err != nil {
		return err
	}
	defer sc.s.end()
	return f()
}

/**
* emit llama a fn con una versión: sin valor (scanKeys), el de la hoja, o el leído de su cadena de desborde (con el
* snapshot y el store tomados, como la hoja).
* @param ver btree.Version, fn func(id string, data []byte) (bool, error)
* @return bool: si sigue, error
**/
func (sc *scanner) emit(ver btree.Version, fn func(id string, data []byte) (bool, error)) (bool, error) {
	var data []byte
	switch {
	case sc.mode == scanKeys:
	case ver.Flags&btree.EntryOverflow != 0:
		err := sc.locked(func() error {
			var err error
			data, err = sc.s.valueOf(ver)
			return sc.s.e.readError(err)
		})
		if err != nil {
			return false, err
		}
	default:
		data = ver.Value
		if data == nil {
			data = []byte{}
		}
	}
	return fn(ver.ID, data)
}

/**
* forEachAt recorre el rango con el snapshot v entregando copias de los valores.
* @param v *Snapshot, r scanRange, fn func(id string, data []byte) (bool, error)
* @return error
**/
func (s *Store) forEachAt(v *Snapshot, r scanRange, fn func(id string, data []byte) (bool, error)) error {
	return s.scanAt(v, r, scanValues, fn)
}

/**
* withSnapshot corre f con un snapshot propio, sin tiempo máximo, y lo cierra al terminar (también si f entra en
* pánico): lo que ve un recorrido suelto no cambia mientras recorre.
* @param f func(v *Snapshot) error
* @return error
**/
func (s *Store) withSnapshot(f func(v *Snapshot) error) error {
	v := s.e.snapshotFor(0)
	defer v.close()
	if v.err != nil {
		return v.err
	}
	return f(v)
}

/**
* forEach recorre el rango con un snapshot propio.
* @param r scanRange, mode scanMode, fn func(id string, data []byte) (bool, error)
* @return error
**/
func (s *Store) forEach(r scanRange, mode scanMode, fn func(id string, data []byte) (bool, error)) error {
	return s.withSnapshot(func(v *Snapshot) error {
		return s.scanAt(v, r, mode, fn)
	})
}

/**
* keys devuelve los ids del rango con un snapshot propio.
* @param r scanRange
* @return []string: nunca nil, error
**/
func (s *Store) keys(r scanRange) ([]string, error) {
	ids := []string{}
	err := s.forEach(r, scanKeys, func(id string, _ []byte) (bool, error) {
		ids = append(ids, id)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

/**
* keysAfter devuelve hasta limit ids mayores que after, en orden, con un snapshot propio.
* @param after string: "" = desde el primero, limit int: ≤ 0 = sin límite
* @return []string, error
**/
func (s *Store) keysAfter(after string, limit int) ([]string, error) {
	ids := []string{}
	err := s.forEach(scanRange{from: after, asc: true}, scanKeys, func(id string, _ []byte) (bool, error) {
		if id == after {
			return true, nil
		}
		ids = append(ids, id)
		return limit <= 0 || len(ids) < limit, nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

/**
* countAt cuenta los ids visibles para v recorriendo el store entero (O(n), sin leer valores).
* @param v *Snapshot
* @return int64, error
**/
func (s *Store) countAt(v *Snapshot) (int64, error) {
	var n int64
	err := s.scanAt(v, scanRange{asc: true}, scanKeys, func(string, []byte) (bool, error) {
		n++
		return true, nil
	})
	return n, err
}

/**
* existsWith dice si id tiene una versión viva visible para snap (nil = la más nueva confirmada), sin leer el valor.
* @param id string, snap *xact.Snapshot
* @return bool, error
**/
func (s *Store) existsWith(id string, snap *xact.Snapshot) (bool, error) {
	if err := checkID(id); err != nil {
		return false, err
	}
	if err := s.begin(); err != nil {
		return false, err
	}
	defer s.end()
	defer s.e.observeRead(time.Now())
	found, err := s.tree.Exists(id, snap)
	return found, s.e.readError(err)
}

/**
* exists dice si id existe según lo último confirmado (Exists suelto, sin snapshot, como Read).
* @param id string
* @return bool, error
**/
func (s *Store) exists(id string) (bool, error) {
	return s.existsWith(id, nil)
}

/**
* existsIn dice si id existe para el lote b: su propio cambio, o la foto tomada al crearlo.
* @param b *Batch, id string
* @return bool, error
**/
func (s *Store) existsIn(b *Batch, id string) (bool, error) {
	if err := checkID(id); err != nil {
		return false, err
	}
	if err := b.acquire(s.e); err != nil {
		return false, err
	}
	defer b.mu.Unlock()
	return s.existsWith(id, &b.snap)
}

/**
* existsAt dice si id existe para el snapshot v.
* @param v *Snapshot, id string
* @return bool, error: ErrOtherEngine, ErrTxDone, ErrTimeout…
**/
func (s *Store) existsAt(v *Snapshot, id string) (bool, error) {
	if v.e != s.e {
		return false, ErrOtherEngine
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.err != nil {
		return false, v.err
	}
	return s.existsWith(id, &v.snap)
}
