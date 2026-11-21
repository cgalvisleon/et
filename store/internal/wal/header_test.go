package wal

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestSegmentHeaders: cada segmento empieza con su encabezado (número y lsn de su primer registro), el primer
// registro va en SegmentHeaderSize y el encabezado se lee igual tras reabrir.
func TestSegmentHeaders(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	firsts := map[uint64]page.LSN{}
	for i := 1; i <= 300; i++ {
		rec := testRecord(i)
		r, err := w.Reserve(&rec, nil)
		if err != nil {
			t.Fatal(err)
		}
		res := r.(*reservation)
		if _, seen := firsts[res.seg.num]; !seen {
			if res.offset != SegmentHeaderSize {
				t.Fatalf("primer registro del segmento %d en %d", res.seg.num, res.offset)
			}
			firsts[res.seg.num] = r.LSN()
		}
		if err := r.Write(); err != nil {
			t.Fatal(err)
		}
	}
	if len(firsts) < 3 {
		t.Fatalf("solo %d segmentos", len(firsts))
	}
	mustClose(t, w)
	for num, first := range firsts {
		h, state, err := readHeaderOf(fsys, filepath.Join(dir, Dir), num)
		if err != nil || state != headerValid || h.Num != num || h.First != first || h.Version != FormatVersion {
			t.Fatalf("encabezado del segmento %d: %+v %d %v, se esperaba el primer lsn %d", num, h, state, err, first)
		}
	}
	_, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 300)
	if end.LSN != 301 {
		t.Fatalf("fin %+v", end)
	}
}

// TestDecodeHeader: un encabezado se lee igual que se escribió; ceros o un archivo corto es «ausente»; un bit
// cambiado, «roto»; otra versión se reconoce antes del crc.
func TestDecodeHeader(t *testing.T) {
	buf := encodeHeader(7, 1234)
	if h, state := decodeHeader(buf); state != headerValid || h.Num != 7 || h.First != 1234 {
		t.Fatalf("ida y vuelta: %+v %d", h, state)
	}
	if _, state := decodeHeader(make([]byte, SegmentHeaderSize)); state != headerAbsent {
		t.Fatalf("ceros: %d", state)
	}
	if _, state := decodeHeader(nil); state != headerAbsent {
		t.Fatalf("vacío: %d", state)
	}
	for _, off := range []int{offHeaderCRC, offHeaderReserved, offHeaderNum, offHeaderFirst + 7} {
		bad := encodeHeader(7, 1234)
		bad[off] ^= 0x01
		if _, state := decodeHeader(bad); state != headerBroken {
			t.Fatalf("byte %d cambiado: %d", off, state)
		}
	}
	other := encodeHeader(7, 1234)
	other[offHeaderVersion] = FormatVersion + 1
	if h, state := decodeHeader(other); state != headerVersion || h.Version != FormatVersion+1 {
		t.Fatalf("otra versión: %+v %d", h, state)
	}
}

// TestHeaderOtherVersion: un segmento con el encabezado de otra versión hace fallar Open con ErrFormatVersion (el del
// punto de rehacer y uno siguiente), y OtherVersion lo encuentra sin abrir el WAL.
func TestHeaderOtherVersion(t *testing.T) {
	for _, num := range []uint64{1, 2} {
		dir, fsys := t.TempDir(), vfs.OS()
		w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
		writeHistory(t, w, 200)
		mustClose(t, w)
		setVersion(t, dir, num, FormatVersion+1)
		if _, err := openErr(dir, smallOptions(0)); !errors.Is(err, ErrFormatVersion) {
			t.Fatalf("segmento %d de otra versión: %v", num, err)
		}
		name, version, found, err := OtherVersion(fsys, dir)
		if err != nil || !found || version != FormatVersion+1 || name != filepath.Join(Dir, segmentName(num)) {
			t.Fatalf("OtherVersion: %s %d %v %v", name, version, found, err)
		}
	}
	if _, _, found, err := OtherVersion(vfs.OS(), t.TempDir()); found || err != nil {
		t.Fatalf("sin WAL: %v %v", found, err)
	}
}

