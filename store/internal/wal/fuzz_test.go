package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// Fuzzing del formato de registro y de la recuperación. Las semillas (f.Add y testdata/fuzz/) corren como pruebas
// normales; para fuzzear de verdad, de a un fuzzer y con poco tiempo:
//
//	go test ./store/internal/wal -run '^$' -fuzz FuzzDecodeRecord -fuzztime 1m

// FuzzRecordRoundTrip: todo registro que encode acepta se decodifica igual.
func FuzzRecordRoundTrip(f *testing.F) {
	f.Add(uint64(1), uint8(KindLeafWrite), uint8(FlagCommit), uint8(1), uint8(BlockWillInit), []byte("delta"), []byte("main"), false)
	f.Add(uint64(0), uint8(KindCheckpoint), uint8(0), uint8(0), uint8(0), []byte(nil), make([]byte, 48), false)
	f.Add(uint64(7), uint8(KindSplit), uint8(0), uint8(3), uint8(0), []byte("x"), []byte(nil), true)
	f.Add(uint64(7), uint8(0), uint8(0xFF), uint8(9), uint8(0xFF), []byte("x"), []byte(nil), true)
	f.Fuzz(func(t *testing.T, xid uint64, kind, flags, blocks, bflags uint8, delta, main []byte, image bool) {
		rec := Record{XID: page.XID(xid), Kind: Kind(kind), Flags: Flags(flags), Main: main}
		for i := range int(blocks) % (MaxBlocks + 2) {
			b := BlockRef{File: smgr.FileNo(i + 1), Fork: smgr.Fork(i), Block: page.BlockNumber(xid) + page.BlockNumber(i),
				Flags: BlockFlags(bflags), Delta: delta}
			if image {
				if err := b.SetImage(testPage(64+i, 4000, xid)); err != nil {
					t.Fatal(err)
				}
			}
			rec.Blocks = append(rec.Blocks, b)
		}
		size, err := recordSize(&rec)
		if err != nil {
			return
		}
		buf := make([]byte, size)
		encodeRecord(&rec, buf)
		sealRecord(buf, page.LSN(xid|1))
		var got Record
		n, ok := decodeRecord(buf, &got)
		rec.LSN = page.LSN(xid | 1)
		if !ok || n != size || !sameRecord(&got, &rec) {
			t.Fatalf("ida y vuelta: ok %v, n %d de %d", ok, n, size)
		}
	})
}

// FuzzDecodeRecord: decode no entra en pánico con ningún byte, y lo que acepta es canónico: volver a codificarlo da
// exactamente los mismos bytes.
func FuzzDecodeRecord(f *testing.F) {
	for _, seed := range recordSeeds(f) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		checkDecode(t, data)
		checkDecode(t, withCRC(data)) // sin esto el fuzzer casi nunca pasa el crc y no explora la estructura
	})
}

// withCRC es una copia de data con el crc correcto para su totalLen (si lo tiene y cabe).
func withCRC(data []byte) []byte {
	data = bytes.Clone(data)
	if len(data) < HeaderSize {
		return data
	}
	total := totalLenOf(data)
	if total < offLSN || total > len(data) {
		return data
	}
	binary.LittleEndian.PutUint32(data[offCRC:], crc32.Checksum(data[offLSN:total], castagnoli))
	return data
}

// checkDecode: decodeRecord no entra en pánico, y si acepta data, volver a codificar da los mismos bytes.
func checkDecode(t *testing.T, data []byte) {
	var rec Record
	n, ok := decodeRecord(data, &rec)
	if !ok {
		return
	}
	if n > len(data) || n < HeaderSize {
		t.Fatalf("largo %d de %d", n, len(data))
	}
	size, err := recordSize(&rec)
	if err != nil || size != n {
		t.Fatalf("aceptado pero no codificable: %v, %d de %d", err, size, n)
	}
	buf := make([]byte, size)
	encodeRecord(&rec, buf)
	sealRecord(buf, rec.LSN)
	if !bytes.Equal(buf, data[:n]) {
		t.Fatal("aceptado pero no canónico")
	}
}

