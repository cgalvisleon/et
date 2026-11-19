package crashfs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/vfs"
)

const dir = "/db"

var allModes = []Mode{LoseUnsynced, TornWrites, ReorderUnsynced}

// newDisk crea un disco con el directorio dir.
func newDisk(t *testing.T) FS {
	t.Helper()
	m := New()
	if err := m.MkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	return m
}

// open abre (o crea) un archivo de dir para leer y escribir.
func open(t *testing.T, m vfs.FS, name string) vfs.File {
	t.Helper()
	f, err := m.OpenFile(filepath.Join(dir, name), os.O_RDWR|os.O_CREATE)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// write escribe en f y falla la prueba si no puede.
func write(t *testing.T, f vfs.File, data []byte, off int64) {
	t.Helper()
	if _, err := f.WriteAt(data, off); err != nil {
		t.Fatal(err)
	}
}

// contents lee entero el archivo name de dir; ok es false si no existe.
func contents(t *testing.T, m vfs.FS, name string) ([]byte, bool) {
	t.Helper()
	f, err := m.OpenFile(filepath.Join(dir, name), os.O_RDONLY)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	size, _ := f.Size()
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, 0); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	return buf, true
}

// fill devuelve n bytes iguales a b.
func fill(b byte, n int) []byte {
	return bytes.Repeat([]byte{b}, n)
}

// TestConformance hace las mismas operaciones sobre el disco real y el simulado y compara lo que se ve.
func TestConformance(t *testing.T) {
	run := func(fsys vfs.FS, root string) string {
		var log bytes.Buffer
		note := func(format string, args ...any) { fmt.Fprintf(&log, format+"\n", args...) }
		d := filepath.Join(root, "x", "y")
		note("mkdir %v", fsys.MkdirAll(d) == nil)
		_, err := fsys.OpenFile(filepath.Join(d, "f"), os.O_RDWR)
		note("open missing notexist=%v", errors.Is(err, fs.ErrNotExist))
		f, _ := fsys.OpenFile(filepath.Join(d, "f"), os.O_RDWR|os.O_CREATE)
		_, err = fsys.OpenFile(filepath.Join(d, "f"), os.O_RDWR|os.O_CREATE|os.O_EXCL)
		note("excl exist=%v", errors.Is(err, fs.ErrExist))
		f.WriteAt([]byte("abc"), 4)
		size, _ := f.Size()
		buf := make([]byte, 10)
		n, err := f.ReadAt(buf, 0)
		note("size=%d read=%d %q eof=%v", size, n, buf[:n], err == io.EOF)
		n, err = f.ReadAt(buf, 100)
		note("read past end=%d eof=%v", n, err == io.EOF)
		note("sync=%v truncate=%v", f.Sync(), f.Truncate(2))
		size, _ = f.Size()
		note("size=%d close=%v", size, f.Close())
		g, _ := fsys.OpenFile(filepath.Join(d, "g"), os.O_RDWR|os.O_CREATE|os.O_TRUNC)
		g.WriteAt([]byte("g"), 0)
		g.Close()
		note("rename=%v", fsys.Rename(filepath.Join(d, "g"), filepath.Join(d, "f")))
		fsys.MkdirAll(filepath.Join(d, "sub"))
		names, err := fsys.ReadDir(d)
		note("readdir=%v %v syncdir=%v", names, err, fsys.SyncDir(d))
		note("remove=%v", fsys.Remove(filepath.Join(d, "f")))
		err = fsys.Remove(filepath.Join(d, "f"))
		note("remove again notexist=%v", errors.Is(err, fs.ErrNotExist))
		err = fsys.MkdirAll(filepath.Join(d, "sub", "x"))
		note("mkdir nested=%v", err)
		h, _ := fsys.OpenFile(filepath.Join(d, "h"), os.O_RDWR|os.O_CREATE)
		h.Close()
		note("mkdir over file fails=%v", fsys.MkdirAll(filepath.Join(d, "h", "z")) != nil)
		return log.String()
	}
	onDisk := run(vfs.OS(), t.TempDir())
	mem := run(New(), "/root")
	if onDisk != mem {
		t.Fatalf("real and simulated differ:\nreal:\n%s\nmem:\n%s", onDisk, mem)
	}
}

