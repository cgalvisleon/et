package btree

import (
	"bytes"
	"encoding/binary"
	"sync/atomic"
	"unsafe"

	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
)

// le es el orden de bytes de todo el formato.
var le = binary.LittleEndian

// knownEntryFlags son las marcas que el formato conoce; cualquier otra es daño.
const knownEntryFlags = EntryTombstone | EntryOverflow | EntryXidCommitted | EntryXidAborted | EntryFrozen | EntryFreeing |
	EntryCompressed

// Entrada de la forma 2 (hojas con LayoutVersion 2, formato 2; doc.go, «Entrada de hoja de la forma 2»): word0(4) =
// marcas (bits 0–7) | largo del sufijo (bits 8–19) | largo en línea (bits 20–31), lsn(8) y, salvo en una versión
// congelada sin desborde, xid(8); después el sufijo del id y el valor en línea o la referencia first(4) len(4).
const (
	entry2Header     = 20 // con xid
	entry2Frozen     = 12 // congelada en línea: sin xid
	entry2RefSize    = 8  // first(4) len(4)
	off2LSN          = 4
	off2XID          = 12
	word0FlagMask    = 0xFF
	word0SuffixShift = 8
	word0InlineShift = 20
	word0LenMask     = 0xFFF
)

// entry es una entrada de hoja (una versión) ya validada por parseEntry o parseEntry2. Es una vista: si viene de una
// página, vale mientras la página siga bloqueada. En la forma 2 el id es pre (el prefijo de la hoja) + el sufijo
// guardado en la entrada.
type entry struct {
	b      []byte // el ítem
	pre    []byte // forma 2: el prefijo de la hoja (vista); nil en la forma 1
	w0     uint32 // forma 2: word0 leída al validar (los largos no cambian nunca; las marcas se releen con flags)
	sfxOff uint16 // b[sfxOff:sfxEnd] es lo que la entrada guarda del id; b[sfxEnd:], el valor o la referencia
	sfxEnd uint16
	v2     bool
}

/**
* parseEntry valida una entrada de hoja de la forma 1 sin confiar en sus bytes: largo mínimo, marcas conocidas, largo
* del id y largo total según sea lápida, desborde o valor en línea.
* @param b []byte: el ítem de la página
* @return entry, error: ErrCorrupt con el detalle
**/
func parseEntry(b []byte) (entry, error) {
	if len(b) < EntryHeaderSize {
		return entry{}, corrupt("leaf entry of %d bytes", len(b))
	}
	e := entry{b: b}
	flags, idLen, valueLen := e.flags(), int(le.Uint16(b[OffEntryIDLen:])), int(le.Uint32(b[OffEntryValueLen:]))
	if flags&^knownEntryFlags != 0 || idLen < 1 || idLen > MaxIDLen {
		return entry{}, corrupt("leaf entry flags %#x id length %d", flags, idLen)
	}
	want := EntryHeaderSize + idLen
	switch {
	case flags&EntryTombstone != 0:
		if valueLen != 0 || flags&(EntryOverflow|EntryFreeing|EntryCompressed) != 0 {
			return entry{}, corrupt("tombstone with a value")
		}
	case flags&EntryOverflow != 0:
		want += OverflowRefSize
	case flags&EntryFreeing != 0:
		return entry{}, corrupt("freeing entry without overflow")
	default:
		want += valueLen
	}
	if len(b) != want {
		return entry{}, corrupt("leaf entry of %d bytes, expected %d", len(b), want)
	}
	e.sfxOff, e.sfxEnd = EntryHeaderSize, uint16(EntryHeaderSize+idLen)
	return e, nil
}

