package control

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// sampleCatalog tiene nombres límite (1 y 255 bytes) y UTF-8 de varios bytes por carácter.
func sampleCatalog() Catalog {
	return Catalog{
		NextFileNo: 10,
		CountLSN:   12345,
		Entries: []Entry{
			{Name: "a", File: 1, MetaBlock: 0, Count: 7, Flags: 0},
			{Name: strings.Repeat("n", MaxNameLen), File: 9, MetaBlock: 3, Count: math.MaxInt64, Flags: 0xffffffff},
			{Name: "colección ñ €", File: 4, Count: 0},
		},
	}
}

func catalogPath(dir string) string { return filepath.Join(dir, CatalogFileName) }

func TestCatalogRoundTrip(t *testing.T) {
	for _, c := range []Catalog{sampleCatalog(), {NextFileNo: 1, Entries: []Entry{}}} {
		b, err := encodeCatalog(c)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeCatalog(b)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, c) {
			t.Fatalf("got %+v, want %+v", got, c)
		}
	}
}

func TestCatalogRejectsInvalid(t *testing.T) {
	bad := map[string]func(c *Catalog){
		"next file 0":    func(c *Catalog) { c.NextFileNo = 0 },
		"empty name":     func(c *Catalog) { c.Entries[0].Name = "" },
		"name 256 bytes": func(c *Catalog) { c.Entries[0].Name = strings.Repeat("x", MaxNameLen+1) },
		"file 0":         func(c *Catalog) { c.Entries[0].File = 0 },
		"file >= next":   func(c *Catalog) { c.Entries[0].File = c.NextFileNo },
		"negative count": func(c *Catalog) { c.Entries[0].Count = -1 },
		"dup name":       func(c *Catalog) { c.Entries[1].Name = c.Entries[0].Name },
		"dup file":       func(c *Catalog) { c.Entries[1].File = c.Entries[0].File },
	}
	for name, change := range bad {
		c := sampleCatalog()
		change(&c)
		if _, err := encodeCatalog(c); !errors.Is(err, ErrBadCatalog) {
			t.Fatalf("%s: encode got %v", name, err)
		}
		if _, err := LoadStores(c); !errors.Is(err, ErrBadCatalog) {
			t.Fatalf("%s: LoadStores got %v", name, err)
		}
		if err := NewCatalogFile(vfs.OS(), t.TempDir()).Save(c); !errors.Is(err, ErrBadCatalog) {
			t.Fatalf("%s: Save got %v", name, err)
		}
	}
}

// TestCatalogRejectsBytes cambia bytes y recalcula el CRC: la validación no es solo el CRC.
func TestCatalogRejectsBytes(t *testing.T) {
	good, err := encodeCatalog(sampleCatalog())
	if err != nil {
		t.Fatal(err)
	}
	first := CatalogHeaderSize
	cases := map[string]func(b []byte) []byte{
		"magic":           func(b []byte) []byte { b[0] ^= 1; return b },
		"version":         func(b []byte) []byte { b[catOffVersion] = FormatVersion + 1; return b },
		"reserved header": func(b []byte) []byte { b[catOffReserved] = 1; return b },
		"count too big":   func(b []byte) []byte { le.PutUint32(b[catOffCount:], math.MaxUint32); return b },
		"count +1":        func(b []byte) []byte { le.PutUint32(b[catOffCount:], 4); return b },
		"count -1":        func(b []byte) []byte { le.PutUint32(b[catOffCount:], 2); return b },
		"name len 0":      func(b []byte) []byte { le.PutUint16(b[first+entOffNameLen:], 0); return b },
		"name len 256":    func(b []byte) []byte { le.PutUint16(b[first+entOffNameLen:], 256); return b },
		"name len past":   func(b []byte) []byte { le.PutUint16(b[first+entOffNameLen:], 255); return b },
		"reserved entry":  func(b []byte) []byte { b[first+entOffReserved] = 1; return b },
		"negative count":  func(b []byte) []byte { b[first+entOffCount+7] = 0x80; return b },
		"extra byte":      func(b []byte) []byte { return append(b[:len(b)-crcSize:len(b)-crcSize], 0, 0, 0, 0, 0) },
		"truncated":       func(b []byte) []byte { return append(b[:len(b)-crcSize-3:len(b)-crcSize-3], 0, 0, 0, 0) },
	}
	for name, change := range cases {
		b := change(append([]byte(nil), good...))
		le.PutUint32(b[len(b)-crcSize:], crcOf(b[:len(b)-crcSize]))
		if _, err := decodeCatalog(b); !errors.Is(err, ErrBadCatalog) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
	for n := 0; n < CatalogHeaderSize+crcSize; n++ {
		if _, err := decodeCatalog(good[:n]); !errors.Is(err, ErrBadCatalog) {
			t.Fatalf("length %d: got %v", n, err)
		}
	}
}

func TestCatalogEveryByteCorrupted(t *testing.T) {
	dir := t.TempDir()
	f := NewCatalogFile(vfs.OS(), dir)
	if err := f.Save(sampleCatalog()); err != nil {
		t.Fatal(err)
	}
	orig := readFile(t, catalogPath(dir))
	for i := range orig {
		b := append([]byte(nil), orig...)
		b[i] ^= 0x10
		if _, err := decodeCatalog(b); !errors.Is(err, ErrBadCatalog) {
			t.Fatalf("byte %d: got %v", i, err)
		}
	}
	writeFile(t, catalogPath(dir), append(orig[:10:10], 1))
	if _, err := f.Load(); !errors.Is(err, ErrBadCatalog) {
		t.Fatalf("Load: got %v", err)
	}
}

func TestCatalogSaveLoad(t *testing.T) {
	dir := t.TempDir()
	f := NewCatalogFile(vfs.OS(), dir)
	if _, err := f.Load(); err == nil {
		t.Fatal("missing catalog: want an error")
	}
	want := sampleCatalog()
	if err := f.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := NewCatalogFile(vfs.OS(), dir).Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}

	// Un catalog.tmp sobrante (Save cortado antes del rename) se ignora y se borra.
	writeFile(t, filepath.Join(dir, CatalogTempName), []byte("a medio escribir"))
	got, err = f.Load()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, CatalogTempName)); !os.IsNotExist(err) {
		t.Fatalf("catalog.tmp still there: %v", err)
	}
}

