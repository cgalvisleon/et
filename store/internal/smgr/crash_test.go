package smgr

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
)

// crashModes son los tres modos de corte de crashfs, con su nombre para los subtests.
var crashModes = map[string]crashfs.Mode{
	"LoseUnsynced":    crashfs.LoseUnsynced,
	"ReorderUnsynced": crashfs.ReorderUnsynced,
	"TornWrites":      crashfs.TornWrites,
}

// openCrash abre un gestor sobre el disco simulado, en /db.
func openCrash(t *testing.T, fs crashfs.FS) *manager {
	t.Helper()
	m, err := open(fs, "/db", 8, testBlocksPerFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m
}

// checkSectors comprueba que cada sector del bloque tenga el contenido viejo o el nuevo (old o new repetidos): un
// corte puede dejar una página rota por sectores, nunca bytes de otro lado.
func checkSectors(t *testing.T, m *manager, block page.BlockNumber, old, new byte) {
	t.Helper()
	buf := make(page.Page, page.Size)
	if err := m.Read(1, ForkMain, block, buf); err != nil {
		t.Fatalf("Read(%d): %v", block, err)
	}
	for off := 0; off < page.Size; off += crashfs.SectorSize {
		sector := buf[off : off+crashfs.SectorSize]
		if !bytes.Equal(sector, bytes.Repeat([]byte{old}, len(sector))) &&
			!bytes.Equal(sector, bytes.Repeat([]byte{new}, len(sector))) {
			t.Fatalf("bloque %d, byte %d: %d, se esperaba %d o %d", block, off, sector[0], old, new)
		}
	}
}

// TestSyncedSurvivesCrash: lo sincronizado con SyncAll sobrevive a un corte en los tres modos. Lo reescrito y
// extendido después (dentro de los archivos que ya existían) queda viejo o nuevo, sector a sector; con LoseUnsynced,
// viejo.
func TestSyncedSurvivesCrash(t *testing.T) {
	for name, mode := range crashModes {
		for seed := int64(0); seed < 8; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				fs := crashfs.New()
				m := openCrash(t, fs)
				if err := m.Create(1, ForkMain); err != nil {
					t.Fatal(err)
				}
				for b := page.BlockNumber(0); b < 6; b++ { // el archivo 1 entero y medio archivo 2
					if err := m.Extend(1, ForkMain, b, pageOf(byte(b+1))); err != nil {
						t.Fatal(err)
					}
				}
				if err := m.SyncAll(); err != nil {
					t.Fatal(err)
				}
				for b := page.BlockNumber(0); b < 6; b++ {
					if err := m.Write(1, ForkMain, b, pageOf(byte(b+100))); err != nil {
						t.Fatal(err)
					}
				}
				for b := page.BlockNumber(6); b < 8; b++ { // completa el archivo 2, sin crear otro
					if err := m.Extend(1, ForkMain, b, pageOf(byte(b+100))); err != nil {
						t.Fatal(err)
					}
				}

				fs.Crash(mode, seed)
				m = openCrash(t, fs)
				n, err := m.NBlocks(1, ForkMain)
				if err != nil || n < 6 || n > 8 || (mode == crashfs.LoseUnsynced && n != 6) {
					t.Fatalf("NBlocks tras el corte = %d, %v", n, err)
				}
				for b := page.BlockNumber(0); b < n; b++ {
					old := byte(b + 1)
					if b >= 6 {
						old = 0 // extendido sin fsync: lo que no llegó queda en ceros
					}
					if mode == crashfs.LoseUnsynced {
						mustRead(t, m, 1, ForkMain, b, old)
						continue
					}
					checkSectors(t, m, b, old, byte(b+100))
				}
			})
		}
	}
}

// TestExtendNewFileCrash: un Extend que hace nacer un archivo hace fsync de los anteriores (llenos) y de data/, así
// que tras un corte el archivo nuevo existe aunque sus datos no hayan tenido fsync, ningún archivo del medio queda
// corto y el fork se puede abrir con al menos los bloques sincronizados.
func TestExtendNewFileCrash(t *testing.T) {
	for name, mode := range crashModes {
		for seed := int64(0); seed < 8; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				fs := crashfs.New()
				m := openCrash(t, fs)
				if err := m.Create(1, ForkMain); err != nil {
					t.Fatal(err)
				}
				for b := page.BlockNumber(0); b < 6; b++ {
					if err := m.Extend(1, ForkMain, b, pageOf(byte(b+1))); err != nil {
						t.Fatal(err)
					}
				}
				if err := m.SyncAll(); err != nil {
					t.Fatal(err)
				}
				for b := page.BlockNumber(6); b < 10; b++ { // completa el archivo 2 y hace nacer el 3
					if err := m.Extend(1, ForkMain, b, pageOf(byte(b+100))); err != nil {
						t.Fatal(err)
					}
				}

				fs.Crash(mode, seed)
				names, err := fs.ReadDir(filepath.Join("/db", DataDir))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Contains(names, "0000000001.0003") {
					t.Fatalf("el archivo nuevo se perdió pese al fsync de data/: %v", names)
				}
				m = openCrash(t, fs)
				n, err := m.NBlocks(1, ForkMain)
				if err != nil || n < 6 {
					t.Fatalf("NBlocks tras el corte = %d, %v; se esperaban al menos 6", n, err)
				}
				for b := page.BlockNumber(0); b < 6; b++ {
					mustRead(t, m, 1, ForkMain, b, byte(b+1))
				}
			})
		}
	}
}
