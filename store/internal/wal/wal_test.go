package wal

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"os"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// testPage es una página con encabezado válido, hueco [lower, upper) en ceros y el resto con bytes de semilla.
func testPage(lower, upper int, seed uint64) page.Page {
	p := make(page.Page, page.Size)
	r := rand.New(rand.NewPCG(seed, seed))
	for i := range p {
		p[i] = byte(r.IntN(4)) // poco variado: comprime
	}
	clear(p[lower:upper])
	p[page.OffLower], p[page.OffLower+1] = byte(lower), byte(lower>>8)
	p[page.OffUpper], p[page.OffUpper+1] = byte(upper), byte(upper>>8)
	return p
}

// TestRecordRoundTrip: lo que se codifica se decodifica igual, con y sin bloques, imágenes y datos principales.
func TestRecordRoundTrip(t *testing.T) {
	img, flags, err := MakeImage(testPage(200, 6000, 1))
	if err != nil {
		t.Fatal(err)
	}
	recs := []Record{
		{Kind: KindAbort, XID: 9},
		{Kind: KindCheckpoint, Main: bytes.Repeat([]byte{1}, 48)},
		testRecord(3),
		{Kind: KindSplit, XID: 4, Blocks: []BlockRef{
			{File: 2, Block: 10, Flags: flags, Image: img, Delta: []byte("izq")},
			{File: 2, Block: 11, Flags: BlockWillInit, Delta: []byte("der")},
			{File: 2, Fork: smgr.ForkVM, Block: 12},
		}},
	}
	for i := range recs {
		buf := encoded(t, recs[i], page.LSN(i+1))
		var got Record
		n, ok := decodeRecord(append(buf, 0xAA, 0xBB), &got) // bytes de más detrás: no son del registro
		recs[i].LSN = page.LSN(i + 1)
		if !ok || n != len(buf) || !sameRecord(&got, &recs[i]) {
			t.Fatalf("registro %d: ok %v, n %d de %d, %+v", i, ok, n, len(buf), got)
		}
	}
}

// TestRecordRejects: encode rechaza registros mal armados, y decode cualquier byte tocado.
func TestRecordRejects(t *testing.T) {
	bad := []Record{
		{},
		{Kind: KindAbort, Flags: 1 << 7},
		{Kind: KindAbort, Blocks: make([]BlockRef, MaxBlocks+1)},
		{Kind: KindAbort, Blocks: []BlockRef{{Flags: 1 << 6}}},
		{Kind: KindAbort, Blocks: []BlockRef{{Flags: BlockImageCompressed}}},
		{Kind: KindAbort, Blocks: []BlockRef{{Image: make([]byte, page.Size+1)}}},
		{Kind: KindAbort, Main: make([]byte, MaxRecordSize)},
	}
	for i := range bad {
		if _, err := recordSize(&bad[i]); err == nil {
			t.Fatalf("aceptó el registro %d", i)
		}
	}

	buf := encoded(t, testRecord(5), 5)
	var rec Record
	for i := range buf {
		damaged := bytes.Clone(buf)
		damaged[i] ^= 0x10
		if _, ok := decodeRecord(damaged, &rec); ok {
			t.Fatalf("aceptó el registro con el byte %d cambiado", i)
		}
	}
	if _, ok := decodeRecord(buf[:len(buf)-1], &rec); ok {
		t.Fatal("aceptó un registro cortado")
	}
	if _, ok := decodeRecord(encoded(t, testRecord(5), 0), &rec); ok {
		t.Fatal("aceptó lsn 0")
	}
}

// TestImage: la imagen sin hueco se restaura igual a la página; una página en ceros o sin hueco válido va entera;
// una que no comprime va sin comprimir; las imágenes dañadas se rechazan.
func TestImage(t *testing.T) {
	random := make(page.Page, page.Size)
	for i := range random {
		random[i] = byte(rand.IntN(256))
	}
	random[page.OffLower], random[page.OffLower+1] = 0xFF, 0xFF // hueco inválido
	pages := []page.Page{testPage(200, 6000, 1), testPage(page.HeaderSize, page.Size, 2), make(page.Page, page.Size), random}
	for i, p := range pages {
		var b BlockRef
		if err := b.SetImage(p); err != nil {
			t.Fatal(err)
		}
		if i == 0 && (b.Flags&BlockImageCompressed == 0 || len(b.Image) >= page.Size-5800) {
			t.Fatalf("página con hueco: %d bytes, flags %b", len(b.Image), b.Flags)
		}
		if i == 3 && (b.Flags&BlockImageCompressed != 0 || len(b.Image) != page.Size) {
			t.Fatalf("página al azar: %d bytes, flags %b", len(b.Image), b.Flags)
		}
		got := make(page.Page, page.Size)
		for j := range got {
			got[j] = 0xEE // lo que había en el búfer no debe quedar
		}
		if err := b.RestoreImage(got); err != nil || !bytes.Equal(got, p) {
			t.Fatalf("página %d restaurada distinta: %v", i, err)
		}
	}

	img, flags, _ := MakeImage(testPage(200, 6000, 1))
	dst := make(page.Page, page.Size)
	for _, c := range []struct {
		img   []byte
		flags BlockFlags
	}{
		{img[:len(img)/2], flags},
		{nil, flags},
		{img, BlockHasImage}, // comprimida leída como si no
		{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, flags},
	} {
		if err := RestoreImage(c.img, c.flags, dst); err == nil {
			t.Fatalf("restauró una imagen dañada de %d bytes", len(c.img))
		}
	}
}

