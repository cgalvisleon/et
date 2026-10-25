package store

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

type recordHeader struct {
	LSN     uint64 `json:"lsn"`
	DataLen uint32 `json:"data_len"`
	CRC     uint32 `json:"crc"`
	IDLen   uint16 `json:"id_len"`
	ID      string `json:"id"`
	Status  byte   `json:"status"`
}

/**
* headerSize: Tamaño del encabezado, incluido el id.
* @return int64
**/
func (s *recordHeader) headerSize() int64 {
	return int64(fixedHeaderSize) + int64(s.IDLen)
}

/**
* recordSize: Tamaño total del registro (encabezado + datos).
* @return int64
**/
func (s *recordHeader) recordSize() int64 {
	return s.headerSize() + int64(s.DataLen)
}

type recordRef struct {
	segment int
	offset  int64
	length  uint32
	crc     uint32 // CRC de los datos; vale solo con hasCRC
	hasCRC  bool   // false si la referencia vino del snapshot (no guarda el CRC)
}

/**
* differs: Indica, sin leer el disco, que data es distinto de lo guardado en la referencia.
* false no asegura que sea igual: hay que comparar los bytes.
* @param data []byte
* @return bool
**/
func (s *recordRef) differs(data []byte) bool {
	return s.hasCRC && (s.length != uint32(len(data)) || s.crc != checksum(data))
}

/**
* toJson: Retorna la referencia como JSON.
* @return et.Json
**/
func (s *recordRef) toJson() et.Json {
	return et.Json{
		"segment": s.segment,
		"offset":  s.offset,
		"length":  s.length,
	}
}

/**
* toString: Retorna la referencia como texto.
* @return string
**/
func (s *recordRef) toString() string {
	return s.toJson().ToString()
}

type segment struct {
	file      *os.File
	size      int64
	name      string
	readOnly  bool
	writeErr  error
	errMu     sync.Mutex
	closeOnce sync.Once
	readers   atomic.Int32           // lectores en curso (Get/Read/ForEach) que no sostienen indexMu
	retired   atomic.Bool            // fuera del store (compaction/close): se cierra con el último lector
	mapped    atomic.Pointer[[]byte] // contenido mapeado en memoria (sellado, o activo con capacidad MaxSegment); nil = ReadAt
	visible   atomic.Int64           // bytes ya escritos con write: lo que se puede leer del mapeo
}

/**
* mapReadOnly: Mapea en memoria el segmento, que ya no recibe escrituras: las lecturas pasan a
* ser copias de memoria en vez de llamadas al sistema. Si no se puede, se sigue con ReadAt.
**/
func (s *segment) mapReadOnly() {
	if s.file == nil || s.size == 0 || s.mapped.Load() != nil {
		return
	}
	b, err := mapFile(s.file, s.size)
	if err != nil {
		return
	}
	s.mapped.Store(&b)
}

/**
* mapActive: Mapea en memoria el segmento activo con capacidad para capacity bytes (o su tamaño
* actual, si es mayor). El mapeo es compartido: lo que write agrega se ve en él, y readAt solo lee
* de él hasta visible; lo que queda fuera (un registro mayor que la capacidad) se lee con ReadAt.
* @param capacity int64
**/
func (s *segment) mapActive(capacity int64) {
	if s.file == nil || s.mapped.Load() != nil {
		return
	}
	size := max(capacity, s.size)
	if size <= 0 {
		return
	}
	b, err := mapFile(s.file, size)
	if err != nil {
		return
	}
	s.mapped.Store(&b)
}

/**
* acquire: Registra un lector; solo se llama mientras el segmento sigue en el store (bajo indexMu).
**/
func (s *segment) acquire() {
	s.readers.Add(1)
}

/**
* release: Libera un lector y cierra el archivo si el segmento fue retirado y era el último.
**/
func (s *segment) release() {
	if s.readers.Add(-1) == 0 && s.retired.Load() {
		s.close()
	}
}

/**
* retire: Saca el segmento del store; el archivo se cierra cuando no quedan lectores.
**/
func (s *segment) retire() {
	s.retired.Store(true)
	if s.readers.Load() == 0 {
		s.close()
	}
}

/**
* newSegment: Crea un segmento de escritura.
* @param file *os.File, size int64, name string
* @return *segment
**/
func newSegment(file *os.File, size int64, name string) *segment {
	result := &segment{
		file:     file,
		size:     size,
		name:     name,
		readOnly: false,
	}
	result.visible.Store(size)
	return result
}

