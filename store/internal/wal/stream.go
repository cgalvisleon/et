package wal

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/page"
)

// Lo que la réplica (tarea 46) necesita del WAL: leer los registros tal como están en los segmentos, con su
// posición (Read, Locate), y en la réplica escribirlos en la misma posición (Append), así el WAL de la réplica es una
// copia byte a byte del primario y los puntos de rehacer de sus checkpoints sirven igual en los dos.

// errEnough corta un recorrido de Read o Locate: no es un error para quien llama.
var errEnough = errors.New("wal: enough")

/**
* Read recorre los registros desde from (la posición exacta de un registro) hasta el de lsn stop inclusive, o hasta
* juntar maxBytes (siempre entrega al menos uno si lo hay), y llama a fn con la posición, los bytes tal como están en
* el segmento y el registro decodificado (raw y rec valen solo durante fn). Lee del disco, no de la memoria: el
* llamador pone stop ≤ Durable (o Published), así nunca mira bytes que se están escribiendo.
* @param from Position, stop page.LSN, maxBytes int
* @param fn func(at Position, raw []byte, rec *Record) error
* @return Position: la siguiente al último entregado (from si ninguno), error: ErrRemoved si el segmento de from ya no está
**/
func (w *WAL) Read(from Position, stop page.LSN, maxBytes int, fn func(at Position, raw []byte, rec *Record) error) (Position, error) {
	if stop < from.LSN {
		return from, nil
	}
	total, next := 0, from
	_, err := scan(w.fs, w.dir, from, stop, func(at Position, raw []byte, rec *Record) error {
		if total > 0 && total+len(raw) > maxBytes {
			return errEnough
		}
		if err := fn(at, raw, rec); err != nil {
			return err
		}
		total += len(raw)
		next = Position{Segment: at.Segment, Offset: at.Offset + int64(len(raw)), LSN: at.LSN + 1}
		return nil
	}, nil)
	if errors.Is(err, errEnough) {
		err = nil
	}
	return next, err
}

/**
* Locate busca la posición del registro lsn, que tiene que estar escrito (≤ Published): el segmento es el mayor (sin
* contar los repuestos) cuyo primer registro tiene lsn ≤ lsn, y dentro se recorre hasta él.
* @param lsn page.LSN
* @return Position, error: ErrRemoved si es anterior al segmento más viejo que queda
**/
func (w *WAL) Locate(lsn page.LSN) (Position, error) {
	nums, err := listSegments(w.fs, w.dir)
	if err != nil {
		return Position{}, err
	}
	active := w.ActiveSegment()
	s := getScanner()
	defer putScanner(s)
	for i := len(nums) - 1; i >= 0; i-- {
		num := nums[i]
		if num > active {
			continue // un repuesto reciclado: registros viejos
		}
		first, ok, err := firstLSN(w.fs, w.dir, num)
		if err != nil {
			return Position{}, err
		}
		if !ok || first > lsn {
			continue
		}
		return w.locateIn(s, Position{Segment: num, Offset: SegmentHeaderSize, LSN: first}, lsn)
	}
	return Position{}, fmt.Errorf(MSG_LSN_GONE, ErrRemoved, lsn)
}

/**
* SegmentOf es el segmento donde está (o va a estar) el registro lsn: el mayor, sin contar los repuestos, cuyo primer
* registro tiene lsn ≤ lsn; el activo si lsn todavía no se escribió o el activo está vacío. Solo lee el primer
* registro de cada segmento que mira, del más nuevo al más viejo (lo usan las ranuras para retener).
* @param lsn page.LSN
* @return uint64, error: ErrRemoved si es anterior al segmento más viejo que queda
**/
func (w *WAL) SegmentOf(lsn page.LSN) (uint64, error) {
	active := w.ActiveSegment()
	if lsn > w.Published() {
		return active, nil
	}
	nums, err := listSegments(w.fs, w.dir)
	if err != nil {
		return 0, err
	}
	for i := len(nums) - 1; i >= 0; i-- {
		num := nums[i]
		if num > active {
			continue
		}
		first, ok, err := firstLSN(w.fs, w.dir, num)
		if err != nil {
			return 0, err
		}
		if ok && first <= lsn {
			return num, nil
		}
	}
	return 0, fmt.Errorf(MSG_LSN_GONE, ErrRemoved, lsn)
}

