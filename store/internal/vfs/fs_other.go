//go:build !unix

package vfs

import (
	"errors"
	"io"
	"os"
	"syscall"
)

// isNoSpaceErrno dice si err es ENOSPC del sistema operativo (fuera de unix, el que traduce Go).
func isNoSpaceErrno(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}

// syncDir fuera de unix no hace nada: no hay fsync de directorios.
func syncDir(dir string) error {
	return nil
}

// lockFile fuera de unix solo crea el archivo, sin bloquear (no hay flock).
func lockFile(name string) (io.Closer, error) {
	return os.OpenFile(name, os.O_CREATE|os.O_RDWR, filePerm)
}
