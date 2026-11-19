package control

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Las pruebas usan vfs.OS() como disco real y crashfs para fallas y cortes. Quedan dos envolturas mínimas para lo
// que crashfs no hace: cortar una escritura en un byte exacto (tornFS) y cortar la luz en un Sync dado (cutFS).

// errTorn es el error de la escritura que tornFS corta.
var errTorn = errors.New("escritura cortada (prueba)")

// tornFS envuelve el disco real y, armado, corta la próxima escritura: alcanza a escribir sus primeros torn bytes y
// falla. crashfs no lo simula (una escritura que falla no escribe nada, y TornWrites corta por sectores enteros).
type tornFS struct {
	vfs.FS
	mu    sync.Mutex
	armed bool
	torn  int
}

// newTornFS envuelve vfs.OS().
func newTornFS() *tornFS { return &tornFS{FS: vfs.OS()} }

// arm corta la próxima escritura tras torn bytes.
func (f *tornFS) arm(torn int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed, f.torn = true, torn
}

func (f *tornFS) OpenFile(name string, flag int) (vfs.File, error) {
	file, err := f.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return tornFile{file, f}, nil
}

type tornFile struct {
	vfs.File
	fs *tornFS
}

func (f tornFile) WriteAt(b []byte, off int64) (int, error) {
	f.fs.mu.Lock()
	armed, torn := f.fs.armed, f.fs.torn
	f.fs.armed = false
	f.fs.mu.Unlock()
	if !armed {
		return f.File.WriteAt(b, off)
	}
	n, _ := f.File.WriteAt(b[:min(torn, len(b))], off)
	return n, errTorn
}

// errCut es el error del Sync en el que cutFS cortó la luz.
var errCut = errors.New("corte de luz (prueba)")

// cutFS envuelve crashfs y corta la luz justo antes de un Sync o SyncDir: el n-ésimo desde arm (0 el próximo) llama
// a Crash(mode, seed) en vez de hacerse y falla con errCut. Así el corte cae en medio de Write, Create o Save, con lo
// escrito y no sincronizado pendiente. crashfs solo corta cuando la prueba llama a Crash, entre operaciones.
type cutFS struct {
	crashfs.FS
	mu    sync.Mutex
	left  int // Sync y SyncDir que faltan para cortar; < 0 sin corte
	mode  crashfs.Mode
	seed  int64
	fired bool
}

// newCutFS crea un disco simulado con el directorio dir.
func newCutFS(t *testing.T, dir string) *cutFS {
	t.Helper()
	f := &cutFS{FS: crashfs.New(), left: -1}
	if err := f.MkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	return f
}

// arm programa el corte en el n-ésimo Sync o SyncDir.
func (f *cutFS) arm(n int, mode crashfs.Mode, seed int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.left, f.mode, f.seed, f.fired = n, mode, seed, false
}

// cut descuenta un Sync y, si toca, corta la luz.
func (f *cutFS) cut() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.left < 0 {
		return false
	}
	if f.left > 0 {
		f.left--
		return false
	}
	f.left, f.fired = -1, true
	f.FS.Crash(f.mode, f.seed)
	return true
}

func (f *cutFS) OpenFile(name string, flag int) (vfs.File, error) {
	file, err := f.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return cutFile{file, f}, nil
}

func (f *cutFS) SyncDir(dir string) error {
	if f.cut() {
		return errCut
	}
	return f.FS.SyncDir(dir)
}

type cutFile struct {
	vfs.File
	fs *cutFS
}

func (f cutFile) Sync() error {
	if f.fs.cut() {
		return errCut
	}
	return f.File.Sync()
}

// crashModes son los tres modos de corte de crashfs, con su nombre para los mensajes.
var crashModes = map[string]crashfs.Mode{
	"LoseUnsynced":    crashfs.LoseUnsynced,
	"ReorderUnsynced": crashfs.ReorderUnsynced,
	"TornWrites":      crashfs.TornWrites,
}

// writeFile y readFile son atajos para manipular los archivos de prueba en el disco real.
func writeFile(t testingT, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t testingT, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// testingT es lo que usan los atajos de *testing.T y *testing.F.
type testingT interface {
	Helper()
	Fatal(args ...any)
}

func controlPath(dir string) string { return filepath.Join(dir, FileName) }
