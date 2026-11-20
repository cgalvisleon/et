package smgr

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
)

// testBlocksPerFile achica los archivos a 4 bloques (32 KB) para probar el paso de un archivo al siguiente sin
// escribir 1 GB.
const testBlocksPerFile = 4

// newTestManager abre un gestor sobre un directorio temporal.
func newTestManager(t *testing.T, maxFiles int) (*manager, *countFS, string) {
	t.Helper()
	dir := t.TempDir()
	fs := newCountFS()
	m, err := open(fs, dir, maxFiles, testBlocksPerFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m, fs, dir
}

// reopen cierra m y abre otro gestor sobre el mismo directorio.
func reopen(t *testing.T, m *manager, fs *countFS, dir string) *manager {
	t.Helper()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	m2, err := open(fs, dir, 8, testBlocksPerFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m2.Close() })
	return m2
}

// pageOf devuelve una página llena con b.
func pageOf(b byte) page.Page {
	return page.Page(bytes.Repeat([]byte{b}, page.Size))
}

// mustRead lee un bloque y comprueba que esté lleno con want.
func mustRead(t *testing.T, m *manager, file FileNo, fork Fork, block page.BlockNumber, want byte) {
	t.Helper()
	buf := make(page.Page, page.Size)
	if err := m.Read(file, fork, block, buf); err != nil {
		t.Fatalf("Read(%d, %d): %v", file, block, err)
	}
	if !bytes.Equal(buf, pageOf(want)) {
		t.Fatalf("Read(%d, %d): contenido %d…, se esperaba %d", file, block, buf[0], want)
	}
}

func TestCreateExistsNBlocks(t *testing.T) {
	m, fs, dir := newTestManager(t, 8)

	if ok, err := m.Exists(1, ForkMain); err != nil || ok {
		t.Fatalf("Exists antes de Create = %v, %v", ok, err)
	}
	if _, err := m.NBlocks(1, ForkMain); !errors.Is(err, ErrNoFork) {
		t.Fatalf("NBlocks sin Create: %v", err)
	}
	before := fs.dirSyncs.Load()
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if fs.dirSyncs.Load() != before {
		t.Fatal("Create hizo fsync de data/: lo hace SyncAll")
	}
	if err := m.SyncAll(); err != nil || fs.dirSyncs.Load() != before+1 {
		t.Fatalf("SyncAll tras Create no hizo fsync de data/: %v", err)
	}
	if err := m.SyncAll(); err != nil || fs.dirSyncs.Load() != before+1 {
		t.Fatalf("SyncAll sin cambios hizo fsync de data/: %v", err)
	}
	if err := m.Create(1, ForkMain); err != nil { // idempotente
		t.Fatal(err)
	}
	if ok, _ := m.Exists(1, ForkMain); !ok {
		t.Fatal("Exists tras Create = false")
	}
	if n, err := m.NBlocks(1, ForkMain); err != nil || n != 0 {
		t.Fatalf("NBlocks = %d, %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "0000000001.0001")); err != nil {
		t.Fatal(err)
	}
	if err := m.Create(1, ForkVM); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "0000000001_vm.0001")); err != nil {
		t.Fatal(err)
	}
	if err := m.Create(1, Fork(7)); !errors.Is(err, ErrBadFork) {
		t.Fatalf("fork desconocido: %v", err)
	}
}

func TestExtendReadWrite(t *testing.T) {
	m, _, _ := newTestManager(t, 8)
	if err := m.Extend(1, ForkMain, 0, pageOf(1)); !errors.Is(err, ErrNoFork) {
		t.Fatalf("Extend sin Create: %v", err)
	}
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	for b := page.BlockNumber(0); b < 3; b++ {
		if err := m.Extend(1, ForkMain, b, pageOf(byte(b+1))); err != nil {
			t.Fatal(err)
		}
	}
	for b := page.BlockNumber(0); b < 3; b++ {
		mustRead(t, m, 1, ForkMain, b, byte(b+1))
	}
	if err := m.Write(1, ForkMain, 1, pageOf(9)); err != nil {
		t.Fatal(err)
	}
	mustRead(t, m, 1, ForkMain, 1, 9)

	buf := make(page.Page, page.Size)
	if err := m.Read(1, ForkMain, 3, buf); !errors.Is(err, ErrBlockOutOfRange) {
		t.Fatalf("Read más allá del final: %v", err)
	}
	if err := m.Write(1, ForkMain, 3, pageOf(1)); !errors.Is(err, ErrBlockOutOfRange) {
		t.Fatalf("Write más allá del final: %v", err)
	}
	if err := m.Extend(1, ForkMain, 2, pageOf(1)); !errors.Is(err, ErrBlockExists) {
		t.Fatalf("Extend de un bloque que existe: %v", err)
	}
	if err := m.Read(1, ForkMain, 0, make(page.Page, 100)); !errors.Is(err, ErrBadPage) {
		t.Fatalf("buffer corto: %v", err)
	}
	if err := m.Extend(1, ForkMain, ^page.BlockNumber(0), pageOf(1)); !errors.Is(err, ErrBlockOutOfRange) {
		t.Fatalf("bloque inválido: %v", err)
	}
}