// TestSyncedSurvives: lo que tuvo Sync (y su nombre SyncDir) sobrevive a cualquier corte, con cualquier seed.
func TestSyncedSurvives(t *testing.T) {
	for _, mode := range allModes {
		for seed := int64(0); seed < 200; seed++ {
			m := newDisk(t)
			f := open(t, m, "data")
			write(t, f, fill('a', 8192), 0)
			f.Sync()
			m.SyncDir(dir)
			write(t, f, fill('b', 8192), 8192) // sin Sync
			m.Crash(mode, seed)

			got, ok := contents(t, m, "data")
			if !ok || len(got) < 8192 || !bytes.Equal(got[:8192], fill('a', 8192)) {
				t.Fatalf("mode %d seed %d: synced page lost (len %d)", mode, seed, len(got))
			}
			if mode == LoseUnsynced && len(got) != 8192 {
				t.Fatalf("seed %d: LoseUnsynced kept unsynced bytes (len %d)", seed, len(got))
			}
		}
	}
}

// TestUnsyncedLost: con LoseUnsynced se pierde lo escrito sin Sync y el archivo creado sin SyncDir.
func TestUnsyncedLost(t *testing.T) {
	m := newDisk(t)
	f := open(t, m, "data")
	write(t, f, []byte("durable"), 0)
	f.Sync()
	m.SyncDir(dir)
	write(t, f, []byte("PERDIDO"), 0)
	g := open(t, m, "nuevo")
	write(t, g, []byte("x"), 0)
	g.Sync() // el fsync del archivo no hace durable su nombre
	m.Crash(LoseUnsynced, 1)

	if got, _ := contents(t, m, "data"); string(got) != "durable" {
		t.Fatalf("data = %q", got)
	}
	if _, ok := contents(t, m, "nuevo"); ok {
		t.Fatal("file created without SyncDir survived LoseUnsynced")
	}
}

// TestUnsyncedMaySurvive: con los otros modos lo no sincronizado a veces queda y a veces no.
func TestUnsyncedMaySurvive(t *testing.T) {
	for _, mode := range []Mode{TornWrites, ReorderUnsynced} {
		kept, lost := 0, 0
		for seed := int64(0); seed < 100; seed++ {
			m := newDisk(t)
			f := open(t, m, "data")
			m.SyncDir(dir)
			write(t, f, []byte("x"), 0)
			m.Crash(mode, seed)
			if got, _ := contents(t, m, "data"); string(got) == "x" {
				kept++
			} else {
				lost++
			}
		}
		if kept == 0 || lost == 0 {
			t.Fatalf("mode %d: kept %d lost %d; want both", mode, kept, lost)
		}
	}
}

// TestRenameWithoutSyncDir: un renombre sin SyncDir puede revertirse; con SyncDir no.
func TestRenameWithoutSyncDir(t *testing.T) {
	prepare := func() FS {
		m := newDisk(t)
		f := open(t, m, "catalog.tmp")
		write(t, f, []byte("nuevo"), 0)
		f.Sync()
		m.SyncDir(dir)
		if err := m.Rename(filepath.Join(dir, "catalog.tmp"), filepath.Join(dir, "catalog")); err != nil {
			t.Fatal(err)
		}
		return m
	}

	m := prepare()
	m.Crash(LoseUnsynced, 1)
	if _, ok := contents(t, m, "catalog"); ok {
		t.Fatal("rename without SyncDir survived LoseUnsynced")
	}
	if got, ok := contents(t, m, "catalog.tmp"); !ok || string(got) != "nuevo" {
		t.Fatalf("catalog.tmp = %q %v; want reverted", got, ok)
	}

	// En cualquier modo el renombre es atómico: queda un nombre o el otro, nunca los dos ni ninguno.
	for _, mode := range allModes {
		for seed := int64(0); seed < 50; seed++ {
			m := prepare()
			m.Crash(mode, seed)
			_, tmp := contents(t, m, "catalog.tmp")
			_, final := contents(t, m, "catalog")
			if tmp == final {
				t.Fatalf("mode %d seed %d: tmp %v final %v", mode, seed, tmp, final)
			}
		}
	}

	for _, mode := range allModes {
		m := prepare()
		m.SyncDir(dir)
		m.Crash(mode, 7)
		if got, ok := contents(t, m, "catalog"); !ok || string(got) != "nuevo" {
			t.Fatalf("mode %d: synced rename lost", mode)
		}
	}
}

