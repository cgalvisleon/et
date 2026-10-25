//go:build !unix

package store

import (
	"errors"
	"os"
)

/**
* mapFile: Sin mmap en esta plataforma: los segmentos se leen con ReadAt.
* @param file *os.File, size int64
* @return []byte, error
**/
func mapFile(file *os.File, size int64) ([]byte, error) {
	return nil, errors.ErrUnsupported
}

/**
* unmapFile: Sin mmap en esta plataforma.
* @param b []byte
* @return error
**/
func unmapFile(b []byte) error {
	return nil
}
