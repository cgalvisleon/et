package wal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Start es la posición del primer registro de un WAL nuevo: el punto de rehacer de un motor que nunca hizo un
// checkpoint.
var Start = Position{Segment: 1, Offset: SegmentHeaderSize, LSN: 1}

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
	return scan(r.fs, r.dir, from, 0, recordsOnly(fn), nil)
}

// scan recorre el WAL de dir desde from (ver Reader.Scan). Con stop > 0 termina antes del primer registro de lsn
// mayor que stop (el objetivo de una recuperación, Options.StopLSN), como si el WAL terminara ahí. onEnd, si no es
// nil, recibe dónde termina la parte válida de cada segmento recorrido (Open lo usa para SizeSince y SegmentInfo).
func scan(fsys vfs.FS, dir string, from Position, stop page.LSN, fn visitFunc, onEnd func(end Position)) (Position, error) {
	if from.LSN == 0 || from.Segment == 0 || from.Offset < SegmentHeaderSize {
		return from, fmt.Errorf(MSG_BAD_POSITION, from)
	}
	nums, err := listSegments(fsys, dir)
	if err != nil {
		return from, err
	}
	if !slices.Contains(nums, from.Segment) {
		return from, fmt.Errorf(MSG_SEGMENT_GONE, ErrRemoved, from.Segment)
	}
	if err := checkStartHeader(fsys, dir, from); err != nil {
		return from, err
	}

	s := getScanner()
	defer putScanner(s)
	s.stop = stop
	pos := from
	for {
		pos, err = s.records(fsys, dir, pos, fn)
		if err != nil {
			return pos, err
		}
		if onEnd != nil {
			onEnd(pos)
		}
		if s.stopped(pos) {
			return pos, nil
		}

		// El registro siguiente no está en este segmento: o no cabía y empieza el siguiente, o el WAL terminó.
		next := pos.Segment + 1
		if !slices.Contains(nums, next) {
			return pos, nil
		}
		h, state, err := readHeaderOf(fsys, dir, next)
		if err != nil {
			return pos, err
		}
		switch {
		case state == headerVersion:
			return pos, versionError(next, h.Version)
		case state != headerValid || h.Num != next || h.First < pos.LSN:
			return pos, nil // sin encabezado, roto o de un repuesto reciclado: el final del WAL (checkSealed lo revisa)
		case h.First > pos.LSN:
			return pos, fmt.Errorf(MSG_GAP, ErrCorrupted, next, h.First, pos.LSN)
		}
		pos = Position{Segment: next, Offset: SegmentHeaderSize, LSN: pos.LSN}
	}
}

// segmentScanner guarda los búferes de un recorrido: no se asigna uno por registro ni por segmento.
type segmentScanner struct {
	buf  []byte // un registro: MaxRecordSize, así ningún largo leído del disco hace pedir más memoria
	rd   *bufio.Reader
	rec  Record
	stop page.LSN // 0 = sin objetivo; si no, no se pasa de este lsn
}

// stopped dice si el recorrido llegó al objetivo: el registro de pos ya pasa de stop.
func (s *segmentScanner) stopped(pos Position) bool {
	return s.stop > 0 && pos.LSN > s.stop
}

// newSegmentScanner arma los búferes de un recorrido.
func newSegmentScanner() *segmentScanner {
	return &segmentScanner{buf: make([]byte, MaxRecordSize), rd: bufio.NewReaderSize(nil, scanBufferSize)}
}

// scanners guarda los búferes de los recorridos terminados para reutilizarlos: un lector que pide el WAL a cada rato
// (WalSince, las ranuras) no asigna 1 MiB por pedido.
var scanners = sync.Pool{New: func() any { return newSegmentScanner() }}

// getScanner toma un recorrido de scanners; putScanner lo devuelve (sin referencias al último archivo).
func getScanner() *segmentScanner {
	s := scanners.Get().(*segmentScanner)
	s.stop = 0
	return s
}

// putScanner devuelve s a scanners.
func putScanner(s *segmentScanner) {
	s.rd.Reset(nil)
	scanners.Put(s)
}

// visitFunc recibe cada registro de un recorrido con su posición y sus bytes tal como están en el segmento (raw y
// rec valen solo durante la llamada).
type visitFunc func(at Position, raw []byte, rec *Record) error

// recordsOnly adapta una función que solo mira el registro a visitFunc.
func recordsOnly(fn func(rec *Record) error) visitFunc {
	return func(_ Position, _ []byte, rec *Record) error { return fn(rec) }
}

// records llama a fn con cada registro válido del segmento pos.Segment desde pos.Offset, mientras cada uno tenga el
// lsn esperado, y devuelve la posición siguiente al último válido.
func (s *segmentScanner) records(fsys vfs.FS, dir string, pos Position, fn visitFunc) (Position, error) {
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
	for !s.stopped(pos) {
		n, ok, err := s.next(size - pos.Offset)
		if err != nil || !ok || s.rec.LSN != pos.LSN {
			return pos, err
		}
		if err := fn(pos, s.buf[:n], &s.rec); err != nil {
			return pos, err
		}
		pos.Offset += int64(n)
		pos.LSN++
	}
	return pos, nil
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

// firstLSN lee del encabezado del segmento num el lsn de su primer registro (el que tiene o el que va a tener, si
// todavía no tiene ninguno). Devuelve si el encabezado es válido y de este segmento; otra versión es ErrFormatVersion.
func firstLSN(fsys vfs.FS, dir string, num uint64) (page.LSN, bool, error) {
	h, state, err := readHeaderOf(fsys, dir, num)
	if err != nil {
		return 0, false, err
	}
	if state == headerVersion {
		return 0, false, versionError(num, h.Version)
	}
	return h.First, state == headerValid && h.Num == num, nil
}

// readError: un final de archivo (completo o a medias) no es un error de lectura.
func readError(err error) error {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil
	}
	return err
}
