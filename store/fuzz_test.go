package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Fuzzing (pruebas con datos al azar de `go test -fuzz`) del formato de registro y de la recuperación (F-23). Las
// semillas corren como pruebas normales con `go test ./store`; para fuzzear de verdad:
//
//	go test ./store -run '^$' -fuzz FuzzScanSegment -fuzztime 2m

/**
* validSeed: Un registro completo y sellado, para las semillas.
* @param k kind, id string, data []byte, lsn uint64
* @return []byte
**/
func validSeed(k kind, id string, data []byte, lsn uint64) []byte {
	buf := encodeRecord(k, id, data)
	sealRecord(buf, lsn)
	return buf
}

/**
* recordSeeds: Los casos reales que se dan a los fuzzers de bytes: registros válidos, cortados a la mitad, en ceros,
* con el crc cambiado, con idLen 0 y con dataLen máximo (y uno más).
* @return [][]byte
**/
func recordSeeds() [][]byte {
	put := validSeed(kindPut, "k01", []byte("valor"), 7)
	del := validSeed(kindDelete, "k01", nil, 8)
	badCRC := bytes.Clone(put)
	badCRC[0] ^= 0xFF
	noID := validSeed(kindPut, "", []byte("x"), 9)
	maxData := bytes.Clone(put[:headerSize])
	binary.BigEndian.PutUint32(maxData[15:19], maxDataLen)
	overData := bytes.Clone(put[:headerSize])
	binary.BigEndian.PutUint32(overData[15:19], math.MaxUint32)
	return [][]byte{
		put,
		append(bytes.Clone(put), del...),
		put[:len(put)/2],
		put[:headerSize-1],
		make([]byte, headerSize),
		make([]byte, 3*len(put)),
		badCRC,
		append(bytes.Clone(put), badCRC...),
		noID,
		maxData,
		append(bytes.Clone(put), maxData...),
		overData,
	}
}

/**
* decodeNext: El modelo del recorrido de scan sobre un slice: decodifica el registro que empieza en buf, con las
* mismas reglas (idLen 0 o dataLen mayor que maxDataLen detienen) pero sin asignar nada: lo que no cabe en buf es un
* registro incompleto.
* @param buf []byte
* @return header, bool (si hay un registro completo y válido)
**/
func decodeNext(buf []byte) (header, bool) {
	if len(buf) < headerSize {
		return header{}, false
	}
	h := decodeHeader(buf)
	if h.idLen == 0 || h.dataLen > maxDataLen || h.size() > len(buf) {
		return header{}, false
	}
	return h, validRecord(h, buf[:h.size()])
}

/**
* modelScan: Los registros válidos del principio de buf según decodeNext, y dónde termina la parte válida.
* @param buf []byte
* @return []header, int
**/
func modelScan(buf []byte) ([]header, int) {
	var recs []header
	off := 0
	for {
		h, ok := decodeNext(buf[off:])
		if !ok {
			return recs, off
		}
		recs = append(recs, h)
		off += h.size()
	}
}

/**
* checkRoundTrip: Un registro válido vuelve a codificarse byte a byte igual con encodeRecord y sealRecord.
* @param t *testing.T, h header, rec []byte (el registro completo)
**/
func checkRoundTrip(t *testing.T, h header, rec []byte) {
	t.Helper()
	again := encodeRecord(h.kind, recordID(h, rec), recordData(h, rec))
	sealRecord(again, h.lsn)
	if !bytes.Equal(again, rec) {
		t.Fatalf("ida y vuelta distinta:\n%x\n%x", rec, again)
	}
}

/**
* FuzzRecordRoundTrip: Todo registro que se puede escribir se decodifica con los mismos campos y pasa la validación;
* cambiar un byte cualquiera lo invalida (el crc lo cubre todo menos a sí mismo).
**/
func FuzzRecordRoundTrip(f *testing.F) {
	f.Add(true, "k01", []byte("valor"), uint64(1), uint16(0))
	f.Add(false, "k01", []byte{}, uint64(math.MaxUint64), uint16(5))
	f.Add(true, "x", []byte{}, uint64(0), uint16(30))
	f.Add(true, "", []byte("sin id"), uint64(3), uint16(1))
	f.Fuzz(func(t *testing.T, put bool, id string, data []byte, lsn uint64, flip uint16) {
		k := kindPut
		if !put {
			k, data = kindDelete, nil
		}
		if checkPut(id, data) != nil {
			return // Put y Remove lo rechazan antes de codificar
		}
		rec := validSeed(k, id, data, lsn)
		h := decodeHeader(rec)
		if h.size() != len(rec) || h.kind != k || h.lsn != lsn || recordID(h, rec) != id ||
			!bytes.Equal(recordData(h, rec), data) {
			t.Fatalf("campos distintos: %+v", h)
		}
		if validRecord(h, rec) != (lsn != 0) {
			t.Fatalf("validRecord = %v con lsn %d", !(lsn != 0), lsn)
		}
		bad := bytes.Clone(rec)
		bad[int(flip)%len(bad)] ^= 0x01
		if validRecord(decodeHeader(bad), bad) {
			t.Fatalf("un byte cambiado en %d sigue válido", int(flip)%len(bad))
		}
	})
}