/**
* newReadOnlySegment: Crea un segmento que no acepta escrituras.
* @param file *os.File, size int64, name string
* @return *segment
**/
func newReadOnlySegment(file *os.File, size int64, name string) *segment {
	result := &segment{
		file:     file,
		size:     size,
		name:     name,
		readOnly: true,
	}
	result.visible.Store(size)
	return result
}

/**
* flush: Fuerza a disco lo escrito (fsync).
* @return error
**/
func (s *segment) flush() error {
	if s.readOnly {
		return nil
	}
	if s.file == nil {
		return errors.New(msg.MSG_STORE_FILE_IS_NIL)
	}
	return s.file.Sync()
}

/**
* seal: Hace fsync y deja el segmento en solo lectura sin cerrar el archivo.
* @return error
**/
func (s *segment) seal() error {
	if s.readOnly {
		return nil
	}

	if err := s.writeError(); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	s.readOnly = true
	s.mapReadOnly()
	return nil
}

/**
* close: Hace fsync si es de escritura y cierra el archivo.
* @return error
**/
func (s *segment) close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		if s.file == nil {
			return
		}
		// Sin lectores (retire/release): nadie usa el mapeo.
		if b := s.mapped.Swap(nil); b != nil {
			unmapFile(*b)
		}
		if s.readOnly {
			closeErr = s.file.Close()
			return
		}
		if err := s.flush(); err != nil {
			closeErr = err
			return
		}
		closeErr = s.file.Close()
	})

	return closeErr
}

/**
* toJson: Retorna el segmento como JSON.
* @return et.Json
**/
func (s *segment) toJson() et.Json {
	return et.Json{
		"file": s.file.Name(),
		"size": s.size,
		"name": s.name,
	}
}

/**
* toString: Retorna el segmento como texto.
* @return string
**/
func (s *segment) toString() string {
	return s.toJson().ToString()
}

/**
* readAt: Lee len(b) bytes desde off.
* @param b []byte, off int64
* @return int, error
**/
func (s *segment) readAt(b []byte, off int64) (int, error) {
	// Del mapeo solo lo que ya se escribió (más allá del archivo el acceso fallaría); el resto,
	// incluido leer pasado el final (io.EOF), con ReadAt.
	if m := s.mapped.Load(); m != nil {
		end := min(s.visible.Load(), int64(len(*m)))
		if off >= 0 && off+int64(len(b)) <= end {
			return copy(b, (*m)[off:]), nil
		}
	}
	if s.file == nil {
		return 0, errors.New(msg.MSG_STORE_FILE_IS_NIL)
	}
	return s.file.ReadAt(b, off)
}

/**
* write: Escribe b de forma síncrona; tras el primer error, toda escritura falla con ese error.
* @param b []byte
* @return error
**/
func (s *segment) write(b []byte) error {
	if s.file == nil {
		return errors.New(msg.MSG_STORE_FILE_IS_NIL)
	}
	if s.readOnly {
		return errors.New(msg.MSG_STORE_IS_READ_ONLY)
	}
	if err := s.writeError(); err != nil {
		return err
	}

	if _, err := s.file.Write(b); err != nil {
		s.errMu.Lock()
		s.writeErr = err
		s.errMu.Unlock()
		return err
	}
	s.visible.Add(int64(len(b)))

	return nil
}

/**
* writeError: Retorna el primer error de escritura, si lo hubo.
* @return error
**/
func (s *segment) writeError() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.writeErr
}

/**
* writeRecord: Escribe encabezado y datos de un registro en una sola escritura.
* @param lsn uint64, id string, data []byte, status byte
* @return *recordRef, error
**/
func (s *segment) writeRecord(lsn uint64, id string, data []byte, status byte) (*recordRef, error) {
	h, header, err := newRecordHeaderAt(lsn, id, data, status)
	if err != nil {
		return nil, err
	}

	// Encabezado y datos en una sola escritura: nunca se lee uno sin el otro.
	offset := s.size
	record := make([]byte, 0, len(header)+len(data))
	record = append(record, header...)
	record = append(record, data...)
	if err := s.write(record); err != nil {
		return nil, err
	}

	s.size += h.recordSize()
	return &recordRef{
		offset: offset,
		length: h.DataLen,
		crc:    h.CRC,
		hasCRC: true,
	}, nil
}

