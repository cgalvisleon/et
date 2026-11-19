package page

// MaxLines es el máximo de punteros a ítems que caben en una página (todo el espacio menos el encabezado).
const MaxLines = (Size - HeaderSize) / ItemIDSize

// ItemID es un puntero a ítem tal como está en la página: lp_off en los bits 0–14, lp_flags en 15–16 y lp_len en
// 17–31.
type ItemID uint32

// Off devuelve lp_off: la posición del ítem desde el inicio de la página.
func (id ItemID) Off() int { return int(id & 0x7FFF) }

// Flags devuelve lp_flags.
func (id ItemID) Flags() LineFlags { return LineFlags(id>>15) & 3 }

// Len devuelve lp_len: el largo del ítem sin la alineación.
func (id ItemID) Len() int { return int(id >> 17) }

// HasStorage dice si el puntero tiene bytes en la página (lp_len > 0), como ItemIdHasStorage.
func (id ItemID) HasStorage() bool { return id.Len() > 0 }

/**
* makeItemID arma un puntero a ítem.
* @param off int, flags LineFlags, n int: lp_off, lp_flags y lp_len
* @return ItemID
**/
func makeItemID(off int, flags LineFlags, n int) ItemID {
	return ItemID(uint32(off)&0x7FFF | uint32(flags&3)<<15 | uint32(n)<<17)
}

/**
* MaxOffset devuelve el número del último puntero a ítem (0 si no hay), como PageGetMaxOffsetNumber.
* @return OffsetNumber: nunca más de MaxLines, aunque pd_lower esté dañado
**/
func (p Page) MaxOffset() OffsetNumber {
	lower := min(p.Lower(), Size)
	if lower <= HeaderSize {
		return 0
	}
	return OffsetNumber((lower - HeaderSize) / ItemIDSize)
}

// linePos es la posición en la página del puntero número off.
func linePos(off OffsetNumber) int { return HeaderSize + int(off-1)*ItemIDSize }

/**
* ItemID devuelve el puntero número off.
* @param off OffsetNumber: de FirstOffset a MaxOffset
* @return ItemID, error: ErrInvalidOffset fuera de rango
**/
func (p Page) ItemID(off OffsetNumber) (ItemID, error) {
	if off < FirstOffset || off > p.MaxOffset() {
		return 0, ErrInvalidOffset
	}
	return p.line(off), nil
}

// line lee el puntero off, ya validado.
func (p Page) line(off OffsetNumber) ItemID { return ItemID(le.Uint32(p[linePos(off):])) }

// setLine escribe el puntero off, ya validado.
func (p Page) setLine(off OffsetNumber, id ItemID) { le.PutUint32(p[linePos(off):], uint32(id)) }

/**
* Item devuelve los bytes del ítem off, dentro de la página (sin copiar; la capacidad termina en el ítem). Sirve
* para ítems normales y muertos con contenido. Revisa los límites: un puntero dañado da error, nunca pánico.
* @param off OffsetNumber
* @return []byte, error: ErrInvalidOffset, ErrNoItem (sin contenido) o ErrCorrupt
**/
func (p Page) Item(off OffsetNumber) ([]byte, error) {
	id, err := p.ItemID(off)
	if err != nil {
		return nil, err
	}
	start, end, err := p.storage(id)
	if err != nil {
		return nil, err
	}
	return p[start:end:end], nil
}

/**
* storage valida dónde están los bytes de un puntero: alineado, entre pd_upper y pd_special, dentro de la página.
* @param id ItemID
* @return int, int, error: inicio y fin del ítem; ErrNoItem sin contenido, ErrCorrupt fuera de lugar
**/
func (p Page) storage(id ItemID) (int, int, error) {
	if !id.HasStorage() || (id.Flags() != LineNormal && id.Flags() != LineDead) {
		return 0, 0, ErrNoItem
	}
	start, end := id.Off(), id.Off()+id.Len()
	if start < HeaderSize || start < p.Upper() || start%ItemAlign != 0 || end > p.Special() || end > Size {
		return 0, 0, corrupt("item at %d len %d outside [%d, %d)", start, id.Len(), p.Upper(), p.Special())
	}
	return start, end, nil
}

