package page

import (
	"bytes"
	"errors"
	"hash/crc32"
	"math/rand/v2"
	"testing"
)

// newPage devuelve una página iniciada con espacio especial de árbol.
func newPage(t testing.TB) Page {
	t.Helper()
	p := make(Page, Size)
	if err := p.Init(OpaqueSize); err != nil {
		t.Fatal(err)
	}
	return p
}

// mustItem devuelve el ítem off o falla.
func mustItem(t testing.TB, p Page, off OffsetNumber) []byte {
	t.Helper()
	b, err := p.Item(off)
	if err != nil {
		t.Fatalf("Item(%d): %v", off, err)
	}
	return b
}

func TestInit(t *testing.T) {
	p := newPage(t)
	if p.Lower() != HeaderSize || p.Upper() != Size-OpaqueSize || p.Special() != Size-OpaqueSize {
		t.Fatalf("lower %d upper %d special %d", p.Lower(), p.Upper(), p.Special())
	}
	if p.IsNew() || p.MaxOffset() != 0 || p.SpecialSize() != OpaqueSize {
		t.Fatal("unexpected empty page state")
	}
	if p.ExactFreeSpace() != Size-HeaderSize-OpaqueSize || p.FreeSpace() != Size-HeaderSize-OpaqueSize-ItemIDSize {
		t.Fatalf("free %d exact %d", p.FreeSpace(), p.ExactFreeSpace())
	}
	if err := p.Init(0); err != nil || p.Special() != Size || p.SpecialSize() != 0 {
		t.Fatalf("Init(0): %v special %d", err, p.Special())
	}
	for _, bad := range []int{-8, 4, Size} {
		if err := p.Init(bad); !errors.Is(err, ErrSpecialSize) {
			t.Fatalf("Init(%d): %v", bad, err)
		}
	}
	if err := Page(make([]byte, 100)).Init(0); !errors.Is(err, ErrSize) {
		t.Fatalf("short page: %v", err)
	}
}

func TestHeaderFields(t *testing.T) {
	p := newPage(t)
	p.SetLSN(0x0102030405060708)
	p.SetPruneXID(42)
	p.SetFlags(FlagAllVisible)
	if p.LSN() != 0x0102030405060708 || p.PruneXID() != 42 || p.Flags() != FlagAllVisible {
		t.Fatal("header fields do not round trip")
	}
	if p[0] != 0x08 || p[OffPruneXID] != 42 {
		t.Fatal("header is not little-endian")
	}
}

func TestChecksum(t *testing.T) {
	p := newPage(t)
	if _, err := p.AddItem([]byte("hola")); err != nil {
		t.Fatal(err)
	}
	p.SetChecksum(3)
	if err := Verify(p, 3); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := Verify(p, 4); !errors.Is(err, ErrChecksum) {
		t.Fatalf("another block: %v", err)
	}
	// La suma no depende de lo que haya en pd_checksum.
	sum := p.Checksum()
	p[OffChecksum] ^= 0xFF
	if p.ComputeChecksum(3) != sum {
		t.Fatal("checksum depends on pd_checksum")
	}
	p[OffChecksum] ^= 0xFF
	// Un bit cambiado en cualquier lugar se nota.
	for i := 0; i < Size; i += 97 {
		if i == OffChecksum || i == OffChecksum+1 {
			continue
		}
		p[i] ^= 0x10
		if Verify(p, 3) == nil {
			t.Fatalf("flip at byte %d not detected", i)
		}
		p[i] ^= 0x10
	}
}

func TestVerifyNewPage(t *testing.T) {
	p := make(Page, Size)
	if err := Verify(p, 9); err != nil {
		t.Fatalf("zero page: %v", err)
	}
	p[100] = 1
	if err := Verify(p, 9); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("pd_upper 0 with data: %v", err)
	}
	if err := Verify(make([]byte, Size-1), 0); !errors.Is(err, ErrSize) {
		t.Fatalf("short: %v", err)
	}
}