// recordSeeds son registros reales para los fuzzers de bytes: válidos, cortados, en ceros, con el crc cambiado,
// con totalLen máximo y uno de más.
func recordSeeds(t testing.TB) [][]byte {
	img, flags, _ := MakeImage(testPage(100, 7000, 1))
	split := Record{Kind: KindSplit, XID: 3, Blocks: []BlockRef{
		{File: 1, Block: 2, Flags: flags, Image: img, Delta: []byte("d")}, {File: 1, Block: 3, Flags: BlockWillInit},
	}}
	leaf := encoded(t, testRecord(1), 1)
	badCRC := bytes.Clone(leaf)
	badCRC[0] ^= 0xFF
	maxLen := bytes.Clone(leaf[:HeaderSize])
	maxLen[offTotalLen+2] = 1 // 64 KiB
	return [][]byte{
		leaf,
		encoded(t, split, 2),
		encoded(t, Record{Kind: KindAbort}, 3),
		append(bytes.Clone(leaf), encoded(t, testRecord(2), 2)...),
		leaf[:len(leaf)/2],
		make([]byte, HeaderSize),
		badCRC,
		maxLen,
		append(bytes.Clone(maxLen), 0xFF),
	}
}

// fuzzHistory es la historia de FuzzRecoverSegment: cuántos registros y dónde quedó cada uno.
const fuzzRecords = 120

// recordPlace es dónde quedó un registro: segmento, desplazamiento y largo.
type recordPlace struct {
	segment uint64
	offset  int64
	size    int64
}

