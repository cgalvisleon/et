package store

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// diskImage es el contenido de un disco simulado: los archivos por ruta y los directorios.
type diskImage struct {
	files map[string][]byte
	paths []string // las rutas de files, ordenadas (el fuzzer elige una por su índice)
	dirs  []string
}

var (
	damagedBaseOnce sync.Once
	damagedBase     diskImage
)

/**
* readImage copia a memoria todo lo que hay bajo root en fsys.
* @param t testing.TB, fsys vfs.FS, root string
* @return diskImage
**/
func readImage(t testing.TB, fsys vfs.FS, root string) diskImage {
	img := diskImage{files: map[string][]byte{}}
	var walk func(dir string)
	walk = func(dir string) {
		names, err := fsys.ReadDir(dir)
		mustOK(t, err)
		img.dirs = append(img.dirs, dir)
		for _, name := range names {
			p := filepath.Join(dir, name)
			if _, err := fsys.ReadDir(p); err == nil {
				walk(p)
				continue
			}
			f, err := fsys.OpenFile(p, os.O_RDONLY)
			mustOK(t, err)
			size, err := f.Size()
			mustOK(t, err)
			buf := make([]byte, size)
			if _, err := f.ReadAt(buf, 0); err != nil && err != io.EOF {
				t.Fatalf("read %s: %v", p, err)
			}
			mustOK(t, f.Close())
			img.files[p] = buf
			img.paths = append(img.paths, p)
		}
	}
	walk(root)
	sort.Strings(img.paths)
	return img
}

/**
* writeImage arma un disco simulado nuevo con img, todo durable.
* @param t testing.TB, img diskImage
* @return crashfs.FS
**/
func writeImage(t testing.TB, img diskImage) crashfs.FS {
	fsys := crashfs.New()
	for _, d := range img.dirs {
		mustOK(t, fsys.MkdirAll(d))
	}
	for _, p := range img.paths {
		f, err := fsys.OpenFile(p, os.O_CREATE|os.O_RDWR)
		mustOK(t, err)
		if data := img.files[p]; len(data) > 0 {
			_, err = f.WriteAt(data, 0)
			mustOK(t, err)
		}
		mustOK(t, f.Sync())
		mustOK(t, f.Close())
	}
	for _, d := range img.dirs {
		mustOK(t, fsys.SyncDir(d))
	}
	return fsys
}

/**
* buildDamagedBase arma una vez el disco de partida: dos stores con valores chicos y de desborde, un checkpoint,
* más cambios después (para que Open tenga WAL que rehacer) y un corte sin Close.
* @param t testing.TB
* @return diskImage
**/
func buildDamagedBase(t testing.TB) diskImage {
	damagedBaseOnce.Do(func() {
		base := crashfs.New()
		proc := newProc(base)
		e := openTest(t, proc, testConfig())
		users := mustStore(t, e, "users")
		docs := mustStore(t, e, "docs")
		for i := range 300 {
			mustOK(t, users.Put(fmt.Sprintf("u%04d", i), []byte(fmt.Sprintf("user %d", i))))
		}
		for i := range 4 {
			mustOK(t, docs.Put(fmt.Sprintf("d%d", i), bytes.Repeat([]byte{byte('a' + i)}, 12_000)))
		}
		mustOK(t, e.Checkpoint())
		for i := range 100 {
			mustOK(t, users.Put(fmt.Sprintf("u%04d", i), []byte(fmt.Sprintf("user %d v2", i))))
		}
		_, err := docs.Delete("d1")
		mustOK(t, err)
		proc.kill(crashfs.LoseUnsynced, 1)
		e.Close()
		damagedBase = readImage(t, base, testDir)
	})
	return damagedBase
}

// FuzzOpenDamaged daña bytes de un archivo del motor (datos, WAL, control o catálogo) y lo abre: Open puede fallar,
// y las lecturas, Count, Engine.Verify y Engine.Repair pueden devolver error o hallazgos, pero nada
// puede hacer caer el proceso ni colgarse.
func FuzzOpenDamaged(f *testing.F) {
	f.Add(uint8(0), uint32(0), []byte{0xff})
	f.Add(uint8(1), uint32(8192), []byte{0, 0, 0, 0})
	f.Add(uint8(2), uint32(24), []byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(uint8(3), uint32(100), []byte{0x80})
	f.Fuzz(func(t *testing.T, which uint8, off uint32, xor []byte) {
		img := buildDamagedBase(t)
		if len(xor) == 0 || len(xor) > 64 {
			return
		}
		damaged := diskImage{files: map[string][]byte{}, paths: img.paths, dirs: img.dirs}
		for p, data := range img.files {
			damaged.files[p] = data
		}
		p := img.paths[int(which)%len(img.paths)]
		data := append([]byte(nil), img.files[p]...)
		if len(data) == 0 {
			return
		}
		at := int(off) % len(data)
		for i, b := range xor {
			if at+i < len(data) {
				data[at+i] ^= b
			}
		}
		damaged.files[p] = data

		e, err := open(writeImage(t, damaged), testDir, testConfig())
		if err != nil {
			return // un error claro al abrir es aceptable
		}
		defer e.Close()
		for _, name := range e.Stores() {
			s, err := e.Store(name)
			if err != nil {
				continue
			}
			s.Count()
			for i := range 300 {
				s.Read(fmt.Sprintf("u%04d", i))
			}
			for i := range 4 {
				s.Read(fmt.Sprintf("d%d", i))
			}
		}
		if _, err := e.Verify(); err != nil {
			return
		}
		if _, err := e.Repair(); err != nil {
			return
		}
		e.Verify()
	})
}
