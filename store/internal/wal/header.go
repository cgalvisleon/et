package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"slices"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Encabezado del segmento (pregunta 76 de F-23, como el long page header de PostgreSQL): los primeros
// SegmentHeaderSize bytes de cada segmento. Dice de qué versión del formato es el segmento, cuál es su número y el lsn
// de su primer registro, así un segmento suelto (archivado, en un respaldo, recibido por una réplica) se valida sin el
// control. Ver doc.go «Encabezado del segmento».
const (
	// SegmentHeaderSize es el largo del encabezado; el primer registro de un segmento empieza aquí.
	SegmentHeaderSize = 32

	// FormatVersion es la versión del formato en disco del motor (la misma del control, que la toma de aquí): la que
	// este binario escribe. 2 desde la tarea 132 (formato 2, en desarrollo: valores comprimidos).
	FormatVersion = 2

	// MinFormatVersion es la versión más vieja que este binario lee: un directorio de [MinFormatVersion,
	// FormatVersion) se abre y pasa a FormatVersion al escribir (ver «Formato 2» en store/doc.go).
	MinFormatVersion = 1

	headerMagic = 0x47455357 // «WSEG» en little-endian

	offHeaderCRC      = 0  // CRC-32C de [4, SegmentHeaderSize)
	offHeaderMagic    = 4  // headerMagic
	offHeaderVersion  = 8  // FormatVersion
	offHeaderReserved = 12 // 4 bytes en cero
	offHeaderNum      = 16 // número del segmento
	offHeaderFirst    = 24 // lsn del primer registro del segmento
)

// ErrFormatVersion: un segmento con el encabezado de otra versión del formato.
var ErrFormatVersion = errors.New("wal: unsupported on-disk format version")

// SegmentHeader es el encabezado leído de un segmento.
type SegmentHeader struct {
	Version uint32
	Num     uint64
	First   page.LSN
}

// headerState es lo que se encontró al leer un encabezado.
type headerState int

const (
	headerValid   headerState = iota // crc, magia, versión y ceros correctos
	headerAbsent                     // el archivo es más corto que el encabezado o está en ceros: nunca se escribió
	headerBroken                     // hay bytes pero no son un encabezado válido
	headerVersion                    // la magia es la del encabezado y la versión, otra
)

/**
* encodeHeader arma el encabezado del segmento num cuyo primer registro es first.
* @param num uint64, first page.LSN
* @return []byte: SegmentHeaderSize bytes
**/
func encodeHeader(num uint64, first page.LSN) []byte {
	buf := make([]byte, SegmentHeaderSize)
	binary.LittleEndian.PutUint32(buf[offHeaderMagic:], headerMagic)
	binary.LittleEndian.PutUint32(buf[offHeaderVersion:], FormatVersion)
	binary.LittleEndian.PutUint64(buf[offHeaderNum:], num)
	binary.LittleEndian.PutUint64(buf[offHeaderFirst:], uint64(first))
	binary.LittleEndian.PutUint32(buf[offHeaderCRC:], crc32.Checksum(buf[offHeaderMagic:], castagnoli))
	return buf
}

/**
* decodeHeader lee un encabezado sin confiar en nada. La versión se mira antes que el crc: otra versión puede tener
* otro encabezado, y solo la magia y la versión están fijas para siempre.
* @param buf []byte: lo leído del principio del segmento (puede ser corto)
* @return SegmentHeader, headerState
**/
func decodeHeader(buf []byte) (SegmentHeader, headerState) {
	if len(buf) < SegmentHeaderSize {
		if bytes.Count(buf, []byte{0}) == len(buf) {
			return SegmentHeader{}, headerAbsent
		}
		return SegmentHeader{}, headerBroken
	}
	buf = buf[:SegmentHeaderSize]
	h := SegmentHeader{
		Version: binary.LittleEndian.Uint32(buf[offHeaderVersion:]),
		Num:     binary.LittleEndian.Uint64(buf[offHeaderNum:]),
		First:   page.LSN(binary.LittleEndian.Uint64(buf[offHeaderFirst:])),
	}
	switch {
	case binary.LittleEndian.Uint32(buf[offHeaderMagic:]) != headerMagic:
		if bytes.Count(buf, []byte{0}) == len(buf) {
			return h, headerAbsent
		}
		return h, headerBroken
	case !Readable(h.Version):
		return h, headerVersion
	case binary.LittleEndian.Uint32(buf[offHeaderCRC:]) != crc32.Checksum(buf[offHeaderMagic:], castagnoli),
		binary.LittleEndian.Uint32(buf[offHeaderReserved:]) != 0, h.Num == 0, h.First == 0:
		return h, headerBroken
	}
	return h, headerValid
}