// TestWriteAndReopen: lo escrito sigue al reabrir, en orden y con lsn contiguos, y se sigue escribiendo a
// continuación.
func TestWriteAndReopen(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	if end != Start || len(recs) != 0 {
		t.Fatalf("WAL nuevo: %+v, %d registros", end, len(recs))
	}
	writeHistory(t, w, 500)
	if w.Published() != 500 {
		t.Fatalf("published %d", w.Published())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if w.Durable() != 500 {
		t.Fatalf("durable tras Close %d", w.Durable())
	}

	w, end, recs = openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 500)
	if end.LSN != 501 || len(segments(t, dir)) < 2 {
		t.Fatalf("fin %+v, segmentos %v", end, segments(t, dir))
	}
	if lsn := appendRecord(t, w, testRecord(501)); lsn != 501 {
		t.Fatalf("lsn al seguir: %d", lsn)
	}
	w.Close()
	_, _, recs = openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 501)
}

// TestConcurrentWriters: 16 goroutines reservan y escriben a la vez con rotación: no se pierde ninguno, published
// llega al último y al reabrir están todos, con lsn contiguos.
func TestConcurrentWriters(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	const writers, each = 16, 200
	var mu sync.Mutex
	byLSN := map[page.LSN]Record{}
	var wg sync.WaitGroup
	for g := range writers {
		wg.Go(func() {
			for i := range each {
				rec := testRecord(g*each + i + 1)
				r, err := w.Reserve(&rec, nil)
				if err != nil {
					t.Error(err)
					return
				}
				if err := r.Write(); err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				byLSN[r.LSN()] = rec
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if w.Published() != writers*each {
		t.Fatalf("published %d", w.Published())
	}
	if n := len(segments(t, dir)); n < 3 {
		t.Fatalf("no rotó: %d segmentos", n)
	}
	w.Close()

	_, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	if len(recs) != writers*each || end.LSN != writers*each+1 {
		t.Fatalf("al reabrir: %d registros, fin %+v", len(recs), end)
	}
	for i := range recs {
		want := byLSN[recs[i].LSN]
		if recs[i].LSN != page.LSN(i+1) || !sameRecord(&recs[i], &want) {
			t.Fatalf("registro %d distinto", i+1)
		}
	}
}

// TestRecoverTruncatesTail: un registro a medio escribir al final se descarta y se corta; se sigue escribiendo.
func TestRecoverTruncatesTail(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	writeHistory(t, w, 2)
	w.Close()

	path := segPath(dir, 1)
	valid := fileSize(t, path)
	partial := encoded(t, testRecord(3), 3)
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	f.Write(partial[:len(partial)-2])
	f.Close()

	w, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 2)
	if end != (Position{Segment: 1, Offset: valid, LSN: 3}) || fileSize(t, path) != valid {
		t.Fatalf("fin %+v, archivo %d, válido %d", end, fileSize(t, path), valid)
	}
	if lsn := appendRecord(t, w, testRecord(3)); lsn != 3 {
		t.Fatalf("lsn %d", lsn)
	}
}

// TestRecoverStopsAtHole: un hueco (una reserva que no llegó al disco) seguido de registros completos: lo que sigue
// al hueco nunca se confirmó y se descarta.
func TestRecoverStopsAtHole(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	appendRecord(t, w, testRecord(1))
	hole := w.active.size
	appendRecord(t, w, testRecord(2))
	holeEnd := w.active.size
	appendRecord(t, w, testRecord(3))
	w.Close()

	f, _ := os.OpenFile(segPath(dir, 1), os.O_WRONLY, 0o644)
	f.WriteAt(make([]byte, holeEnd-hole), hole)
	f.Close()

	_, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 1)
	if end.Offset != hole || fileSize(t, segPath(dir, 1)) != hole {
		t.Fatalf("no se cortó en el hueco: %+v", end)
	}
}

// TestRecoverOutOfOrderLSN: la copia de un registro viejo con crc válido al final no pasa: su lsn no es el siguiente.
func TestRecoverOutOfOrderLSN(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	writeHistory(t, w, 3)
	w.Close()
	f, _ := os.OpenFile(segPath(dir, 1), os.O_WRONLY|os.O_APPEND, 0o644)
	f.Write(encoded(t, testRecord(2), 2))
	f.Close()

	_, end, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 3)
	if end.LSN != 4 {
		t.Fatalf("fin %+v", end)
	}
}

// TestRecoverCorruptSealedFails: un registro dañado en un segmento sellado deja un hueco antes de registros
// confirmados: Open falla con ErrCorrupted en vez de perder datos en silencio.
func TestRecoverCorruptSealedFails(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	writeHistory(t, w, 200)
	w.Close()
	if len(segments(t, dir)) < 3 {
		t.Fatal("hacen falta varios segmentos")
	}

	f, _ := os.OpenFile(segPath(dir, 1), os.O_WRONLY, 0o644)
	f.WriteAt([]byte{0xFF}, 2000) // un byte cambiado en medio del primer segmento
	f.Close()
	if _, _, err := Open(fsys, dir, smallOptions(0), Start, nil); !errors.Is(err, ErrCorrupted) {
		t.Fatalf("abrió un segmento sellado dañado: %v", err)
	}
}

// TestGroupCommit: 64 escritores que esperan durabilidad comparten los fsync, y todo está al reabrir.
func TestGroupCommit(t *testing.T) {
	dir, fsys := t.TempDir(), &syncCountFS{FS: vfs.OS()}
	w, _, _ := openTest(t, fsys, dir, Options{SegmentSize: 1 << 20}, Start)
	const writers, each = 64, 20
	before := fsys.syncs.Load()
	var wg sync.WaitGroup
	for g := range writers {
		wg.Go(func() {
			for i := range each {
				rec := testRecord(g*each + i + 1)
				r, err := w.Reserve(&rec, nil)
				if err == nil {
					err = r.Write()
				}
				if err == nil {
					err = w.WaitDurable(r.LSN())
				}
				if err != nil {
					t.Error(err)
					return
				}
				if w.Durable() < r.LSN() {
					t.Errorf("durable %d < %d", w.Durable(), r.LSN())
				}
			}
		})
	}
	wg.Wait()
	fsyncs := fsys.syncs.Load() - before
	if fsyncs >= writers*each/2 {
		t.Fatalf("%d fsync para %d escrituras: no se comparten", fsyncs, writers*each)
	}
	t.Logf("%d escrituras con %d fsync", writers*each, fsyncs)
	w.Close()
	_, _, recs := openTest(t, fsys, dir, Options{SegmentSize: 1 << 20}, Start)
	if len(recs) != writers*each {
		t.Fatalf("al reabrir: %d", len(recs))
	}
}

// TestWriteFailure: una escritura fallida deja el WAL inutilizable: published no pasa el hueco, quien espera un lsn
// posterior se entera, y no se aceptan más reservas.
func TestWriteFailure(t *testing.T) {
	dir, fsys := "/db", crashfs.New()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	appendRecord(t, w, testRecord(1))

	rec2, rec3 := testRecord(2), testRecord(3)
	r2, _ := w.Reserve(&rec2, nil)
	r3, _ := w.Reserve(&rec3, nil)
	if err := r3.Write(); err != nil {
		t.Fatal(err)
	}
	fsys.FailAfter(0)
	if err := r2.Write(); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("Write con el disco fallando: %v", err)
	}
	fsys.FailAfter(-1)
	if w.Published() != 1 || !errors.Is(w.Failed(), crashfs.ErrIO) {
		t.Fatalf("published %d, failed %v", w.Published(), w.Failed())
	}
	if err := w.WaitDurable(3); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("WaitDurable tras el hueco: %v", err)
	}
	rec4 := testRecord(4)
	if _, err := w.Reserve(&rec4, nil); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("Reserve tras el fallo: %v", err)
	}
	if _, err := w.BeginCheckpoint(nil); err == nil {
		t.Fatal("checkpoint tras el fallo")
	}
	w.Close()

	// Al reabrir, lo posterior al hueco se descarta.
	_, _, recs := openTest(t, fsys, dir, smallOptions(0), Start)
	checkHistory(t, recs, 1)
}

