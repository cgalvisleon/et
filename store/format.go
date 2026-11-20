package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/cgalvisleon/et/store/internal/control"
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
* copia con su crc válido), el catálogo, backup_label, archive_status y slots. Un archivo que no existe no cuenta; uno
* sin la magia de su formato tampoco (es daño, y lo informa quien lo lee después).
* @return error: ErrFormatVersion
**/
func (e *Engine) checkFormat() error {
	names, err := e.fs.ReadDir(e.dir)
	if err != nil {
		return err
	}
	if slices.Contains(names, control.FileName) {
		if _, err := e.ctl.Read(); errors.Is(err, control.ErrFormatVersion) {
			return fmt.Errorf(MSG_FORMAT_CONTROL, ErrFormatVersion, err)
		}
	}
	if slices.Contains(names, control.CatalogFileName) {
		if _, err := e.catFile.Read(); errors.Is(err, control.ErrFormatVersion) {
			return fmt.Errorf(MSG_FORMAT_CONTROL, ErrFormatVersion, err)
		}
	}
	for _, h := range formatHeaders {
		if !slices.Contains(names, h.name) {
			continue
		}
		version, ok, err := e.readFormatHeader(h)
		if err != nil {
			return err
		}
		if ok && version != control.FormatVersion {
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
* formatError es el error de un archivo de otra versión: más nueva (actualizar et) o anterior (sin migración).
* @param name string, version uint32
* @return error: ErrFormatVersion
**/
func formatError(name string, version uint32) error {
	if version > control.FormatVersion {
		return fmt.Errorf(MSG_FORMAT_NEWER, ErrFormatVersion, name, version, control.FormatVersion)
	}
	return fmt.Errorf(MSG_FORMAT_OLDER, ErrFormatVersion, name, version, control.FormatVersion)
}
