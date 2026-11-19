package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

/**
* missingDirs: Los directorios de la ruta dir que no existen, del más alto al más bajo (el último es dir mismo).
* @param dir string
* @return []string, error
**/
func missingDirs(dir string) ([]string, error) {
	var missing []string
	for path := filepath.Clean(dir); ; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return nil, &fs.PathError{Op: "mkdir", Path: path, Err: fs.ErrExist}
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		missing = append([]string{path}, missing...)
		if filepath.Dir(path) == path {
			break
		}
	}
	return missing, nil
}

/**
* mkdirAll: Como os.MkdirAll, pero hace durable el nombre de cada directorio que crea con el fsync de su padre: si se
* va la luz justo después, el árbol creado sigue ahí.
* @param dir string
* @return error
**/
func mkdirAll(dir string) error {
	missing, err := missingDirs(dir)
	if err != nil {
		return err
	}
	for _, path := range missing {
		if err := os.Mkdir(path, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		if err := syncDir(filepath.Dir(path)); err != nil {
			return err
		}
	}
	return nil
}