/**
* parseEntry2 valida una entrada de la forma 2 sin confiar en sus bytes: marcas conocidas, el id entero (prefijo de la
* hoja + sufijo) de 1 a MaxIDLen, la cabecera corta solo en una congelada sin desborde, y el largo total según sea
* lápida, desborde o valor en línea.
* @param b []byte: el ítem (o la entrada de un delta, con pre vacío y el id entero)
* @param pre []byte: el prefijo de la hoja
* @return entry, error: ErrCorrupt con el detalle
**/
func parseEntry2(b []byte, pre []byte) (entry, error) {
	if len(b) < entry2Frozen {
		return entry{}, corrupt("leaf entry of %d bytes", len(b))
	}
	e := entry{b: b, pre: pre, v2: true}
	e.w0 = loadWord(b)
	flags, sfx, inline := EntryFlags(e.w0&word0FlagMask), int(e.w0>>word0SuffixShift&word0LenMask), int(e.w0>>word0InlineShift)
	if flags&^knownEntryFlags != 0 || len(pre)+sfx < 1 || len(pre)+sfx > MaxIDLen {
		return entry{}, corrupt("leaf entry flags %#x id length %d", flags, len(pre)+sfx)
	}
	want := e.hdrLen() + sfx
	switch {
	case flags&EntryTombstone != 0:
		if inline != 0 || flags&(EntryOverflow|EntryFreeing|EntryCompressed) != 0 {
			return entry{}, corrupt("tombstone with a value")
		}
	case flags&EntryOverflow != 0:
		if inline != 0 {
			return entry{}, corrupt("overflow entry with an inline length")
		}
		want += entry2RefSize
	case flags&EntryFreeing != 0:
		return entry{}, corrupt("freeing entry without overflow")
	default:
		want += inline
	}
	if len(b) != want {
		return entry{}, corrupt("leaf entry of %d bytes, expected %d", len(b), want)
	}
	h := e.hdrLen()
	e.sfxOff, e.sfxEnd = uint16(h), uint16(h+sfx)
	return e, nil
}

/**
* loadWord lee el uint32 del byte 0 de un ítem. Dentro de una página, con una carga atómica: con bloqueo compartido,
* otro lector puede estar poniendo marcas de visibilidad con un OR atómico (ver buffer); los ítems están alineados a 8
* en páginas alineadas a 8. Una entrada que no está alineada (el delta de un registro del WAL) no la comparte nadie:
* lectura normal (una carga atómica desalineada es un SIGBUS en arm64).
* @param b []byte: de 4 bytes o más
* @return uint32
**/
func loadWord(b []byte) uint32 {
	p := unsafe.Pointer(&b[0])
	if uintptr(p)%4 != 0 {
		return le.Uint32(b)
	}
	return atomic.LoadUint32((*uint32)(p))
}

/**
* orItemFlags agrega marcas a una entrada de una página con un OR atómico sobre su uint32 del byte 0 (las marcas son
* sus bits bajos en las dos formas; con bloqueo compartido otro lector puede estar haciendo lo mismo, y la escritura
* de la página al disco la copia con cargas atómicas). Una entrada desalineada no viene de una página: no se marca.
* @param b []byte: el ítem, add EntryFlags
* @return bool: si se marcó
**/
func orItemFlags(b []byte, add EntryFlags) bool {
	p := unsafe.Pointer(&b[0])
	if uintptr(p)%4 != 0 {
		return false
	}
	atomic.OrUint32((*uint32)(p), uint32(add))
	return true
}

// flags lee las marcas, con una carga atómica dentro de una página (loadWord).
func (e *entry) flags() EntryFlags {
	if e.v2 {
		return EntryFlags(loadWord(e.b) & word0FlagMask)
	}
	return EntryFlags(loadWord(e.b))
}

// orFlags agrega marcas con un OR atómico (orItemFlags).
func (e *entry) orFlags(add EntryFlags) bool { return orItemFlags(e.b, add) }

// freeing dice si la versión está muerta y su cadena de desborde se está liberando.
func (e *entry) freeing() bool { return e.flags()&EntryFreeing != 0 }

// hdrLen es el largo de los campos fijos: 28 en la forma 1; 20, o 12 en una congelada sin desborde, en la 2.
func (e *entry) hdrLen() int {
	switch {
	case !e.v2:
		return EntryHeaderSize
	case EntryFlags(e.w0)&(EntryFrozen|EntryOverflow) == EntryFrozen:
		return entry2Frozen
	}
	return entry2Header
}

