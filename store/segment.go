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
* escrituras reservadas que aún no terminan, para poder sellarlo.
**/
type segment struct {
	num      int
	path     string
	file     *os.File
	size     int64
	inflight sync.WaitGroup
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
* createSegment: Crea el archivo del segmento num en dir y hace durable su nombre (fsync del directorio).
* @param dir string, num int
* @return *segment, error
**/
func createSegment(dir string, num int) (*segment, error) {
	path := filepath.Join(dir, segmentName(num))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syncDir(dir); err != nil {
		file.Close()
		return nil, err
	}
	return &segment{num: num, path: path, file: file}, nil
}

/**
* openSegment: Abre un segmento existente para leerlo y, si es el activo, seguir escribiendo.
* @param dir string, num int
* @return *segment, error
**/
func openSegment(dir string, num int) (*segment, error) {
	path := filepath.Join(dir, segmentName(num))
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	return &segment{num: num, path: path, file: file}, nil
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
* close: Cierra el archivo.
* @return error
**/
func (s *segment) close() error {
	return s.file.Close()
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
* primer registro inválido o incompleto y devuelve dónde termina la parte válida; reached indica si se llegó al final
* del archivo sin encontrar nada inválido.
* @param fn func(rec scanned)
* @return int64 (fin de la parte válida), bool (reached), error (solo de lectura del disco)
**/
func (s *segment) scan(fn func(rec scanned)) (int64, bool, error) {
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return 0, false, err
	}
	r := bufio.NewReaderSize(s.file, 1<<20)
	var off int64
	head := make([]byte, headerSize)
	for {
		if _, err := io.ReadFull(r, head); err != nil {
			return off, errors.Is(err, io.EOF), readError(err)
		}
		h := decodeHeader(head)
		if h.idLen == 0 || h.dataLen > maxDataLen {
			return off, false, nil
		}
		buf := make([]byte, h.size())
		copy(buf, head)
		if _, err := io.ReadFull(r, buf[headerSize:]); err != nil {
			return off, false, readError(err)
		}
		if !validRecord(h, buf) {
			return off, false, nil
		}
		id := string(buf[headerSize : headerSize+h.idLen])
		fn(scanned{id: id, ref: ref{segment: s.num, offset: off, size: uint32(h.size()), lsn: h.lsn}})
		off += int64(h.size())
	}
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
