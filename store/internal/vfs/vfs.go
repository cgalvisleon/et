package vfs

import (
	"errors"
	"io"
)

// ErrLocked es el error de Lock cuando otro ya tiene el bloqueo (se compara con errors.Is).
var ErrLocked = errors.New("vfs: locked by another holder")

// File es un archivo abierto. Las escrituras no son durables hasta que Sync vuelve sin error.
type File interface {
	io.ReaderAt
	io.WriterAt
	io.Closer

	// Sync deja en el disco todo lo escrito en el archivo (en macOS con F_FULLFSYNC). Si falla, el llamador no
	// reintenta: el motor queda inutilizable (ver store/doc.go, «Fallos de escritura»).
	Sync() error

	// Truncate fija el largo del archivo (corta la cola o la extiende con ceros).
	Truncate(size int64) error

	// Size es el largo actual del archivo en bytes.
	Size() (int64, error)
}

// FS es un sistema de archivos. Los nombres son rutas completas.
type FS interface {
	// OpenFile abre o crea un archivo con las marcas de os.OpenFile (os.O_RDWR, os.O_CREATE…).
	OpenFile(name string, flag int) (File, error)

	// Remove borra un archivo. El borrado es durable tras SyncDir del directorio que lo contenía.
	Remove(name string) error

	// Rename cambia el nombre de un archivo, reemplazando el destino si existe. Durable tras SyncDir.
	Rename(oldName, newName string) error

	// MkdirAll crea el directorio y los padres que falten, con fsync del padre de cada uno que crea.
	MkdirAll(dir string) error

	// SyncDir hace fsync de un directorio: vuelve durables las altas, bajas y renombres de sus entradas.
	SyncDir(dir string) error

	// ReadDir lista los nombres (sin ruta) de las entradas de un directorio, en orden alfabético.
	ReadDir(dir string) ([]string, error)

	// Lock toma un bloqueo exclusivo y sin espera del archivo name, creándolo si falta (flock en unix); si otro lo
	// tiene, también en el mismo proceso, falla con ErrLocked. Cerrar lo devuelto lo suelta.
	Lock(name string) (io.Closer, error)
}