// TestDirPrefix: de los cambios sin SyncDir de un directorio sobrevive un prefijo, en orden.
func TestDirPrefix(t *testing.T) {
	seen := map[string]bool{}
	for seed := int64(0); seed < 200; seed++ {
		m := newDisk(t)
		for _, name := range []string{"a", "b", "c"} {
			open(t, m, name).Close()
		}
		m.Crash(ReorderUnsynced, seed)
		names, _ := m.ReadDir(dir)
		got := fmt.Sprint(names)
		if !slices.Contains([]string{"[]", "[a]", "[a b]", "[a b c]"}, got) {
			t.Fatalf("seed %d: %s is not a prefix", seed, got)
		}
		seen[got] = true
	}
	if len(seen) != 4 {
		t.Fatalf("prefixes seen %v; want all four", seen)
	}
}

// TestReorder: una escritura posterior puede quedar sin una anterior, y cada una queda entera.
func TestReorder(t *testing.T) {
	reordered := false
	for seed := int64(0); seed < 200; seed++ {
		m := newDisk(t)
		f := open(t, m, "wal")
		write(t, f, fill(0, 2048), 0)
		f.Sync()
		m.SyncDir(dir)
		write(t, f, fill('1', 1024), 0)
		write(t, f, fill('2', 1024), 1024)
		m.Crash(ReorderUnsynced, seed)

		got, _ := contents(t, m, "wal")
		first, second := got[:1024], got[1024:]
		if !bytes.Equal(first, fill(0, 1024)) && !bytes.Equal(first, fill('1', 1024)) {
			t.Fatalf("seed %d: first write torn", seed)
		}
		if !bytes.Equal(second, fill(0, 1024)) && !bytes.Equal(second, fill('2', 1024)) {
			t.Fatalf("seed %d: second write torn", seed)
		}
		if bytes.Equal(first, fill(0, 1024)) && bytes.Equal(second, fill('2', 1024)) {
			reordered = true
		}
	}
	if !reordered {
		t.Fatal("never saw the second write without the first")
	}
}

// TestTornWrites: una página sin Sync queda como mezcla de sectores enteros viejos y nuevos.
func TestTornWrites(t *testing.T) {
	const pageSize = 8192
	torn := false
	for seed := int64(0); seed < 200; seed++ {
		m := newDisk(t)
		f := open(t, m, "page")
		write(t, f, fill('o', pageSize), 0)
		f.Sync()
		m.SyncDir(dir)
		write(t, f, fill('n', pageSize), 0)
		m.Crash(TornWrites, seed)

		got, _ := contents(t, m, "page")
		olds, news := 0, 0
		for off := 0; off < pageSize; off += SectorSize {
			switch sector := got[off : off+SectorSize]; {
			case bytes.Equal(sector, fill('o', SectorSize)):
				olds++
			case bytes.Equal(sector, fill('n', SectorSize)):
				news++
			default:
				t.Fatalf("seed %d: sector at %d is neither old nor new", seed, off)
			}
		}
		if olds > 0 && news > 0 {
			torn = true
		}
	}
	if !torn {
		t.Fatal("never saw a torn page")
	}
}

