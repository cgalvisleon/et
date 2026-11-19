package store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
)

// kind: qué hace un registro.
type kind byte

const (
	kindPut kind = 1 // guarda data para id
)

const (
	// headerSize: crc(4) + lsn(8) + kind(1) + idLen(2) + dataLen(4).
	headerSize = 19
	maxIDLen   = math.MaxUint16 // idLen se guarda en 2 bytes
	maxDataLen = 1 << 30        // 1 GiB: un valor más grande no es un caso de uso de un store de registros
)

// castagnoli: la tabla del CRC-32C, más rápido que el IEEE en CPUs con instrucción propia.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// header: los campos fijos de un registro.
type header struct {
	crc     uint32
	lsn     uint64
	kind    kind
	idLen   int
	dataLen int
}

/**
* size: Largo total del registro en disco.
* @return int
**/
func (h header) size() int {
	return headerSize + h.idLen + h.dataLen
}

/**
* checkPut: Valida id y data antes de escribir.
* @param id string, data []byte
* @return error
**/
func checkPut(id string, data []byte) error {
	switch {
	case id == "":
		return errors.New(MSG_ID_REQUIRED)
	case len(id) > maxIDLen:
		return fmt.Errorf(MSG_ID_TOO_LONG, maxIDLen)
	case len(data) > maxDataLen:
		return fmt.Errorf(MSG_DATA_TOO_LONG, maxDataLen)
	}
	return nil
}

/**
* encodeRecord: Arma un registro sin lsn ni crc: se completan con sealRecord cuando se conoce el lsn.
* @param k kind, id string, data []byte
* @return []byte
**/
func encodeRecord(k kind, id string, data []byte) []byte {
	buf := make([]byte, headerSize+len(id)+len(data))
	buf[12] = byte(k)
	binary.BigEndian.PutUint16(buf[13:15], uint16(len(id)))
	binary.BigEndian.PutUint32(buf[15:19], uint32(len(data)))
	copy(buf[headerSize:], id)
	copy(buf[headerSize+len(id):], data)
	return buf
}

/**
* sealRecord: Escribe el lsn y el crc de todo lo que le sigue.
* @param buf []byte, lsn uint64
**/
func sealRecord(buf []byte, lsn uint64) {
	binary.BigEndian.PutUint64(buf[4:12], lsn)
	binary.BigEndian.PutUint32(buf[0:4], crc32.Checksum(buf[4:], castagnoli))
}

/**
* decodeHeader: Lee los campos fijos de un registro.
* @param buf []byte (al menos headerSize bytes)
* @return header
**/
func decodeHeader(buf []byte) header {
	return header{
		crc:     binary.BigEndian.Uint32(buf[0:4]),
		lsn:     binary.BigEndian.Uint64(buf[4:12]),
		kind:    kind(buf[12]),
		idLen:   int(binary.BigEndian.Uint16(buf[13:15])),
		dataLen: int(binary.BigEndian.Uint32(buf[15:19])),
	}
}

/**
* validRecord: El registro completo pasa la verificación: crc correcto, lsn distinto de cero y tipo conocido.
* @param h header, buf []byte (el registro completo)
* @return bool
**/
func validRecord(h header, buf []byte) bool {
	return h.lsn != 0 && h.kind == kindPut && crc32.Checksum(buf[4:], castagnoli) == h.crc
}