// TestCatalogSaveCut: un Save que falla en el Truncate, la escritura o el Sync de catalog.tmp deja el catálogo
// anterior (también tras un corte) y no reintenta.
func TestCatalogSaveCut(t *testing.T) {
	const dir = "/db"
	for n := 0; n < 3; n++ { // Truncate, WriteAt y Sync de catalog.tmp
		fs := newCutFS(t, dir)
		f := NewCatalogFile(fs, dir)
		old := sampleCatalog()
		if err := f.Save(old); err != nil {
			t.Fatal(err)
		}
		next := sampleCatalog()
		next.Entries = next.Entries[:1]
		fs.FailAfter(n)
		if err := f.Save(next); !errors.Is(err, crashfs.ErrIO) {
			t.Fatalf("fail at %d: got %v", n, err)
		}
		fs.FailAfter(-1)
		got, err := NewCatalogFile(fs, dir).Load()
		if err != nil || !reflect.DeepEqual(got, old) {
			t.Fatalf("fail at %d: got %+v, %v", n, got, err)
		}
		if err := f.Save(next); !errors.Is(err, ErrFailed) {
			t.Fatalf("fail at %d: got %v, want ErrFailed", n, err)
		}
		fs.Crash(crashfs.ReorderUnsynced, int64(n))
		got, err = NewCatalogFile(fs, dir).Load()
		if err != nil || !reflect.DeepEqual(got, old) {
			t.Fatalf("fail at %d, after crash: got %+v, %v", n, got, err)
		}
	}
}

// TestCatalogSaveDiskFull: un Save que no encuentra lugar para catalog.tmp deja el catálogo anterior y no queda
// fallado: con lugar, el siguiente Save anda.
func TestCatalogSaveDiskFull(t *testing.T) {
	const dir = "/db"
	fs := newCutFS(t, dir)
	f := NewCatalogFile(fs, dir)
	old := sampleCatalog()
	if err := f.Save(old); err != nil {
		t.Fatal(err)
	}
	next := sampleCatalog()
	next.Entries = next.Entries[:1]
	fs.SetCapacity(fs.Used() + 1)
	if err := f.Save(next); !vfs.IsNoSpace(err) || errors.Is(err, ErrFailed) {
		t.Fatalf("Save with the disk full: %v", err)
	}
	if got, err := NewCatalogFile(fs, dir).Load(); err != nil || !reflect.DeepEqual(got, old) {
		t.Fatalf("after the full Save: %+v, %v", got, err)
	}
	fs.SetCapacity(0)
	if err := f.Save(next); err != nil {
		t.Fatalf("Save with room: %v", err)
	}
	if got, err := NewCatalogFile(fs, dir).Load(); err != nil || !reflect.DeepEqual(got, next) {
		t.Fatalf("after the Save with room: %+v, %v", got, err)
	}
}

