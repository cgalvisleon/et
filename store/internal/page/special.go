package page

/**
* SpecialSize es el largo del espacio especial (Size - pd_special).
* @return int: 0 si no tiene o si pd_special está dañado
**/
func (p Page) SpecialSize() int {
	return max(Size-p.Special(), 0)
}

/**
* Opaque lee el espacio especial de una página de un store.
* @return Opaque, error: ErrNoOpaque si el espacio especial no mide OpaqueSize
**/
func (p Page) Opaque() (Opaque, error) {
	if p.Special() != Size-OpaqueSize {
		return Opaque{}, ErrNoOpaque
	}
	s := p[Size-OpaqueSize:]
	return Opaque{
		Prev:    BlockNumber(le.Uint32(s[OffOpaquePrev:])),
		Next:    BlockNumber(le.Uint32(s[OffOpaqueNext:])),
		Level:   le.Uint32(s[OffOpaqueLevel:]),
		Flags:   le.Uint16(s[OffOpaqueFlags:]),
		CycleID: le.Uint16(s[OffOpaqueCycleID:]),
	}, nil
}

/**
* SetOpaque escribe el espacio especial de una página de un store.
* @param o Opaque
* @return error: ErrNoOpaque si el espacio especial no mide OpaqueSize
**/
func (p Page) SetOpaque(o Opaque) error {
	if p.Special() != Size-OpaqueSize {
		return ErrNoOpaque
	}
	s := p[Size-OpaqueSize:]
	le.PutUint32(s[OffOpaquePrev:], uint32(o.Prev))
	le.PutUint32(s[OffOpaqueNext:], uint32(o.Next))
	le.PutUint32(s[OffOpaqueLevel:], o.Level)
	le.PutUint16(s[OffOpaqueFlags:], o.Flags)
	le.PutUint16(s[OffOpaqueCycleID:], o.CycleID)
	return nil
}
