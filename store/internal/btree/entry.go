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
const knownEntryFlags = EntryTombstone | EntryOverflow | EntryXidCommitted | EntryXidAborted | EntryFrozen | EntryFreeing

// entry es una entrada de hoja (una versión) ya validada por parseEntry. Es una vista: si viene de una página, vale
// mientras la página siga bloqueada.
type entry []byte

/**
* parseEntry valida una entrada de hoja sin confiar en sus bytes: largo mínimo, marcas conocidas, largo del id y
* largo total según sea lápida, desborde o valor en línea.
* @param b []byte: el ítem de la página
* @return entry, error: ErrCorrupt con el detalle
**/
func parseEntry(b []byte) (entry, error) {
	if len(b) < EntryHeaderSize {
		return nil, corrupt("leaf entry of %d bytes", len(b))
	}
	e := entry(b)
	flags, idLen, valueLen := e.flags(), int(le.Uint16(b[OffEntryIDLen:])), int(le.Uint32(b[OffEntryValueLen:]))
	if flags&^knownEntryFlags != 0 || idLen < 1 || idLen > MaxIDLen {
		return nil, corrupt("leaf entry flags %#x id length %d", flags, idLen)
	}
	want := EntryHeaderSize + idLen
	switch {
	case flags&EntryTombstone != 0:
		if valueLen != 0 || flags&(EntryOverflow|EntryFreeing) != 0 {
			return nil, corrupt("tombstone with a value")
		}
	case flags&EntryOverflow != 0:
		want += OverflowRefSize
	case flags&EntryFreeing != 0:
		return nil, corrupt("freeing entry without overflow")
	default:
		want += valueLen
	}
	if len(b) != want {
		return nil, corrupt("leaf entry of %d bytes, expected %d", len(b), want)
	}
	return e, nil
}

/**
* flags lee las marcas. Dentro de una página, con una carga atómica: con bloqueo compartido, otro lector puede estar
* poniendo marcas de visibilidad con un OR atómico (ver buffer); los ítems están alineados a 8 en páginas alineadas a
* 8. Una entrada que no está alineada (el delta de un registro del WAL) no la comparte nadie: lectura normal (una
* carga atómica desalineada es un SIGBUS en arm64).
* @return EntryFlags
**/
func (e entry) flags() EntryFlags {
	p := unsafe.Pointer(&e[OffEntryFlags])
	if uintptr(p)%4 != 0 {
		return EntryFlags(le.Uint32(e[OffEntryFlags:]))
	}
	return EntryFlags(atomic.LoadUint32((*uint32)(p)))
}

/**
* orFlags agrega marcas a una entrada de una página con un OR atómico (con bloqueo compartido otro lector puede estar
* haciendo lo mismo, y la escritura de la página al disco la copia con cargas atómicas). Una entrada desalineada no
* viene de una página: no se marca.
* @param add EntryFlags
* @return bool: si se marcó
**/
func (e entry) orFlags(add EntryFlags) bool {
	p := unsafe.Pointer(&e[OffEntryFlags])
	if uintptr(p)%4 != 0 {
		return false
	}
	atomic.OrUint32((*uint32)(p), uint32(add))
	return true
}

// freeing dice si la versión está muerta y su cadena de desborde se está liberando.
func (e entry) freeing() bool { return e.flags()&EntryFreeing != 0 }

// id es la vista del id.
func (e entry) id() []byte {
	n := int(le.Uint16(e[OffEntryIDLen:]))
	return e[EntryHeaderSize : EntryHeaderSize+n]
}

// lsn es el lsn del registro que escribió la versión.
func (e entry) lsn() page.LSN { return page.LSN(le.Uint64(e[OffEntryLSN:])) }

// xid es la transacción que escribió la versión.
func (e entry) xid() page.XID { return page.XID(le.Uint64(e[OffEntryXID:])) }

// key es la clave de la versión: (id, lsn).
func (e entry) key() key { return key{id: e.id(), lsn: e.lsn()} }

// tombstone dice si la versión es un borrado.
func (e entry) tombstone() bool { return e.flags()&EntryTombstone != 0 }

/**
* version copia la entrada a un Version que sobrevive a soltar la página: el valor en línea se copia; el de desborde
* queda como su Ref.
* @return Version
**/
func (e entry) version() Version {
	v := Version{ID: string(e.id()), LSN: e.lsn(), XID: e.xid(), Flags: e.flags()}
	rest := e[EntryHeaderSize+len(e.id()):]
	switch {
	case v.Flags&EntryTombstone != 0:
	case v.Flags&EntryOverflow != 0:
		v.Overflow = overflow.Ref{First: page.BlockNumber(le.Uint32(rest)), Len: le.Uint32(e[OffEntryValueLen:])}
	default:
		v.Value = bytes.Clone(rest)
		if v.Value == nil {
			v.Value = []byte{}
		}
	}
	return v
}

/**
* encodeEntry arma la entrada de una escritura con lsn 0: el lsn lo pone la aplicación del registro, cuando ya se
* conoce (paso 8 del orden de un cambio). Quien llama ya validó largos con checkWrite.
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
* checkWrite valida el id y que la versión quepa en línea (o venga en desborde).
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