// TestSyncFailure: un fsync fallido deja el WAL inutilizable y no se reintenta, aunque el disco vuelva a andar.
func TestSyncFailure(t *testing.T) {
	disk := crashfs.New()
	dir, fsys := "/db", &syncCountFS{FS: disk}
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	lsn := appendRecord(t, w, testRecord(1))
	disk.FailSyncAfter(0)
	if err := w.WaitDurable(lsn); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("WaitDurable con fsync fallando: %v", err)
	}
	disk.FailSyncAfter(-1)
	before := fsys.syncs.Load()
	if err := w.WaitDurable(lsn); err == nil || w.Durable() >= lsn {
		t.Fatalf("WaitDurable reintentó: %v, durable %d", err, w.Durable())
	}
	if fsys.syncs.Load() != before {
		t.Fatal("hizo otro fsync tras uno fallido")
	}
}

// TestNeedImage: un checkpoint que empieza entre que se arma el registro y se reserva obliga a traer la imagen:
// ErrNeedImage sin reservar nada; con la imagen, reserva. BlockWillInit y las páginas ya escritas tras el punto de
// rehacer no la necesitan.
func TestNeedImage(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	writeHistory(t, w, 5)

	// La página se escribió por última vez en el lsn 3. Antes de que empiece el checkpoint, no hace falta imagen.
	leaf := func() Record {
		return Record{Kind: KindLeafWrite, XID: 1, Blocks: []BlockRef{{File: 1, Block: 7, PageLSN: 3, Delta: []byte("d")}}}
	}
	rec := leaf()
	r, err := w.Reserve(&rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Write()

	// Empieza un checkpoint: el punto de rehacer pasa al lsn 7 y la página (lsn 3) necesita su imagen.
	redo, err := w.BeginCheckpoint(nil)
	if err != nil || redo.LSN != 7 || w.RedoPoint() != redo {
		t.Fatalf("redo %+v, %v", redo, err)
	}
	rec = leaf()
	if _, err := w.Reserve(&rec, nil); !errors.Is(err, ErrNeedImage) {
		t.Fatalf("sin imagen tras el checkpoint: %v", err)
	}
	if w.lastLSN.Load() != 6 || w.active.size != redo.Offset {
		t.Fatal("ErrNeedImage reservó algo")
	}
	if err := rec.Blocks[0].SetImage(testPage(100, 7000, 3)); err != nil {
		t.Fatal(err)
	}
	if r, err = w.Reserve(&rec, nil); err != nil || r.LSN() != 7 {
		t.Fatalf("con imagen: %v", err)
	}
	r.Write()

	for _, b := range []BlockRef{{Block: 8, PageLSN: 2, Flags: BlockWillInit}, {Block: 9, PageLSN: 7}} {
		rec := Record{Kind: KindLeafWrite, Blocks: []BlockRef{b}}
		r, err := w.Reserve(&rec, nil)
		if err != nil {
			t.Fatalf("bloque %d: %v", b.Block, err)
		}
		r.Write()
	}
}

// TestUnderLock: underLock recibe el lsn bajo writeMu, en orden de reserva.
func TestUnderLock(t *testing.T) {
	w, _, _ := openTest(t, vfs.OS(), t.TempDir(), smallOptions(0), Start)
	var seen []page.LSN
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			rec := testRecord(i + 1)
			r, err := w.Reserve(&rec, func(lsn page.LSN) { seen = append(seen, lsn) }) // sin lock propio: writeMu
			if err == nil {
				err = r.Write()
			}
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for i, lsn := range seen {
		if lsn != page.LSN(i+1) {
			t.Fatalf("underLock fuera de orden: %v", seen)
		}
	}
}

// TestCheckpointAcrossSegments: un punto de rehacer al final de un segmento lleno sigue en el siguiente; recuperar
// desde ahí lee solo lo posterior, y SizeSince cuenta los bytes desde el punto de rehacer.
func TestCheckpointAcrossSegments(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	w, _, _ := openTest(t, fsys, dir, smallOptions(0), Start)
	i := 0
	for w.active.size+int64(len(encoded(t, testRecord(i+1), 1))) <= MaxRecordSize { // llenar el primer segmento
		i++
		appendRecord(t, w, testRecord(i))
	}
	redo, _ := w.BeginCheckpoint(nil)
	if redo.Segment != 1 || w.SizeSince(redo) != 0 {
		t.Fatalf("redo %+v, size %d", redo, w.SizeSince(redo))
	}
	writeHistoryFrom(t, w, i+1, i+30)
	if w.active.num != 2 {
		t.Fatalf("no rotó: activo %d", w.active.num)
	}
	want := w.sizes[1].Offset - redo.Offset + w.active.size
	if got := w.SizeSince(redo); got != want || got <= 0 {
		t.Fatalf("SizeSince %d, se esperaba %d", got, want)
	}
	if w.SizeSince(Start) != w.sizes[1].Offset+w.active.size {
		t.Fatalf("SizeSince(Start) %d", w.SizeSince(Start))
	}
	w.Close()

	_, end, recs := openTest(t, fsys, dir, smallOptions(0), redo)
	if len(recs) != 30 || recs[0].LSN != redo.LSN || end.LSN != page.LSN(i+31) {
		t.Fatalf("desde el redo: %d registros, primero %d, fin %+v", len(recs), recs[0].LSN, end)
	}

	recs = nil
	end2, err := NewReader(fsys, dir).Scan(Start, func(rec *Record) error {
		recs = append(recs, cloneRecord(rec))
		return nil
	})
	if err != nil || end2 != end {
		t.Fatalf("Reader: %+v %v", end2, err)
	}
	checkHistory(t, recs, i+30)
}

// writeHistoryFrom escribe los registros from..to de testRecord.
func writeHistoryFrom(t testing.TB, w *WAL, from, to int) {
	t.Helper()
	for i := from; i <= to; i++ {
		appendRecord(t, w, testRecord(i))
	}
}

// TestRecycle: los segmentos anteriores al punto de rehacer se reciclan (hasta Spares) o se borran; al rotar se
// reutilizan, y su contenido viejo, con crc válido, no pasa por registros nuevos al recuperar.
func TestRecycle(t *testing.T) {
	dir, fsys := t.TempDir(), vfs.OS()
	opts := smallOptions(2)
	w, _, _ := openTest(t, fsys, dir, opts, Start)
	writeHistory(t, w, 400)
	before := segments(t, dir)
	if len(before) < 5 {
		t.Fatalf("hacen falta varios segmentos: %v", before)
	}
	active := w.active.num
	redo, _ := w.BeginCheckpoint(nil)
	if err := w.Recycle(redo); err != nil {
		t.Fatal(err)
	}
	after := segments(t, dir)
	if after[0] != active || after[len(after)-1] != active+2 {
		t.Fatalf("tras reciclar: %v (activo %d)", after, active)
	}
	if fileSize(t, segPath(dir, active+1)) == 0 {
		t.Fatal("el repuesto no conserva su contenido viejo")
	}
	if err := w.Recycle(Position{Segment: active + 5}); err == nil {
		t.Fatal("recicló más allá del activo")
	}

	// Un registro en el repuesto y se corta la luz: lo viejo detrás no cuenta.
	n := 400
	for w.active.num == active {
		n++
		appendRecord(t, w, testRecord(n))
	}
	w.Close()
	_, end, recs := openTest(t, fsys, dir, opts, redo)
	if len(recs) != n-400 || end.LSN != page.LSN(n+1) || end.Segment != active+1 {
		t.Fatalf("tras reutilizar el repuesto: %d registros, fin %+v", len(recs), end)
	}
	if fileSize(t, segPath(dir, active+2)) != 0 {
		t.Fatal("no se cortaron los segmentos después del final")
	}
	if _, _, err := Open(fsys, dir, opts, Start, nil); err == nil {
		t.Fatal("abrió desde un segmento reciclado")
	}
}

// TestOpenRejects: sin segmentos solo se abre desde Start; un segmento demasiado chico no se acepta; un lsn nunca
// reservado no se espera; Reserve tras Close devuelve ErrClosed y cerrar dos veces no hace nada.
func TestOpenRejects(t *testing.T) {
	fsys := vfs.OS()
	if _, _, err := Open(fsys, t.TempDir(), smallOptions(0), Position{Segment: 3, LSN: 50}, nil); err == nil {
		t.Fatal("abrió un WAL perdido")
	}
	if _, _, err := Open(fsys, t.TempDir(), Options{SegmentSize: 1024}, Start, nil); err == nil {
		t.Fatal("aceptó un segmento chico")
	}
	w, _, _ := openTest(t, fsys, t.TempDir(), smallOptions(0), Start)
	if err := w.WaitDurable(10); err == nil {
		t.Fatal("esperó un lsn nunca reservado")
	}
	w.Close()
	rec := testRecord(1)
	if _, err := w.Reserve(&rec, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("Reserve tras Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
