package vfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// filePerm y dirPerm son los permisos con que se crean archivos y directorios.
const (
	filePerm = 0o644
	dirPerm  = 0o755
)

// OS devuelve el sistema de archivos real, el del sistema operativo.
func OS() FS {
	return osFS{}
}

// osFS implementa FS sobre el paquete os.
type osFS struct{}

// osFile es un archivo real; Sync y Size son propios, lo demás es el de *os.File.
type osFile struct {
	*os.File
}

// Sync deja el archivo en el disco: F_FULLFSYNC en macOS, fdatasync en Linux (ver sync_*.go).
func (f osFile) Sync() error {
	return syncFile(f.File)
}

// Size devuelve el largo actual del archivo.
func (f osFile) Size() (int64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// OpenFile abre o crea name. Crear el archivo no hace durable su entrada: eso lo hace SyncDir.
func (osFS) OpenFile(name string, flag int) (File, error) {
	f, err := os.OpenFile(name, flag, filePerm)
	if err != nil {
		return nil, err
	}
	return osFile{f}, nil
}

// Remove borra el archivo name.
func (osFS) Remove(name string) error {
	return os.Remove(name)
}

// Rename cambia oldName por newName, reemplazando el destino si existe.
func (osFS) Rename(oldName, newName string) error {
	return os.Rename(oldName, newName)
}

// SyncDir hace durables las altas, bajas y renombres de las entradas de dir.
func (osFS) SyncDir(dir string) error {
	return syncDir(dir)
}

// ReadDir lista los nombres de las entradas de dir en orden alfabético (os.ReadDir ya los ordena).
func (osFS) ReadDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

// Lock toma el bloqueo exclusivo de name (ver lock_*.go).
func (osFS) Lock(name string) (io.Closer, error) {
	return lockFile(name)
}

// MkdirAll crea dir y los padres que falten; tras crear cada uno hace fsync de su padre, para que su nombre
// sobreviva a un corte.
func (osFS) MkdirAll(dir string) error {
	missing, err := missingDirs(dir)
	if err != nil {
		return err
	}
	for _, path := range missing {
		if err := os.Mkdir(path, dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		if err := syncDir(filepath.Dir(path)); err != nil {
			return err
		}
	}
	return nil
}

// missingDirs devuelve los directorios de la ruta dir que no existen, del más alto al más bajo (el último es dir).
// Falla si algún tramo de la ruta existe y no es un directorio.
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
