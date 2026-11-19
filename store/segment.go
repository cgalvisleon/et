package store

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const segmentExt = ".seg"

/**
* segment: Un archivo del log. size es dónde va la próxima reserva (lo protege writeMu del Store); inflight cuenta las
* escrituras reservadas que aún no terminan, para poder sellarlo. mapped es el archivo mapeado a memoria, de donde lee
* Read (ver mapSegment); nil fuera de unix.
**/
type segment struct {
	num      int
	path     string
	file     *os.File
	size     int64
	inflight sync.WaitGroup
	mapped   []byte
}

/**
* segmentName: El nombre del archivo del segmento num: diez dígitos, así el orden alfabético es el numérico.
* @param num int
* @return string
**/
func segmentName(num int) string {
	return fmt.Sprintf("%010d%s", num, segmentExt)
}

/**
* createSegment: Crea el archivo del segmento num en dir, hace durable su nombre (fsync del directorio) y lo mapea con
* capacity bytes (el tamaño de segmento: hasta ahí crece).
* @param dir string, num int, capacity int64
* @return *segment, error
**/
func createSegment(dir string, num int, capacity int64) (*segment, error) {
	path := filepath.Join(dir, segmentName(num))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syncDir(dir); err != nil {
		file.Close()
		return nil, err
	}
	return mapSegment(&segment{num: num, path: path, file: file}, capacity)
}

/**
* openSegment: Abre un segmento existente para leerlo y, si es el activo, seguir escribiendo, y lo mapea con capacity
* bytes o con su largo si es mayor.
* @param dir string, num int, capacity int64
* @return *segment, error
**/
func openSegment(dir string, num int, capacity int64) (*segment, error) {
	path := filepath.Join(dir, segmentName(num))
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	return mapSegment(&segment{num: num, path: path, file: file}, max(capacity, info.Size()))
}

/**
* mapSegment: Mapea a memoria los primeros capacity bytes del archivo de seg, para que Read lea sin llamada al
* sistema. Con pread, todas las lecturas en paralelo pasaban por el kernel y no escalaban con los núcleos (el 99% del
* tiempo de BenchmarkRead con 10 núcleos era el pread; ver doc.go, «Lectura»). El mapa se hace una vez y no cambia
* hasta cerrar: si se hiciera al sellar, habría que rehacerlo con lectores encima.
* @param seg *segment, capacity int64
* @return *segment, error (cierra el archivo si falla)
**/
func mapSegment(seg *segment, capacity int64) (*segment, error) {
	mapped, err := mapFile(seg.file, capacity)
	if err != nil {
		seg.file.Close()
		return nil, err
	}
	seg.mapped = mapped
	return seg, nil
}

/**
* listSegments: Los números de los segmentos de dir, en orden; falla si falta uno en medio.
* @param dir string
* @return []int, error
**/
func listSegments(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var nums []int
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, segmentExt) {
			continue
		}
		num, err := strconv.Atoi(strings.TrimSuffix(name, segmentExt))
		if err != nil || num <= 0 {
			return nil, fmt.Errorf(MSG_BAD_SEGMENT, name)
		}
		nums = append(nums, num)
	}
	sort.Ints(nums)
	for i := 1; i < len(nums); i++ {
		if nums[i] != nums[i-1]+1 {
			return nil, fmt.Errorf(MSG_SEGMENT_MISSING, nums[i-1]+1, nums[i], nums[i-1])
		}
	}
	return nums, nil
}

/**
* writeAt: Escribe un registro completo en su lugar reservado.
* @param buf []byte, off int64
* @return error
**/
func (s *segment) writeAt(buf []byte, off int64) error {
	_, err := s.file.WriteAt(buf, off)
	return err
}

/**
* seal: Espera a que terminen las escrituras reservadas y hace fsync: desde aquí el segmento no cambia.
* Se llama con writeMu tomado (nadie puede reservar mientras tanto).
* @return error
**/
func (s *segment) seal() error {
	s.inflight.Wait()
	return s.file.Sync()
}

