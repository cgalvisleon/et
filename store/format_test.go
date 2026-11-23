package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Pruebas de la versión del formato en disco (tarea 54 de F-23; «Versiones del formato» en doc.go).

// updateFormatV2 genera store/testdata/format-v2 (el directorio de referencia del formato 2). Se usa una sola vez, al
// congelar el formato 2 (tarea 132: después de (b) y (c)); format-v1 se generó al congelar el 1 (2026-11-21) y nunca
// se regenera: este binario ya no escribe el formato 1.
var updateFormatV2 = flag.Bool("update-format-v2", false, "generate store/testdata/format-v2 (only when freezing format 2)")

// formatV1Dir y formatV2Dir son los directorios de referencia de los formatos 1 y 2.
const (
	formatV1Dir = "testdata/format-v1"
	formatV2Dir = "testdata/format-v2"
)

// formatReference es el directorio de referencia: el motor cerrado, una copia de Backup y lo que deben tener.
type formatReference struct {
	primary diskImage
	backup  diskImage
	want    contents
}

var (
	formatRefOnce sync.Once
	formatRef     formatReference
)

/**
* formatConfig es testConfig para el directorio de referencia: WAL de segmentos de 128 KB sin preasignar ni repuestos
* (archivos chicos: un segmento sellado mide entero) y con el archivo continuo (archive_status).
* @return config
**/
func formatConfig() config {
	cfg := testConfig()
	cfg.wal.SegmentSize = 128 << 10
	cfg.wal.Preallocate = false
	cfg.wal.Spares = 0
	cfg.archive = true
	cfg.compress = compression{method: compressZstd, min: defaultCompressMin} // formato 2: valores comprimidos
	return cfg
}

/**
* buildFormatReference arma una vez el directorio de referencia con todo lo que va a disco: hojas, páginas internas,
* meta, desborde (y una cadena liberada), páginas borradas y recicladas, versiones congeladas y el mapa de
* visibilidad, un store vaciado y uno borrado, un lote con origen, una ranura física y una lógica, un segmento sellado
* y archivado (archive_status) y una copia de Backup (backup_label).
* @param t testing.TB
* @return formatReference
**/
func buildFormatReference(t testing.TB) formatReference {
	formatRefOnce.Do(func() {
		fsys := crashfs.New()
		e := openTest(t, fsys, formatConfig())
		_, err := e.CreateSlot("replica")
		mustOK(t, err)
		_, err = e.CreateLogicalSlot("changes")
		mustOK(t, err)

		small := mustStore(t, e, "small")
		for i := range 120 {
			mustOK(t, small.Put(fmt.Sprintf("s%04d", i), bytes.Repeat([]byte{byte('a' + i%26)}, 40)))
		}
		r := rand.New(rand.NewPCG(7, 8))
		big := mustStore(t, e, "big")
		for i := range 3 { // al azar: no achican, van a desborde sin comprimir
			noise := make([]byte, 10_000)
			for j := range noise {
				noise[j] = byte(r.Uint32())
			}
			mustOK(t, big.Put(fmt.Sprintf("b%d", i), noise))
		}
		mustOK(t, big.Put("b0", bytes.Repeat([]byte{'Z'}, 9_000))) // la cadena vieja se libera en la limpieza
		zipped := mustStore(t, e, "zipped")
		for i, size := range []int{300, 4096, 16384, 40_000, 2048, 100} { // en línea, en desborde y sin comprimir
			mustOK(t, zipped.Put(fmt.Sprintf("z%d", i), jsonDoc(r, size)))
		}
		mustOK(t, zipped.Put("z1", jsonDoc(r, 30_000))) // la cadena comprimida vieja se libera en la limpieza
		pages := mustStore(t, e, "pages")
		for i := range 500 {
			mustOK(t, pages.Put(fmt.Sprintf("p%04d", i), bytes.Repeat([]byte{'p'}, 60)))
		}
		for i := 50; i < 450; i++ {
			mustOK(t, pages.Remove(fmt.Sprintf("p%04d", i)))
		}
		emptied := mustStore(t, e, "emptied")
		for i := range 20 {
			mustOK(t, emptied.Put(fmt.Sprintf("e%02d", i), []byte("old")))
		}
		mustOK(t, emptied.Empty())
		for i := range 3 {
			mustOK(t, emptied.Put(fmt.Sprintf("n%d", i), []byte("new")))
		}
		dropped := mustStore(t, e, "dropped")
		mustOK(t, dropped.Put("x", []byte("y")))
		mustOK(t, e.DropStore("dropped"))
		b := e.NewBatch()
		mustOK(t, b.SetOrigin(7))
		for i := range 5 {
			mustOK(t, small.PutIn(b, fmt.Sprintf("o%d", i), []byte("origin")))
		}
		mustOK(t, b.Commit())
		for range 3 { // borra hojas vacías, las recicla y congela
			mustOK(t, e.Vacuum())
		}
		seg, err := e.SwitchWAL()
		mustOK(t, err)
		mustOK(t, e.SetArchived(seg))
		mustOK(t, e.Checkpoint())

		bfs := crashfs.New()
		_, err = e.backup(bfs, testDir)
		mustOK(t, err)
		formatRef.want = dumpEngine(t, e)
		mustOK(t, e.Close())
		formatRef.primary = readImage(t, fsys, testDir)
		formatRef.backup = readImage(t, bfs, testDir)
	})
	return formatRef
}

