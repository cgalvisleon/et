//go:build unix

package store

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/cgalvisleon/et/msg"
)

/**
* syncDir: Hace fsync del directorio path para que sus entradas (archivos creados,
* renombrados o borrados) sobrevivan a un corte de luz.
* @param path string
* @return error
**/
func syncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}

	// En macOS File.Sync usa F_FULLFSYNC, que algunos sistemas de archivos no aceptan sobre
	// un directorio: entonces basta el fsync común.
	syncErr := d.Sync()
	if errors.Is(syncErr, syscall.ENOTSUP) || errors.Is(syncErr, syscall.EINVAL) {
		syncErr = syscall.Fsync(int(d.Fd()))
	}
	closeErr := d.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

/**
* lockDir: Toma el bloqueo exclusivo y no bloqueante (flock) del archivo path y deja en él
* el pid del proceso. Si otro lo tiene, retorna MSG_STORE_LOCKED con su pid.
* @param path string
* @return *os.File, error
**/
func lockDir(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		pid := lockOwner(f)
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf(msg.MSG_STORE_LOCKED, path, pid)
		}
		return nil, err
	}

	// El pid es informativo (para el mensaje de error de otro proceso); el lock es el flock.
	if err := f.Truncate(0); err == nil {
		f.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0)
	}

	return f, nil
}

/**
* unlockDir: Libera el bloqueo de lockDir y cierra el archivo; f nil no hace nada.
* El archivo LOCK se deja en disco: borrarlo abre una carrera con quien lo está tomando.
* @param f *os.File
* @return error
**/
func unlockDir(f *os.File) error {
	if f == nil {
		return nil
	}

	unlockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	closeErr := f.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

/**
* lockOwner: Retorna el pid escrito en el archivo de bloqueo, o «?» si no se puede leer.
* @param f *os.File
* @return string
**/
func lockOwner(f *os.File) string {
	buf := make([]byte, 32)
	n, _ := f.ReadAt(buf, 0)
	pid := strings.TrimSpace(string(buf[:n]))
	if pid == "" {
		return "?"
	}
	return pid
}