// Extend con hueco: los bloques intermedios existen y están en ceros.
func TestExtendGapIsZero(t *testing.T) {
	m, _, _ := newTestManager(t, 8)
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := m.Extend(1, ForkMain, 2, pageOf(5)); err != nil {
		t.Fatal(err)
	}
	if n, _ := m.NBlocks(1, ForkMain); n != 3 {
		t.Fatalf("NBlocks = %d", n)
	}
	mustRead(t, m, 1, ForkMain, 0, 0)
	mustRead(t, m, 1, ForkMain, 1, 0)
	mustRead(t, m, 1, ForkMain, 2, 5)
}

// El paso de un archivo al siguiente: nombres, largos, fsync de data/ y lectura tras reabrir.
func TestSegments(t *testing.T) {
	m, fs, dir := newTestManager(t, 8)
	if err := m.Create(7, ForkMain); err != nil {
		t.Fatal(err)
	}
	dirSyncs := fs.dirSyncs.Load()
	for b := page.BlockNumber(0); b < 10; b++ {
		if err := m.Extend(7, ForkMain, b, pageOf(byte(b+1))); err != nil {
			t.Fatal(err)
		}
	}
	if got := fs.dirSyncs.Load() - dirSyncs; got != 2 { // al nacer .0002 y .0003
		t.Fatalf("fsync de data/ al extender = %d, se esperaban 2", got)
	}
	full := int64(testBlocksPerFile * page.Size)
	for seg, want := range map[string]int64{"0001": full, "0002": full, "0003": 2 * page.Size} {
		info, err := os.Stat(filepath.Join(dir, "data", "0000000007."+seg))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != want {
			t.Fatalf("archivo %s mide %d, se esperaba %d", seg, info.Size(), want)
		}
	}

	m = reopen(t, m, fs, dir)
	if n, err := m.NBlocks(7, ForkMain); err != nil || n != 10 {
		t.Fatalf("NBlocks tras reabrir = %d, %v", n, err)
	}
	for b := page.BlockNumber(0); b < 10; b++ {
		mustRead(t, m, 7, ForkMain, b, byte(b+1))
	}
}

// Un hueco que salta archivos deja los intermedios llenos (1 GB en producción) y en ceros.
func TestExtendGapAcrossSegments(t *testing.T) {
	m, fs, dir := newTestManager(t, 8)
	if err := m.Create(3, ForkVM); err != nil {
		t.Fatal(err)
	}
	if err := m.Extend(3, ForkVM, 1, pageOf(1)); err != nil {
		t.Fatal(err)
	}
	if err := m.Extend(3, ForkVM, 9, pageOf(2)); err != nil { // archivo 3, bloque 1
		t.Fatal(err)
	}
	full := int64(testBlocksPerFile * page.Size)
	for _, seg := range []string{"0001", "0002"} {
		info, err := os.Stat(filepath.Join(dir, "data", "0000000003_vm."+seg))
		if err != nil || info.Size() != full {
			t.Fatalf("archivo %s: %v, %v", seg, info, err)
		}
	}
	m = reopen(t, m, fs, dir)
	if n, _ := m.NBlocks(3, ForkVM); n != 10 {
		t.Fatalf("NBlocks = %d", n)
	}
	mustRead(t, m, 3, ForkVM, 1, 1)
	mustRead(t, m, 3, ForkVM, 5, 0)
	mustRead(t, m, 3, ForkVM, 8, 0)
	mustRead(t, m, 3, ForkVM, 9, 2)
}

// Archivos cortos: uno del medio (al abrir) y el último cortado desde fuera (al leer).
func TestShortFile(t *testing.T) {
	m, fs, dir := newTestManager(t, 8)
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	for b := page.BlockNumber(0); b < 6; b++ {
		if err := m.Extend(1, ForkMain, b, pageOf(1)); err != nil {
			t.Fatal(err)
		}
	}

	// El último cortado mientras el gestor cree que tiene el bloque.
	last := filepath.Join(dir, "data", "0000000001.0002")
	if err := os.Truncate(last, page.Size); err != nil {
		t.Fatal(err)
	}
	buf := make(page.Page, page.Size)
	if err := m.Read(1, ForkMain, 5, buf); !errors.Is(err, ErrShortFile) {
		t.Fatalf("Read de un bloque cortado: %v", err)
	}

	// Uno del medio corto seguido de otro: corrupción al abrir.
	if err := os.Truncate(filepath.Join(dir, "data", "0000000001.0001"), page.Size); err != nil {
		t.Fatal(err)
	}
	m = reopen(t, m, fs, dir)
	if _, err := m.NBlocks(1, ForkMain); !errors.Is(err, ErrShortFile) {
		t.Fatalf("archivo del medio corto: %v", err)
	}
}