/**
* rootFile da el contenido de un archivo de la raíz de img (nil si no está).
* @param img diskImage, name string
* @return []byte
**/
func rootFile(img diskImage, name string) []byte {
	return img.files[filepath.Join(testDir, name)]
}

/**
* checkCodecs comprueba la ida y vuelta de cada archivo de la raíz de img (lo que se decodifica se vuelve a escribir
* igual byte a byte) y la suma de cada página de data/. Una copia de Backup trae backup_label y no trae slots ni
* archive_status; un motor, al revés.
* @param t testing.TB, img diskImage, label bool: si img es una copia de Backup, strict bool: exigir los archivos de
* su clase (false: solo revisar los que estén)
**/
func checkCodecs(t testing.TB, img diskImage, label, strict bool) {
	t.Helper()
	ctl := rootFile(img, control.FileName)
	if len(ctl) != 2*control.CopySize {
		t.Fatalf("control of %d bytes", len(ctl))
	}
	valid := 0
	for off := 0; off < len(ctl); off += control.CopySize {
		c, err := control.DecodeCopy(ctl[off : off+control.CopySize])
		if err != nil {
			continue // la copia sin escribir de un motor nuevo
		}
		valid++
		again, err := control.EncodeCopy(c)
		mustOK(t, err)
		if !bytes.Equal(asWritten(again, ctl[off:off+control.CopySize]), ctl[off:off+control.CopySize]) {
			t.Fatalf("control copy at %d does not round-trip", off)
		}
	}
	if valid == 0 {
		t.Fatal("no valid control copy")
	}
	cat, err := control.DecodeCatalog(rootFile(img, control.CatalogFileName))
	mustOK(t, err)
	again, err := control.EncodeCatalog(cat)
	mustOK(t, err)
	if !bytes.Equal(asWritten(again, rootFile(img, control.CatalogFileName)), rootFile(img, control.CatalogFileName)) {
		t.Fatal("catalog does not round-trip")
	}
	if b := rootFile(img, slotsFileName); b != nil {
		slots, ok := decodeSlots(b)
		if !ok || !bytes.Equal(asWritten(encodeSlots(slots), b), b) {
			t.Fatalf("slots do not round-trip (%d slots, ok %v)", len(slots), ok)
		}
	} else if strict && !label {
		t.Fatal("no slots file")
	}
	if b := rootFile(img, archiveStatusName); b != nil {
		st, ok := decodeArchiveStatus(b)
		if !ok || !bytes.Equal(asWritten(encodeArchiveStatus(st), b), b) {
			t.Fatalf("archive_status does not round-trip: %+v %v", st, ok)
		}
	} else if strict && !label {
		t.Fatal("no archive_status file")
	}
	if b := rootFile(img, backupLabelName); b != nil {
		l, ok := decodeLabel(b)
		if !ok || l.endLSN == 0 || !bytes.Equal(asWritten(encodeLabel(l), b), b) {
			t.Fatalf("backup_label does not round-trip: %+v %v", l, ok)
		}
	} else if strict && label {
		t.Fatal("no backup_label in the copy")
	}
	pages := 0
	for _, p := range img.paths {
		if filepath.Base(filepath.Dir(p)) != smgr.DataDir {
			continue
		}
		data := img.files[p]
		if len(data)%page.Size != 0 {
			t.Fatalf("%s: %d bytes is not a whole number of pages", p, len(data))
		}
		for blk := 0; blk*page.Size < len(data); blk++ {
			mustOK(t, page.Verify(data[blk*page.Size:(blk+1)*page.Size], page.BlockNumber(blk)))
			pages++
		}
	}
	if pages == 0 {
		t.Fatal("no data pages")
	}
	segments := 0
	for _, p := range img.paths {
		if filepath.Base(filepath.Dir(p)) != wal.Dir {
			continue
		}
		h, err := wal.ParseSegmentHeader(img.files[p])
		mustOK(t, err)
		if wal.SegmentName(h.Num) != filepath.Base(p) || !wal.Readable(h.Version) || h.First == 0 {
			t.Fatalf("%s: header %+v", p, h)
		}
		segments++
	}
	if segments == 0 {
		t.Fatal("no WAL segments")
	}
}

