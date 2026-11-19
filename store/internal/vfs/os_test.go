package vfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestFileReadWrite escribe, lee, sincroniza, mide y trunca un archivo real.
func TestFileReadWrite(t *testing.T) {
	name := filepath.Join(t.TempDir(), "data")
	f, err := OS().OpenFile(name, os.O_RDWR|os.O_CREATE)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := f.WriteAt([]byte("hola"), 8192); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if size, err := f.Size(); err != nil || size != 8196 {
		t.Fatalf("Size = %d, %v; want 8196", size, err)
	}
	buf := make([]byte, 6)
	n, err := f.ReadAt(buf, 8190)
	if n != 6 || err != nil || !bytes.Equal(buf, []byte("\x00\x00hola")) {
		t.Fatalf("ReadAt = %d %q %v", n, buf, err)
	}
	if n, err := f.ReadAt(buf, 8194); n != 2 || err != io.EOF {
		t.Fatalf("ReadAt past the end = %d, %v; want 2, EOF", n, err)
	}
	if err := f.Truncate(10); err != nil {
		t.Fatal(err)
	}
	if size, _ := f.Size(); size != 10 {
		t.Fatalf("Size after Truncate = %d", size)
	}
}

// TestDirOps prueba MkdirAll, SyncDir, ReadDir, Rename y Remove sobre directorios reales.
func TestDirOps(t *testing.T) {
	fsys := OS()
	dir := filepath.Join(t.TempDir(), "a", "b", "c")
	if err := fsys.MkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := fsys.MkdirAll(dir); err != nil {
		t.Fatalf("MkdirAll again: %v", err)
	}
	for _, name := range []string{"z", "m", "a"} {
		f, err := fsys.OpenFile(filepath.Join(dir, name), os.O_RDWR|os.O_CREATE)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteAt([]byte(name), 0)
		f.Close()
	}
	if err := fsys.MkdirAll(filepath.Join(dir, "sub")); err != nil {
		t.Fatal(err)
	}
	if err := fsys.SyncDir(dir); err != nil {
		t.Fatalf("SyncDir: %v", err)
	}
	names, err := fsys.ReadDir(dir)
	if err != nil || !slices.Equal(names, []string{"a", "m", "sub", "z"}) {
		t.Fatalf("ReadDir = %v, %v", names, err)
	}

	// Rename reemplaza el destino.
	if err := fsys.Rename(filepath.Join(dir, "a"), filepath.Join(dir, "z")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "z")); string(got) != "a" {
		t.Fatalf("after Rename z = %q", got)
	}
	if err := fsys.Remove(filepath.Join(dir, "m")); err != nil {
		t.Fatal(err)
	}
	if err := fsys.Remove(filepath.Join(dir, "m")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Remove missing = %v", err)
	}
	names, _ = fsys.ReadDir(dir)
	if !slices.Equal(names, []string{"sub", "z"}) {
		t.Fatalf("ReadDir = %v", names)
	}
}

// TestMkdirAllOverFile falla si un tramo de la ruta es un archivo.
func TestMkdirAllOverFile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(name, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := OS().MkdirAll(filepath.Join(name, "x")); err == nil {
		t.Fatal("MkdirAll over a file: want error")
	}
}

// TestOpenMissing no crea sin O_CREATE.
func TestOpenMissing(t *testing.T) {
	_, err := OS().OpenFile(filepath.Join(t.TempDir(), "nope"), os.O_RDWR)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OpenFile missing = %v", err)
	}
}
