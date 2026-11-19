package control

import (
	"errors"
	"hash/crc32"
	"strings"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// sampleControl devuelve un control con todos los campos distintos de cero; n cambia los valores.
func sampleControl(n int64) Control {
	c := Control{
		State:           StateInProduction,
		CheckpointTime:  1_700_000_000_000_000_000 + n,
		Redo:            wal.Position{Segment: 7 + uint64(n), Offset: 4096 + n, LSN: page.LSN(1000 + n)},
		CheckpointLSN:   page.LSN(1010 + n),
		NextXID:         page.XID(500 + n),
		OldestActiveXID: page.XID(490 + n),
		FrozenXID:       page.XID(3),
	}
	c.Slots[0] = Slot{Name: "replica-1", RestartLSN: 900}
	c.Slots[7] = Slot{Name: strings.Repeat("z", SlotNameLen), RestartLSN: 1}
	return c
}

// newControl crea un control en un directorio temporal y devuelve el directorio y el ControlFile.
func newControl(t *testing.T) (string, ControlFile) {
	t.Helper()
	dir := t.TempDir()
	f := NewControlFile(vfs.OS(), dir)
	if err := f.Create(sampleControl(0)); err != nil {
		t.Fatal(err)
	}
	return dir, f
}

// mustRead lee el control con un ControlFile nuevo (como un Open tras un corte).
func mustRead(t *testing.T, dir string) Control {
	t.Helper()
	c, err := NewControlFile(vfs.OS(), dir).Read()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// withSeq devuelve c con la seq dada, para comparar.
func withSeq(c Control, seq uint64) Control {
	c.Seq = seq
	return c
}

func TestControlRoundTrip(t *testing.T) {
	c := withSeq(sampleControl(1), 42)
	b, err := encodeControl(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != CopySize {
		t.Fatalf("copy length %d", len(b))
	}
	got, err := decodeControl(b)
	if err != nil {
		t.Fatal(err)
	}
	if got != c {
		t.Fatalf("got %+v, want %+v", got, c)
	}
}

func TestControlCreateWriteRead(t *testing.T) {
	dir, f := newControl(t)
	if got := mustRead(t, dir); got != withSeq(sampleControl(0), 2) {
		t.Fatalf("after Create got %+v", got)
	}
	if size := len(readFile(t, controlPath(dir))); size != 2*CopySize {
		t.Fatalf("file size %d", size)
	}
	for seq := uint64(3); seq < 10; seq++ {
		if err := f.Write(sampleControl(int64(seq))); err != nil {
			t.Fatal(err)
		}
		if got := mustRead(t, dir); got != withSeq(sampleControl(int64(seq)), seq) {
			t.Fatalf("seq %d: got %+v", seq, got)
		}
	}

	// Un ControlFile nuevo escribe sin Read previo: aprende la seq vigente solo.
	if err := NewControlFile(vfs.OS(), dir).Write(sampleControl(99)); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, dir); got.Seq != 10 || got.Redo != sampleControl(99).Redo {
		t.Fatalf("got %+v", got)
	}
}

// TestControlEveryByteCorrupted altera cada byte del archivo: siempre queda la otra copia.
func TestControlEveryByteCorrupted(t *testing.T) {
	dir, f := newControl(t)
	if err := f.Write(sampleControl(3)); err != nil { // seq 3, en la copia B
		t.Fatal(err)
	}
	orig := readFile(t, controlPath(dir))
	newest := withSeq(sampleControl(3), 3)
	older := withSeq(sampleControl(0), 2)
	for i := range orig {
		for _, x := range []byte{0x01, 0x80, 0xff} {
			b := append([]byte(nil), orig...)
			b[i] ^= x
			writeFile(t, controlPath(dir), b)
			want := older // el byte cayó en la copia B, la más nueva
			if i < CopySize {
				want = newest
			}
			if got := mustRead(t, dir); got != want {
				t.Fatalf("byte %d ^%02x: got seq %d, want %d", i, x, got.Seq, want.Seq)
			}
		}
	}
}

func TestControlBothBroken(t *testing.T) {
	dir, _ := newControl(t)
	orig := readFile(t, controlPath(dir))
	cases := map[string][]byte{
		"both flipped": func() []byte {
			b := append([]byte(nil), orig...)
			b[100] ^= 1
			b[CopySize+100] ^= 1
			return b
		}(),
		"zeros":   make([]byte, 2*CopySize),
		"empty":   nil,
		"short":   orig[:CopySize-1],
		"garbage": []byte(strings.Repeat("basura!", 200)),
	}
	for name, b := range cases {
		writeFile(t, controlPath(dir), b)
		_, err := NewControlFile(vfs.OS(), dir).Read()
		if !errors.Is(err, ErrNoValidCopy) {
			t.Fatalf("%s: got %v, want ErrNoValidCopy", name, err)
		}
	}

	if _, err := NewControlFile(vfs.OS(), t.TempDir()).Read(); err == nil {
		t.Fatal("missing control: want an error")
	}
}

// TestControlShortFileKeepsCopyA: un archivo cortado en la copia B todavía tiene la A entera.
func TestControlShortFileKeepsCopyA(t *testing.T) {
	dir, _ := newControl(t)
	writeFile(t, controlPath(dir), readFile(t, controlPath(dir))[:CopySize+10])
	if got := mustRead(t, dir); got.Seq != 2 {
		t.Fatalf("got seq %d, want 2", got.Seq)
	}
}

// TestControlTornWrite corta la escritura de una copia en distintos bytes: se lee la anterior entera, y el
// ControlFile que falló no vuelve a escribir (escribiría sobre la única copia buena).
func TestControlTornWrite(t *testing.T) {
	for _, torn := range []int{0, 1, 8, 100, 300, CopySize - 4, CopySize - 1} {
		dir := t.TempDir()
		fs := newTornFS()
		f := NewControlFile(fs, dir)
		if err := f.Create(sampleControl(0)); err != nil {
			t.Fatal(err)
		}
		if err := f.Write(sampleControl(3)); err != nil {
			t.Fatal(err)
		}
		fs.arm(torn)
		if err := f.Write(sampleControl(4)); !errors.Is(err, errTorn) {
			t.Fatalf("torn %d: got %v, want errTorn", torn, err)
		}
		if got := mustRead(t, dir); got != withSeq(sampleControl(3), 3) {
			t.Fatalf("torn %d: got seq %d", torn, got.Seq)
		}
		if err := f.Write(sampleControl(5)); !errors.Is(err, ErrFailed) {
			t.Fatalf("torn %d: write after failure: got %v, want ErrFailed", torn, err)
		}
	}
}

// TestControlCutBetweenCopies: Create escribió la seq 1 y se cortó antes de la 2 (la A queda rota o vacía).
func TestControlCutBetweenCopies(t *testing.T) {
	dir, _ := newControl(t)
	b := readFile(t, controlPath(dir))
	copy(b[:CopySize], make([]byte, CopySize))
	writeFile(t, controlPath(dir), b)
	if got := mustRead(t, dir); got != withSeq(sampleControl(0), 1) {
		t.Fatalf("got %+v", got)
	}

	// Y la escritura siguiente (seq 2) va a la copia A, sin tocar la B.
	f := NewControlFile(vfs.OS(), dir)
	if err := f.Write(sampleControl(2)); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, dir); got != withSeq(sampleControl(2), 2) {
		t.Fatalf("got %+v", got)
	}
	if after := readFile(t, controlPath(dir)); string(after[CopySize:]) != string(b[CopySize:]) {
		t.Fatal("copy B changed")
	}
}

// TestControlSyncFails: un Sync fallido deja el ControlFile inutilizable, y tras un corte queda la copia anterior
// (crashfs olvida lo que no llegó a sincronizarse, como Linux).
func TestControlSyncFails(t *testing.T) {
	const dir = "/db"
	fs := newCutFS(t, dir)
	f := NewControlFile(fs, dir)
	if err := f.Create(sampleControl(0)); err != nil {
		t.Fatal(err)
	}
	fs.FailSyncAfter(0)
	if err := f.Write(sampleControl(1)); !errors.Is(err, crashfs.ErrIO) {
		t.Fatalf("got %v, want crashfs.ErrIO", err)
	}
	fs.FailSyncAfter(-1)
	if err := f.Write(sampleControl(1)); !errors.Is(err, ErrFailed) {
		t.Fatalf("got %v, want ErrFailed", err)
	}
	fs.Crash(crashfs.ReorderUnsynced, 1)
	if got, err := NewControlFile(fs, dir).Read(); err != nil || got != withSeq(sampleControl(0), 2) {
		t.Fatalf("after crash got %+v, %v", got, err)
	}
}

// TestControlCrashDuringWrite corta la luz justo antes del Sync de Write, en los tres modos: tras el corte se lee la
// copia anterior o la nueva entera (512 bytes, un sector), nunca un error; con LoseUnsynced, siempre la anterior. Lo
// que Write confirmó antes sobrevive.
func TestControlCrashDuringWrite(t *testing.T) {
	const dir = "/db"
	for name, mode := range crashModes {
		for seed := int64(0); seed < 20; seed++ {
			fs := newCutFS(t, dir)
			f := NewControlFile(fs, dir)
			if err := f.Create(sampleControl(0)); err != nil {
				t.Fatal(err)
			}
			if err := f.Write(sampleControl(3)); err != nil {
				t.Fatal(err)
			}
			fs.arm(0, mode, seed)
			if err := f.Write(sampleControl(4)); !errors.Is(err, errCut) {
				t.Fatalf("%s/%d: got %v, want errCut", name, seed, err)
			}
			got, err := NewControlFile(fs, dir).Read()
			if err != nil {
				t.Fatalf("%s/%d: read after crash: %v", name, seed, err)
			}
			old, next := withSeq(sampleControl(3), 3), withSeq(sampleControl(4), 4)
			if got != old && (got != next || mode == crashfs.LoseUnsynced) {
				t.Fatalf("%s/%d: got seq %d", name, seed, got.Seq)
			}
		}
	}
}

// TestControlCrashDuringCreate corta la luz en cada Sync de Create (el del archivo y el del directorio): tras el
// corte, o hay un control válido igual al creado, o no hay ninguno y un Create nuevo lo reemplaza.
func TestControlCrashDuringCreate(t *testing.T) {
	const dir = "/db"
	for name, mode := range crashModes {
		for cut := 0; cut < 2; cut++ {
			for seed := int64(0); seed < 10; seed++ {
				fs := newCutFS(t, dir)
				fs.arm(cut, mode, seed)
				if err := NewControlFile(fs, dir).Create(sampleControl(0)); !errors.Is(err, errCut) {
					t.Fatalf("%s/%d/%d: got %v, want errCut", name, cut, seed, err)
				}
				f := NewControlFile(fs, dir)
				got, err := f.Read()
				if err == nil {
					if got != withSeq(sampleControl(0), 1) && got != withSeq(sampleControl(0), 2) {
						t.Fatalf("%s/%d/%d: got %+v", name, cut, seed, got)
					}
					continue
				}
				if err := f.Create(sampleControl(7)); err != nil {
					t.Fatalf("%s/%d/%d: create after crash: %v", name, cut, seed, err)
				}
				if got, err := NewControlFile(fs, dir).Read(); err != nil || got != withSeq(sampleControl(7), 2) {
					t.Fatalf("%s/%d/%d: got %+v, %v", name, cut, seed, got, err)
				}
			}
		}
	}
}

func TestControlCreate(t *testing.T) {
	dir, f := newControl(t)
	if err := f.Create(sampleControl(1)); !errors.Is(err, ErrExists) {
		t.Fatalf("got %v, want ErrExists", err)
	}
	if err := NewControlFile(vfs.OS(), dir).Create(sampleControl(1)); !errors.Is(err, ErrExists) {
		t.Fatalf("got %v, want ErrExists", err)
	}

	// Un control sin copias válidas (un Create cortado) se reemplaza.
	writeFile(t, controlPath(dir), []byte("x"))
	if err := NewControlFile(vfs.OS(), dir).Create(sampleControl(5)); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, dir); got != withSeq(sampleControl(5), 2) {
		t.Fatalf("got %+v", got)
	}
}

