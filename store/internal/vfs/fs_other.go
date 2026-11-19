//go:build !unix

package vfs

import (
	"io"
	"os"
)

// syncDir fuera de unix no hace nada: no hay fsync de directorios.
func syncDir(dir string) error {
	return nil
}

// lockFile fuera de unix solo crea el archivo, sin bloquear (no hay flock).
func lockFile(name string) (io.Closer, error) {
	return os.OpenFile(name, os.O_CREATE|os.O_RDWR, filePerm)
}