/**
* asWritten es again (lo que escribe este binario, con FormatVersion) con la versión de orig en el byte 4 y su suma
* recalculada: un archivo de una versión anterior que este binario lee (formato 1) da ida y vuelta salvo la versión.
* @param again, orig []byte
* @return []byte
**/
func asWritten(again, orig []byte) []byte {
	if len(orig) < 8 || len(again) != len(orig) || le.Uint32(orig[4:]) == control.FormatVersion {
		return again
	}
	return setVersion(again, len(again), le.Uint32(orig[4:]))
}

/**
* checkReference abre el directorio de referencia (el motor y la copia de Backup, cada uno en un disco nuevo) y
* exige el contenido esperado, Verify limpio y que al reabrir siga igual.
* @param t testing.TB, ref formatReference
**/
func checkReference(t testing.TB, ref formatReference) {
	t.Helper()
	checkCodecs(t, ref.primary, false, true)
	checkCodecs(t, ref.backup, true, true)
	for _, img := range []diskImage{ref.primary, ref.backup} {
		fsys := writeImage(t, img)
		e := openTest(t, fsys, formatConfig())
		checkContents(t, e, ref.want)
		checkClean(t, e)
		mustOK(t, e.Close())
		e = openTest(t, fsys, formatConfig())
		checkContents(t, e, ref.want)
		mustOK(t, e.Close())
		checkCodecs(t, readImage(t, fsys, testDir), false, false) // lo que escribe al cerrar también da ida y vuelta
	}
}

// TestFormatRoundTrip: el directorio de referencia generado ahora (todas las clases de página, desborde, páginas
// borradas, ranuras, archivo continuo y backup_label) da ida y vuelta en cada archivo de la raíz, sus páginas tienen
// la suma bien, y abre con el contenido esperado, también tras reabrir.
func TestFormatRoundTrip(t *testing.T) {
	checkReference(t, buildFormatReference(t))
}

// TestFormatV1Reference abre el directorio de referencia v1 guardado en testdata/format-v1 (congelado el
// 2026-11-21): este binario lo lee y lo migra al 2 (TestFormatV1Migrates). Si falla, el cambio rompió la lectura del
// formato 1.
func TestFormatV1Reference(t *testing.T) {
	if _, err := os.Stat(formatV1Dir); errors.Is(err, os.ErrNotExist) {
		t.Fatalf("falta %s: el formato 1 está congelado (tarea 54; doc.go, «Versiones del formato»)", formatV1Dir)
	}
	checkReference(t, readReference(t, formatV1Dir))
}

// TestFormatV2Reference abre el directorio de referencia v2 guardado en testdata/format-v2 (congelado el 2026-11-22,
// tarea 134: compresión de valores y hojas de la forma 2). Se generó una vez con -update-format-v2 y nunca se
// regenera: si falla, el cambio rompió el formato 2.
func TestFormatV2Reference(t *testing.T) {
	if *updateFormatV2 {
		writeReference(t, buildFormatReference(t), formatV2Dir)
	}
	if _, err := os.Stat(formatV2Dir); errors.Is(err, os.ErrNotExist) {
		t.Fatalf("falta %s: el formato 2 está congelado (tarea 134; doc.go, «Versiones del formato»)", formatV2Dir)
	}
	checkReference(t, readReference(t, formatV2Dir))
}

