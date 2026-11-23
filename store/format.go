package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Versión del formato en disco (tarea 54 de F-23): ver «Versiones del formato» en doc.go. Open revisa, antes de leer
// o cambiar nada, que cada archivo de la raíz sea de la versión que este binario lee.

// formatHeader es un archivo de la raíz que empieza con magic(4) version(4).
type formatHeader struct {
	name  string
	magic uint32
}

// formatHeaders son los archivos de la raíz con encabezado propio, además del control (que tiene dos copias con su
// crc y lo revisa su paquete) y del catálogo (que revisa su paquete).
var formatHeaders = []formatHeader{
	{name: backupLabelName, magic: backupLabelMagic},
	{name: archiveStatusName, magic: archiveStatusMagic},
	{name: slotsFileName, magic: slotsMagic},
}

/**
* checkFormat comprueba, sin escribir nada, la versión del formato de los archivos de la raíz: el control (cualquier
* copia con su crc válido), el catálogo, el encabezado de los segmentos del WAL que Open va a leer (desde el punto de
* rehacer del control, más los repuestos, y hacia atrás hasta el lsn más viejo que retiene una ranura: no los que
* retiene solo el archivo continuo atrasado), backup_label, archive_status y slots. Un archivo que no existe no cuenta; uno sin la magia de
* su formato tampoco (es daño, y lo informa quien lo lee después).
* @return error: ErrFormatVersion
**/
func (e *Engine) checkFormat() error {
	names, err := e.fs.ReadDir(e.dir)
	if err != nil {
		return err
	}
	redoSegment := uint64(0) // sin control legible se miran todos los segmentos
	if slices.Contains(names, control.FileName) {
		ctl, err := e.ctl.Read()
		if errors.Is(err, control.ErrFormatVersion) {
			return fmt.Errorf(MSG_FORMAT_CONTROL, ErrFormatVersion, err)
		}
		if err == nil {
			redoSegment = ctl.Redo.Segment
		}
	}
	if slices.Contains(names, control.CatalogFileName) {
		if _, err := e.catFile.Read(); errors.Is(err, control.ErrFormatVersion) {
			return fmt.Errorf(MSG_FORMAT_CONTROL, ErrFormatVersion, err)
		}
		// Un catálogo del formato 1 (Open lo reescribe en el 2): sus stores pueden tener hojas de la forma 1, así
		// que la limpieza de fondo corre hasta una pasada entera (migrate.go)
		catalog := formatHeader{name: control.CatalogFileName, magic: control.CatalogMagic}
		if version, ok, err := e.readFormatHeader(catalog); err == nil && ok && version < control.FormatVersion {
			e.leaves.fromV1.Store(true)
		}
	}
	if name, version, found, err := wal.OtherVersion(e.fs, e.dir, redoSegment, e.slotsKeep()); err != nil || found {
		if err != nil {
			return err
		}
		return formatError(name, version)
	}
	for _, h := range formatHeaders {
		if !slices.Contains(names, h.name) {
			continue
		}
		version, ok, err := e.readFormatHeader(h)
		if err != nil {
			return err
		}
		if ok && !wal.Readable(version) {
			return formatError(h.name, version)
		}
	}
	return nil
}

/**
* readFormatHeader lee los 8 primeros bytes de un archivo de la raíz.
* @param h formatHeader
* @return uint32: la versión, bool: si la magia es la del formato, error
**/
func (e *Engine) readFormatHeader(h formatHeader) (uint32, bool, error) {
	f, err := e.fs.OpenFile(filepath.Join(e.dir, h.name), os.O_RDONLY)
	if err != nil {
		return 0, false, err
	}
	defer f.Close()
	b := make([]byte, 8)
	if n, err := f.ReadAt(b, 0); n < len(b) {
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, false, err
		}
		return 0, false, nil // corto: no es de ningún formato
	}
	if le.Uint32(b[0:]) != h.magic {
		return 0, false, nil
	}
	return le.Uint32(b[4:]), true, nil
}

/**
* formatError es el error de un archivo de una versión que este binario no lee: más nueva (actualizar et) o anterior
* a wal.MinFormatVersion (sin migración).
* @param name string, version uint32
* @return error: ErrFormatVersion
**/
func formatError(name string, version uint32) error {
	if version > control.FormatVersion {
		return fmt.Errorf(MSG_FORMAT_NEWER, ErrFormatVersion, name, version, control.FormatVersion)
	}
	return fmt.Errorf(MSG_FORMAT_OLDER, ErrFormatVersion, name, version, control.FormatVersion)
}

/**
* slotsKeep es el menor lsn que retiene una ranura válida del archivo slots, sin escribir nada: hasta ahí lee Open los
* encabezados del WAL al poner las retenciones (wal.SegmentOf). Un archivo que no se lee da 0 (lo informa loadSlots).
* @return page.LSN: 0 si no hay ranuras
**/
func (e *Engine) slotsKeep() page.LSN {
	list, err := e.readSlots()
	if err != nil {
		return 0
	}
	keep := page.LSN(0)
	for _, s := range list {
		if !s.invalid && s.restart > 0 && (keep == 0 || s.restart < keep) {
			keep = s.restart
		}
	}
	return keep
}
