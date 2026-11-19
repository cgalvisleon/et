package page

import "hash/crc32"

// castagnoli es la tabla de CRC-32C.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// zeroChecksum son los dos bytes que reemplazan a pd_checksum al calcular la suma.
var zeroChecksum [2]byte

/**
* ComputeChecksum calcula la suma de la página: CRC-32C de la página con pd_checksum en cero, seguida del número de
* bloque en 4 bytes little-endian, plegada a 16 bits (uint16(c) ^ uint16(c>>16)). No asigna memoria.
* @param blk BlockNumber: el bloque donde vive la página (una página copiada a otro bloque no pasa)
* @return uint16
**/
func (p Page) ComputeChecksum(blk BlockNumber) uint16 {
	c := crc32.Update(0, castagnoli, p[:OffChecksum])
	c = crc32.Update(c, castagnoli, zeroChecksum[:])
	c = crc32.Update(c, castagnoli, p[OffChecksum+2:Size])
	c = updateUint32(c, uint32(blk))
	return uint16(c) ^ uint16(c>>16)
}

/**
* updateUint32 suma a la CRC-32C c los 4 bytes little-endian de v, byte a byte con la tabla, como crc32.Update pero
* sin pasar un slice (que obligaría a llevar los bytes al heap).
* @param c uint32: la suma hasta ahora
* @param v uint32
* @return uint32
**/
func updateUint32(c, v uint32) uint32 {
	c = ^c
	for range 4 {
		c = castagnoli[byte(c)^byte(v)] ^ c>>8
		v >>= 8
	}
	return ^c
}

/**
* SetChecksum calcula la suma y la guarda en pd_checksum. Se llama al escribir al disco, sobre una copia.
* @param blk BlockNumber
**/
func (p Page) SetChecksum(blk BlockNumber) {
	le.PutUint16(p[OffChecksum:], p.ComputeChecksum(blk))
}

/**
* Verify valida una página leída del disco sin confiar en sus bytes: largo, suma y encabezado (como
* PageIsVerified). Una página toda en ceros es nueva y válida. No revisa los punteros a ítems: eso es CheckItems,
* para las páginas que los tienen; aun sin él, ningún método entra en pánico ante punteros dañados.
* @param b []byte: los bytes leídos
* @param blk BlockNumber: el bloque leído
* @return error: nil, ErrSize, ErrChecksum o ErrCorrupt
**/
func Verify(b []byte, blk BlockNumber) error {
	p := Page(b)
	if len(p) != Size {
		return ErrSize
	}
	if p.IsNew() {
		if !allZero(p) {
			return corrupt("pd_upper is 0 but the page is not all zeros")
		}
		return nil
	}
	if p.Checksum() != p.ComputeChecksum(blk) {
		return ErrChecksum
	}
	return p.checkHeader()
}

/**
* allZero dice si todos los bytes de b son cero.
* @param b []byte: largo múltiplo de 8
* @return bool
**/
func allZero(b []byte) bool {
	for i := 0; i+8 <= len(b); i += 8 {
		if le.Uint64(b[i:]) != 0 {
			return false
		}
	}
	return true
}
