package store

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// tarSink es un BackupSink que arma un tar en memoria; con max > 0 deja de aceptar archivos después de max (un
// respaldo cortado).
type tarSink struct {
	buf   bytes.Buffer
	tw    *tar.Writer
	files int
	max   int
}

/**
* newTarSink arma un tarSink vacío.
* @param max int: 0 = sin tope
* @return *tarSink
**/
func newTarSink(max int) *tarSink {
	s := &tarSink{max: max}
	s.tw = tar.NewWriter(&s.buf)
	return s
}

// tarFile es un archivo en curso del tar: cuenta lo escrito para comprobar el largo anunciado.
type tarFile struct {
	tw   *tar.Writer
	left int64
}

func (f *tarFile) Write(p []byte) (int, error) {
	f.left -= int64(len(p))
	return f.tw.Write(p)
}

func (f *tarFile) Close() error {
	if f.left != 0 {
		return fmt.Errorf("tar: %d bytes left", f.left)
	}
	return nil
}

/**
* File empieza un archivo del tar con su largo.
* @param name string, size int64
* @return io.WriteCloser, error
**/
func (s *tarSink) File(name string, size int64) (io.WriteCloser, error) {
	if s.max > 0 && s.files >= s.max {
		return nil, errors.New("tar: cut")
	}
	s.files++
	if err := s.tw.WriteHeader(&tar.Header{Name: name, Size: size, Mode: 0o644}); err != nil {
		return nil, err
	}
	return &tarFile{tw: s.tw, left: size}, nil
}

/**
* extract saca el tar en el directorio dir de fsys (los archivos repetidos: el último).
* @param t testing.TB, data []byte, fsys vfs.FS, dir string
**/
func extract(t testing.TB, data []byte, fsys vfs.FS, dir string) {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return
		}
		mustOK(t, err)
		body, err := io.ReadAll(tr)
		mustOK(t, err)
		path := filepath.Join(dir, filepath.FromSlash(h.Name))
		mustOK(t, fsys.MkdirAll(filepath.Dir(path)))
		putFile(t, fsys, path, body)
	}
}

// TestBackupTo (tarea 77): un respaldo en caliente hacia un tar, con escritores mientras copia, se restaura
// sacando el tar y abriendo el directorio: el contenido es el del fin del respaldo y Verify da limpio. Un respaldo
// cortado deja el marcador «en curso»: la copia no se abre (ErrBackupIncomplete).
func TestBackupTo(t *testing.T) {
	e := openTest(t, crashfs.New(), smallWAL())
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b"}, 200)
	mustOK(t, stores[0].Put("big", bytes.Repeat([]byte{7}, 40_000)))
	want := dumpEngine(t, e)
	sink := newTarSink(0)
	info, err := e.BackupTo(sink)
	mustOK(t, err)
	mustOK(t, sink.tw.Close())
	mustOK(t, stores[1].Put("after", []byte("not in the copy")))
	dst := crashfs.New()
	extract(t, sink.buf.Bytes(), dst, testDir)
	r := restore(t, dst, testConfig())
	checkContents(t, r, want)
	mustOK(t, r.Close())
	t.Logf("backup to a tar: %d files, %d bytes (%+v)", sink.files, sink.buf.Len(), info)

	cut := newTarSink(4)
	if _, err := e.BackupTo(cut); err == nil {
		t.Fatal("a backup to a sink that fails gave no error")
	}
	mustOK(t, cut.tw.Close())
	part := crashfs.New()
	extract(t, cut.buf.Bytes(), part, testDir)
	if _, err := open(part, testDir, testConfig()); !errors.Is(err, ErrBackupIncomplete) {
		t.Fatalf("open of a cut backup: %v", err)
	}
	mustVerifyClean(t, e)
}

// TestOpenAtTime (tarea 77): un respaldo hacia un tar, dos tandas de lotes con una hora en medio y los segmentos
// archivados después; la copia abierta con un objetivo por hora llega hasta la primera tanda y no más; una hora antes
// del fin del respaldo da ErrRecoveryTarget sin cambiar nada.
func TestOpenAtTime(t *testing.T) {
	base := crashfs.New()
	cfg := archiveConfig()
	e := openTest(t, base, cfg)
	defer e.Close()
	a := mustStore(t, e, "a")
	arch := walArchive{}
	archiveAll(t, e, base, arch)
	before := time.Now()
	sink := newTarSink(0)
	_, err := e.BackupTo(sink)
	mustOK(t, err)
	mustOK(t, sink.tw.Close())
	batch := func(prefix string, n int) {
		for i := range n {
			b := e.NewBatch()
			mustOK(t, a.PutIn(b, fmt.Sprintf("%s-%03d", prefix, i), bytes.Repeat([]byte{1}, 500)))
			mustOK(t, b.Commit())
		}
	}
	batch("one", 200)
	want1 := dumpEngine(t, e)
	time.Sleep(5 * time.Millisecond)
	target := time.Now()
	time.Sleep(5 * time.Millisecond)
	batch("two", 200)
	_, err = e.SwitchWAL()
	mustOK(t, err)
	archiveAll(t, e, base, arch)

	dst := crashfs.New()
	extract(t, sink.buf.Bytes(), dst, testDir)
	addArchived(t, dst, arch)
	early := cfg
	early.archive, early.stopTime = false, before.Add(-time.Hour)
	if _, err := open(dst, testDir, early); !errors.Is(err, ErrRecoveryTarget) {
		t.Fatalf("a time before the backup end: %v", err)
	}
	at := cfg
	at.archive, at.stopTime = false, target
	r := restore(t, dst, at)
	defer r.Close()
	checkContents(t, r, want1)
}
