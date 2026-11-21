//go:build !linux && !windows

package vfs

import (
	"errors"
	"os"
	"syscall"
)

// syncFile hace fsync de f con (*os.File).Sync. En macOS Go usa fcntl(F_FULLFSYNC), que además vacía la caché del
// disco (el fsync de macOS no lo hace), y solo cae a fsync si el sistema de archivos no admite F_FULLFSYNC.
func syncFile(f *os.File) error {
	return f.Sync()
}

// flushFile hace el fsync simple de f (sin F_FULLFSYNC en macOS): los datos llegan al dispositivo y salen sus
// errores, pero pueden quedar en la caché del disco.
func flushFile(f *os.File) error {
	conn, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var syncErr error
	err = conn.Control(func(fd uintptr) {
		for {
			syncErr = syscall.Fsync(int(fd))
			if !errors.Is(syncErr, syscall.EINTR) {
				return
			}
		}
	})
	if err != nil {
		return err
	}
	if syncErr != nil {
		return &os.PathError{Op: "fsync", Path: f.Name(), Err: syncErr}
	}
	return nil
}