// TestCatalogCrashDuringSave corta la luz en el Sync de catalog.tmp y en el SyncDir tras el rename, en los tres
// modos: queda el catálogo anterior o el nuevo, entero; antes del rename, siempre el anterior. Un Save confirmado
// sobrevive al corte.
func TestCatalogCrashDuringSave(t *testing.T) {
	const dir = "/db"
	old := sampleCatalog()
	next := sampleCatalog()
	next.Entries = next.Entries[:1]
	for name, mode := range crashModes {
		for cut := 0; cut < 2; cut++ { // 0: Sync de catalog.tmp; 1: SyncDir tras el rename
			for seed := int64(0); seed < 10; seed++ {
				fs := newCutFS(t, dir)
				if err := NewCatalogFile(fs, dir).Save(old); err != nil {
					t.Fatal(err)
				}
				fs.arm(cut, mode, seed)
				if err := NewCatalogFile(fs, dir).Save(next); !errors.Is(err, errCut) {
					t.Fatalf("%s/%d/%d: got %v, want errCut", name, cut, seed, err)
				}
				f := NewCatalogFile(fs, dir)
				got, err := f.Load()
				if err != nil {
					t.Fatalf("%s/%d/%d: load after crash: %v", name, cut, seed, err)
				}
				if !reflect.DeepEqual(got, old) && (cut == 0 || !reflect.DeepEqual(got, next)) {
					t.Fatalf("%s/%d/%d: got %+v", name, cut, seed, got)
				}
				if err := f.Save(next); err != nil {
					t.Fatal(err)
				}
				fs.Crash(mode, seed)
				if got, err := NewCatalogFile(fs, dir).Load(); err != nil || !reflect.DeepEqual(got, next) {
					t.Fatalf("%s/%d/%d: confirmed save lost: %+v, %v", name, cut, seed, got, err)
				}
			}
		}
	}
}

func TestCatalogThousands(t *testing.T) {
	s := NewStores()
	const n = 5000
	for i := 0; i < n; i++ {
		file, err := s.AllocFileNo()
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("store-%05d-%s", i, strings.Repeat("x", i%200))
		if ok, err := s.Create(name, file); !ok || err != nil {
			t.Fatalf("create %s: %v %v", name, ok, err)
		}
		if _, err := s.AddCount(file, int64(i)); err != nil {
			t.Fatal(err)
		}
	}
	s.SetCountLSN(777)
	dir := t.TempDir()
	f := NewCatalogFile(vfs.OS(), dir)
	want := s.Catalog()
	if err := f.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := f.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || len(got.Entries) != n || got.NextFileNo != n+1 {
		t.Fatalf("catalog differs: %d entries, next %d", len(got.Entries), got.NextFileNo)
	}
	loaded, err := LoadStores(got)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.Catalog(), want) || loaded.Len() != n || loaded.CountLSN() != 777 {
		t.Fatal("LoadStores differs")
	}
}

func TestStoresNames(t *testing.T) {
	s := NewStores()
	for _, name := range []string{"", strings.Repeat("x", MaxNameLen+1), "a\x00b", "\xff\xfe"} {
		if _, err := s.Create(name, 1); !errors.Is(err, ErrBadName) {
			t.Fatalf("len %d: got %v", len(name), err)
		}
	}
	for i, name := range []string{"x", strings.Repeat("y", MaxNameLen)} {
		if ok, err := s.Create(name, smgr.FileNo(i+1)); !ok || err != nil {
			t.Fatalf("len %d: %v %v", len(name), ok, err)
		}
	}
	if got := s.Names(); !reflect.DeepEqual(got, []string{"x", strings.Repeat("y", MaxNameLen)}) {
		t.Fatalf("names %q", got)
	}
}