/**
* FuzzDecodeRecord: Con bytes arbitrarios, decodificar y validar nunca entra en pánico ni se sale de buf, y lo que
* pasa la validación tiene un tipo conocido, lsn distinto de cero e ida y vuelta exacta.
**/
func FuzzDecodeRecord(f *testing.F) {
	for _, seed := range recordSeeds() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		recs, end := modelScan(buf)
		if end > len(buf) {
			t.Fatalf("fin %d fuera de %d bytes", end, len(buf))
		}
		off := 0
		for _, h := range recs {
			rec := buf[off : off+h.size()]
			if h.lsn == 0 || (h.kind != kindPut && h.kind != kindDelete) || (h.kind == kindDelete && h.dataLen != 0) {
				t.Fatalf("pasó la validación un registro imposible: %+v", h)
			}
			checkRoundTrip(t, h, rec)
			off += h.size()
		}
	})
}

/**
* scanFile: Escribe buf en un segmento temporal y lo recorre con scan, como Open.
* @param t *testing.T, buf []byte
* @return []scanned, int64 (fin), bool (reached), uint64 (bytes asignados por scan)
**/
func scanFile(t *testing.T, buf []byte) ([]scanned, int64, bool, uint64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), segmentName(1))
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var recs []scanned
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	end, reached, err := (&segment{num: 1, path: path, file: file}).scan(func(rec scanned) bool {
		recs = append(recs, rec)
		return true
	})
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return recs, end, reached, after.TotalAlloc - before.TotalAlloc
}

/**
* FuzzScanSegment: scan sobre un archivo con bytes arbitrarios da lo mismo que el modelo (modelScan): los mismos
* registros, el mismo fin, y llegó al final solo si todo era válido. Las refs caen dentro del archivo y no asigna
* mucho más que el archivo (un dataLen enorme no debe reservar memoria antes de validar).
**/
func FuzzScanSegment(f *testing.F) {
	for _, seed := range recordSeeds() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		recs, end, reached, allocated := scanFile(t, buf)
		want, wantEnd := modelScan(buf)
		if end != int64(wantEnd) || len(recs) != len(want) || reached != (wantEnd == len(buf)) {
			t.Fatalf("scan: fin %d, %d registros, reached %v; modelo: fin %d, %d registros",
				end, len(recs), reached, wantEnd, len(want))
		}
		for i, rec := range recs {
			at := rec.ref
			if at.offset < 0 || at.offset+int64(at.size) > int64(len(buf)) || at.lsn != want[i].lsn ||
				at.deleted != (want[i].kind == kindDelete) || int(at.size) != want[i].size() {
				t.Fatalf("ref %d %+v no coincide con %+v", i, at, want[i])
			}
			if got := recordID(want[i], buf[at.offset:]); got != rec.id {
				t.Fatalf("id %q, en disco %q", rec.id, got)
			}
		}
		if limit := uint64(1<<20+64<<10) + 2*uint64(len(buf)); allocated > limit {
			t.Fatalf("scan asignó %d bytes para un archivo de %d", allocated, len(buf))
		}
	})
}

// fuzzSegmentSize: segmentos chicos, para que la historia ocupe varios y haya sellados que dañar.
const fuzzSegmentSize = 160

// fuzzOp: una escritura de la historia que se recupera.
type fuzzOp struct {
	id   string
	data []byte
	del  bool
}

/**
* fuzzHistory: La historia fija que FuzzRecoverSegment escribe: guardados, reemplazos y borrados de pocos ids, en
* varios segmentos. El lsn de la escritura i es i+1.
* @return []fuzzOp
**/
func fuzzHistory() []fuzzOp {
	var ops []fuzzOp
	for i := range 18 {
		id := fmt.Sprintf("k%d", i%5)
		if i%7 == 6 {
			ops = append(ops, fuzzOp{id: id, del: true})
			continue
		}
		ops = append(ops, fuzzOp{id: id, data: fmt.Appendf(nil, "v%02d-%s", i, bytes.Repeat([]byte{'x'}, i%4))})
	}
	return ops
}