/**
* view: Los size bytes desde off, como una vista del mapa (sin copiar ni llamar al sistema), si caben en él. Solo se
* pide lo que ya está escrito: un registro al que apunta el índice (ver doc.go, «Lectura»).
* @param off int64, size int
* @return []byte, bool (false si no hay mapa o el rango se sale de él)
**/
func (s *segment) view(off int64, size int) ([]byte, bool) {
	end := off + int64(size)
	if off < 0 || end > int64(len(s.mapped)) {
		return nil, false
	}
	return s.mapped[off:end:end], true
}

/**
* readAt: Lee len(buf) bytes desde off (pread). Es el respaldo de view: fuera de unix, o para un registro más grande
* que el tamaño de segmento, que no cabe en el mapa.
* @param buf []byte, off int64
* @return error
**/
func (s *segment) readAt(buf []byte, off int64) error {
	_, err := s.file.ReadAt(buf, off)
	return err
}

/**
* close: Deshace el mapa y cierra el archivo. Ninguna lectura puede estar usando el mapa (ver readers.go).
* @return error
**/
func (s *segment) close() error {
	err := unmapFile(s.mapped)
	s.mapped = nil
	return errors.Join(err, s.file.Close())
}

/**
* scanned: Un registro válido encontrado al recorrer un segmento.
**/
type scanned struct {
	id  string
	ref ref
}

/**
* scan: Recorre el segmento desde el principio y llama a fn con cada registro válido, en orden. Se detiene en el
* primer registro inválido o incompleto, o en el que fn rechaza (devuelve false), y devuelve dónde termina la parte
* válida; reached indica si se llegó al final del archivo sin encontrar nada inválido. Un registro que no cabe en lo
* que queda del archivo es incompleto y se detecta antes de asignar memoria: un dataLen con basura (hasta 1 GiB) no
* debe hacer que Open pida esa memoria por un archivo de pocos bytes. El búfer se reutiliza entre registros: recuperar
* no asigna uno por registro.
* @param fn func(rec scanned) bool (false: el registro no vale y el recorrido termina ahí)
* @return int64 (fin de la parte válida), bool (reached), error (solo de lectura del disco)
**/
func (s *segment) scan(fn func(rec scanned) bool) (int64, bool, error) {
	info, err := s.file.Stat()
	if err != nil {
		return 0, false, err
	}
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return 0, false, err
	}
	r := bufio.NewReaderSize(s.file, 1<<20)
	var off int64
	buf := make([]byte, headerSize, 4096)
	for {
		if _, err := io.ReadFull(r, buf[:headerSize]); err != nil {
			return off, errors.Is(err, io.EOF), readError(err)
		}
		h := decodeHeader(buf)
		if h.idLen == 0 || h.dataLen > maxDataLen || off+int64(h.size()) > info.Size() {
			return off, false, nil
		}
		buf = grow(buf, h.size())
		if _, err := io.ReadFull(r, buf[headerSize:]); err != nil {
			return off, false, readError(err)
		}
		if !validRecord(h, buf) {
			return off, false, nil
		}
		at := ref{segment: s.num, offset: off, size: uint32(h.size()), deleted: h.kind == kindDelete, lsn: h.lsn}
		if !fn(scanned{id: recordID(h, buf), ref: at}) {
			return off, false, nil
		}
		off += int64(h.size())
	}
}

/**
* grow: buf con largo size, conservando sus primeros bytes; solo asigna si no cabe en su capacidad.
* @param buf []byte, size int
* @return []byte
**/
func grow(buf []byte, size int) []byte {
	if size <= cap(buf) {
		return buf[:size]
	}
	bigger := make([]byte, size)
	copy(bigger, buf)
	return bigger
}

/**
* readError: Un final de archivo (completo o a medias) no es un error de lectura: es donde termina el log.
* @param err error
* @return error
**/
func readError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return nil
	}
	return err
}

/**
* truncate: Corta el segmento en off y hace fsync: lo que seguía no se había confirmado.
* @param off int64
* @return error
**/
func (s *segment) truncate(off int64) error {
	if err := s.file.Truncate(off); err != nil {
		return err
	}
	return s.file.Sync()
}