/**
* decodeFixedHeader: Decodifica la parte fija del encabezado.
* Formato: [LSN:8][DataLen:4][CRC:4][IDLen:2][ID][Status:1][Data]
* @param b []byte (at least 18 bytes)
* @return recordHeader
**/
func decodeFixedHeader(b []byte) recordHeader {
	return recordHeader{
		LSN:     binary.BigEndian.Uint64(b[0:8]),
		DataLen: getUint32(b[8:12]),
		CRC:     getUint32(b[12:16]),
		IDLen:   getUint16(b[16:18]),
	}
}

/**
* decodeRecord: Valida un registro completo (id, longitud, estado Active y CRC) y retorna sus datos.
* @param buf []byte, ref *recordRef, id string
* @return []byte, error
**/
func decodeRecord(buf []byte, ref *recordRef, id string) ([]byte, error) {
	h := decodeFixedHeader(buf)
	idLen := int(h.IDLen)
	if idLen != len(id) || h.DataLen != ref.length || len(buf) != int(fixedHeaderSize)+idLen+int(h.DataLen) {
		return nil, errors.New(msg.MSG_STORE_CORRUPTED_RECORD)
	}
	if string(buf[18:18+idLen]) != id || buf[18+idLen] != Active {
		return nil, errors.New(msg.MSG_STORE_CORRUPTED_RECORD)
	}

	data := buf[int(fixedHeaderSize)+idLen:]
	if checksum(data) != h.CRC {
		return nil, errors.New(msg.MSG_STORE_CORRUPTED_RECORD)
	}

	return data, nil
}

/**
* readRecord: Lee el registro de id con una sola lectura a disco.
* @param ref *recordRef, id string
* @return []byte, error
**/
func (s *segment) readRecord(ref *recordRef, id string) ([]byte, error) {
	buf := make([]byte, int(fixedHeaderSize)+len(id)+int(ref.length))
	if _, err := s.readAt(buf, ref.offset); err != nil {
		return nil, err
	}

	return decodeRecord(buf, ref, id)
}

/**
* readVersion: readRecord más el LSN del registro (su versión).
* @param ref *recordRef, id string
* @return []byte, uint64, error
**/
func (s *segment) readVersion(ref *recordRef, id string) ([]byte, uint64, error) {
	buf := make([]byte, int(fixedHeaderSize)+len(id)+int(ref.length))
	if _, err := s.readAt(buf, ref.offset); err != nil {
		return nil, 0, err
	}

	data, err := decodeRecord(buf, ref, id)
	if err != nil {
		return nil, 0, err
	}
	return data, decodeFixedHeader(buf).LSN, nil
}

/**
* read: Lee el registro de ref cuando no se conoce el id (dos lecturas).
* @param ref *recordRef
* @return []byte, error
**/
func (s *segment) read(ref *recordRef) ([]byte, error) {
	head := make([]byte, 18)
	if _, err := s.readAt(head, ref.offset); err != nil {
		return nil, err
	}

	idLen := int(getUint16(head[16:18]))
	buf := make([]byte, int(fixedHeaderSize)+idLen+int(ref.length))
	if _, err := s.readAt(buf, ref.offset); err != nil {
		return nil, err
	}

	return decodeRecord(buf, ref, string(buf[18:18+idLen]))
}

/**
* scan: Recorre los registros desde offset en orden de escritura; se detiene en silencio
* al final del archivo o en el primer registro corrupto.
* @param offset int64, fn func(offset int64, h recordHeader, data []byte) error
* @return error
**/
func (s *segment) scan(offset int64, fn func(offset int64, h recordHeader, data []byte) error) error {
	for {
		fixed := make([]byte, fixedHeaderSize)
		n, err := s.readAt(fixed, offset)
		if err != nil {
			if errors.Is(err, io.EOF) || n < len(fixed) {
				return nil
			}
			return err
		}

		h := decodeFixedHeader(fixed)
		if h.IDLen == 0 || h.IDLen > maxIdLen {
			return nil // corrupción → parar seguro
		}

		idBytes := make([]byte, h.IDLen)
		if _, err := s.readAt(idBytes, offset+18); err != nil {
			return nil
		}
		h.ID = string(idBytes)

		statusByte := make([]byte, 1)
		if _, err := s.readAt(statusByte, offset+18+int64(h.IDLen)); err != nil {
			return nil
		}
		h.Status = statusByte[0]

		var data []byte
		if h.DataLen > 0 {
			data = make([]byte, h.DataLen)
			if _, err := s.readAt(data, offset+18+int64(h.IDLen)+1); err != nil {
				return nil
			}
			if checksum(data) != h.CRC {
				return nil // registro corrupto → detener el escaneo
			}
		}

		if err := fn(offset, h, data); err != nil {
			return err
		}

		offset += h.recordSize()
	}
}
