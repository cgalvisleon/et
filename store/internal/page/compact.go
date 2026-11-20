package page

import "slices"

// stored es un ítem con contenido: su número y dónde está.
type stored struct {
	off   OffsetNumber
	start uint16
	n     uint16
}

/**
* storedItems junta los ítems con contenido, los ordena por posición de mayor a menor y comprueba que cada uno está
* en su lugar (storage) y que no se pisan. Revisa también los punteros sin contenido: libres en cero, sin
* LineRedirect. Usa buf (en la pila de quien llama) para no asignar memoria.
* @param buf *[MaxLines]stored
* @return []stored, int, error: los ítems, cuántos punteros están libres y ErrCorrupt si algo está mal
**/
func (p Page) storedItems(buf *[MaxLines]stored) ([]stored, int, error) {
	items := buf[:0]
	unused := 0
	for off := FirstOffset; off <= p.MaxOffset(); off++ {
		id := p.line(off)
		switch {
		case id.Flags() == LineRedirect:
			return nil, 0, corrupt("item %d is a redirect", off)
		case id.Flags() == LineUnused:
			if id != 0 {
				return nil, 0, corrupt("unused item %d is not zero", off)
			}
			unused++
		case !id.HasStorage():
			if id.Off() != 0 {
				return nil, 0, corrupt("item %d without storage points to %d", off, id.Off())
			}
		default:
			start, _, err := p.storage(id)
			if err != nil {
				return nil, 0, err
			}
			items = append(items, stored{off: off, start: uint16(start), n: uint16(id.Len())})
		}
	}
	slices.SortFunc(items, func(a, b stored) int { return int(b.start) - int(a.start) })
	for i := 1; i < len(items); i++ {
		if int(items[i].start)+int(items[i].n) > int(items[i-1].start) {
			return nil, 0, corrupt("items %d and %d overlap", items[i].off, items[i-1].off)
		}
	}
	return items, unused, nil
}

/**
* CheckItems valida los punteros a ítems de una página que los tiene (no meta ni desborde): encabezado, cada ítem
* alineado entre pd_upper y pd_special, ninguno pisa a otro, libres en cero. Verify no lo hace porque hay páginas sin
* ítems; quien lee una página de ítems del disco puede llamarlo después. No asigna memoria.
* @return error: ErrCorrupt con el detalle
**/
func (p Page) CheckItems() error {
	if err := p.checkLayout(); err != nil {
		return err
	}
	// Las mismas comprobaciones que storedItems, pero los pisados con un mapa de bits de la página en unidades de
	// ItemAlign (128 bytes en la pila) en lugar de ordenar: el árbol lo llama en cada bloqueo (tarea 100 de F-23)
	var used [Size / ItemAlign / 64]uint64
	for off := FirstOffset; off <= p.MaxOffset(); off++ {
		id := p.line(off)
		switch {
		case id.Flags() == LineRedirect:
			return corrupt("item %d is a redirect", off)
		case id.Flags() == LineUnused:
			if id != 0 {
				return corrupt("unused item %d is not zero", off)
			}
		case !id.HasStorage():
			if id.Off() != 0 {
				return corrupt("item %d without storage points to %d", off, id.Off())
			}
		default:
			start, end, err := p.storage(id)
			if err != nil {
				return err
			}
			if !markUsed(&used, start/ItemAlign, (end+ItemAlign-1)/ItemAlign) {
				return corrupt("item %d overlaps another", off)
			}
		}
	}
	return nil
}

/**
* markUsed marca las unidades [from, to) en used.
* @param used *[Size / ItemAlign / 64]uint64, from, to int
* @return bool: false si alguna ya estaba marcada (dos ítems se pisan)
**/
func markUsed(used *[Size / ItemAlign / 64]uint64, from, to int) bool {
	for u := from; u < to; {
		w, b := u/64, u%64
		n := min(64-b, to-u)
		mask := ^uint64(0)
		if n < 64 {
			mask = (uint64(1)<<n - 1) << b
		}
		if used[w]&mask != 0 {
			return false
		}
		used[w] |= mask
		u += n
	}
	return true
}

/**
* Repair compacta los ítems contra el espacio especial para juntar todo el hueco entre pd_lower y pd_upper, sin
* cambiar números ni lp_flags, como PageRepairFragmentation. Pone o apaga FlagHasFreeLines según haya punteros
* libres. No asigna memoria.
* @return error: ErrCorrupt si los punteros están dañados (entonces no cambia nada)
**/
func (p Page) Repair() error {
	if err := p.checkLayout(); err != nil {
		return err
	}
	var buf [MaxLines]stored
	items, unused, err := p.storedItems(&buf)
	if err != nil {
		return err
	}
	// De la posición más alta a la más baja: cada ítem baja hacia el final y nunca pisa uno que falte mover.
	upper := p.Special()
	for _, it := range items {
		n := int(it.n)
		upper -= Align(n)
		copy(p[upper:upper+n], p[it.start:int(it.start)+n])
		clear(p[upper+n : upper+Align(n)])
		p.setLine(it.off, makeItemID(upper, p.line(it.off).Flags(), n))
	}
	p.setU16(OffUpper, upper)
	if unused > 0 {
		p.SetFlags(p.Flags() | FlagHasFreeLines)
	} else {
		p.SetFlags(p.Flags() &^ FlagHasFreeLines)
	}
	return nil
}
