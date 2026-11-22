package wal

import (
	"fmt"
	"github.com/cgalvisleon/et/store/internal/page"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// segmentExt es la extensión de los archivos de segmento.
const segmentExt = ".seg"

// segment es el segmento activo (o el que se está sellando). size es dónde va la próxima reserva y allocated hasta
// dónde el archivo ya está escrito (con ceros o registros), así que escribir por debajo no pide lugar nuevo en el
// disco (los dos bajo writeMu).
type segment struct {
	num       uint64
	file      vfs.File
	size      int64
	allocated int64
}

// segmentName es el nombre del segmento num: diez dígitos, así el orden alfabético es el numérico.
func segmentName(num uint64) string {
	return fmt.Sprintf("%010d%s", num, segmentExt)
}

// segmentPath es la ruta del segmento num dentro de dir (el directorio del WAL).
func segmentPath(dir string, num uint64) string {
	return filepath.Join(dir, segmentName(num))
}

// listSegments devuelve los números de los segmentos de dir, en orden; falla si falta uno en medio.
func listSegments(fsys vfs.FS, dir string) ([]uint64, error) {
	names, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var nums []uint64
	for _, name := range names {
		if !strings.HasSuffix(name, segmentExt) {
			continue
		}
		num, err := strconv.ParseUint(strings.TrimSuffix(name, segmentExt), 10, 64)
		if err != nil || num == 0 || name != segmentName(num) {
			return nil, fmt.Errorf(MSG_BAD_SEGMENT, name)
		}
		nums = append(nums, num)
	}
	slices.Sort(nums)
	for i := 1; i < len(nums); i++ {
		if nums[i] != nums[i-1]+1 {
			return nil, fmt.Errorf(MSG_SEGMENT_MISSING, nums[i-1]+1, nums[i], nums[i-1])
		}
	}
	return nums, nil
}

// createSegment crea el archivo del segmento num (no debe existir) y hace durable su nombre con fsync del directorio.
func createSegment(fsys vfs.FS, dir string, num uint64) (vfs.File, error) {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDWR|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return nil, err
	}
	if err := fsys.SyncDir(dir); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

// seal espera a que todo lo reservado esté escrito en el archivo (las reservas en curso las escriben sus dueños, y lo
// pendiente se escribe aquí) y hace fsync de s: desde aquí el segmento no cambia. Con writeMu tomado, así nadie
// reserva mientras tanto.
func (w *WAL) seal(s *segment) error {
	if err := w.waitPublished(page.LSN(w.lastLSN.Load())); err != nil {
		return err
	}
	return s.file.Sync()
}

// segmentSize es el largo del archivo del segmento num.
func segmentSize(fsys vfs.FS, dir string, num uint64) (int64, error) {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDONLY)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.Size()
}

// truncateSegment corta el segmento num en size y hace fsync: lo que seguía nunca se confirmó.
func truncateSegment(fsys vfs.FS, dir string, num uint64, size int64) error {
	file, err := fsys.OpenFile(segmentPath(dir, num), os.O_RDWR)
	if err != nil {
		return err
	}
	err = file.Truncate(size)
	if err == nil {
		err = file.Sync()
	}
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	return err
}