// TestHeaderBrokenFails: un encabezado roto es corrupción: el del segmento del punto de rehacer, y el de un segmento
// que sigue a uno sellado (su encabezado fue durable antes del sellado), aunque esté en ceros.
func TestHeaderBrokenFails(t *testing.T) {
	cases := map[string]func(dir string){
		"inicio":          func(dir string) { flipByte(t, dir, 1, offHeaderNum) },
		"siguiente":       func(dir string) { flipByte(t, dir, 2, offHeaderFirst) },
		"siguiente-ceros": func(dir string) { zeroHeader(t, dir, 2) },
		"último-en-ceros": func(dir string) { zeroHeader(t, dir, 3) },
		"último-roto":     func(dir string) { flipByte(t, dir, 3, offHeaderNum) },
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			dir, fsys := t.TempDir(), vfs.OS()
			w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
			writeHistory(t, w, 200)
			if w.ActiveSegment() < 3 {
				t.Fatalf("activo %d", w.ActiveSegment())
			}
			mustClose(t, w)
			damage(dir)
			if _, err := openErr(dir, smallOptions(0)); !errors.Is(err, ErrCorrupted) {
				t.Fatalf("Open con el encabezado dañado: %v", err)
			}
		})
	}
}

// TestHeaderMissingRewritten: un segmento creado y cortado antes de que su encabezado fuera durable (el primero de un
// motor nuevo) abre bien: Open le escribe el encabezado.
func TestHeaderMissingRewritten(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(segPath(dir, 1), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	w, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	if len(recs) != 0 || end != Start {
		t.Fatalf("fin %+v, %d registros", end, len(recs))
	}
	if h, state, err := readHeaderOf(fsys, filepath.Join(dir, Dir), 1); err != nil || state != headerValid || h.First != 1 {
		t.Fatalf("encabezado: %+v %d %v", h, state, err)
	}
	writeHistory(t, w, 10)
	mustClose(t, w)
	_, _, recs = openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 10)
}

// TestHeaderDurableAtRotate: tras una rotación sin ningún fsync posterior, un corte (los tres modos) deja el
// encabezado del segmento nuevo, y Open sigue en él aunque sus registros se perdieran.
func TestHeaderDurableAtRotate(t *testing.T) {
	for name, mode := range crashModes {
		t.Run(name, func(t *testing.T) {
			const dir = "/db"
			fsys := crashfs.New()
			w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
			i := 0
			for w.ActiveSegment() == 1 {
				i++
				appendRecord(t, w, testRecord(i))
			}
			fsys.Crash(mode, 1)
			w.Close()
			h, state, err := readHeaderOf(fsys, filepath.Join(dir, Dir), 2)
			if err != nil || state != headerValid || h.First != page.LSN(i) {
				t.Fatalf("encabezado del segmento 2: %+v %d %v, se esperaba el primer lsn %d", h, state, err, i)
			}
			w, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
			checkHistory(t, recs, len(recs))
			if len(recs) < i-1 || end.Segment != 2 {
				t.Fatalf("%d registros (sellados %d), fin %+v", len(recs), i-1, end)
			}
			appendRecord(t, w, testRecord(len(recs)+1))
		})
	}
}

/**
* setVersion cambia la versión del encabezado del segmento num del motor de dir (sin rehacer el crc: la versión se
* mira antes).
* @param t testing.TB, dir string, num uint64, version uint32
**/
func setVersion(t testing.TB, dir string, num uint64, version uint32) {
	t.Helper()
	writeAt(t, segPath(dir, num), offHeaderVersion, []byte{byte(version), byte(version >> 8), byte(version >> 16), byte(version >> 24)})
}

/**
* zeroHeader pone en ceros el encabezado del segmento num del motor de dir.
* @param t testing.TB, dir string, num uint64
**/
func zeroHeader(t testing.TB, dir string, num uint64) {
	t.Helper()
	writeAt(t, segPath(dir, num), 0, make([]byte, SegmentHeaderSize))
}