// suffix es la vista de lo que la entrada guarda del id: el id entero en la forma 1, el resto tras el prefijo en la 2.
func (e *entry) suffix() []byte { return e.b[e.sfxOff:e.sfxEnd] }

// tail es lo que sigue al id: el valor en línea (vacío en una lápida) o la referencia a desborde.
func (e *entry) tail() []byte { return e.b[e.sfxEnd:] }

// lsn es el lsn del registro que escribió la versión.
func (e *entry) lsn() page.LSN {
	if e.v2 {
		return page.LSN(le.Uint64(e.b[off2LSN:]))
	}
	return page.LSN(le.Uint64(e.b[OffEntryLSN:]))
}

// xid es la transacción que escribió la versión; 0 en una congelada de la forma 2 sin desborde (no lo guarda: es
// visible para todos).
func (e *entry) xid() page.XID {
	switch {
	case !e.v2:
		return page.XID(le.Uint64(e.b[OffEntryXID:]))
	case e.hdrLen() == entry2Frozen:
		return 0
	}
	return page.XID(le.Uint64(e.b[off2XID:]))
}

// key es la clave de la versión: (id, lsn); en la forma 2 con el prefijo de la hoja en pre.
func (e *entry) key() key { return key{pre: e.pre, id: e.suffix(), lsn: e.lsn()} }

/**
* idEqual dice si el id entero de la versión es b, sin armarlo.
* @param b []byte
* @return bool
**/
func (e *entry) idEqual(b []byte) bool {
	n := len(e.pre)
	return len(b) == n+len(e.suffix()) && bytes.Equal(b[:n], e.pre) && bytes.Equal(b[n:], e.suffix())
}

/**
* compareID ordena por bytes el id entero de la versión con b, sin armarlo.
* @param b []byte
* @return int: -1, 0 o 1
**/
func (e *entry) compareID(b []byte) int {
	n := len(e.pre)
	if n == 0 {
		return bytes.Compare(e.suffix(), b)
	}
	if len(b) < n {
		if c := bytes.Compare(e.pre[:len(b)], b); c != 0 {
			return c
		}
		return 1 // b es un prefijo propio del id
	}
	if c := bytes.Compare(e.pre, b[:n]); c != 0 {
		return c
	}
	return bytes.Compare(e.suffix(), b[n:])
}

// appendID agrega el id entero a dst.
func (e *entry) appendID(dst []byte) []byte { return append(append(dst, e.pre...), e.suffix()...) }

/**
* idString es el id entero como string, con una sola asignación.
* @return string
**/
func (e *entry) idString() string {
	sfx := e.suffix()
	if len(e.pre) == 0 || len(sfx) == 0 {
		return string(e.pre) + string(sfx)
	}
	b := make([]byte, len(e.pre)+len(sfx))
	copy(b[copy(b, e.pre):], sfx)
	return unsafe.String(&b[0], len(b))
}

// tombstone dice si la versión es un borrado.
func (e *entry) tombstone() bool { return e.flags()&EntryTombstone != 0 }

/**
* overflowRef es la referencia a desborde de una versión con EntryOverflow: el primer bloque y el largo guardado.
* @return overflow.Ref
**/
func (e *entry) overflowRef() overflow.Ref {
	rest := e.tail()
	if e.v2 {
		return overflow.Ref{First: page.BlockNumber(le.Uint32(rest)), Len: le.Uint32(rest[4:])}
	}
	return overflow.Ref{First: page.BlockNumber(le.Uint32(rest)), Len: le.Uint32(e.b[OffEntryValueLen:])}
}

/**
* version copia la entrada a un Version que sobrevive a soltar la página: el valor en línea se copia; el de desborde
* queda como su Ref.
* @return Version
**/
func (e *entry) version() Version {
	return e.versionOf(e.idString())
}

/**
* versionOf es version con el id ya conocido (una búsqueda exacta: no arma ni asigna el id de la página).
* @param id string: el id entero de la entrada
* @return Version
**/
func (e *entry) versionOf(id string) Version {
	v := e.headerOf(id)
	if v.Flags&(EntryTombstone|EntryOverflow) == 0 {
		v.Value = bytes.Clone(e.inlineValue())
		if v.Value == nil {
			v.Value = []byte{}
		}
	}
	return v
}

