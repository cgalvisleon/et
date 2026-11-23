package wal

import (
	"fmt"

	"github.com/cgalvisleon/et/store/internal/page"
)

// Lo que el archivo continuo del motor (tarea 66) necesita del WAL: qué segmentos están sellados, qué rango de lsn
// tiene cada uno y sellar el activo a pedido. La retención es Hold (ver «Modo respaldo y retención» en doc.go).

// SegmentInfo describe un segmento sellado: su número, el lsn de su primer registro y la posición siguiente al
// último (End.Offset son los bytes con registros válidos; lo que sigue en el archivo son ceros de la preasignación o
// registros viejos de un repuesto reciclado, que Scan descarta por su lsn).
type SegmentInfo struct {
	Num   uint64
	First page.LSN
	End   Position
}

// SegmentName es el nombre del archivo del segmento num dentro de Dir ("0000000003.seg").
func SegmentName(num uint64) string {
	return segmentName(num)
}

// ActiveSegment es el número del segmento activo, sin candados: todos los menores que existen están sellados (sus
// escrituras terminadas y con fsync) y no cambian más.
func (w *WAL) ActiveSegment() uint64 {
	return w.activeNum.Load()
}

// Segments devuelve los números de los segmentos que hay en el directorio, en orden (los mayores que el activo son
// repuestos reciclados).
func (w *WAL) Segments() ([]uint64, error) {
	return listSegments(w.fs, w.dir)
}

// Switch sella el segmento activo y pasa al siguiente aunque no esté lleno (como pg_switch_wal): así el archivo
// continuo puede copiarlo sin esperar a que se llene. Con el activo vacío no hace nada. Bajo writeMu, como una
// rotación: con el disco lleno para preasignar el siguiente devuelve ErrNoRoom sin tocar nada.
// Devuelve el número del segmento sellado (0 si no selló nada).
func (w *WAL) Switch() (uint64, error) {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return 0, err
	}
	if w.active.size <= SegmentHeaderSize {
		return 0, nil
	}
	num := w.active.num
	if err := w.rotate(); err != nil {
		return 0, w.reserveError(err)
	}
	return num, nil
}

// StartSegment deja el WAL listo para que el próximo registro caiga en un segmento sin registros de antes y con
// encabezado de FormatVersion (Promote de una réplica, tarea 138): con registros en el activo lo sella y rota como
// Switch; sin registros, si su encabezado es de una versión anterior lo reescribe con fsync (como upgrade en Open).
// Bajo writeMu: con el disco lleno para preasignar el siguiente devuelve ErrNoRoom sin tocar nada.
// Devuelve el número del segmento sellado (0 si no selló nada).
func (w *WAL) StartSegment() (uint64, error) {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return 0, err
	}
	if w.active.size > SegmentHeaderSize {
		num := w.active.num
		if err := w.rotate(); err != nil {
			return 0, w.reserveError(err)
		}
		return num, nil
	}
	h, state, err := readHeader(w.active.file)
	if err != nil {
		return 0, err
	}
	if state == headerValid && h.Version >= FormatVersion {
		return 0, nil
	}
	if err := w.writeHeader(w.active.file, w.active.num, page.LSN(w.lastLSN.Load())+1); err != nil {
		return 0, w.reserveError(err)
	}
	return 0, nil
}

// SegmentInfo describe el segmento sellado num. El final sale de lo anotado al sellarlo o al recorrerlo en Open; un
// segmento anterior al punto de rehacer de Open (retenido por el archivo) se recorre una vez, sin writeMu (ya no
// cambia), y queda anotado.
func (w *WAL) SegmentInfo(num uint64) (SegmentInfo, error) {
	w.writeMu.Lock()
	active := w.active.num
	end, known := w.sizes[num]
	w.writeMu.Unlock()
	if num >= active {
		return SegmentInfo{}, fmt.Errorf(MSG_NOT_SEALED, num, active)
	}

	s := newSegmentScanner()
	first, ok, err := firstLSN(w.fs, w.dir, num)
	if err != nil {
		return SegmentInfo{}, err
	}
	if !ok {
		return SegmentInfo{}, fmt.Errorf(MSG_SEGMENT_EMPTY, ErrCorrupted, num)
	}
	if !known {
		start := Position{Segment: num, Offset: SegmentHeaderSize, LSN: first}
		if end, err = s.records(w.fs, w.dir, start, func(Position, []byte, *Record) error { return nil }); err != nil {
			return SegmentInfo{}, err
		}
		w.writeMu.Lock()
		w.sizes[num] = end // si Recycle lo quitó mientras tanto, su próxima vuelta borra la anotación
		w.writeMu.Unlock()
	}
	return SegmentInfo{Num: num, First: first, End: end}, nil
}