/**
* AddItem agrega un ítem sin elegir lugar: reusa el primer puntero libre si FlagHasFreeLines lo indica (y la apaga
* si ya no hay), si no lo pone al final. No toca FlagPageFull: eso lo decide quien llama.
* @param item []byte: de 1 byte en adelante
* @return OffsetNumber, error: el número del ítem; ErrEmptyItem, ErrNoSpace o ErrCorrupt
**/
func (p Page) AddItem(item []byte) (OffsetNumber, error) {
	if err := p.checkLayout(); err != nil {
		return 0, err
	}
	if p.Flags()&FlagHasFreeLines != 0 {
		for off := FirstOffset; off <= p.MaxOffset(); off++ {
			if p.line(off).Flags() == LineUnused {
				return off, p.placeItem(item, off, false)
			}
		}
		p.SetFlags(p.Flags() &^ FlagHasFreeLines)
	}
	off := p.MaxOffset() + 1
	return off, p.placeItem(item, off, true)
}

/**
* InsertItem pone un ítem en el número off y corre una posición los que estaban desde ahí (como el árbol B, que
* guarda sus ítems en orden). off = MaxOffset + 1 lo agrega al final.
* @param item []byte: de 1 byte en adelante
* @param off OffsetNumber: de FirstOffset a MaxOffset + 1
* @return error: ErrEmptyItem, ErrInvalidOffset, ErrNoSpace o ErrCorrupt
**/
func (p Page) InsertItem(item []byte, off OffsetNumber) error {
	if err := p.checkLayout(); err != nil {
		return err
	}
	if off < FirstOffset || off > p.MaxOffset()+1 {
		return ErrInvalidOffset
	}
	return p.placeItem(item, off, true)
}

/**
* placeItem guarda el ítem justo debajo de pd_upper y lo apunta desde off, con un puntero nuevo (corriendo los
* siguientes) o en uno libre que ya existe. La página ya pasó checkLayout.
* @param item []byte, off OffsetNumber, newLine bool
* @return error: ErrEmptyItem o ErrNoSpace
**/
func (p Page) placeItem(item []byte, off OffsetNumber, newLine bool) error {
	n := len(item)
	if n == 0 {
		return ErrEmptyItem
	}
	lower, upper := p.Lower(), p.Upper()-Align(n)
	if newLine {
		lower += ItemIDSize
	}
	if n > Size || lower > upper {
		return ErrNoSpace
	}
	if newLine && off <= p.MaxOffset() {
		pos := linePos(off)
		copy(p[pos+ItemIDSize:lower], p[pos:lower-ItemIDSize])
	}
	copy(p[upper:], item)
	clear(p[upper+n : upper+Align(n)])
	p.setLine(off, makeItemID(upper, LineNormal, n))
	p.setU16(OffLower, lower)
	p.setU16(OffUpper, upper)
	return nil
}

/**
* ReplaceItem cambia el contenido del ítem off por item, que puede ser de otro largo, sin cambiar su número ni sus
* lp_flags; mueve los ítems de más abajo para no dejar hueco, como PageIndexTupleOverwrite.
* @param off OffsetNumber: un ítem con contenido
* @param item []byte: el contenido nuevo, de 1 byte en adelante
* @return error: ErrInvalidOffset, ErrNoItem, ErrEmptyItem, ErrNoSpace o ErrCorrupt
**/
func (p Page) ReplaceItem(off OffsetNumber, item []byte) error {
	if err := p.checkLayout(); err != nil {
		return err
	}
	id, err := p.ItemID(off)
	if err != nil {
		return err
	}
	start, _, err := p.storage(id)
	if err != nil {
		return err
	}
	n := len(item)
	if n == 0 {
		return ErrEmptyItem
	}
	diff := Align(id.Len()) - Align(n)
	if n > Size || -diff > p.ExactFreeSpace() {
		return ErrNoSpace
	}
	if diff != 0 {
		p.shiftItems(start, diff)
	}
	start += diff
	copy(p[start:], item)
	clear(p[start+n : start+Align(n)])
	p.setLine(off, makeItemID(start, id.Flags(), n))
	return nil
}