/**
* header copia la entrada a un Version sin el valor en línea (Value nil): id, lsn, xid, marcas y la referencia a
* desborde. Lo usan los recorridos que no necesitan el valor y los que lo copian a su propio búfer.
* @return Version
**/
func (e *entry) header() Version {
	return e.headerOf(e.idString())
}

/**
* headerOf es header con el id ya conocido.
* @param id string: el id entero de la entrada
* @return Version
**/
func (e *entry) headerOf(id string) Version {
	v := Version{ID: id, LSN: e.lsn(), XID: e.xid(), Flags: e.flags()}
	if v.Flags&EntryOverflow != 0 && v.Flags&EntryTombstone == 0 {
		v.Overflow = e.overflowRef()
	}
	return v
}

// inlineValue es la vista del valor en línea (vacía en una lápida; en desborde, los bytes de la referencia).
func (e *entry) inlineValue() []byte { return e.tail() }

/**
* storedSize es lo que ocupa en una hoja de la forma 2 con un prefijo de prefixLen bytes (sin alinear), y congelada
* si freeze: lo que va a medir reencode.
* @param prefixLen int, freeze bool
* @return int
**/
func (e *entry) storedSize(prefixLen int, freeze bool) int {
	flags := e.flags()
	if freeze {
		flags |= EntryFrozen
	}
	n := entry2Header
	if flags&(EntryFrozen|EntryOverflow) == EntryFrozen {
		n = entry2Frozen
	}
	n += len(e.pre) + len(e.suffix()) - prefixLen
	switch {
	case flags&EntryTombstone != 0:
	case flags&EntryOverflow != 0:
		n += entry2RefSize
	default:
		n += len(e.inlineValue())
	}
	return n
}

/**
* reencode arma la entrada en la forma 2 para una hoja con un prefijo de prefixLen bytes (appendReencode en un búfer
* propio).
* @param prefixLen int, freeze bool
* @return []byte: copia
**/
func (e *entry) reencode(prefixLen int, freeze bool) []byte {
	return e.appendReencode(make([]byte, 0, e.storedSize(prefixLen, freeze)), prefixLen, freeze)
}

/**
* appendReencode agrega a dst la entrada en la forma 2 para una hoja con un prefijo de prefixLen bytes (el id entero de
* la versión empieza con él: quien llama lo garantiza), desde cualquiera de las dos formas, con sus marcas actuales; con
* freeze, además congelada (EntryFrozen, y sin xid si no tiene desborde). Es lo que hacen la migración de una hoja de
* la forma 1, el cambio de prefijo de una partición o de un prefijo que se acorta, y el congelado de la limpieza.
* @param dst []byte, prefixLen int, freeze bool
* @return []byte
**/
func (e *entry) appendReencode(dst []byte, prefixLen int, freeze bool) []byte {
	flags := e.flags()
	if freeze {
		flags |= EntryFrozen
	}
	inline := 0
	if flags&(EntryTombstone|EntryOverflow) == 0 {
		inline = len(e.tail())
	}
	sfxLen := len(e.pre) + len(e.suffix()) - prefixLen
	dst = le.AppendUint32(dst, uint32(flags)|uint32(sfxLen)<<word0SuffixShift|uint32(inline)<<word0InlineShift)
	dst = le.AppendUint64(dst, uint64(e.lsn()))
	if flags&(EntryFrozen|EntryOverflow) != EntryFrozen {
		dst = le.AppendUint64(dst, uint64(e.xid()))
	}
	if prefixLen >= len(e.pre) {
		dst = append(dst, e.suffix()[prefixLen-len(e.pre):]...)
	} else {
		dst = append(append(dst, e.pre[prefixLen:]...), e.suffix()...)
	}
	switch {
	case flags&EntryTombstone != 0:
	case flags&EntryOverflow != 0:
		ref := e.overflowRef()
		dst = le.AppendUint32(dst, uint32(ref.First))
		dst = le.AppendUint32(dst, ref.Len)
	default:
		dst = append(dst, e.tail()...)
	}
	return dst
}