// TestSectorsAligned: los trozos de una escritura no cruzan límites de sector del archivo.
func TestSectorsAligned(t *testing.T) {
	op := fileOp{off: 500, data: fill('x', 1100)}
	var got []string
	for _, piece := range op.sectors(SectorSize) {
		got = append(got, fmt.Sprintf("%d+%d", piece.off, len(piece.data)))
	}
	if want := "[500+12 512+512 1024+512 1536+64]"; fmt.Sprint(got) != want {
		t.Fatalf("sectors = %v; want %s", got, want)
	}
}

// TestReproducible: la misma historia con la misma seed da el mismo disco.
func TestReproducible(t *testing.T) {
	history := func(seed int64) string {
		m := newDisk(t)
		for i, name := range []string{"a", "b", "c"} {
			f := open(t, m, name)
			write(t, f, fill(byte('a'+i), 3000), int64(i*700))
			if i == 1 {
				f.Sync()
			}
		}
		m.Crash(TornWrites, seed)
		names, _ := m.ReadDir(dir)
		var out bytes.Buffer
		for _, name := range names {
			got, _ := contents(t, m, name)
			fmt.Fprintf(&out, "%s=%x;", name, got)
		}
		return out.String()
	}
	for seed := int64(0); seed < 20; seed++ {
		if history(seed) != history(seed) {
			t.Fatalf("seed %d: different results", seed)
		}
	}
}

// TestCrashInvalidatesFilesAndLocks: tras un corte los archivos abiertos fallan y los bloqueos se sueltan.
func TestCrashInvalidatesFilesAndLocks(t *testing.T) {
	m := newDisk(t)
	f := open(t, m, "data")
	lock, err := m.Lock(filepath.Join(dir, "LOCK"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Lock(filepath.Join(dir, "LOCK")); !errors.Is(err, vfs.ErrLocked) {
		t.Fatalf("second Lock = %v; want ErrLocked", err)
	}
	m.Crash(LoseUnsynced, 1)

	if _, err := f.WriteAt([]byte("x"), 0); !errors.Is(err, ErrCrashed) {
		t.Fatalf("WriteAt after crash = %v", err)
	}
	if err := f.Sync(); !errors.Is(err, ErrCrashed) {
		t.Fatalf("Sync after crash = %v", err)
	}
	again, err := m.Lock(filepath.Join(dir, "LOCK"))
	if err != nil {
		t.Fatalf("Lock after crash: %v", err)
	}
	lock.Close() // el bloqueo viejo no suelta el nuevo
	if _, err := m.Lock(filepath.Join(dir, "LOCK")); !errors.Is(err, vfs.ErrLocked) {
		t.Fatalf("old Close released the new lock: %v", err)
	}
	again.Close()
	if l, err := m.Lock(filepath.Join(dir, "LOCK")); err != nil {
		t.Fatalf("Lock after Close: %v", err)
	} else {
		l.Close()
	}
}

// TestFailAfter: tras n escrituras o Sync, todo falla con ErrIO hasta el corte.
func TestFailAfter(t *testing.T) {
	m := newDisk(t)
	f := open(t, m, "data")
	m.FailAfter(2)
	write(t, f, []byte("a"), 0)
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("b"), 1); !errors.Is(err, ErrIO) {
		t.Fatalf("third op = %v; want ErrIO", err)
	}
	if err := f.Sync(); !errors.Is(err, ErrIO) {
		t.Fatalf("Sync = %v; want ErrIO", err)
	}
	if err := m.SyncDir(dir); !errors.Is(err, ErrIO) {
		t.Fatalf("SyncDir = %v; want ErrIO", err)
	}
	if got, _ := f.Size(); got != 1 {
		t.Fatalf("failed write wrote: size %d", got)
	}
	m.Crash(LoseUnsynced, 1)
	g := open(t, m, "otro")
	if _, err := g.WriteAt([]byte("x"), 0); err != nil {
		t.Fatalf("after crash the failure must be gone: %v", err)
	}
}

// TestFailedSyncForgets: un Sync fallido olvida lo pendiente; el siguiente Sync «funciona» pero no lo guarda.
func TestFailedSyncForgets(t *testing.T) {
	m := newDisk(t)
	f := open(t, m, "data")
	m.SyncDir(dir)
	write(t, f, []byte("perdido"), 0)
	m.FailSyncAfter(0)
	if err := f.Sync(); !errors.Is(err, ErrIO) {
		t.Fatalf("Sync = %v; want ErrIO", err)
	}
	write(t, f, []byte("ok"), 100) // FailSyncAfter no toca las escrituras
	m.FailSyncAfter(-1)
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if got, _ := contents(t, m, "data"); !bytes.HasPrefix(got, []byte("perdido")) {
		t.Fatalf("reads must still see the write: %q", got)
	}
	m.Crash(LoseUnsynced, 1)
	got, _ := contents(t, m, "data")
	if bytes.Contains(got, []byte("perdido")) || !bytes.HasSuffix(got, []byte("ok")) {
		t.Fatalf("after crash = %q; want the failed write gone and the later one kept", got)
	}
}

// TestCapacity: lo que no cabe falla con ErrNoSpace sin escribir.
func TestCapacity(t *testing.T) {
	m := newDisk(t)
	m.SetCapacity(100)
	f := open(t, m, "data")
	write(t, f, fill('x', 60), 0)
	g := open(t, m, "other")
	if _, err := g.WriteAt(fill('y', 50), 0); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("WriteAt = %v; want ErrNoSpace", err)
	}
	if err := g.Truncate(41); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("Truncate = %v; want ErrNoSpace", err)
	}
	write(t, g, fill('y', 40), 0)
	write(t, f, fill('z', 60), 0) // sobrescribir no crece
	if size, _ := g.Size(); size != 40 {
		t.Fatalf("size = %d", size)
	}
}