// Un bloque escrito a medias al final del último archivo no cuenta.
func TestPartialTailBlock(t *testing.T) {
	m, fs, dir := newTestManager(t, 8)
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := m.Extend(1, ForkMain, 0, pageOf(1)); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(filepath.Join(dir, "data", "0000000001.0001"), page.Size+100); err != nil {
		t.Fatal(err)
	}
	m = reopen(t, m, fs, dir)
	if n, _ := m.NBlocks(1, ForkMain); n != 1 {
		t.Fatalf("NBlocks = %d", n)
	}
	if err := m.Extend(1, ForkMain, 1, pageOf(2)); err != nil {
		t.Fatal(err)
	}
	mustRead(t, m, 1, ForkMain, 1, 2)
}

// La caché de descriptores no pasa de su tope con muchos stores.
func TestFileCacheBound(t *testing.T) {
	const maxFiles = 3
	m, fs, _ := newTestManager(t, maxFiles)
	for f := FileNo(1); f <= 50; f++ {
		if err := m.Create(f, ForkMain); err != nil {
			t.Fatal(err)
		}
		for b := page.BlockNumber(0); b < 6; b++ { // dos archivos por store
			if err := m.Extend(f, ForkMain, b, pageOf(byte(f))); err != nil {
				t.Fatal(err)
			}
		}
	}
	for f := FileNo(1); f <= 50; f++ {
		mustRead(t, m, f, ForkMain, 5, byte(f))
	}
	if got := fs.maxOpen.Load(); got > maxFiles+1 { // +1: el recién abierto antes de expulsar al viejo
		t.Fatalf("hubo %d descriptores abiertos a la vez, tope %d", got, maxFiles)
	}
	if got := m.files.count(); got > maxFiles {
		t.Fatalf("la caché tiene %d descriptores, tope %d", got, maxFiles)
	}
	if err := m.SyncAll(); err != nil {
		t.Fatal(err)
	}
}