/**
* stateAfter: El estado de los ids tras las primeras n escrituras de ops.
* @param ops []fuzzOp, n int
* @return map[string]string
**/
func stateAfter(ops []fuzzOp, n int) map[string]string {
	state := map[string]string{}
	for _, op := range ops[:n] {
		if op.del {
			delete(state, op.id)
		} else {
			state[op.id] = string(op.data)
		}
	}
	return state
}

/**
* writeHistory: Escribe ops en un store nuevo en dir y lo cierra.
* @param t testing.TB, dir string, ops []fuzzOp
**/
func writeHistory(t testing.TB, dir string, ops []fuzzOp) {
	t.Helper()
	s, err := open(dir, testOptions(fuzzSegmentSize, false))
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		if op.del {
			err = s.Remove(op.id)
		} else {
			err = s.Put(op.id, op.data)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

/**
* recordEnds: Dónde termina cada registro de dir, en orden de lsn: segmento y fin dentro de él.
* @param t testing.TB, dir string
* @return []ref (segment y offset = fin del registro)
**/
func recordEnds(t testing.TB, dir string) []ref {
	t.Helper()
	nums, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}
	var ends []ref
	for _, num := range nums {
		file, err := os.Open(filepath.Join(dir, segmentName(num)))
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = (&segment{num: num, file: file}).scan(func(rec scanned) bool {
			ends = append(ends, ref{segment: num, offset: rec.ref.offset + int64(rec.ref.size), lsn: rec.ref.lsn})
			return true
		})
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	return ends
}

// damage: dónde y cómo se dañó un segmento.
type damage struct {
	segment int   // número del segmento dañado
	offset  int64 // desde dónde
	sealed  bool  // si no es el último
	changed bool  // si algún byte cambió de verdad
}

/**
* corrupt: Aplica bytes del fuzzer a dir: al final del último segmento (mode par) o encima, desde pos, de uno de los
* segmentos elegido por which (mode impar; puede alargarlo).
* @param t *testing.T, dir string, junk []byte, mode, which uint8, pos uint16
* @return damage
**/
func corrupt(t *testing.T, dir string, junk []byte, mode, which uint8, pos uint16) damage {
	t.Helper()
	nums, err := listSegments(dir)
	if err != nil {
		t.Fatal(err)
	}
	num := nums[len(nums)-1]
	if mode%2 == 1 {
		num = nums[int(which)%len(nums)]
	}
	path := filepath.Join(dir, segmentName(num))
	old, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	at := int64(len(old))
	if mode%2 == 1 && len(old) > 0 {
		at = int64(pos) % int64(len(old))
	}
	updated := append([]byte{}, old...)
	if need := int(at) + len(junk); need > len(updated) {
		updated = append(updated, make([]byte, need-len(updated))...)
	}
	copy(updated[at:], junk)
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		t.Fatal(err)
	}
	return damage{segment: num, offset: at, sealed: num != nums[len(nums)-1], changed: !bytes.Equal(old, updated)}
}

/**
* copyStore: Copia los segmentos de src a un directorio nuevo (sin el LOCK): cada ejecución daña su propia copia.
* @param t *testing.T, src string
* @return string
**/
func copyStore(t *testing.T, src string) string {
	t.Helper()
	nums, err := listSegments(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, num := range nums {
		data, err := os.ReadFile(filepath.Join(src, segmentName(num)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, segmentName(num)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

/**
* survivors: Cuántas escrituras quedan enteras antes del daño (las que deben sobrevivir a la recuperación).
* @param ends []ref, d damage
* @return int
**/
func survivors(ends []ref, d damage) int {
	n := 0
	for _, e := range ends {
		if e.segment > d.segment || (e.segment == d.segment && e.offset > d.offset) {
			break
		}
		n++
	}
	return n
}

/**
* checkState: El store tiene exactamente want: cada id con su valor y ningún otro en el índice.
* @param t *testing.T, s *Store, want map[string]string, ids []string
**/
func checkState(t *testing.T, s *Store, want map[string]string, ids []string) {
	t.Helper()
	if n := s.index.entries(); n != len(want) {
		t.Fatalf("el índice tiene %d ids, se esperaban %d", n, len(want))
	}
	for _, id := range ids {
		got, ok, err := s.Read(id)
		if err != nil {
			t.Fatalf("Read(%s): %v", id, err)
		}
		if v, exists := want[id]; ok != exists || string(got) != v {
			t.Fatalf("%s = %q (%v), se esperaba %q (%v)", id, got, ok, v, exists)
		}
	}
}

/**
* forgesRecord: Los bytes del fuzzer empiezan con un registro válido (lo traen las semillas: el fuzzer no adivina un
* crc) y se escriben encima de un segmento (mode impar). Si cae justo en el lugar de un registro con su mismo lsn, el
* store lo acepta con razón (crc y lsn válidos) y la historia deja de ser la de ops. Anexado al final (mode par) no
* hay excepción: su lsn (7 a 9) no es el siguiente del log y la regla de lsn contiguos lo corta.
* @param junk []byte, mode uint8
* @return bool
**/
func forgesRecord(junk []byte, mode uint8) bool {
	_, ok := decodeNext(junk)
	return ok && mode%2 == 1
}

/**
* FuzzRecoverSegment: Escribe una historia fija en varios segmentos, la daña con bytes del fuzzer y reabre. Open
* nunca entra en pánico; falla si y solo si se dañó un segmento sellado; si abre, el estado es el de un prefijo de la
* historia (ningún id inventado ni valor distinto, nada posterior a un hueco), no se pierde nada anterior al daño, y
* se puede seguir escribiendo y reabrir.
**/
func FuzzRecoverSegment(f *testing.F) {
	for i, seed := range recordSeeds() {
		f.Add(seed, uint8(i), uint8(i), uint16(i*37))
	}
	f.Add([]byte{0xFF}, uint8(1), uint8(0), uint16(headerSize+1))
	f.Add(make([]byte, 64), uint8(1), uint8(9), uint16(40))
	// La historia se escribe una vez por proceso (cada escritura cuesta fsyncs); cada ejecución daña una copia.
	ops := fuzzHistory()
	template := f.TempDir()
	writeHistory(f, template, ops)
	ends := recordEnds(f, template)
	if len(ends) != len(ops) {
		f.Fatalf("historia: %d registros en disco de %d", len(ends), len(ops))
	}
	f.Fuzz(func(t *testing.T, junk []byte, mode, which uint8, pos uint16) {
		if len(junk) > 4096 {
			return
		}
		dir := copyStore(t, template)
		d := corrupt(t, dir, junk, mode, which, pos)

		s, err := open(dir, testOptions(fuzzSegmentSize, false))
		if forgesRecord(junk, mode) {
			if err == nil {
				s.Close()
			}
			return // solo se exige que Open no entre en pánico
		}
		if err != nil {
			if !d.sealed || !d.changed {
				t.Fatalf("Open falló sin dañar un segmento sellado (%+v): %v", d, err)
			}
			return
		}
		defer s.Close()
		if d.sealed && d.changed {
			t.Fatalf("Open aceptó un segmento sellado dañado (%+v)", d)
		}
		recovered := int(s.lastLSN)
		if floor := survivors(ends, d); recovered < floor || recovered > len(ops) ||
			(!d.changed && recovered != len(ops)) {
			t.Fatalf("recuperó %d de %d escrituras; antes del daño había %d (%+v)", recovered, len(ops), floor, d)
		}

		ids := []string{"k0", "k1", "k2", "k3", "k4", "nuevo"}
		want := stateAfter(ops, recovered)
		checkState(t, s, want, ids)
		writeAfter(t, s, want)
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}

		s, err = open(dir, testOptions(fuzzSegmentSize, false))
		if err != nil {
			t.Fatalf("no reabrió después de seguir escribiendo: %v", err)
		}
		defer s.Close()
		checkState(t, s, want, ids)
		if s.lastLSN != uint64(recovered+2) {
			t.Fatalf("lsn al reabrir %d, se esperaba %d", s.lastLSN, recovered+2)
		}
	})
}

/**
* writeAfter: Sigue escribiendo tras recuperar (un Put nuevo y un Remove) y anota el efecto en want.
* @param t *testing.T, s *Store, want map[string]string
**/
func writeAfter(t *testing.T, s *Store, want map[string]string) {
	t.Helper()
	if err := s.Put("nuevo", []byte("después")); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("k0"); err != nil {
		t.Fatal(err)
	}
	want["nuevo"] = "después"
	delete(want, "k0")
}
