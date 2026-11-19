//go:build !unix

package store

import (
	"os"
	"runtime"
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
)

// noLockOnce: el aviso de que no hay bloqueo del directorio sale una sola vez por proceso.
var noLockOnce sync.Once

/**
* syncDir: Sin fsync de directorios en esta plataforma (Windows no lo permite): no hace nada.
* @param path string
* @return error
**/
func syncDir(path string) error {
	return nil
}

/**
* lockDir: Sin bloqueo del directorio en esta plataforma: el store abre igual y se avisa una vez en el log.
* @param path string
* @return *os.File, error
**/
func lockDir(path string) (*os.File, error) {
	noLockOnce.Do(func() {
		logs.Log(packageName, msg.MSG_STORE_NO_LOCK, ":", runtime.GOOS)
	})
	return nil, nil
}

/**
* unlockDir: Sin bloqueo del directorio en esta plataforma.
* @param f *os.File
* @return error
**/
func unlockDir(f *os.File) error {
	return nil
}