func TestVerifyHeader(t *testing.T) {
	cases := map[string]func(p Page){
		"version":       func(p Page) { p.setU16(OffPageSizeVersion, Size|2) },
		"flags":         func(p Page) { p.SetFlags(0x80) },
		"lower<header":  func(p Page) { p.setU16(OffLower, 8) },
		"lower>upper":   func(p Page) { p.setU16(OffLower, p.Upper()+4) },
		"upper>special": func(p Page) { p.setU16(OffUpper, Size) },
		"special>size":  func(p Page) { p.setU16(OffSpecial, Size+8) },
		"special align": func(p Page) { p.setU16(OffSpecial, Size-4); p.setU16(OffUpper, Size-4) },
		"upper is zero": func(p Page) { p.setU16(OffUpper, 0) },
	}
	for name, damage := range cases {
		p := newPage(t)
		damage(p)
		p.SetChecksum(1)
		if err := Verify(p, 1); !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAddAndInsert(t *testing.T) {
	p := newPage(t)
	for _, s := range []string{"b", "d"} {
		if _, err := p.AddItem([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.InsertItem([]byte("a"), 1); err != nil {
		t.Fatal(err)
	}
	if err := p.InsertItem([]byte("c"), 3); err != nil {
		t.Fatal(err)
	}
	if err := p.InsertItem([]byte("e"), 5); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"a", "b", "c", "d", "e"} {
		got := mustItem(t, p, OffsetNumber(i+1))
		if string(got) != want {
			t.Fatalf("item %d = %q, want %q", i+1, got, want)
		}
		if id, _ := p.ItemID(OffsetNumber(i + 1)); id.Off()%ItemAlign != 0 {
			t.Fatalf("item %d not aligned: %d", i+1, id.Off())
		}
	}
	for _, off := range []OffsetNumber{0, 7} {
		if err := p.InsertItem([]byte("x"), off); !errors.Is(err, ErrInvalidOffset) {
			t.Fatalf("InsertItem(%d): %v", off, err)
		}
	}
	if _, err := p.AddItem(nil); !errors.Is(err, ErrEmptyItem) {
		t.Fatalf("empty item: %v", err)
	}
	// El slice devuelto no deja escribir sobre el ítem vecino con append.
	b := mustItem(t, p, 1)
	if cap(b) != len(b) {
		t.Fatalf("cap %d != len %d", cap(b), len(b))
	}
}

func TestNoSpace(t *testing.T) {
	p := newPage(t)
	item := make([]byte, 1000)
	n := 0
	for {
		_, err := p.AddItem(item)
		if errors.Is(err, ErrNoSpace) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if want := (Size - HeaderSize - OpaqueSize) / SpaceFor(1000); n != want {
		t.Fatalf("fit %d items, want %d", n, want)
	}
	if p.Flags()&FlagPageFull != 0 {
		t.Fatal("AddItem must not set FlagPageFull")
	}
	if _, err := p.AddItem(make([]byte, Size)); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("huge item: %v", err)
	}
	// Un ítem que llena justo el espacio entra.
	q := newPage(t)
	if _, err := q.AddItem(make([]byte, q.FreeSpace()&^(ItemAlign-1))); err != nil || q.FreeSpace() != 0 {
		t.Fatalf("exact fit: %v free %d", err, q.ExactFreeSpace())
	}
}

func TestReplace(t *testing.T) {
	p := newPage(t)
	for _, s := range []string{"uno", "dos", "tres"} {
		if _, err := p.AddItem([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	free := p.ExactFreeSpace()
	big := bytes.Repeat([]byte("D"), 20)
	if err := p.ReplaceItem(2, big); err != nil {
		t.Fatal(err)
	}
	if p.ExactFreeSpace() != free-(Align(20)-Align(3)) {
		t.Fatalf("free %d after grow", p.ExactFreeSpace())
	}
	if err := p.ReplaceItem(2, []byte("d")); err != nil {
		t.Fatal(err)
	}
	if p.ExactFreeSpace() != free {
		t.Fatalf("free %d after shrink, want %d", p.ExactFreeSpace(), free)
	}
	for i, want := range []string{"uno", "d", "tres"} {
		if got := mustItem(t, p, OffsetNumber(i+1)); string(got) != want {
			t.Fatalf("item %d = %q", i+1, got)
		}
	}
	if err := p.ReplaceItem(1, make([]byte, Size)); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("too big: %v", err)
	}
	_ = p.SetUnused(3)
	if err := p.ReplaceItem(3, []byte("x")); !errors.Is(err, ErrNoItem) {
		t.Fatalf("unused: %v", err)
	}
	if err := p.CheckItems(); err != nil {
		t.Fatal(err)
	}
}

func TestDelete(t *testing.T) {
	p := newPage(t)
	for _, s := range []string{"a", "bb", "ccc", "dddd"} {
		if _, err := p.AddItem([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.DeleteItem(2); err != nil {
		t.Fatal(err)
	}
	if err := p.DeleteItems([]OffsetNumber{1, 3}); err != nil {
		t.Fatal(err)
	}
	if p.MaxOffset() != 1 || string(mustItem(t, p, 1)) != "ccc" {
		t.Fatalf("left %d items", p.MaxOffset())
	}
	if p.ExactFreeSpace() != Size-OpaqueSize-HeaderSize-SpaceFor(3) {
		t.Fatalf("free %d", p.ExactFreeSpace())
	}
	if err := p.DeleteItems([]OffsetNumber{1, 1}); !errors.Is(err, ErrUnsorted) {
		t.Fatalf("unsorted: %v", err)
	}
	if err := p.DeleteItem(2); !errors.Is(err, ErrInvalidOffset) {
		t.Fatalf("missing: %v", err)
	}
}

func TestUnusedAndRepair(t *testing.T) {
	p := newPage(t)
	for _, s := range []string{"a", "b", "c"} {
		if _, err := p.AddItem([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	free := p.ExactFreeSpace()
	if err := p.SetUnused(2); err != nil {
		t.Fatal(err)
	}
	if err := p.MarkDead(3); err != nil {
		t.Fatal(err)
	}
	if err := p.MarkDead(2); !errors.Is(err, ErrNoItem) {
		t.Fatalf("MarkDead unused: %v", err)
	}
	if p.Flags()&FlagHasFreeLines == 0 || p.ExactFreeSpace() != free {
		t.Fatal("SetUnused must only free the pointer")
	}
	if err := p.Repair(); err != nil {
		t.Fatal(err)
	}
	if p.ExactFreeSpace() != free+ItemAlign {
		t.Fatalf("Repair freed %d", p.ExactFreeSpace()-free)
	}
	if id, _ := p.ItemID(3); id.Flags() != LineDead || string(mustItem(t, p, 3)) != "c" {
		t.Fatal("Repair lost the dead item")
	}
	off, err := p.AddItem([]byte("z"))
	if err != nil || off != 2 {
		t.Fatalf("AddItem reused %d: %v", off, err)
	}
	// Con la marca prendida pero sin libres, AddItem la apaga y agrega al final.
	off, err = p.AddItem([]byte("w"))
	if err != nil || off != 4 || p.Flags()&FlagHasFreeLines != 0 {
		t.Fatalf("AddItem %d: %v flags %#x", off, err, p.Flags())
	}
}

func TestOpaque(t *testing.T) {
	p := newPage(t)
	o := Opaque{Prev: 1, Next: 2, Level: 3, Flags: OpaqueLeaf | OpaqueRoot, CycleID: 9}
	if err := p.SetOpaque(o); err != nil {
		t.Fatal(err)
	}
	got, err := p.Opaque()
	if err != nil || got != o {
		t.Fatalf("Opaque %+v: %v", got, err)
	}
	if p[Size-OpaqueSize] != 1 || p[Size-4] != byte(OpaqueLeaf|OpaqueRoot) {
		t.Fatal("opaque layout")
	}
	_ = p.Init(0)
	if _, err := p.Opaque(); !errors.Is(err, ErrNoOpaque) {
		t.Fatalf("no opaque: %v", err)
	}
	if err := p.SetOpaque(o); !errors.Is(err, ErrNoOpaque) {
		t.Fatalf("set no opaque: %v", err)
	}
}

func TestSetLower(t *testing.T) {
	p := newPage(t)
	if err := p.SetLower(HeaderSize + 48); err != nil {
		t.Fatal(err)
	}
	p.SetChecksum(0)
	if err := Verify(p, 0); err != nil {
		t.Fatalf("meta-like page: %v", err)
	}
	// Una página de desborde llena hasta pd_special, con pd_lower sin alinear.
	if err := p.SetLower(Size - OpaqueSize - 3); err != nil {
		t.Fatal(err)
	}
	p.SetChecksum(0)
	if err := Verify(p, 0); err != nil {
		t.Fatalf("overflow-like page: %v", err)
	}
	if _, err := p.AddItem([]byte("x")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("AddItem on unaligned pd_lower: %v", err)
	}
	for _, bad := range []int{HeaderSize - 1, Size} {
		if err := p.SetLower(bad); !errors.Is(err, ErrInvalidOffset) {
			t.Fatalf("SetLower(%d): %v", bad, err)
		}
	}
}

func TestCheckItemsDetects(t *testing.T) {
	build := func() Page {
		p := newPage(t)
		for _, s := range []string{"aaaa", "bbbb", "cccc"} {
			_, _ = p.AddItem([]byte(s))
		}
		return p
	}
	cases := map[string]func(p Page){
		"overlap":       func(p Page) { id := p.line(1); p.setLine(2, makeItemID(id.Off(), LineNormal, 4)) },
		"past special":  func(p Page) { p.setLine(1, makeItemID(Size-8, LineNormal, 8)) },
		"below upper":   func(p Page) { p.setLine(1, makeItemID(HeaderSize+16, LineNormal, 4)) },
		"unaligned":     func(p Page) { id := p.line(1); p.setLine(1, makeItemID(id.Off()+1, LineNormal, 2)) },
		"redirect":      func(p Page) { p.setLine(1, makeItemID(2, LineRedirect, 0)) },
		"dirty unused":  func(p Page) { p.setLine(1, makeItemID(8, LineUnused, 0)) },
		"empty pointer": func(p Page) { p.setLine(1, makeItemID(64, LineDead, 0)) },
	}
	for name, damage := range cases {
		p := build()
		damage(p)
		if err := p.CheckItems(); !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: CheckItems %v", name, err)
		}
		before := bytes.Clone(p)
		if err := p.Repair(); !errors.Is(err, ErrCorrupt) || !bytes.Equal(before, p) {
			t.Errorf("%s: Repair %v (must fail without changes)", name, err)
		}
	}
}

// TestGarbageNoPanic recorre páginas con encabezado válido y punteros al azar: nada entra en pánico.
func TestGarbageNoPanic(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 200; i++ {
		b := make([]byte, Size)
		for j := range b {
			b[j] = byte(r.Uint32())
		}
		p := Page(b)
		if i%2 == 0 {
			p.setU16(OffPageSizeVersion, pageSizeVersion)
			p.SetFlags(0)
			p.setU16(OffLower, HeaderSize+ItemIDSize*r.IntN(40))
			p.setU16(OffUpper, Align(Size/2+r.IntN(Size/2-OpaqueSize)))
			p.setU16(OffSpecial, Size-OpaqueSize)
		}
		exercise(p)
	}
}

// TestRandomOps corre el modelo con operaciones al azar, sin fuzzer.
func TestRandomOps(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for i := 0; i < 200; i++ {
		ops := make([]byte, 4*400)
		for j := range ops {
			ops[j] = byte(r.Uint32())
		}
		for j := 2; j < len(ops); j += 4 {
			ops[j] %= 2 // ítems de hasta ~512 B, para que entren muchos
		}
		runOps(t, ops)
	}
}

func TestNoAllocs(t *testing.T) {
	p := newPage(t)
	item := make([]byte, 40)
	allocs := testing.AllocsPerRun(100, func() {
		_ = p.Init(OpaqueSize)
		for range 20 {
			_, _ = p.AddItem(item)
		}
		_ = p.ReplaceItem(3, item[:10])
		_ = p.DeleteItem(5)
		_ = p.SetUnused(2)
		_ = p.Repair()
		_ = p.CheckItems()
		_, _ = p.Item(4)
		p.SetChecksum(5)
		_ = Verify(p, 5)
	})
	if allocs != 0 {
		t.Fatalf("%v allocations per run", allocs)
	}
}

/**
* exercise llama a todo lo que lee o cambia una página, sobre copias, sin mirar los errores: solo no debe entrar en
* pánico. Si CheckItems acepta la página, Repair debe andar y conservar los ítems.
* @param p Page: bytes cualesquiera de largo Size
**/
func exercise(p Page) {
	_ = Verify(p, 1)
	_ = p.IsNew()
	_, _ = p.Opaque()
	_, _ = p.FreeSpace(), p.SpecialSize()
	for off := OffsetNumber(0); off <= p.MaxOffset()+1; off++ {
		_, _ = p.Item(off)
		_, _ = p.ItemID(off)
	}
	checked := p.CheckItems() == nil
	var buf [MaxLines]stored
	_, _, serr := p.storedItems(&buf)
	if slow := p.checkLayout() == nil && serr == nil; slow != checked {
		panic("CheckItems disagrees with storedItems")
	}
	q := Page(bytes.Clone(p))
	if err := q.Repair(); checked && err != nil {
		panic("Repair failed on a page CheckItems accepted: " + err.Error())
	}
	if checked {
		if err := q.CheckItems(); err != nil {
			panic("Repair left a bad page: " + err.Error())
		}
		for off := FirstOffset; off <= p.MaxOffset(); off++ {
			a, _ := p.Item(off)
			b, _ := q.Item(off)
			if !bytes.Equal(a, b) {
				panic("Repair changed an item")
			}
		}
	}
	for off := OffsetNumber(0); off <= min(p.MaxOffset()+1, 4); off++ {
		q = Page(bytes.Clone(p))
		_, _ = q.AddItem([]byte("abc"))
		_ = q.InsertItem([]byte("abcdefghijk"), off)
		_ = q.ReplaceItem(off, []byte("x"))
		_ = q.ReplaceItem(off, make([]byte, 300))
		_ = q.DeleteItem(off)
		_ = q.MarkDead(off)
		_ = q.SetUnused(off)
		_ = q.DeleteItems([]OffsetNumber{1, off + 1})
		_ = q.Repair()
		_ = q.SetLower(int(off) * 8)
		_ = q.SetOpaque(Opaque{Next: 1})
		for o := OffsetNumber(0); o <= q.MaxOffset(); o++ {
			_, _ = q.Item(o)
		}
	}
}

// TestChecksumReference compara la suma con CRC-32C calculada de una vez sobre la página y el bloque.
func TestChecksumReference(t *testing.T) {
	p := newPage(t)
	_, _ = p.AddItem([]byte("referencia"))
	p.SetChecksum(0)
	buf := append(bytes.Clone(p), 0x78, 0x56, 0x34, 0x12)
	buf[OffChecksum], buf[OffChecksum+1] = 0, 0
	c := crc32.Checksum(buf, crc32.MakeTable(crc32.Castagnoli))
	if got, want := p.ComputeChecksum(0x12345678), uint16(c)^uint16(c>>16); got != want {
		t.Fatalf("checksum %#x, want %#x", got, want)
	}
}