// TestFormatV1Migrates: el directorio de referencia v1 (el motor cerrado limpio y la copia de Backup, que se recupera)
// abre con este binario, pasa al formato 2 (control, catálogo y el segmento activo del WAL; los segmentos viejos
// siguen en 1), acepta valores comprimidos y, tras reabrir, los devuelve; Verify limpio. Sus hojas son de la forma 1
// hasta que la limpieza las reescribe (migración perezosa): Verify y Stats cuentan las que faltan.
func TestFormatV1Migrates(t *testing.T) {
	ref := readReference(t, formatV1Dir)
	r := rand.New(rand.NewPCG(9, 10))
	doc := jsonDoc(r, 8192)
	for name, img := range map[string]diskImage{"primary": ref.primary, "backup": ref.backup} {
		t.Run(name, func(t *testing.T) {
			fsys := writeImage(t, img)
			e := openTest(t, fsys, formatConfig())
			if !e.leaves.fromV1.Load() {
				t.Fatal("a format 1 directory does not ask for the leaf migration")
			}
			if rep := checkClean(t, e); rep.LeavesForm1 == 0 {
				t.Fatalf("a format 1 directory without leaves of form 1:\n%s", rep)
			}
			if st := e.Stats(); st.LeafForm1Stores == 0 {
				t.Fatalf("Stats does not count the stores to migrate: %+v", st)
			}
			mustOK(t, e.Vacuum())
			if rep := checkClean(t, e); rep.LeavesForm1 != 0 {
				t.Fatalf("leaves of form 1 after Vacuum:\n%s", rep)
			}
			if st := e.Stats(); st.LeafForm1Stores != 0 || st.LeafRewrites == 0 {
				t.Fatalf("Stats after the migration: %d stores, %d rewrites", st.LeafForm1Stores, st.LeafRewrites)
			}
			s := mustStore(t, e, "small")
			mustOK(t, s.Put("migrated", doc))
			mustOK(t, e.Close())
			after := readImage(t, fsys, testDir)
			if v := newestControlVersion(t, rootFile(after, control.FileName)); v != control.FormatVersion {
				t.Fatalf("control version %d after open", v)
			}
			if v := le.Uint32(rootFile(after, control.CatalogFileName)[4:]); v != control.FormatVersion {
				t.Fatalf("catalog version %d after open", v)
			}
			last := ""
			for _, p := range after.paths {
				if filepath.Base(filepath.Dir(p)) == wal.Dir && filepath.Base(p) > last {
					last = filepath.Base(p)
				}
			}
			h, err := wal.ParseSegmentHeader(after.files[filepath.Join(testDir, wal.Dir, last)])
			mustOK(t, err)
			if h.Version != control.FormatVersion {
				t.Fatalf("active segment %s has version %d", last, h.Version)
			}
			cfg := formatConfig()
			cfg.compress = compression{} // sin compresión: lee lo comprimido igual
			e = openTest(t, fsys, cfg)
			s = mustStore(t, e, "small")
			got, found, err := s.Read("migrated")
			mustOK(t, err)
			if !found || !bytes.Equal(got, doc) {
				t.Fatalf("migrated value: found %v, %d bytes", found, len(got))
			}
			checkClean(t, e)
			mustOK(t, e.Close())
		})
	}
}

/**
* newestControlVersion es la versión de la copia válida más nueva del control (la de mayor seq).
* @param t testing.TB, ctl []byte: el archivo de control
* @return uint32
**/
func newestControlVersion(t testing.TB, ctl []byte) uint32 {
	t.Helper()
	var version uint32
	var seq uint64
	for off := 0; off+control.CopySize <= len(ctl); off += control.CopySize {
		c, err := control.DecodeCopy(ctl[off : off+control.CopySize])
		if err == nil && c.Seq >= seq {
			seq, version = c.Seq, le.Uint32(ctl[off+4:])
		}
	}
	return version
}