func TestStoresIdempotent(t *testing.T) {
	s := NewStores()
	if ok, _ := s.Create("a", 5); !ok {
		t.Fatal("create")
	}
	if s.NextFileNo() != 6 {
		t.Fatalf("next %d, want 6", s.NextFileNo())
	}
	if ok, _ := s.Create("a", 7); ok {
		t.Fatal("second create of a must do nothing")
	}
	if _, err := s.Create("b", 5); !errors.Is(err, ErrBadFileNo) {
		t.Fatalf("file in use: got %v", err)
	}
	if _, err := s.Create("b", 0); !errors.Is(err, ErrBadFileNo) {
		t.Fatalf("file 0: got %v", err)
	}
	if _, err := s.AddCount(5, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddCount(5, -4); !errors.Is(err, ErrNegativeCount) {
		t.Fatalf("got %v", err)
	}
	if _, err := s.AddCount(5, math.MinInt64); !errors.Is(err, ErrNegativeCount) {
		t.Fatalf("got %v", err)
	}
	if ok, _ := s.AddCount(99, 1); ok {
		t.Fatal("count of a missing file")
	}
	if !s.SetMeta(5, 2) || !s.SetFlags(5, 1) || s.SetMeta(99, 1) {
		t.Fatal("SetMeta/SetFlags")
	}

	// Vaciar: solo si el archivo es el esperado; repetirlo no hace nada.
	if _, err := s.Empty("a", 5, 5); !errors.Is(err, ErrBadFileNo) {
		t.Fatalf("same file: got %v", err)
	}
	if ok, err := s.Empty("a", 5, 8); !ok || err != nil {
		t.Fatalf("empty: %v %v", ok, err)
	}
	if e, _ := s.Get("a"); e != (Entry{Name: "a", File: 8, Flags: 1}) {
		t.Fatalf("after empty %+v", e)
	}
	if ok, _ := s.Empty("a", 5, 8); ok {
		t.Fatal("second empty must do nothing")
	}
	if _, ok := s.ByFile(5); ok {
		t.Fatal("old file still mapped")
	}

	// Borrar: solo si el archivo es el esperado.
	if _, ok := s.Drop("a", 5); ok {
		t.Fatal("drop with an old file number")
	}
	if e, ok := s.Drop("a", 8); !ok || e.File != 8 {
		t.Fatal("drop")
	}
	if _, ok := s.Drop("a", 8); ok {
		t.Fatal("second drop must do nothing")
	}
	if s.Len() != 0 || s.NextFileNo() != 9 {
		t.Fatalf("len %d next %d", s.Len(), s.NextFileNo())
	}
}

// storeOp es una operación del catálogo como la anota el WAL.
type storeOp struct {
	kind     byte // 'c' crear, 'd' borrar, 'e' vaciar
	name     string
	old, new smgr.FileNo
}

// apply aplica op a s como lo haría la recuperación.
func (op storeOp) apply(t *testing.T, s *Stores) {
	t.Helper()
	var err error
	switch op.kind {
	case 'c':
		_, err = s.Create(op.name, op.new)
	case 'd':
		s.Drop(op.name, op.old)
	case 'e':
		_, err = s.Empty(op.name, op.old, op.new)
	}
	if err != nil {
		t.Fatalf("%+v: %v", op, err)
	}
}

// TestStoresReplay: rehacer el WAL desde el punto de rehacer j sobre un catálogo guardado en cualquier momento
// i ≥ j (el catálogo se guarda al crear, borrar y en cada checkpoint) termina en el mismo catálogo.
func TestStoresReplay(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	live := NewStores()
	var ops []storeOp
	snaps := []Catalog{live.Catalog()} // snaps[i] = el catálogo tras ops[:i]
	for len(ops) < 300 {
		name := string(rune('a' + rng.Intn(4)))
		e, exists := live.Get(name)
		var op storeOp
		switch {
		case !exists:
			file, _ := live.AllocFileNo()
			op = storeOp{kind: 'c', name: name, new: file}
		case rng.Intn(2) == 0:
			op = storeOp{kind: 'd', name: name, old: e.File}
		default:
			file, _ := live.AllocFileNo()
			op = storeOp{kind: 'e', name: name, old: e.File, new: file}
		}
		op.apply(t, live)
		ops = append(ops, op)
		snaps = append(snaps, live.Catalog())
	}
	want := live.Catalog()
	for j := 0; j < len(snaps); j += 7 {
		for i := j; i < len(snaps); i += 5 {
			s, err := LoadStores(snaps[i])
			if err != nil {
				t.Fatal(err)
			}
			for _, op := range ops[j:] {
				op.apply(t, s)
			}
			if got := s.Catalog(); !reflect.DeepEqual(got, want) {
				t.Fatalf("redo from %d on catalog %d:\n got %+v\nwant %+v", j, i, got, want)
			}
		}
	}
}

func TestStoresConcurrent(t *testing.T) {
	s := NewStores()
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				name := fmt.Sprintf("s%d-%d", w, i%10)
				file, err := s.AllocFileNo()
				if err != nil {
					t.Error(err)
					return
				}
				if _, err := s.Create(name, file); err != nil {
					t.Error(err)
					return
				}
				e, _ := s.Get(name)
				_, _ = s.AddCount(e.File, 1)
				if i%3 == 0 {
					s.Drop(name, e.File)
				}
				_ = s.Catalog()
				_ = s.Names()
			}
		}(w)
	}
	wg.Wait()
	if _, err := encodeCatalog(s.Catalog()); err != nil {
		t.Fatal(err)
	}
}

func TestStoresFileNoExhausted(t *testing.T) {
	s, err := LoadStores(Catalog{NextFileNo: math.MaxUint32})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AllocFileNo(); !errors.Is(err, ErrBadFileNo) {
		t.Fatalf("got %v", err)
	}
	if _, err := s.Create("a", math.MaxUint32); !errors.Is(err, ErrBadFileNo) {
		t.Fatalf("got %v", err)
	}
}
