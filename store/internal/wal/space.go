package wal

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// zeros es el trozo de ceros con que se preasignan los segmentos. Nadie lo cambia.
var zeros = make([]byte, 64<<10)

// Room asegura lugar en el disco para seguir escribiendo antes de empezar una operación: al menos medio segmento ya
// preasignado entre el resto del activo y el siguiente (si está listo). Si falta, lo preasigna ahora: rellena con
// ceros el resto del activo y, si no alcanza, prepara el siguiente. Con el disco lleno devuelve ErrNoRoom sin dejar el
// WAL inutilizable; un fsync fallido sí lo deja. Sin Options.Preallocate no hace nada.
func (w *WAL) Room() error {
	if !w.opts.Preallocate {
		return nil
	}
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if err := w.writable(); err != nil {
		return err
	}
	need := w.opts.SegmentSize / 2
	if w.room() >= need {
		return nil
	}
	err := w.fillActive()
	if err == nil && w.room() < need {
		err = w.prepareNext()
	}
	if w.room() >= need {
		return nil
	}
	return w.reserveError(err)
}

// room son los bytes preasignados que quedan para registros: el resto del activo más el siguiente si está listo. Con
// writeMu tomado.
func (w *WAL) room() int64 {
	room := w.active.allocated - w.active.size
	if w.nextReady {
		room += w.opts.SegmentSize
	}
	return room
}

// fillActive rellena con ceros el resto del segmento activo (hasta SegmentSize) y hace fsync. Con el disco lleno
// queda preasignado lo que alcanzó. Con writeMu tomado.
func (w *WAL) fillActive() error {
	a := w.active
	if a.allocated >= w.opts.SegmentSize {
		return nil
	}
	end, err := w.fill(a.file, a.allocated)
	a.allocated = end
	return err
}

// prepareNext deja el segmento siguiente al activo creado (o el repuesto que ya existe) y relleno con ceros hasta
// SegmentSize, con fsync. Un segmento a medio rellenar por un disco lleno queda: el próximo intento sigue desde su
// largo, y como empieza en ceros (o con registros viejos de lsn menor) Scan lo toma por el final del WAL. Con writeMu
// tomado.
func (w *WAL) prepareNext() error {
	if w.nextReady {
		return nil
	}
	file, err := w.openNext(w.active.num + 1)
	if err != nil {
		return err
	}
	defer file.Close()
	size, err := file.Size()
	if err != nil {
		return w.spaceError(err)
	}
	if size < w.opts.SegmentSize {
		if _, err := w.fill(file, size); err != nil {
			return err
		}
	}
	w.nextReady = true
	return nil
}

// openNext abre el segmento num si ya existe (un repuesto) o lo crea y hace durable su nombre (fsync del directorio).
// Con writeMu tomado.
func (w *WAL) openNext(num uint64) (vfs.File, error) {
	path := segmentPath(w.dir, num)
	if num <= w.maxNum {
		file, err := w.fs.OpenFile(path, os.O_RDWR)
		if err != nil {
			return nil, w.spaceError(err)
		}
		return file, nil
	}
	file, err := w.fs.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return nil, w.spaceError(err)
	}
	w.maxNum = num // existe aunque el fsync del directorio falle: el WAL queda inutilizable igual
	if err := w.fs.SyncDir(w.dir); err != nil {
		file.Close()
		w.fail(err)
		return nil, failedError(err)
	}
	return file, nil
}

// fill escribe ceros en file desde from hasta SegmentSize y hace fsync. Devuelve hasta dónde quedó escrito. Una
// escritura que no encuentra lugar devuelve el error sin dejar el WAL inutilizable (no hay registros en juego); otro
// fallo de escritura o un fsync fallido sí lo dejan. Con writeMu tomado.
func (w *WAL) fill(file vfs.File, from int64) (int64, error) {
	for from < w.opts.SegmentSize {
		n := min(int64(len(zeros)), w.opts.SegmentSize-from)
		written, err := file.WriteAt(zeros[:n], from)
		from += int64(written)
		if err != nil {
			return from, w.spaceError(err)
		}
	}
	if err := file.Sync(); err != nil {
		w.fail(err)
		return from, failedError(err)
	}
	return from, nil
}

// spaceError devuelve err tal cual si es de disco lleno (no deja el WAL inutilizable); si no, deja el WAL
// inutilizable con err.
func (w *WAL) spaceError(err error) error {
	if vfs.IsNoSpace(err) {
		return err
	}
	w.fail(err)
	return failedError(err)
}

// reserveError es el error de una preasignación que no alcanzó: el del WAL si quedó inutilizable, si no ErrNoRoom
// con la causa.
func (w *WAL) reserveError(cause error) error {
	if err := w.Failed(); err != nil {
		return failedError(err)
	}
	need := w.opts.SegmentSize / 2
	return fmt.Errorf(MSG_NO_ROOM, ErrNoRoom, w.room(), need, cause)
}
