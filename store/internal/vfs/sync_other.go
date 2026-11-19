//go:build !linux

package vfs

import "os"

// syncFile hace fsync de f con (*os.File).Sync. En macOS Go usa fcntl(F_FULLFSYNC), que además vacía la caché del
// disco (el fsync de macOS no lo hace), y solo cae a fsync si el sistema de archivos no admite F_FULLFSYNC.
func syncFile(f *os.File) error {
	return f.Sync()
}
