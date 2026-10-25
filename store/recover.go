package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

/**
* recover: Repara en disco un store después de una falla (caída del proceso, corte
* de energía, compactación interrumpida) y retorna un reporte de lo que hizo.
* El store debe estar cerrado, también en otros procesos. Pasos:
*  1. Compactación interrumpida: restaura o limpia los directorios de segmentos y
*     borra el directorio temporal.
*  2. Borra el snapshot para que el próximo Open reconstruya el índice completo.
*  3. Recorre cada segmento: si tiene bytes que no forman registros válidos (escritura
*     cortada o corrupción), los copia a PathRecover y trunca el segmento en el último
*     registro válido. No se borra nada sin dejar copia.
* Después de Recover, abrir el store con Open.
* @param pathData, pathWald, name string
* @return et.Json, error
**/
func recover(pathData, pathWald, name string) (et.Json, error) {
	s := newFileStore(pathData, pathWald, name, ReadWrite)
	report := et.Json{
		"name":                 s.Name,
		"restored_segments":    false,
		"removed_old_segments": false,
		"removed_compact_tmp":  false,
		"removed_snapshot":     false,
	}

	// ---- 1. Compactación interrumpida ----
	restored, removedOld, removedTmp, err := s.repairCompaction()
	if err != nil {
		return nil, err
	}
	report["restored_segments"] = restored
	report["removed_old_segments"] = removedOld
	report["removed_compact_tmp"] = removedTmp
	if !exists(s.Path) {
		return nil, fmt.Errorf("%w: %s", errors.New(msg.MSG_STORE_NOT_FOUND), s.Path)
	}

	// ---- 2. Snapshot ----
	if exists(s.snapshotPath()) {
		if err := s.removeSnapshot(); err != nil {
			return nil, err
		}
		report["removed_snapshot"] = true
	}

	// ---- 3. Segmentos ----
	files, err := os.ReadDir(s.Path)
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	stamp := time.Now().Format("20060102150405")
	segments := []et.Json{}
	records, truncated := 0, int64(0)
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		result, err := s.recoverSegment(f.Name(), stamp)
		if err != nil {
			return nil, err
		}
		segments = append(segments, result)
		records += result.Int("records")
		truncated += result.Int64("truncated")
	}

	report["segments"] = segments
	report["records"] = records
	report["truncated_bytes"] = truncated

	return report, nil
}

/**
* recoverSegment: Recorre un segmento y lo trunca en el último registro válido,
* guardando antes en PathRecover los bytes descartados.
* @param name string, stamp string
* @return et.Json, error
**/
func (s *FileStore) recoverSegment(name, stamp string) (et.Json, error) {
	path := filepath.Join(s.Path, name)
	fd, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	st, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()

	// scan se detiene en el primer registro cortado o corrupto: lo anterior es válido.
	seg := newReadOnlySegment(fd, size, name)
	validEnd, records := int64(0), 0
	err = seg.scan(0, func(offset int64, h recordHeader, data []byte) error {
		validEnd = offset + h.recordSize()
		records++
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := et.Json{
		"name":      name,
		"size":      size,
		"records":   records,
		"valid":     validEnd,
		"truncated": size - validEnd,
	}
	if validEnd >= size {
		return result, nil
	}

	quarantine, err := s.quarantineTail(fd, name, validEnd, size, stamp)
	if err != nil {
		return nil, err
	}
	result["quarantine"] = quarantine

	return result, nil
}

/**
* repairCompaction: Deja los segmentos como antes o como después de una compactación
* interrumpida: si falta el directorio de segmentos y existe .segments.old, lo restaura
* (la caída fue entre los dos renombres); si existen los dos, borra .segments.old (el
* intercambio terminó). Borra además el directorio temporal de la compactación. Lo usan
* Recover y Open.
* @return bool (restauró), bool (borró .segments.old), bool (borró el temporal), error
**/
func (s *FileStore) repairCompaction() (bool, bool, bool, error) {
	restored, removedOld, removedTmp := false, false, false
	pathExists := exists(s.Path)
	oldDir := s.oldSegmentsPath()
	switch {
	case !pathExists && exists(oldDir):
		if err := os.Rename(oldDir, s.Path); err != nil {
			return false, false, false, err
		}
		restored = true
	case pathExists && exists(oldDir):
		if err := os.RemoveAll(oldDir); err != nil {
			return false, false, false, err
		}
		removedOld = true
	}

	if tmp := s.compactTmpPath(); exists(tmp) {
		if err := os.RemoveAll(tmp); err != nil {
			return restored, removedOld, false, err
		}
		removedTmp = true
	}

	return restored, removedOld, removedTmp, nil
}

/**
* quarantineTail: Copia a PathRecover los bytes del segmento desde validEnd hasta size y
* lo trunca en validEnd, con fsync. No se descarta nada sin dejar copia.
* @param fd *os.File, name string, validEnd, size int64, stamp string
* @return string (archivo de cuarentena), error
**/
func (s *FileStore) quarantineTail(fd *os.File, name string, validEnd, size int64, stamp string) (string, error) {
	tail := make([]byte, size-validEnd)
	if n, err := fd.ReadAt(tail, validEnd); err != nil && !(errors.Is(err, io.EOF) && n == len(tail)) {
		return "", err
	}
	if err := os.MkdirAll(s.PathRecover, 0755); err != nil {
		return "", err
	}
	quarantine := filepath.Join(s.PathRecover, fmt.Sprintf("%s.%s.corrupt", name, stamp))
	if err := os.WriteFile(quarantine, tail, 0644); err != nil {
		return "", err
	}

	if err := fd.Truncate(validEnd); err != nil {
		return "", err
	}
	if err := fd.Sync(); err != nil {
		return "", err
	}

	return quarantine, nil
}

/**
* exists: Indica si la ruta existe.
* @param path string
* @return bool
**/
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
