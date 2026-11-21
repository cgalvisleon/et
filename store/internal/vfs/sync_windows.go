//go:build windows

package vfs

import "os"

// syncFile hace fsync de f con (*os.File).Sync, que en Windows es FlushFileBuffers (vacía también la caché del disco).
func syncFile(f *os.File) error {
	return f.Sync()
}

// flushFile en Windows es lo mismo que syncFile: no hay un fsync más liviano que FlushFileBuffers.
func flushFile(f *os.File) error {
	return f.Sync()
}