// writeTemplate escribe la historia de prueba en dir y devuelve el lugar de cada registro.
func writeTemplate(t testing.TB, fsys vfs.FS, dir string) []recordPlace {
	w, _, err := Open(fsys, dir, smallOptions(0), Start, nil)
	if err != nil {
		t.Fatal(err)
	}
	var places []recordPlace
	for i := 1; i <= fuzzRecords; i++ {
		rec := testRecord(i)
		r, err := w.Reserve(&rec, nil)
		if err != nil {
			t.Fatal(err)
		}
		res := r.(*reservation)
		places = append(places, recordPlace{res.seg.num, res.offset, int64(len(res.buf))})
		if err := r.Write(); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return places
}

// readAll lee un archivo entero de fsys.
func readAll(t testing.TB, fsys vfs.FS, path string) []byte {
	f, err := fsys.OpenFile(path, os.O_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	size, err := f.Size()
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	if _, err := f.ReadAt(data, 0); err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	return data
}

// writeAll reemplaza el contenido de un archivo de fsys por data (lo crea si falta).
func writeAll(t testing.TB, fsys vfs.FS, path string, data []byte) {
	f, err := fsys.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err == nil {
		_, err = f.WriteAt(data, 0)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

// copyDir copia los archivos de src/wal en el disco from a dst/wal en el disco to.
func copyDir(t testing.TB, from vfs.FS, src string, to vfs.FS, dst string) {
	names, err := from.ReadDir(filepath.Join(src, Dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := to.MkdirAll(filepath.Join(dst, Dir)); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		writeAll(t, to, filepath.Join(dst, Dir, name), readAll(t, from, filepath.Join(src, Dir, name)))
	}
}

// damage es un daño aplicado a un segmento: desde qué byte cambia (from) y si deja datos detrás (append).
type damage struct {
	segment uint64
	from    int64
	append  bool
}

// corrupt daña el segmento which de dir según mode: 0 pisa con junk en pos, 1 corta en pos, 2 pone ceros en pos, 3
// agrega junk al final.
func corrupt(t *testing.T, fsys vfs.FS, dir string, last uint64, mode, which uint8, pos uint16, junk []byte) damage {
	num := uint64(which)%last + 1
	path := segPath(dir, num)
	data := readAll(t, fsys, path)
	at := int(pos) % (len(data) + 1)
	d := damage{segment: num, from: int64(at)}
	switch mode % 4 {
	case 0:
		data = append(data[:at], append(bytes.Clone(junk), data[min(at+len(junk), len(data)):]...)...)
	case 1:
		data = data[:at]
	case 2:
		clear(data[at:min(at+max(len(junk), 1), len(data))])
	case 3:
		d.from, d.append = int64(len(data)), true
		data = append(data, junk...)
	}
	writeAll(t, fsys, path, data)
	return d
}

// forgesRecord dice si junk contiene en algún lugar un registro válido: el fuzzer pudo fabricar uno.
func forgesRecord(junk []byte) bool {
	var rec Record
	for i := range junk {
		if _, ok := decodeRecord(junk[i:], &rec); ok {
			return true
		}
	}
	return false
}

// FuzzRecoverSegment: con un segmento dañado (pisado, cortado, en ceros o con basura al final), Open devuelve un
// prefijo exacto de la historia con todo lo anterior al daño, o falla con ErrCorrupted si el daño está en un
// segmento sellado; y tras recuperar se sigue escribiendo y reabriendo bien. Corre sobre crashfs: en memoria, sin
// fsync real.
func FuzzRecoverSegment(f *testing.F) {
	const template, dir = "/template", "/db"
	templateFS := crashfs.New()
	places := writeTemplate(f, templateFS, template)
	last := places[len(places)-1].segment

	f.Add(uint8(0), uint8(0), uint16(100), []byte("basura"))
	f.Add(uint8(1), uint8(last-1), uint16(5000), []byte(nil))
	f.Add(uint8(2), uint8(0), uint16(0), make([]byte, 64))
	f.Add(uint8(3), uint8(last-1), uint16(0), encoded(f, testRecord(fuzzRecords+1), fuzzRecords+1))
	f.Add(uint8(3), uint8(0), uint16(0), encoded(f, testRecord(5), 5))
	f.Fuzz(func(t *testing.T, mode, which uint8, pos uint16, junk []byte) {
		fsys := crashfs.New()
		copyDir(t, templateFS, template, fsys, dir)
		d := corrupt(t, fsys, dir, last, mode, which, pos, junk)
		forged := forgesRecord(junk)

		var recs []Record
		w, end, err := Open(fsys, dir, smallOptions(0), Start, func(rec *Record) error {
			recs = append(recs, cloneRecord(rec))
			return nil
		})
		if err != nil {
			if !errors.Is(err, ErrCorrupted) || (!forged && (d.segment == last || d.append)) {
				t.Fatalf("Open falló con el daño %+v: %v", d, err)
			}
			return
		}
		survivors := 0
		for _, p := range places {
			if p.segment < d.segment || (p.segment == d.segment && p.offset+p.size <= d.from) {
				survivors++
			}
		}
		if len(recs) < survivors {
			t.Fatalf("se perdieron registros anteriores al daño %+v: %d de %d", d, len(recs), survivors)
		}
		for i := range recs {
			want := testRecord(i + 1)
			want.LSN = page.LSN(i + 1)
			if i >= fuzzRecords || !sameRecord(&recs[i], &want) {
				if forged {
					break
				}
				t.Fatalf("registro %d distinto tras el daño %+v", i+1, d)
			}
		}
		if end.LSN != page.LSN(len(recs)+1) {
			t.Fatalf("fin %+v con %d registros", end, len(recs))
		}

		next := testRecord(fuzzRecords + 7)
		if _, err := appendOne(w, next); err != nil {
			t.Fatal(err)
		}
		w.Close()
		n := 0
		w, _, err = Open(fsys, dir, smallOptions(0), Start, func(rec *Record) error {
			n++
			return nil
		})
		if err != nil || n != len(recs)+1 {
			t.Fatalf("tras seguir escribiendo: %d registros, %v", n, err)
		}
		w.Close()
	})
}

// appendOne reserva y escribe rec sin testing.TB.
func appendOne(w *WAL, rec Record) (page.LSN, error) {
	r, err := w.Reserve(&rec, nil)
	if err != nil {
		return 0, err
	}
	return r.LSN(), r.Write()
}