func TestControlRejectsInvalid(t *testing.T) {
	bad := map[string]func(c *Control){
		"state 0":            func(c *Control) { c.State = 0 },
		"state 9":            func(c *Control) { c.State = 9 },
		"checkpoint < redo":  func(c *Control) { c.CheckpointLSN = c.Redo.LSN - 1 },
		"negative offset":    func(c *Control) { c.Redo.Offset = -1 },
		"slot name too long": func(c *Control) { c.Slots[1].Name = strings.Repeat("a", SlotNameLen+1) },
		"slot name with NUL": func(c *Control) { c.Slots[1].Name = "a\x00b" },
	}
	for name, change := range bad {
		c := withSeq(sampleControl(0), 3)
		change(&c)
		if _, err := encodeControl(c); !errors.Is(err, ErrBadControl) {
			t.Fatalf("%s: encode got %v", name, err)
		}
		_, f := newControl(t)
		if err := f.Write(c); !errors.Is(err, ErrBadControl) {
			t.Fatalf("%s: write got %v", name, err)
		}
	}
	if _, err := encodeControl(sampleControl(0)); !errors.Is(err, ErrBadControl) {
		t.Fatalf("seq 0: got %v", err)
	}
}

// TestControlRejectsFields cambia campos de una copia y recalcula el CRC: la validación no es solo el CRC.
func TestControlRejectsFields(t *testing.T) {
	good, err := encodeControl(withSeq(sampleControl(0), 2))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(b []byte){
		"magic":          func(b []byte) { le.PutUint32(b[offMagic:], 1) },
		"version":        func(b []byte) { le.PutUint32(b[offVersion:], FormatVersion+1) },
		"page size":      func(b []byte) { le.PutUint32(b[offPageSize:], 4096) },
		"blocks":         func(b []byte) { le.PutUint32(b[offBlocksPerFile:], 1) },
		"reserved 1":     func(b []byte) { b[offReserved1] = 1 },
		"reserved 2":     func(b []byte) { b[offCRC-1] = 1 },
		"slot padding":   func(b []byte) { b[offSlots+1*SlotSize+5] = 'x' }, // nombre vacío y basura después
		"state":          func(b []byte) { le.PutUint32(b[offState:], 0) },
		"seq 0":          func(b []byte) { le.PutUint64(b[offSeq:], 0) },
		"checkpoint lsn": func(b []byte) { le.PutUint64(b[offCheckpointLSN:], 0) },
	}
	for name, change := range cases {
		b := append([]byte(nil), good...)
		change(b)
		le.PutUint32(b[offCRC:], crc32.Checksum(b[:offCRC], castagnoli))
		if _, err := decodeControl(b); !errors.Is(err, ErrBadControl) {
			t.Fatalf("%s: got %v", name, err)
		}
	}

	// Una copia válida en el lugar equivocado (seq par en la B) no cuenta.
	dir, _ := newControl(t)
	file := make([]byte, 2*CopySize)
	copy(file[CopySize:], good)
	writeFile(t, controlPath(dir), file)
	if _, err := NewControlFile(vfs.OS(), dir).Read(); !errors.Is(err, ErrNoValidCopy) {
		t.Fatalf("wrong place: got %v", err)
	}
}

func TestControlConcurrent(t *testing.T) {
	dir, f := newControl(t)
	const writers, writes = 8, 20
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(2)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				if err := f.Write(sampleControl(int64(w*writes + i))); err != nil {
					t.Error(err)
					return
				}
			}
		}(w)
		go func() {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				if _, err := f.Read(); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if got := mustRead(t, dir); got.Seq != 2+writers*writes {
		t.Fatalf("seq %d, want %d", got.Seq, 2+writers*writes)
	}
}