/**
* Readable dice si este binario lee la versión del formato v: de MinFormatVersion a FormatVersion.
* @param v uint32
* @return bool
**/
func Readable(v uint32) bool {
	return v >= MinFormatVersion && v <= FormatVersion
}

/**
* ParseSegmentHeader lee el encabezado del principio de un segmento (para herramientas y pruebas fuera del paquete).
* @param buf []byte: al menos SegmentHeaderSize bytes del principio del segmento
* @return SegmentHeader, error: ErrFormatVersion (otra versión) o ErrCorrupted (ausente o roto)
**/
func ParseSegmentHeader(buf []byte) (SegmentHeader, error) {
	h, state := decodeHeader(buf)
	switch state {
	case headerValid:
		return h, nil
	case headerVersion:
		return h, versionError(h.Num, h.Version)
	}
	return h, fmt.Errorf(MSG_HEADER_RAW, ErrCorrupted)
}

/**
* readHeader lee el encabezado de un archivo de segmento abierto.
* @param file vfs.File
* @return SegmentHeader, headerState, error: de lectura
**/
func readHeader(file vfs.File) (SegmentHeader, headerState, error) {
	buf := make([]byte, SegmentHeaderSize)
	n, err := file.ReadAt(buf, 0)
	if err = readError(err); err != nil {
		return SegmentHeader{}, headerBroken, err
	}
	h, state := decodeHeader(buf[:n])
	return h, state, nil
}

/**
* readHeaderOf lee el encabezado del segmento num de dir (el directorio del WAL).
* @param fsys vfs.FS, dir string, num uint64
* @return SegmentHeader, headerState, error: de lectura o de apertura
**/
func readHeaderOf(fsys vfs.FS, dir string, num uint64) (SegmentHeader, headerState, error) {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDONLY)
	if err != nil {
		return SegmentHeader{}, headerBroken, err
	}
	defer file.Close()
	return readHeader(file)
}

/**
* writeHeader escribe el encabezado del segmento num (primer registro first) y hace fsync: el segmento anterior se
* sella recién después, así un segmento sellado siempre tiene el siguiente con su encabezado durable. Con el disco
* lleno devuelve el error sin dejar el WAL inutilizable; un fsync fallido sí lo deja. Con writeMu tomado (o en Open).
* @param file vfs.File, num uint64, first page.LSN
* @return error
**/
func (w *WAL) writeHeader(file vfs.File, num uint64, first page.LSN) error {
	if _, err := file.WriteAt(encodeHeader(num, first), 0); err != nil {
		return w.spaceError(err)
	}
	if err := file.Sync(); err != nil {
		w.fail(err)
		return failedError(err)
	}
	return nil
}

/**
* versionError es el error de un segmento de otra versión del formato.
* @param num uint64, version uint32
* @return error: ErrFormatVersion
**/
func versionError(num uint64, version uint32) error {
	return fmt.Errorf(MSG_HEADER_VERSION, ErrFormatVersion, segmentName(num), version, FormatVersion)
}

/**
* checkStartHeader comprueba el encabezado del segmento donde empieza un recorrido (el punto de rehacer, la posición
* de un lector): tiene que ser de este segmento, de esta versión y con un primer registro no posterior a from. Un
* encabezado ausente solo vale si from es el principio del segmento (un segmento nuevo cortado antes de escribirlo:
* Open lo escribe). Uno roto es corrupción.
* @param fsys vfs.FS, dir string, from Position
* @return error: ErrCorrupted, ErrFormatVersion
**/
func checkStartHeader(fsys vfs.FS, dir string, from Position) error {
	h, state, err := readHeaderOf(fsys, dir, from.Segment)
	if err != nil {
		return err
	}
	switch state {
	case headerVersion:
		return versionError(from.Segment, h.Version)
	case headerAbsent:
		if from.Offset == SegmentHeaderSize {
			return nil
		}
	case headerValid:
		if h.Num == from.Segment && h.First <= from.LSN {
			return nil
		}
	case headerBroken:
		if version, other, err := segmentVersion(fsys, dir, from.Segment); err != nil || other {
			if err != nil {
				return err
			}
			return versionError(from.Segment, version) // un segmento anterior al encabezado
		}
	}
	return fmt.Errorf(MSG_HEADER_BAD, ErrCorrupted, from.Segment, from)
}