// TestMkdirAllSyncsParent: crear un subdirectorio hace durables los cambios pendientes del padre.
func TestMkdirAllSyncsParent(t *testing.T) {
	m := newDisk(t)
	open(t, m, "pending").Close()
	if err := m.MkdirAll(filepath.Join(dir, "wal")); err != nil {
		t.Fatal(err)
	}
	m.Crash(LoseUnsynced, 1)
	names, _ := m.ReadDir(dir)
	if !slices.Equal(names, []string{"pending", "wal"}) {
		t.Fatalf("ReadDir = %v", names)
	}
}

// TestCrossDirRename: un renombre entre directorios es durable por partes.
func TestCrossDirRename(t *testing.T) {
	m := newDisk(t)
	m.MkdirAll(filepath.Join(dir, "other"))
	f := open(t, m, "a")
	write(t, f, []byte("v"), 0)
	f.Sync()
	m.SyncDir(dir)
	if err := m.Rename(filepath.Join(dir, "a"), filepath.Join(dir, "other", "b")); err != nil {
		t.Fatal(err)
	}
	m.SyncDir(filepath.Join(dir, "other")) // solo el destino
	m.Crash(LoseUnsynced, 1)
	if _, ok := contents(t, m, "a"); !ok {
		t.Fatal("the old name must survive: its directory had no SyncDir")
	}
	if _, ok := contents(t, m, filepath.Join("other", "b")); !ok {
		t.Fatal("the new name must survive: its directory had SyncDir")
	}
}

// TestConcurrent escribe y sincroniza en paralelo (para -race).
func TestConcurrent(t *testing.T) {
	m := newDisk(t)
	f := open(t, m, "data")
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 50 {
				f.WriteAt(fill(byte(i), 64), int64((i*50+j)*64))
				if j%10 == 0 {
					f.Sync()
				}
				m.ReadDir(dir)
			}
		}()
	}
	wg.Wait()
	f.Sync()
	if size, _ := f.Size(); size != 8*50*64 {
		t.Fatalf("size = %d", size)
	}
}
