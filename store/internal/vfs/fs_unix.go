//go:build unix

package vfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

// syncDir hace fsync del directorio dir: el fsync de un archivo no cubre su entrada en el directorio.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	err = d.Sync()
	// Algunos sistemas de archivos no admiten fsync sobre un directorio: no hay nada más que hacer.
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) {
		return nil
	}
	return err
}

// lockFile toma un flock exclusivo y sin espera sobre name, creándolo si falta. Otro proceso, o el mismo con otra
// apertura, que lo intente recibe ErrLocked. Se suelta al cerrar lo devuelto o si el proceso muere.
func lockFile(name string) (io.Closer, error) {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, filePerm)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w: %s", ErrLocked, name)
		}
		return nil, err
	}
	return f, nil
}
