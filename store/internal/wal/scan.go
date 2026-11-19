package wal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Start es la posición del primer registro de un WAL nuevo: el punto de rehacer de un motor que nunca hizo un
// checkpoint.
var Start = Position{Segment: 1, Offset: 0, LSN: 1}

// scanBufferSize es el búfer de lectura de un recorrido: la recuperación lee los segmentos en secuencia.
const scanBufferSize = 1 << 20

// reader lee los segmentos de un directorio del WAL.
type reader struct {
	fs  vfs.FS
	dir string
}

// NewReader devuelve un Reader sobre el WAL del motor de dir. Lee los segmentos que hay al empezar cada Scan: un
// segmento creado después no se ve hasta el próximo.
func NewReader(fsys vfs.FS, dir string) Reader {
	return &reader{fs: fsys, dir: filepath.Join(dir, Dir)}
}

// Scan implementa Reader (ver su documentación en wal.go).
func (r *reader) Scan(from Position, fn func(rec *Record) error) (Position, error) {
	return scan(r.fs, r.dir, from, fn, nil)
}

// scan recorre el WAL de dir desde from (ver Reader.Scan). onEnd, si no es nil, recibe dónde termina la parte válida
// de cada segmento recorrido (Open lo usa para SizeSince).
func scan(fsys vfs.FS, dir string, from Position, fn func(rec *Record) error, onEnd func(num uint64, end int64)) (Position, error) {
	if from.LSN == 0 || from.Segment == 0 || from.Offset < 0 {
		return from, fmt.Errorf(MSG_BAD_POSITION, from)
	}
	nums, err := listSegments(fsys, dir)
	if err != nil {
		return from, err
	}
	if !slices.Contains(nums, from.Segment) {
		return from, fmt.Errorf(MSG_SEGMENT_GONE, from.Segment)
	}

	s := newSegmentScanner()
	pos := from
	for {
		pos, err = s.records(fsys, dir, pos, fn)
		if err != nil {
			return pos, err
		}
		if onEnd != nil {
			onEnd(pos.Segment, pos.Offset)
		}

		// El registro siguiente no está en este segmento: o no cabía y empieza el siguiente, o el WAL terminó.
		next := pos.Segment + 1
		if !slices.Contains(nums, next) {
			return pos, nil
		}
		first, ok, err := s.firstLSN(fsys, dir, next)
		if err != nil {
			return pos, err
		}
		switch {
		case !ok || first < pos.LSN:
			return pos, nil // inválido o reciclado: el final del WAL
		case first > pos.LSN:
			return pos, fmt.Errorf(MSG_GAP, ErrCorrupted, next, first, pos.LSN)
		}
		pos = Position{Segment: next, Offset: 0, LSN: pos.LSN}
	}
}

// segmentScanner guarda los búferes de un recorrido: no se asigna uno por registro ni por segmento.
type segmentScanner struct {
	buf []byte // un registro: MaxRecordSize, así ningún largo leído del disco hace pedir más memoria
	rd  *bufio.Reader
	rec Record
}

// newSegmentScanner arma los búferes de un recorrido.
func newSegmentScanner() *segmentScanner {
	return &segmentScanner{buf: make([]byte, MaxRecordSize), rd: bufio.NewReaderSize(nil, scanBufferSize)}
}

// records llama a fn con cada registro válido del segmento pos.Segment desde pos.Offset, mientras cada uno tenga el
// lsn esperado, y devuelve la posición siguiente al último válido.
func (s *segmentScanner) records(fsys vfs.FS, dir string, pos Position, fn func(rec *Record) error) (Position, error) {
	file, err := fsys.OpenFile(segmentPath(dir, pos.Segment), os.O_RDONLY)
	if err != nil {
		return pos, err
	}
	defer file.Close()
	size, err := file.Size()
	if err != nil {
		return pos, err
	}
	if pos.Offset >= size {
		return pos, nil
	}

	s.rd.Reset(io.NewSectionReader(file, pos.Offset, size-pos.Offset))
	for {
		n, ok, err := s.next(size - pos.Offset)
		if err != nil || !ok || s.rec.LSN != pos.LSN {
			return pos, err
		}
		if err := fn(&s.rec); err != nil {
			return pos, err
		}
		pos.Offset += int64(n)
		pos.LSN++
	}
}

// next lee del lector el registro siguiente, si cabe en los left bytes que quedan del segmento. Devuelve su largo y
// si es válido; un final de archivo no es error: es donde termina la parte válida.
func (s *segmentScanner) next(left int64) (int, bool, error) {
	if left < HeaderSize {
		return 0, false, nil
	}
	if _, err := io.ReadFull(s.rd, s.buf[:HeaderSize]); err != nil {
		return 0, false, readError(err)
	}
	total := totalLenOf(s.buf)
	if total < HeaderSize || total > MaxRecordSize || int64(total) > left {
		return 0, false, nil
	}
	if _, err := io.ReadFull(s.rd, s.buf[HeaderSize:total]); err != nil {
		return 0, false, readError(err)
	}
	n, ok := decodeRecord(s.buf[:total], &s.rec)
	return n, ok, nil
}

// firstLSN lee el primer registro del segmento num. Devuelve su lsn y si es válido.
func (s *segmentScanner) firstLSN(fsys vfs.FS, dir string, num uint64) (page.LSN, bool, error) {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDONLY)
	if err != nil {
		return 0, false, err
	}
	defer file.Close()
	size, err := file.Size()
	if err != nil {
		return 0, false, err
	}
	s.rd.Reset(io.NewSectionReader(file, 0, size))
	_, ok, err := s.next(size)
	return s.rec.LSN, ok, err
}

// readError: un final de archivo (completo o a medias) no es un error de lectura.
func readError(err error) error {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil
	}
	return err
}