/**
* writeReference guarda ref en dir (testdata/format-v<n>): primary/ y backup/ con los archivos tal cual y
* want-bytes.json (los valores esperados en base64; format-v1 tiene want.json, con strings).
* @param t testing.TB, ref formatReference, dir string
**/
func writeReference(t testing.TB, ref formatReference, dir string) {
	mustOK(t, os.RemoveAll(dir))
	for sub, img := range map[string]diskImage{"primary": ref.primary, "backup": ref.backup} {
		for _, p := range img.paths {
			rel, err := filepath.Rel(testDir, p)
			mustOK(t, err)
			dst := filepath.Join(dir, sub, rel)
			mustOK(t, os.MkdirAll(filepath.Dir(dst), 0o755))
			mustOK(t, os.WriteFile(dst, img.files[p], 0o644))
		}
		for _, d := range img.dirs {
			rel, err := filepath.Rel(testDir, d)
			mustOK(t, err)
			mustOK(t, os.MkdirAll(filepath.Join(dir, sub, rel), 0o755))
		}
	}
	// want-bytes.json: los valores en base64 (json.Marshal de un string cambia los bytes que no son UTF-8 válido, y
	// desde el formato 2 la referencia tiene valores binarios); format-v1 tiene want.json con strings
	raw := map[string]map[string][]byte{}
	for store, ids := range ref.want {
		raw[store] = map[string][]byte{}
		for id, v := range ids {
			raw[store][id] = []byte(v)
		}
	}
	want, err := json.MarshalIndent(raw, "", "  ")
	mustOK(t, err)
	mustOK(t, os.WriteFile(filepath.Join(dir, "want-bytes.json"), want, 0o644))
}

/**
* readReference lee dir (testdata/format-v<n>) como un formatReference (cada directorio pasa a estar bajo testDir).
* @param t testing.TB, dir string
* @return formatReference
**/
func readReference(t testing.TB, dir string) formatReference {
	var ref formatReference
	for sub, img := range map[string]*diskImage{"primary": &ref.primary, "backup": &ref.backup} {
		root := filepath.Join(dir, sub)
		img.files = map[string][]byte{}
		mustOK(t, filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			path := filepath.Join(testDir, rel)
			if d.IsDir() {
				img.dirs = append(img.dirs, path)
				return nil
			}
			data, err := os.ReadFile(p)
			img.files[path] = data
			img.paths = append(img.paths, path)
			return err
		}))
	}
	if data, err := os.ReadFile(filepath.Join(dir, "want-bytes.json")); err == nil {
		raw := map[string]map[string][]byte{}
		mustOK(t, json.Unmarshal(data, &raw))
		ref.want = contents{}
		for store, ids := range raw {
			ref.want[store] = map[string]string{}
			for id, v := range ids {
				ref.want[store][id] = string(v)
			}
		}
		return ref
	}
	data, err := os.ReadFile(filepath.Join(dir, "want.json"))
	mustOK(t, err)
	mustOK(t, json.Unmarshal(data, &ref.want))
	return ref
}

/**
* setVersion pone version en el byte 4 de cada trozo de chunk bytes de b cuya suma (CRC-32C de lo anterior, en sus
* últimos 4 bytes) es válida, y la recalcula: un archivo bien escrito por un binario de otra versión.
* @param b []byte, chunk int, version uint32
* @return []byte: una copia cambiada
**/
func setVersion(b []byte, chunk int, version uint32) []byte {
	out := bytes.Clone(b)
	table := crc32.MakeTable(crc32.Castagnoli)
	for off := 0; off+chunk <= len(out); off += chunk {
		c := out[off : off+chunk]
		if le.Uint32(c[chunk-4:]) != crc32.Checksum(c[:chunk-4], table) {
			continue
		}
		le.PutUint32(c[4:], version)
		le.PutUint32(c[chunk-4:], crc32.Checksum(c[:chunk-4], table))
	}
	return out
}

// segmentHeaderChunk marca en TestOpenFormatVersion un segmento del WAL: la versión va en su encabezado.
const segmentHeaderChunk = -1

/**
* firstSegment es el primer segmento del WAL de img, relativo al motor («wal/0000000001.seg»).
* @param img diskImage
* @return string
**/
func firstSegment(img diskImage) string {
	first := ""
	for _, p := range img.paths {
		if filepath.Base(filepath.Dir(p)) == wal.Dir && (first == "" || filepath.Base(p) < first) {
			first = filepath.Base(p)
		}
	}
	return filepath.Join(wal.Dir, first)
}

