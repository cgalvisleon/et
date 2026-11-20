package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Marca de sellado: los últimos TrailerSize bytes de un segmento sellado (en SegmentSize − TrailerSize, ver doc.go
// «Marca de sellado»). Dice que el segmento se selló entero y hasta dónde llegan sus registros: un registro inválido
// antes de ese fin no es una cola rota sino corrupción, y Open falla.
const (
	// TrailerSize es el largo de la marca de sellado; los registros de un segmento terminan antes.
	TrailerSize = 32

	trailerMagic = 0x4c414553 // «SEAL» en little-endian

	offTrailerCRC   = 0  // CRC-32C de [4, TrailerSize)
	offTrailerMagic = 4  // trailerMagic
	offTrailerZero  = 8  // 8 bytes en cero: donde un registro tendría totalLen, así nunca pasa por uno
	offTrailerNum   = 16 // número del segmento (uno reciclado conserva la marca de su número viejo)
	offTrailerEnd   = 24 // desplazamiento siguiente al último registro
)

/**
* capacity es hasta dónde puede llegar un registro dentro de un segmento: el resto es de la marca de sellado.
* @return int64
**/
func (w *WAL) capacity() int64 {
	return w.opts.SegmentSize - TrailerSize
}

/**
* encodeTrailer arma la marca de sellado del segmento num, cuyos registros terminan en end.
* @param num uint64, end int64
* @return []byte: TrailerSize bytes
**/
func encodeTrailer(num uint64, end int64) []byte {
	buf := make([]byte, TrailerSize)
	binary.LittleEndian.PutUint32(buf[offTrailerMagic:], trailerMagic)
	binary.LittleEndian.PutUint64(buf[offTrailerNum:], num)
	binary.LittleEndian.PutUint64(buf[offTrailerEnd:], uint64(end))
	binary.LittleEndian.PutUint32(buf[offTrailerCRC:], crc32.Checksum(buf[offTrailerMagic:], castagnoli))
	return buf
}

/**
* decodeTrailer lee una marca de sellado sin confiar en nada: crc, magia, ceros, número y fin dentro del archivo.
* @param buf []byte: TrailerSize bytes, num uint64: el número del archivo, size int64: su largo
* @return int64: el fin de los registros, bool: si es una marca válida de este segmento
**/
func decodeTrailer(buf []byte, num uint64, size int64) (int64, bool) {
	if len(buf) != TrailerSize || binary.LittleEndian.Uint32(buf[offTrailerMagic:]) != trailerMagic {
		return 0, false
	}
	if binary.LittleEndian.Uint32(buf[offTrailerCRC:]) != crc32.Checksum(buf[offTrailerMagic:], castagnoli) {
		return 0, false
	}
	if binary.LittleEndian.Uint64(buf[offTrailerZero:]) != 0 || binary.LittleEndian.Uint64(buf[offTrailerNum:]) != num {
		return 0, false
	}
	end := int64(binary.LittleEndian.Uint64(buf[offTrailerEnd:]))
	if end < 0 || end > size-TrailerSize {
		return 0, false
	}
	return end, true
}

/**
* markSealed escribe la marca de sellado del segmento ya sellado (sus registros con fsync) en SegmentSize −
* TrailerSize y hace fsync: así la marca nunca es durable antes que los registros que promete. Un segmento con
* registros en esa zona (escrito sin la marca) no la lleva. Con el disco lleno devuelve el error sin tocar nada (el
* segmento sigue activo; una marca a medias no es válida por el crc); un fsync fallido deja el WAL inutilizable. Con
* writeMu tomado.
* @param s *segment
* @return error
**/
func (w *WAL) markSealed(s *segment) error {
	at := w.opts.SegmentSize - TrailerSize
	if s.size > at {
		return nil
	}
	if _, err := s.file.WriteAt(encodeTrailer(s.num, s.size), at); err != nil {
		return w.spaceError(err)
	}
	if err := s.file.Sync(); err != nil {
		w.fail(err)
		return failedError(err)
	}
	return nil
}

/**
* readTrailer lee la marca de sellado del segmento num (los últimos TrailerSize bytes del archivo).
* @param fsys vfs.FS, dir string, num uint64
* @return int64: el fin de sus registros, bool: si el segmento está sellado, error: de lectura
**/
func readTrailer(fsys vfs.FS, dir string, num uint64) (int64, bool, error) {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDONLY)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()
	size, err := file.Size()
	if err != nil || size < TrailerSize {
		return 0, false, err
	}
	buf := make([]byte, TrailerSize)
	if _, err := file.ReadAt(buf, size-TrailerSize); err != nil {
		return 0, false, readError(err)
	}
	end, ok := decodeTrailer(buf, num, size)
	return end, ok, nil
}

/**
* checkSealed comprueba que el final válido del WAL (end) no corte un segmento sellado: un registro inválido antes
* del fin que promete su marca, o un segmento sellado después de end, es corrupción (los registros de un segmento
* sellado tuvieron fsync). Solo el final del último segmento sin sellar puede ser una cola rota.
* @param fsys vfs.FS, dir string, end Position, nums []uint64: los segmentos del directorio
* @return error: ErrCorrupted
**/
func checkSealed(fsys vfs.FS, dir string, end Position, nums []uint64) error {
	for _, num := range nums {
		if num < end.Segment {
			continue
		}
		sealedEnd, ok, err := readTrailer(fsys, dir, num)
		if err != nil {
			return err
		}
		switch {
		case !ok:
		case num > end.Segment:
			return fmt.Errorf(MSG_SEALED_AFTER, ErrCorrupted, num, end)
		case end.Offset < sealedEnd:
			return fmt.Errorf(MSG_SEALED_CUT, ErrCorrupted, end.Segment, end.Offset, end.LSN, sealedEnd)
		}
	}
	return nil
}