// Sync hace fsync solo de lo escrito desde el último; expulsar un archivo sucio hace fsync antes de cerrarlo.
func TestSync(t *testing.T) {
	m, fs, dir := newTestManager(t, 2)
	name := func(f FileNo, seg string) string { return filepath.Join(dir, "data", fmt.Sprintf("%010d.%s", f, seg)) }

	for f := FileNo(1); f <= 2; f++ {
		if err := m.Create(f, ForkMain); err != nil {
			t.Fatal(err)
		}
		if err := m.Extend(f, ForkMain, 0, pageOf(1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Sync(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if fs.syncCount(name(1, "0001")) != 1 || fs.syncCount(name(2, "0001")) != 0 {
		t.Fatal("Sync(1) debía sincronizar solo el store 1")
	}
	if err := m.Sync(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if fs.syncCount(name(1, "0001")) != 1 {
		t.Fatal("Sync sin escrituras nuevas no debía hacer fsync")
	}
	if err := m.SyncAll(); err != nil {
		t.Fatal(err)
	}
	if fs.syncCount(name(2, "0001")) != 1 {
		t.Fatal("SyncAll no sincronizó el store 2")
	}

	// Store 2 sucio y expulsado por abrir los stores 3, 4 y 5 (tope 2).
	if err := m.Write(2, ForkMain, 0, pageOf(2)); err != nil {
		t.Fatal(err)
	}
	for f := FileNo(3); f <= 5; f++ {
		if err := m.Create(f, ForkMain); err != nil {
			t.Fatal(err)
		}
	}
	if fs.syncCount(name(2, "0001")) != 2 {
		t.Fatal("expulsar un archivo sucio debía hacer fsync")
	}
	if err := m.SyncAll(); err != nil { // sigue marcado: se vuelve a sincronizar
		t.Fatal(err)
	}
	if fs.syncCount(name(2, "0001")) != 3 {
		t.Fatal("SyncAll debía sincronizar el archivo expulsado")
	}
}

// Un fsync fallido deja el gestor inutilizable: no se reintenta ni da un éxito falso.
func TestSyncFailureIsSticky(t *testing.T) {
	fs := crashfs.New()
	m, err := open(fs, "/db", 8, testBlocksPerFile)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := m.Extend(1, ForkMain, 0, pageOf(1)); err != nil {
		t.Fatal(err)
	}
	fs.FailSyncAfter(0)
	if err := m.SyncAll(); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("SyncAll: %v", err)
	}
	fs.FailSyncAfter(-1)
	if err := m.SyncAll(); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("SyncAll tras el fallo: %v", err)
	}
	if err := m.Read(1, ForkMain, 0, make(page.Page, page.Size)); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("Read tras el fallo: %v", err)
	}
}

func TestUnlink(t *testing.T) {
	m, fs, dir := newTestManager(t, 8)
	for _, fork := range []Fork{ForkMain, ForkVM} {
		if err := m.Create(1, fork); err != nil {
			t.Fatal(err)
		}
		for b := page.BlockNumber(0); b < 9; b++ {
			if err := m.Extend(1, fork, b, pageOf(1)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := m.Create(2, ForkMain); err != nil {
		t.Fatal(err)
	}
	dirSyncs := fs.dirSyncs.Load()
	if err := m.Unlink(1); err != nil {
		t.Fatal(err)
	}
	if err := m.SyncAll(); err != nil || fs.dirSyncs.Load() != dirSyncs+1 {
		t.Fatalf("SyncAll tras Unlink no hizo un fsync de data/: %v", err)
	}
	names, err := fs.ReadDir(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "0000000002.0001" {
		t.Fatalf("quedaron %v", names)
	}
	if ok, _ := m.Exists(1, ForkMain); ok {
		t.Fatal("Exists tras Unlink = true")
	}
	if err := m.Read(1, ForkMain, 0, make(page.Page, page.Size)); !errors.Is(err, ErrNoFork) {
		t.Fatalf("Read tras Unlink: %v", err)
	}
	if err := m.Unlink(1); err != nil { // idempotente
		t.Fatal(err)
	}
	if err := m.SyncAll(); err != nil { // lo sucio del store borrado se olvidó
		t.Fatal(err)
	}
}

func TestClose(t *testing.T) {
	m, fs, _ := newTestManager(t, 8)
	if err := m.Create(1, ForkMain); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if fs.open.Load() != 0 {
		t.Fatalf("quedaron %d descriptores abiertos", fs.open.Load())
	}
	if _, err := m.Exists(1, ForkMain); !errors.Is(err, ErrClosed) {
		t.Fatalf("Exists tras Close: %v", err)
	}
	if err := m.SyncAll(); !errors.Is(err, ErrClosed) {
		t.Fatalf("SyncAll tras Close: %v", err)
	}
}

// Varias goroutines extienden, escriben, leen y sincronizan a la vez, con una caché chica (correr con -race).
func TestConcurrent(t *testing.T) {
	const (
		stores  = 8
		workers = 4
		blocks  = 12 // tres archivos por store
	)
	m, fs, dir := newTestManager(t, 3)
	for f := FileNo(1); f <= stores; f++ {
		if err := m.Create(f, ForkMain); err != nil {
			t.Fatal(err)
		}
	}

	// Cada worker extiende sus bloques (b % workers == w) de cada store, de a uno y en orden.
	var wg sync.WaitGroup
	errs := make(chan error, stores*workers*2)
	var extendMu [stores + 1]sync.Mutex
	next := make([]page.BlockNumber, stores+1)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for f := FileNo(1); f <= stores; f++ {
				for {
					extendMu[f].Lock()
					b := next[f]
					if b >= blocks {
						extendMu[f].Unlock()
						break
					}
					next[f]++
					err := m.Extend(f, ForkMain, b, pageOf(byte(b)))
					extendMu[f].Unlock()
					if err != nil {
						errs <- err
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()

	// Cada worker reescribe y relee los bloques b % workers == w de todos los stores, con Sync en medio.
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			buf := make(page.Page, page.Size)
			for f := FileNo(1); f <= stores; f++ {
				for b := page.BlockNumber(w); b < blocks; b += workers {
					if err := m.Write(f, ForkMain, b, pageOf(byte(b+100))); err != nil {
						errs <- err
						return
					}
					if err := m.Read(f, ForkMain, b, buf); err != nil || buf[0] != byte(b+100) {
						errs <- fmt.Errorf("Read(%d, %d) = %d, %v", f, b, buf[0], err)
						return
					}
				}
				if err := m.Sync(f, ForkMain); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			if err := m.SyncAll(); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	m = reopen(t, m, fs, dir)
	for f := FileNo(1); f <= stores; f++ {
		if n, _ := m.NBlocks(f, ForkMain); n != blocks {
			t.Fatalf("store %d: NBlocks = %d", f, n)
		}
		for b := page.BlockNumber(0); b < blocks; b++ {
			mustRead(t, m, f, ForkMain, b, byte(b+100))
		}
	}
}
