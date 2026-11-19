//go:build linux

package vfs

import (
	"errors"
	"os"
	"syscall"
)

// syncFile hace fdatasync de f: deja en el disco los datos y lo necesario para leerlos (el largo), sin el resto de
// los metadatos (fechas). Reintenta solo si la llamada fue interrumpida (EINTR), que no es una falla del disco.
func syncFile(f *os.File) error {
	conn, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var syncErr error
	err = conn.Control(func(fd uintptr) {
		for {
			syncErr = syscall.Fdatasync(int(fd))
			if !errors.Is(syncErr, syscall.EINTR) {
				return
			}
		}
	})
	if err != nil {
		return err
	}
	if syncErr != nil {
		return &os.PathError{Op: "fdatasync", Path: f.Name(), Err: syncErr}
	}
	return nil
}
