package page

import "encoding/binary"

// le es el orden de bytes de todo el formato.
var le = binary.LittleEndian

// pageSizeVersion es el valor de pd_pagesize_version de una página de este formato.
const pageSizeVersion = Size | LayoutVersion

// validFlags son las marcas de pd_flags que el formato conoce; cualquier otra es daño.
const validFlags = FlagHasFreeLines | FlagPageFull | FlagAllVisible

/**
* Init deja la página vacía: todo en ceros, sin ítems y con specialSize bytes de espacio especial al final.
* @param specialSize int: 0 u OpaqueSize (cualquier múltiplo de ItemAlign que deje lugar al encabezado)
* @return error: ErrSize si p no mide Size, ErrSpecialSize si el tamaño no sirve
**/
func (p Page) Init(specialSize int) error {
	if len(p) != Size {
		return ErrSize
	}
	if specialSize < 0 || specialSize > Size-HeaderSize || specialSize%ItemAlign != 0 {
		return ErrSpecialSize
	}
	clear(p)
	special := Size - specialSize
	p.setU16(OffLower, HeaderSize)
	p.setU16(OffUpper, special)
	p.setU16(OffSpecial, special)
	p.setU16(OffPageSizeVersion, pageSizeVersion)
	return nil
}

/**
* IsNew dice si la página nunca se inició (pd_upper = 0), como PageIsNew.
* @return bool
**/
func (p Page) IsNew() bool {
	return p.Upper() == 0
}

// LSN devuelve pd_lsn.
func (p Page) LSN() LSN { return LSN(le.Uint64(p[OffLSN:])) }

// SetLSN pone pd_lsn.
func (p Page) SetLSN(lsn LSN) { le.PutUint64(p[OffLSN:], uint64(lsn)) }

// Checksum devuelve la suma guardada en pd_checksum (ComputeChecksum la calcula).
func (p Page) Checksum() uint16 { return le.Uint16(p[OffChecksum:]) }

// Flags devuelve pd_flags.
func (p Page) Flags() uint16 { return le.Uint16(p[OffFlags:]) }

// SetFlags reemplaza pd_flags.
func (p Page) SetFlags(flags uint16) { p.setU16(OffFlags, int(flags)) }

// Lower devuelve pd_lower: fin del arreglo de punteros e inicio del espacio libre.
func (p Page) Lower() int { return int(le.Uint16(p[OffLower:])) }

// Upper devuelve pd_upper: fin del espacio libre e inicio de los ítems.
func (p Page) Upper() int { return int(le.Uint16(p[OffUpper:])) }

// Special devuelve pd_special: inicio del espacio especial (Size si la página no tiene).
func (p Page) Special() int { return int(le.Uint16(p[OffSpecial:])) }

// PruneXID devuelve pd_prune_xid.
func (p Page) PruneXID() XID { return XID(le.Uint64(p[OffPruneXID:])) }

// SetPruneXID pone pd_prune_xid.
func (p Page) SetPruneXID(xid XID) { le.PutUint64(p[OffPruneXID:], uint64(xid)) }

/**
* SetLower mueve pd_lower en una página sin punteros a ítems que guarda sus datos desde el byte HeaderSize (meta,
* desborde): así la imagen de página entera del WAL deja fuera el hueco entre pd_lower y pd_upper.
* @param lower int: entre HeaderSize y pd_upper
* @return error: ErrInvalidOffset fuera de ese rango
**/
func (p Page) SetLower(lower int) error {
	if lower < HeaderSize || lower > p.Upper() {
		return ErrInvalidOffset
	}
	p.setU16(OffLower, lower)
	return nil
}

/**
* FreeSpace es el espacio para un ítem nuevo: el hueco libre menos un puntero, como PageGetFreeSpace.
* @return int: 0 si no queda lugar (o el encabezado está dañado)
**/
func (p Page) FreeSpace() int {
	return max(p.ExactFreeSpace()-ItemIDSize, 0)
}

/**
* ExactFreeSpace es el hueco entre pd_lower y pd_upper, como PageGetExactFreeSpace.
* @return int: 0 si el encabezado está dañado
**/
func (p Page) ExactFreeSpace() int {
	return max(p.Upper()-p.Lower(), 0)
}

/**
* Align redondea n hacia arriba al múltiplo de ItemAlign: lo que ocupa un ítem de n bytes.
* @param n int
* @return int
**/
func Align(n int) int {
	return (n + ItemAlign - 1) &^ (ItemAlign - 1)
}

/**
* SpaceFor es lo que consume en la página agregar un ítem de n bytes con un puntero nuevo.
* @param n int: largo del ítem
* @return int
**/
func SpaceFor(n int) int {
	return Align(n) + ItemIDSize
}

/**
* checkHeader valida el encabezado sin confiar en él, como PageIsVerified: versión, marcas conocidas y
* HeaderSize ≤ pd_lower ≤ pd_upper ≤ pd_special ≤ Size con pd_special alineado. No mira los punteros a ítems.
* @return error: ErrSize o ErrCorrupt con el detalle
**/
func (p Page) checkHeader() error {
	if len(p) != Size {
		return ErrSize
	}
	if v := le.Uint16(p[OffPageSizeVersion:]); v != pageSizeVersion {
		return corrupt("pd_pagesize_version %#x", v)
	}
	if f := p.Flags(); f&^validFlags != 0 {
		return corrupt("pd_flags %#x", f)
	}
	lower, upper, special := p.Lower(), p.Upper(), p.Special()
	if lower < HeaderSize || lower > upper || upper > special || special > Size || special%ItemAlign != 0 {
		return corrupt("pd_lower %d pd_upper %d pd_special %d", lower, upper, special)
	}
	return nil
}

/**
* checkLayout valida el encabezado de una página con punteros a ítems: además de checkHeader, pd_lower cae en el
* borde de un puntero y pd_upper está alineado. Lo usan las funciones que cambian ítems antes de tocar nada.
* @return error
**/
func (p Page) checkLayout() error {
	if err := p.checkHeader(); err != nil {
		return err
	}
	if (p.Lower()-HeaderSize)%ItemIDSize != 0 || p.Upper()%ItemAlign != 0 {
		return corrupt("pd_lower %d pd_upper %d not aligned", p.Lower(), p.Upper())
	}
	return nil
}

// setU16 escribe un uint16 del encabezado.
func (p Page) setU16(off, v int) { le.PutUint16(p[off:], uint16(v)) }
