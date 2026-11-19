//go:build unix

package store

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

/**
* syncDir: Hace durable lo que cambió en los nombres de dir (un archivo creado, renombrado o borrado): el fsync de un
* archivo no cubre su entrada en el directorio.
* @param dir string
* @return error
**/
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

/**
* lockDir: Toma un bloqueo exclusivo sin esperar sobre path (flock); otro proceso, o el mismo, que lo intente falla.
* El bloqueo se suelta al cerrar el archivo o si el proceso muere.
* @param path string
* @return *os.File, error
**/
func lockDir(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf(MSG_LOCKED, path)
		}
		return nil, err
	}
	return f, nil
}