/**
* setSegmentVersion pone version en el encabezado de un segmento del WAL (byte 8) y recalcula su suma (CRC-32C de
* [4, 32) en el byte 0): un segmento bien escrito por un binario de otra versión.
* @param b []byte, version uint32
* @return []byte: una copia cambiada
**/
func setSegmentVersion(b []byte, version uint32) []byte {
	out := bytes.Clone(b)
	le.PutUint32(out[8:], version)
	le.PutUint32(out[0:], crc32.Checksum(out[4:wal.SegmentHeaderSize], crc32.MakeTable(crc32.Castagnoli)))
	return out
}

// TestOpenFormatVersion (tarea 54): Open sobre un directorio con un archivo de otra versión del formato (más nueva o
// anterior) devuelve ErrFormatVersion y no toca nada del disco.
func TestOpenFormatVersion(t *testing.T) {
	ref := buildFormatReference(t)
	cases := []struct {
		name    string
		img     diskImage
		file    string
		chunk   int
		version uint32
	}{
		{"control newer", ref.primary, control.FileName, control.CopySize, control.FormatVersion + 1},
		{"control older", ref.primary, control.FileName, control.CopySize, 0},
		{"catalog newer", ref.primary, control.CatalogFileName, 0, control.FormatVersion + 1},
		{"slots newer", ref.primary, slotsFileName, 0, control.FormatVersion + 1},
		{"archive_status older", ref.primary, archiveStatusName, 0, 0},
		{"backup_label newer", ref.backup, backupLabelName, 0, 9},
		{"wal segment newer", ref.primary, firstSegment(ref.primary), segmentHeaderChunk, control.FormatVersion + 1},
		{"wal segment older", ref.backup, firstSegment(ref.backup), segmentHeaderChunk, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img := diskImage{files: map[string][]byte{}, paths: c.img.paths, dirs: c.img.dirs}
			for p, data := range c.img.files {
				img.files[p] = data
			}
			p := filepath.Join(testDir, c.file)
			chunk := c.chunk
			if chunk == 0 {
				chunk = len(img.files[p])
			}
			if chunk == segmentHeaderChunk {
				img.files[p] = setSegmentVersion(img.files[p], c.version)
			} else {
				img.files[p] = setVersion(img.files[p], chunk, c.version)
			}
			fsys := writeImage(t, img)
			_, err := open(fsys, testDir, formatConfig())
			if !errors.Is(err, ErrFormatVersion) {
				t.Fatalf("open: %v, want ErrFormatVersion", err)
			}
			if !strings.Contains(err.Error(), c.file) {
				t.Fatalf("the error does not name %s: %v", c.file, err)
			}
			after := readImage(t, fsys, testDir)
			for _, q := range img.paths {
				if !bytes.Equal(after.files[q], img.files[q]) {
					t.Fatalf("open changed %s", q)
				}
			}
			for _, q := range after.paths {
				if _, ok := img.files[q]; !ok && filepath.Base(q) != lockName {
					t.Fatalf("open created %s", q)
				}
			}
		})
	}
}

// TestOpenBrokenSegmentHeader (pregunta 76): un encabezado de segmento del WAL roto (el del punto de rehacer) hace
// fallar Open con ErrCorrupt; el primario y la copia de Backup, cada uno con su último segmento.
func TestOpenBrokenSegmentHeader(t *testing.T) {
	ref := buildFormatReference(t)
	for name, src := range map[string]diskImage{"primary": ref.primary, "backup": ref.backup} {
		t.Run(name, func(t *testing.T) {
			img := diskImage{files: map[string][]byte{}, paths: src.paths, dirs: src.dirs}
			last := ""
			for p, data := range src.files {
				img.files[p] = data
				if filepath.Base(filepath.Dir(p)) == wal.Dir && p > last {
					last = p
				}
			}
			broken := bytes.Clone(img.files[last])
			broken[20] ^= 0xff // el número del segmento: el crc ya no coincide
			img.files[last] = broken
			_, err := open(writeImage(t, img), testDir, formatConfig())
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("open with a broken header in %s: %v, want ErrCorrupt", last, err)
			}
		})
	}
}
