package control

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

// controlFile implementa ControlFile sobre un vfs.FS. Abre el archivo en cada operación: se usa pocas veces (al abrir
// y en cada checkpoint) y así no retiene un descriptor.
type controlFile struct {
	fs   vfs.FS
	dir  string
	path string

	mu      sync.Mutex
	lastSeq uint64 // seq de la copia vigente; 0 = todavía no se leyó ni se creó
	failed  error  // primera escritura o fsync fallido: desde entonces Write y Create fallan
}

// NewControlFile devuelve el archivo de control del motor en dir.
// fs: el sistema de archivos; dir: el directorio del motor. Devuelve el ControlFile (no toca el disco).
func NewControlFile(fs vfs.FS, dir string) ControlFile {
	return &controlFile{fs: fs, dir: dir, path: filepath.Join(dir, FileName)}
}

// copyOffset es dónde va la copia con seq: las pares en la A (byte 0), las impares en la B (byte CopySize).
func copyOffset(seq uint64) int64 {
	return int64(seq%2) * CopySize
}

// Read devuelve la copia válida con mayor seq. Una copia es válida si decodeControl la acepta y está en el lugar
// que le toca por su seq. Si ninguna lo es, devuelve ErrNoValidCopy con el motivo de cada una.
func (f *controlFile) Read() (Control, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.read()
}

// read es Read con f.mu tomado.
func (f *controlFile) read() (Control, error) {
	b, err := f.readAll()
	if err != nil {
		return Control{}, err
	}
	a, errA := decodeCopy(b, 0)
	bb, errB := decodeCopy(b, 1)
	switch {
	case errA != nil && errB != nil:
		return Control{}, fmt.Errorf("%w: %s: copy A: %v; copy B: %v", ErrNoValidCopy, f.path, errA, errB)
	case errA != nil:
		f.lastSeq = bb.Seq
		return bb, nil
	case errB != nil || a.Seq > bb.Seq:
		f.lastSeq = a.Seq
		return a, nil
	default:
		f.lastSeq = bb.Seq
		return bb, nil
	}
}

// readAll lee los 2 × CopySize bytes del archivo; lo que falte (archivo corto) queda en cero y no valida.
func (f *controlFile) readAll() ([]byte, error) {
	file, err := f.fs.OpenFile(f.path, os.O_RDONLY)
	if err != nil {
		return nil, fmt.Errorf("control: open %s: %w", f.path, err)
	}
	defer file.Close()
	b := make([]byte, 2*CopySize)
	if _, err := file.ReadAt(b, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("control: read %s: %w", f.path, err)
	}
	return b, nil
}

// decodeCopy decodifica la copia i (0 = A, 1 = B) de b y comprueba que su seq le corresponda a ese lugar.
func decodeCopy(b []byte, i int) (Control, error) {
	off := i * CopySize
	c, err := decodeControl(b[off : off+CopySize])
	if err != nil {
		return Control{}, err
	}
	if copyOffset(c.Seq) != int64(off) {
		return Control{}, fmt.Errorf("%w: seq %d is in the wrong copy", ErrBadControl, c.Seq)
	}
	return c, nil
}

// Write escribe c con seq = la vigente + 1 en la otra copia y hace Sync. Si nunca se leyó, lee primero para saber
// la seq vigente. Tras un fallo de escritura o de Sync no reintenta: devuelve ErrFailed hasta reabrir.
func (f *controlFile) Write(c Control) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return fmt.Errorf("%w: %v", ErrFailed, f.failed)
	}
	if f.lastSeq == 0 {
		if _, err := f.read(); err != nil {
			return err
		}
	}
	c.Seq = f.lastSeq + 1
	b, err := encodeControl(c)
	if err != nil {
		return err
	}
	file, err := f.fs.OpenFile(f.path, os.O_RDWR)
	if err != nil {
		return fmt.Errorf("control: open %s: %w", f.path, err)
	}
	if err := f.writeSync(file, []copyAt{{b, copyOffset(c.Seq)}}); err != nil {
		return err
	}
	f.lastSeq = c.Seq
	return nil
}

// Create escribe las dos copias de un motor nuevo (seq 1 en la B y 2 en la A, mismo contenido), hace Sync y SyncDir.
// Si ya hay una copia válida devuelve ErrExists: nunca pisa un motor existente. Un control sin copias válidas (un
// Create anterior cortado) sí se reemplaza.
func (f *controlFile) Create(c Control) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed != nil {
		return fmt.Errorf("%w: %v", ErrFailed, f.failed)
	}
	if _, err := f.read(); err == nil {
		return fmt.Errorf("%w: %s", ErrExists, f.path)
	}
	c.Seq = 1
	one, err := encodeControl(c)
	if err != nil {
		return err
	}
	c.Seq = 2
	two, err := encodeControl(c)
	if err != nil {
		return err
	}
	file, err := f.fs.OpenFile(f.path, os.O_RDWR|os.O_CREATE)
	if err != nil {
		return fmt.Errorf("control: create %s: %w", f.path, err)
	}
	if err := file.Truncate(2 * CopySize); err != nil {
		file.Close()
		return f.fail(fmt.Errorf("control: truncate %s: %w", f.path, err))
	}
	if err := f.writeSync(file, []copyAt{{one, copyOffset(1)}, {two, copyOffset(2)}}); err != nil {
		return err
	}
	if err := f.fs.SyncDir(f.dir); err != nil {
		return f.fail(fmt.Errorf("control: sync dir %s: %w", f.dir, err))
	}
	f.lastSeq = 2
	return nil
}

// copyAt es una copia codificada y dónde va.
type copyAt struct {
	b   []byte
	off int64
}

// writeSync escribe las copias en file, hace Sync y lo cierra. Cualquier fallo deja el archivo marcado como fallido.
func (f *controlFile) writeSync(file vfs.File, copies []copyAt) error {
	for _, c := range copies {
		if _, err := file.WriteAt(c.b, c.off); err != nil {
			file.Close()
			return f.fail(fmt.Errorf("control: write %s: %w", f.path, err))
		}
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return f.fail(fmt.Errorf("control: sync %s: %w", f.path, err))
	}
	if err := file.Close(); err != nil {
		return f.fail(fmt.Errorf("control: close %s: %w", f.path, err))
	}
	return nil
}

// fail anota err como el primer fallo y lo devuelve.
func (f *controlFile) fail(err error) error {
	if f.failed == nil {
		f.failed = err
	}
	return err
}
