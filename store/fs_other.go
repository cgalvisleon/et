//go:build !unix

package store

import "os"

/**
* syncDir: Fuera de unix no hay fsync de directorios: no hace nada.
* @param dir string
* @return error
**/
func syncDir(dir string) error {
	return nil
}

/**
* lockDir: Fuera de unix no hay flock: solo crea el archivo, sin bloquear.
* @param path string
* @return *os.File, error
**/
func lockDir(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
}