/**
* encodeEntry arma la entrada de la forma 1 de una escritura con lsn 0 (los registros y las hojas del formato 1; el
* motor ya no la escribe, la usan las pruebas de la lectura de las dos formas). Quien llama ya validó largos con
* checkWrite.
* @param id []byte, w Write
* @return []byte
**/
func encodeEntry(id []byte, w Write) []byte {
	n := EntryHeaderSize + len(id)
	var flags EntryFlags
	var valueLen int
	switch {
	case w.Tombstone:
		flags = EntryTombstone
	case w.Overflow != nil:
		flags, valueLen, n = EntryOverflow, int(w.Overflow.Len), n+OverflowRefSize
	default:
		valueLen, n = len(w.Value), n+len(w.Value)
	}
	if w.Compressed && !w.Tombstone {
		flags |= EntryCompressed
	}
	b := make([]byte, n)
	le.PutUint32(b[OffEntryFlags:], uint32(flags))
	le.PutUint16(b[OffEntryIDLen:], uint16(len(id)))
	le.PutUint64(b[OffEntryXID:], uint64(w.XID))
	le.PutUint32(b[OffEntryValueLen:], uint32(valueLen))
	rest := b[EntryHeaderSize+copy(b[EntryHeaderSize:], id):]
	switch {
	case w.Tombstone:
	case w.Overflow != nil:
		le.PutUint32(rest, uint32(w.Overflow.First))
	default:
		copy(rest, w.Value)
	}
	return b
}

/**
* encodeEntry2 arma la entrada de la forma 2 de una escritura con el id entero como sufijo y lsn 0: es la que lleva
* el delta de KindLeafWrite (el flujo lógico la lee sin la página); al aplicarla se le quita el prefijo de la hoja y
* se le pone el lsn del registro. Quien llama ya validó largos con checkWrite.
* @param id []byte, w Write
* @return []byte
**/
func encodeEntry2(id []byte, w Write) []byte {
	var flags EntryFlags
	inline, n := 0, entry2Header+len(id)
	switch {
	case w.Tombstone:
		flags = EntryTombstone
	case w.Overflow != nil:
		flags, n = EntryOverflow, n+entry2RefSize
	default:
		inline, n = len(w.Value), n+len(w.Value)
	}
	if w.Compressed && !w.Tombstone {
		flags |= EntryCompressed
	}
	b := make([]byte, n)
	le.PutUint32(b, uint32(flags)|uint32(len(id))<<word0SuffixShift|uint32(inline)<<word0InlineShift)
	le.PutUint64(b[off2XID:], uint64(w.XID))
	rest := b[entry2Header+copy(b[entry2Header:], id):]
	switch {
	case w.Tombstone:
	case w.Overflow != nil:
		le.PutUint32(rest, uint32(w.Overflow.First))
		le.PutUint32(rest[4:], w.Overflow.Len)
	default:
		copy(rest, w.Value)
	}
	return b
}

/**
* checkWrite valida el id y que la versión quepa en línea (o venga en desborde). La regla de en línea es la de la
* forma 1 (28 + id + valor ≤ MaxEntrySize) en las dos formas: la decisión no depende de la hoja donde cae, y una
* entrada de la forma 2 nunca es más larga que la de la 1.
* @param id string, w Write
* @return error: ErrIDLength o ErrEntryTooLarge
**/
func checkWrite(id string, w Write) error {
	if err := checkID(id); err != nil {
		return err
	}
	if !w.Tombstone && w.Overflow == nil && EntryHeaderSize+len(id)+len(w.Value) > MaxEntrySize {
		return ErrEntryTooLarge
	}
	return nil
}

/**
* checkID valida el largo de un id.
* @param id string
* @return error: ErrIDLength
**/
func checkID(id string) error {
	if len(id) < 1 || len(id) > MaxIDLen {
		return ErrIDLength
	}
	return nil
}