/**
* writeAt escribe b en el archivo path desde off.
* @param t testing.TB, path string, off int64, b []byte
**/
func writeAt(t testing.TB, path string, off int64, b []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteAt(b, off); err != nil {
		t.Fatal(err)
	}
}

// TestLegacySegmentIsOlderVersion: un segmento escrito antes del encabezado (registros desde el byte 0) es de una
// versión anterior: OtherVersion lo informa con la versión 0 y Open falla con ErrFormatVersion, no con ErrCorrupted.
func TestLegacySegmentIsOlderVersion(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := append(encoded(t, testRecord(1), 1), encoded(t, testRecord(2), 2)...)
	if err := os.WriteFile(segPath(dir, 1), legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	name, version, found, err := OtherVersion(fsys, dir)
	if err != nil || !found || version != legacyVersion || name != filepath.Join(Dir, segmentName(1)) {
		t.Fatalf("OtherVersion: %s %d %v %v", name, version, found, err)
	}
	if _, err := openErr(dir, smallOptions(0)); !errors.Is(err, ErrFormatVersion) {
		t.Fatalf("Open de un segmento sin encabezado: %v", err)
	}
	if after, _ := os.ReadFile(segPath(dir, 1)); string(after) != string(legacy) {
		t.Fatal("Open cambió el segmento")
	}
}

// orderFS anota, por segmento, si sus registros escritos ya tienen fsync, y falla si un encabezado se escribe
// mientras el segmento anterior tiene registros sin fsync.
type orderFS struct {
	vfs.FS
	t       *testing.T
	mu      sync.Mutex
	pending map[string]bool // segmento con registros escritos sin fsync
	headers int
}

type orderFile struct {
	vfs.File
	fs   *orderFS
	name string
}

func (o *orderFS) OpenFile(name string, flag int) (vfs.File, error) {
	f, err := o.FS.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return orderFile{f, o, filepath.Base(name)}, nil
}

func (f orderFile) WriteAt(b []byte, off int64) (int, error) {
	o := f.fs
	o.mu.Lock()
	num, _ := strconv.ParseUint(strings.TrimSuffix(f.name, segmentExt), 10, 64)
	switch {
	case off == 0 && len(b) == SegmentHeaderSize && num > 1:
		if o.pending[segmentName(num-1)] {
			o.t.Errorf("encabezado del segmento %d escrito con registros sin fsync en el %d", num, num-1)
		}
		o.headers++
	case off >= SegmentHeaderSize && !bytes.Equal(b, zeros[:min(len(b), len(zeros))]):
		o.pending[f.name] = true
	}
	o.mu.Unlock()
	return f.File.WriteAt(b, off)
}

func (f orderFile) Sync() error {
	err := f.File.Sync()
	f.fs.mu.Lock()
	if err == nil {
		delete(f.fs.pending, f.name)
	}
	f.fs.mu.Unlock()
	return err
}

// TestRotateSyncOrder: al rotar, los registros del activo tienen fsync antes de escribir el encabezado del siguiente
// (si no, un corte entre los dos fsync deja un encabezado que promete un lsn que el activo no tiene: un hueco falso).
// Con y sin búfer, con y sin preasignación.
func TestRotateSyncOrder(t *testing.T) {
	for _, opts := range []Options{smallOptions(0), {SegmentSize: minSegmentSize, Preallocate: true, Buffer: 1 << 20}} {
		fsys := &orderFS{FS: vfs.OS(), t: t, pending: map[string]bool{}}
		w, _, _ := openTest(t, fsys, t.TempDir(), opts, Start)
		writeHistory(t, w, 300)
		if _, err := w.Switch(); err != nil {
			t.Fatal(err)
		}
		mustClose(t, w)
		if fsys.headers < 3 {
			t.Fatalf("solo %d encabezados escritos", fsys.headers)
		}
	}
}
