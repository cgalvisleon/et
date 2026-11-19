package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/page"
)

// Posiciones de los campos de un pivote.
const (
	offPivotKeyLen   = 0
	offPivotFlags    = 2
	offPivotDownlink = 4
	offPivotLSN      = 8
)

// pivot es un pivote (enlace de una página interna o clave alta) ya validado por parsePivot. Es una vista.
type pivot []byte

/**
* parsePivot valida un pivote sin confiar en sus bytes: largo, marcas conocidas, largo de la clave y lsn coherente
* con PivotHasLSN (menos infinito no lleva lsn).
* @param b []byte: el ítem de la página
* @return pivot, error: ErrCorrupt con el detalle
**/
func parsePivot(b []byte) (pivot, error) {
	if len(b) < PivotHeaderSize {
		return nil, corrupt("pivot of %d bytes", len(b))
	}
	keyLen, flags := int(le.Uint16(b[offPivotKeyLen:])), le.Uint16(b[offPivotFlags:])
	lsn := page.LSN(le.Uint64(b[offPivotLSN:]))
	switch {
	case flags&^PivotHasLSN != 0, keyLen > MaxIDLen, len(b) != PivotHeaderSize+keyLen:
		return nil, corrupt("pivot flags %#x key length %d in %d bytes", flags, keyLen, len(b))
	case flags&PivotHasLSN == 0 && lsn != 0, flags&PivotHasLSN != 0 && (keyLen == 0 || lsn == MaxLSN):
		return nil, corrupt("pivot lsn %d does not match its flags %#x", lsn, flags)
	}
	return pivot(b), nil
}

/**
* key es la clave del pivote: (prefijo, lsn) con PivotHasLSN, (prefijo, MaxLSN) sin él; con keyLen 0 es menos
* infinito.
* @return key: vista
**/
func (v pivot) key() key {
	lsn := MaxLSN
	if le.Uint16(v[offPivotFlags:])&PivotHasLSN != 0 {
		lsn = page.LSN(le.Uint64(v[offPivotLSN:]))
	}
	return key{id: v[PivotHeaderSize:], lsn: lsn}
}

// downlink es el bloque de la hija (0 en una clave alta).
func (v pivot) downlink() page.BlockNumber {
	return page.BlockNumber(le.Uint32(v[offPivotDownlink:]))
}

// minusInf dice si es el primer enlace de una página interna (clave menos infinito).
func (v pivot) minusInf() bool {
	return le.Uint16(v[offPivotKeyLen:]) == 0
}

/**
* encodePivot arma un pivote.
* @param k key: sin top; lsn MaxLSN = sin PivotHasLSN
* @param downlink page.BlockNumber: 0 en una clave alta
* @return []byte
**/
func encodePivot(k key, downlink page.BlockNumber) []byte {
	b := make([]byte, PivotHeaderSize+len(k.id))
	le.PutUint16(b[offPivotKeyLen:], uint16(len(k.id)))
	le.PutUint32(b[offPivotDownlink:], uint32(downlink))
	if k.lsn != MaxLSN {
		le.PutUint16(b[offPivotFlags:], PivotHasLSN)
		le.PutUint64(b[offPivotLSN:], uint64(k.lsn))
	}
	copy(b[PivotHeaderSize:], k.id)
	return b
}

/**
* withDownlink copia un pivote cambiando su enlace (la clave alta de una hoja pasa a ser el enlace a la derecha en el
* padre).
* @param v pivot, downlink page.BlockNumber
* @return []byte
**/
func withDownlink(v pivot, downlink page.BlockNumber) []byte {
	b := bytes.Clone(v)
	le.PutUint32(b[offPivotDownlink:], uint32(downlink))
	return b
}