/**
* OtherVersion busca, sin escribir nada, un segmento del WAL del motor de dir con el encabezado de otra versión del
* formato (lo usa checkFormat del motor antes de abrir). Solo mira la magia y la versión: un encabezado roto lo
* informa después la recuperación. Mira solo lo que Open va a leer: los segmentos desde from (el del punto de rehacer:
* los que recorre la recuperación, más los repuestos) y, si keep > 0, los anteriores hacia atrás hasta el primero cuyo
* encabezado empieza en keep o antes (lo que lee SegmentOf al poner la retención de una ranura). Así no lee un
* encabezado por cada segmento retenido solo para el archivo continuo; esos los revisa quien los lee (checkStartHeader
* y el paso al siguiente de scan, SegmentInfo).
* @param fsys vfs.FS, dir string: el directorio del motor
* @param from uint64: el primer segmento que se mira entero (0 = todos), keep page.LSN: el menor lsn que retiene una ranura (0 = ninguno)
* @return string: el nombre del segmento (wal/<n>.seg), uint32: su versión, bool: si hay uno, error
**/
func OtherVersion(fsys vfs.FS, dir string, from uint64, keep page.LSN) (string, uint32, bool, error) {
	walDir := filepath.Join(dir, Dir)
	names, err := fsys.ReadDir(dir)
	if err != nil {
		return "", 0, false, err
	}
	if !slices.Contains(names, Dir) {
		return "", 0, false, nil
	}
	nums, err := listSegments(fsys, walDir)
	if err != nil {
		return "", 0, false, err
	}
	below := 0 // nums[:below] son anteriores a from
	for below < len(nums) && nums[below] < from {
		below++
	}
	for _, num := range nums[below:] {
		if name, version, found, err := otherVersionOf(fsys, walDir, num); err != nil || found {
			return name, version, found, err
		}
	}
	for i := below - 1; i >= 0 && keep > 0; i-- {
		num := nums[i]
		if name, version, found, err := otherVersionOf(fsys, walDir, num); err != nil || found {
			return name, version, found, err
		}
		h, state, err := readHeaderOf(fsys, walDir, num)
		if err != nil {
			return "", 0, false, err
		}
		if state == headerValid && h.Num == num && h.First <= keep {
			break // SegmentOf se detiene aquí
		}
	}
	return "", 0, false, nil
}

/**
* otherVersionOf es segmentVersion con el resultado de OtherVersion.
* @param fsys vfs.FS, walDir string, num uint64
* @return string, uint32, bool, error: ver OtherVersion
**/
func otherVersionOf(fsys vfs.FS, walDir string, num uint64) (string, uint32, bool, error) {
	version, other, err := segmentVersion(fsys, walDir, num)
	if err != nil || !other {
		return "", 0, false, err
	}
	return filepath.Join(Dir, segmentName(num)), version, true, nil
}

// legacyVersion es la versión que se informa para un segmento sin encabezado (escrito antes de la pregunta 76): sus
// registros empiezan en el byte 0.
const legacyVersion = 0

/**
* segmentVersion dice si el segmento num es de otra versión: su encabezado lo dice, o no tiene encabezado y en el
* byte 0 hay un registro válido (un segmento anterior al encabezado, legacyVersion).
* @param fsys vfs.FS, dir string, num uint64
* @return uint32: la versión, bool: si es de otra, error
**/
func segmentVersion(fsys vfs.FS, dir string, num uint64) (uint32, bool, error) {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDONLY)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()
	h, state, err := readHeader(file)
	switch {
	case err != nil:
		return 0, false, err
	case state == headerVersion:
		return h.Version, true, nil
	case state != headerBroken:
		return 0, false, nil
	}
	buf := make([]byte, MaxRecordSize)
	n, err := file.ReadAt(buf, 0)
	if err = readError(err); err != nil {
		return 0, false, err
	}
	var rec Record
	if n >= HeaderSize && totalLenOf(buf) <= n {
		if _, ok := decodeRecord(buf[:totalLenOf(buf)], &rec); ok {
			return legacyVersion, true, nil
		}
	}
	return 0, false, nil
}