/**
* locateIn recorre un segmento desde su principio hasta el registro lsn.
* @param s *segmentScanner, start Position: el primer registro del segmento, lsn page.LSN
* @return Position, error
**/
func (w *WAL) locateIn(s *segmentScanner, start Position, lsn page.LSN) (Position, error) {
	if start.LSN == lsn {
		return start, nil
	}
	var at Position
	found := false
	end, err := s.records(w.fs, w.dir, start, func(pos Position, _ []byte, rec *Record) error {
		if rec.LSN != lsn {
			return nil
		}
		at, found = pos, true
		return errEnough
	})
	switch {
	case found:
		return at, nil
	case err != nil && !errors.Is(err, errEnough):
		return Position{}, err
	}
	return Position{}, fmt.Errorf(MSG_LSN_MISSING, lsn, end.LSN)
}

/**
* Append escribe en el WAL de una réplica un registro ya armado (raw: con su lsn y su crc, tal como lo leyó Read del
* primario) en la posición at, que tiene que ser exactamente la del próximo registro: el final del segmento activo,
* o el principio del siguiente (el primario selló el activo antes de llenarlo). Lo publica como Reserve + Write, sin
* fsync (WaitDurable lo hace).
* @param at Position, raw []byte: se copia
* @return error: ErrOutOfOrder (lsn o posición que no siguen), ErrNoRoom, ErrClosed o el fallo del WAL
**/
func (w *WAL) Append(at Position, raw []byte) error {
	var rec Record
	n, ok := decodeRecord(raw, &rec)
	if !ok || n != len(raw) {
		return fmt.Errorf(MSG_APPEND_RAW, ErrOutOfOrder)
	}
	res, err := w.placeRaw(at, rec.LSN, int64(n))
	if err != nil {
		return err
	}
	res.buf = bytes.Clone(raw)
	return res.Write()
}

/**
* placeRaw reserva, bajo writeMu, el lugar de un registro de la réplica en la posición at (rotando si at es el
* principio del segmento siguiente).
* @param at Position, lsn page.LSN, size int64
* @return *reservation, error
**/
func (w *WAL) placeRaw(at Position, lsn page.LSN, size int64) (*reservation, error) {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return nil, err
	}
	next := page.LSN(w.lastLSN.Load()) + 1
	rotate := at.Segment == w.active.num+1 && at.Offset == SegmentHeaderSize
	inPlace := at.Segment == w.active.num && at.Offset == w.active.size
	if lsn != next || at.LSN != lsn || !rotate && !inPlace || at.Offset+size > w.capacity() {
		return nil, fmt.Errorf(MSG_APPEND_ORDER, ErrOutOfOrder, lsn, at, w.active.num, w.active.size, next)
	}
	if rotate {
		if err := w.rotate(); err != nil {
			return nil, w.reserveError(err)
		}
	}
	if w.active.size+size > w.active.allocated {
		if err := w.fillActive(); err != nil {
			return nil, w.reserveError(err)
		}
	}
	seg := w.active
	seg.size += size
	w.written.Add(size)
	w.lastLSN.Store(uint64(lsn))
	return &reservation{w: w, seg: seg, offset: at.Offset, size: size, lsn: lsn}, nil
}

/**
* SetRedo cambia el punto de rehacer vigente sin escribir nada: lo usa la réplica, cuyos checkpoints (restartpoints)
* toman el punto de rehacer de un KindCheckpoint del primario ya rehecho.
* @param redo Position
**/
func (w *WAL) SetRedo(redo Position) {
	w.redo.Store(&redo)
}

/**
* Decode lee un registro entero (los bytes de un segmento, como los entrega Read) sin confiar en nada.
* @param raw []byte
* @return *Record: sus Image, Delta y Main son vistas de raw, error: ErrOutOfOrder si no es exactamente un registro válido
**/
func Decode(raw []byte) (*Record, error) {
	rec := &Record{}
	n, ok := decodeRecord(raw, rec)
	if !ok || n != len(raw) {
		return nil, fmt.Errorf(MSG_APPEND_RAW, ErrOutOfOrder)
	}
	return rec, nil
}