/**
* DeleteItem quita el ítem off y su puntero; los siguientes bajan un número y los bytes de más abajo suben para
* cerrar el hueco, como PageIndexTupleDelete.
* @param off OffsetNumber
* @return error: ErrInvalidOffset o ErrCorrupt
**/
func (p Page) DeleteItem(off OffsetNumber) error {
	if err := p.checkLayout(); err != nil {
		return err
	}
	id, err := p.ItemID(off)
	if err != nil {
		return err
	}
	start, size := 0, 0
	if id.HasStorage() {
		if start, _, err = p.storage(id); err != nil {
			return err
		}
		size = Align(id.Len())
	}
	lower := p.Lower()
	pos := linePos(off)
	copy(p[pos:lower-ItemIDSize], p[pos+ItemIDSize:lower])
	p.setU16(OffLower, lower-ItemIDSize)
	if size > 0 {
		p.shiftItems(start, size)
	}
	return nil
}

/**
* DeleteItems quita varios ítems, como PageIndexMultiDelete: de mayor a menor, así los números de los que faltan
* no cambian.
* @param offs []OffsetNumber: en orden estrictamente ascendente
* @return error: ErrUnsorted, o el primero de DeleteItem (los ya quitados quedan quitados)
**/
func (p Page) DeleteItems(offs []OffsetNumber) error {
	for i := 1; i < len(offs); i++ {
		if offs[i] <= offs[i-1] {
			return ErrUnsorted
		}
	}
	for i := len(offs) - 1; i >= 0; i-- {
		if err := p.DeleteItem(offs[i]); err != nil {
			return err
		}
	}
	return nil
}

/**
* shiftItems corre delta bytes los ítems guardados entre pd_upper y at (los de más abajo que at), y sus punteros;
* ajusta pd_upper. delta > 0 libera espacio, delta < 0 lo toma.
* @param at int: posición del ítem que cambia de largo o se quita
* @param delta int: múltiplo de ItemAlign
**/
func (p Page) shiftItems(at, delta int) {
	upper := p.Upper()
	copy(p[upper+delta:at+delta], p[upper:at])
	p.setU16(OffUpper, upper+delta)
	for off := FirstOffset; off <= p.MaxOffset(); off++ {
		id := p.line(off)
		if id.HasStorage() && id.Off() < at && id.Off() >= upper {
			p.setLine(off, makeItemID(id.Off()+delta, id.Flags(), id.Len()))
		}
	}
}

/**
* MarkDead marca muerto el ítem off (LineDead) sin quitar su contenido: una pista para que la limpieza lo quite.
* @param off OffsetNumber: un ítem normal o ya muerto
* @return error: ErrInvalidOffset o ErrNoItem (puntero libre)
**/
func (p Page) MarkDead(off OffsetNumber) error {
	id, err := p.ItemID(off)
	if err != nil {
		return err
	}
	if id.Flags() != LineNormal && id.Flags() != LineDead {
		return ErrNoItem
	}
	p.setLine(off, makeItemID(id.Off(), LineDead, id.Len()))
	return nil
}

/**
* SetUnused libera el puntero off (LineUnused) sin cambiar los números de los demás y prende FlagHasFreeLines. Sus
* bytes quedan como hueco hasta Repair.
* @param off OffsetNumber
* @return error: ErrInvalidOffset
**/
func (p Page) SetUnused(off OffsetNumber) error {
	if _, err := p.ItemID(off); err != nil {
		return err
	}
	p.setLine(off, 0)
	p.SetFlags(p.Flags() | FlagHasFreeLines)
	return nil
}
