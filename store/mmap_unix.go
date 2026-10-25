//go:build unix

package store

import (
	"os"
	"syscall"
)

/**
* mapFile: Mapea en memoria, solo lectura, los primeros size bytes de file.
* @param file *os.File, size int64
* @return []byte, error
**/
func mapFile(file *os.File, size int64) ([]byte, error) {
	return syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
}

/**
* unmapFile: Libera un mapeo de mapFile.
* @param b []byte
* @return error
**/
func unmapFile(b []byte) error {
	return syscall.Munmap(b)
}
