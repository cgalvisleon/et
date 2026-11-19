package overflow

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// inlineLimit es el valor más grande que va en línea en una hoja con un id de 1 byte (btree.MaxEntrySize - 28 - 1).
const inlineLimit = 2032 - 28 - 1

// TestRoundTrip: valores desde 1 B por encima del límite en línea hasta varios MB vuelven iguales, con la cantidad
// de páginas justa. La caché es chica, así que los valores grandes se leen del disco (con su suma).
func TestRoundTrip(t *testing.T) {
	e := newEnv(t, 64)
	rng := rand.New(rand.NewSource(1))
	sizes := []int{1, inlineLimit + 1, ChunkSize - 1, ChunkSize, ChunkSize + 1, 2 * ChunkSize, 3*ChunkSize + 5, 1 << 20, 3<<20 + 7}
	refs := make([]Ref, len(sizes))
	values := make([][]byte, len(sizes))
	for i, n := range sizes {
		values[i] = randomValue(rng, n)
		ref, err := e.ov.Write(testFile, 7, values[i])
		if err != nil {
			t.Fatalf("Write(%d B): %v", n, err)
		}
		if ref.Len != uint32(n) {
			t.Fatalf("Write(%d B): Len = %d", n, ref.Len)
		}
		if got := len(chainBlocks(t, e.pool, ref)); got != pagesOf(ref.Len) {
			t.Fatalf("Write(%d B): %d páginas, se esperaban %d", n, got, pagesOf(ref.Len))
		}
		refs[i] = ref
	}
	for i, ref := range refs {
		got, err := e.ov.Read(testFile, ref)
		if err != nil {
			t.Fatalf("Read(%d B): %v", sizes[i], err)
		}
		if !bytes.Equal(got, values[i]) {
			t.Fatalf("Read(%d B): el valor no coincide", sizes[i])
		}
	}
}

// TestMaxValue: un valor de MaxValueLen (64 MiB) va y vuelve; uno más grande o vacío se rechaza.
func TestMaxValue(t *testing.T) {
	e := newEnv(t, 256)
	value := randomValue(rand.New(rand.NewSource(2)), MaxValueLen)
	ref, err := e.ov.Write(testFile, 7, value)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.ov.Read(testFile, ref)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, value) {
		t.Fatal("el valor de 64 MiB no coincide")
	}
	for _, n := range []int{0, MaxValueLen + 1} {
		if _, err := e.ov.Write(testFile, 7, make([]byte, n)); !errors.Is(err, ErrBadLen) {
			t.Fatalf("Write(%d B) = %v, se esperaba ErrBadLen", n, err)
		}
	}
}

// TestBadRef: una referencia imposible se rechaza sin tocar páginas.
func TestBadRef(t *testing.T) {
	e := newEnv(t, 16)
	for _, ref := range []Ref{{First: 0, Len: 10}, {First: 1, Len: 0}, {First: 1, Len: MaxValueLen + 1}} {
		if _, err := e.ov.Read(testFile, ref); !errors.Is(err, ErrBadRef) {
			t.Fatalf("Read(%+v) = %v, se esperaba ErrBadRef", ref, err)
		}
		if err := e.ov.Free(testFile, ref, 7); !errors.Is(err, ErrBadRef) {
			t.Fatalf("Free(%+v) = %v, se esperaba ErrBadRef", ref, err)
		}
	}
}

// TestFreeAndReuse: liberar devuelve exactamente las páginas de la cadena a la lista, y la próxima cadena las usa
// sin extender el archivo.
func TestFreeAndReuse(t *testing.T) {
	e := newEnv(t, 64)
	rng := rand.New(rand.NewSource(3))
	a, err := e.ov.Write(testFile, 7, randomValue(rng, 5*ChunkSize-3))
	if err != nil {
		t.Fatal(err)
	}
	blocksA := chainBlocks(t, e.pool, a)
	if err := e.ov.Free(testFile, a, 8); err != nil {
		t.Fatal(err)
	}
	list := freeList(t, e.pool)
	if !sameSet(list, blocksA) {
		t.Fatalf("lista de libres %v, se esperaban las páginas de la cadena %v", list, blocksA)
	}
	before, err := e.sm.NBlocks(testFile, smgr.ForkMain)
	if err != nil {
		t.Fatal(err)
	}

	value := randomValue(rng, 4*ChunkSize-9)
	b, err := e.ov.Write(testFile, 9, value)
	if err != nil {
		t.Fatal(err)
	}
	if blocksB := chainBlocks(t, e.pool, b); !isSubset(blocksB, blocksA) {
		t.Fatalf("la cadena nueva %v no reutiliza las páginas libres %v", blocksB, blocksA)
	}
	if after, _ := e.sm.NBlocks(testFile, smgr.ForkMain); after != before {
		t.Fatalf("el archivo creció de %d a %d bloques con páginas libres disponibles", before, after)
	}
	if got := freeList(t, e.pool); len(got) != 1 {
		t.Fatalf("lista de libres %v, se esperaba una página", got)
	}
	got, err := e.ov.Read(testFile, b)
	if err != nil || !bytes.Equal(got, value) {
		t.Fatalf("Read tras reutilizar: %v", err)
	}
}

// TestDamagedChains: un bucle, un corte, un orden cambiado, una página ajena o un largo que miente devuelven
// ErrBadChain en Read y en Free, y Free no libera nada.
func TestDamagedChains(t *testing.T) {
	e := newEnv(t, 64)
	value := randomValue(rand.New(rand.NewSource(4)), 4*ChunkSize-100)
	ref, err := e.ov.Write(testFile, 7, value)
	if err != nil {
		t.Fatal(err)
	}
	blocks := chainBlocks(t, e.pool, ref)
	other, err := e.ov.Write(testFile, 7, randomValue(rand.New(rand.NewSource(5)), 2*ChunkSize))
	if err != nil {
		t.Fatal(err)
	}

	// setOpaque cambia el espacio especial de la página dañada.
	setOpaque := func(change func(o *page.Opaque)) func(page.Page) {
		return func(p page.Page) {
			o, _ := p.Opaque()
			change(&o)
			p.SetOpaque(o)
		}
	}
	cases := []struct {
		name   string
		ref    Ref
		damage int // página de la cadena que se daña (-1: ninguna)
		change func(page.Page)
	}{
		{"bucle al principio", ref, 3, setOpaque(func(o *page.Opaque) { o.Next = blocks[0] })},
		{"bucle con largo mayor", Ref{First: ref.First, Len: 8 * ChunkSize}, 3, setOpaque(func(o *page.Opaque) { o.Next = blocks[1] })},
		{"bucle sobre sí misma", ref, 1, setOpaque(func(o *page.Opaque) { o.Next = blocks[1] })},
		{"corte", ref, 1, setOpaque(func(o *page.Opaque) { o.Next = 0 })},
		{"salto a otra cadena", ref, 1, setOpaque(func(o *page.Opaque) { o.Next = other.First })},
		{"orden cambiado", ref, 2, setOpaque(func(o *page.Opaque) { o.Level = 3 })},
		{"no es de desborde", ref, 2, setOpaque(func(o *page.Opaque) { o.Flags = page.OpaqueLeaf })},
		{"trozo más corto", ref, 0, func(p page.Page) { p.SetLower(page.HeaderSize + 10) }},
		{"página nueva", ref, 2, func(p page.Page) { clear(p) }},
		{"largo +1", Ref{First: ref.First, Len: ref.Len + 1}, -1, nil},
		{"largo -1", Ref{First: ref.First, Len: ref.Len - 1}, -1, nil},
		{"largo de una página más", Ref{First: ref.First, Len: ref.Len + ChunkSize}, -1, nil},
		{"largo de una página menos", Ref{First: ref.First, Len: ref.Len - ChunkSize}, -1, nil},
		{"largo máximo", Ref{First: ref.First, Len: MaxValueLen}, -1, nil},
		{"empieza en la meta", Ref{First: 0, Len: ref.Len}, -1, nil},
		{"empieza en la segunda", Ref{First: blocks[1], Len: ref.Len}, -1, nil},
		{"empieza fuera del archivo", Ref{First: 100000, Len: ref.Len}, -1, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.damage >= 0 {
				saved := readPage(t, e.pool, blocks[c.damage])
				damaged := append(page.Page(nil), saved...)
				c.change(damaged)
				writePage(t, e.pool, blocks[c.damage], damaged)
				defer writePage(t, e.pool, blocks[c.damage], saved)
			}
			if _, err := e.ov.Read(testFile, c.ref); err == nil {
				t.Fatal("Read de una cadena dañada no devolvió error")
			}
			if err := e.ov.Free(testFile, c.ref, 7); err == nil {
				t.Fatal("Free de una cadena dañada no devolvió error")
			}
			if list := freeList(t, e.pool); len(list) != 0 {
				t.Fatalf("Free de una cadena dañada liberó %v", list)
			}
		})
	}
	if got, err := e.ov.Read(testFile, ref); err != nil || !bytes.Equal(got, value) {
		t.Fatalf("la cadena restaurada no se lee: %v", err)
	}
}

// TestGarbageNoPanic: bytes al azar en las páginas de una cadena nunca provocan un pánico: Read devuelve un error o
// un valor del largo pedido (un trozo cambiado en la caché no se detecta: lo cubre la suma al leer del disco).
func TestGarbageNoPanic(t *testing.T) {
	e := newEnv(t, 64)
	rng := rand.New(rand.NewSource(6))
	ref, err := e.ov.Write(testFile, 7, randomValue(rng, 3*ChunkSize+1))
	if err != nil {
		t.Fatal(err)
	}
	blocks := chainBlocks(t, e.pool, ref)
	for range 300 {
		block := blocks[rng.Intn(len(blocks))]
		saved := readPage(t, e.pool, block)
		damaged := append(page.Page(nil), saved...)
		for range 1 + rng.Intn(4) {
			// La mitad de las veces en el encabezado o el espacio especial, donde están los campos.
			off := rng.Intn(page.Size)
			if rng.Intn(2) == 0 {
				off = []int{rng.Intn(page.HeaderSize), page.Size - 1 - rng.Intn(page.OpaqueSize)}[rng.Intn(2)]
			}
			damaged[off] = byte(rng.Intn(256))
		}
		writePage(t, e.pool, block, damaged)
		if got, err := e.ov.Read(testFile, ref); err == nil && len(got) != int(ref.Len) {
			t.Fatalf("Read sin error devolvió %d bytes, se esperaban %d", len(got), ref.Len)
		}
		writePage(t, e.pool, block, saved)
	}
}

// TestConcurrent: varias goroutines escriben, leen y liberan valores a la vez. Al final cada valor vivo se lee
// igual, las cadenas vivas no comparten páginas y la lista de libres no tiene ninguna de ellas.
func TestConcurrent(t *testing.T) {
	e := newEnv(t, 128)
	const workers, perWorker = 8, 12
	type live struct {
		ref   Ref
		value []byte
	}
	results := make([][]live, workers)
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(100 + w)))
			for i := range perWorker {
				value := randomValue(rng, inlineLimit+1+rng.Intn(6*ChunkSize))
				ref, err := e.ov.Write(testFile, page.XID(w+1), value)
				if err != nil {
					errs <- err
					return
				}
				if got, err := e.ov.Read(testFile, ref); err != nil || !bytes.Equal(got, value) {
					errs <- fmt.Errorf("worker %d valor %d: %v", w, i, err)
					return
				}
				if i%2 == 1 {
					if err := e.ov.Free(testFile, ref, page.XID(w+1)); err != nil {
						errs <- err
						return
					}
					continue
				}
				results[w] = append(results[w], live{ref, value})
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	used := map[page.BlockNumber]bool{}
	for _, lives := range results {
		for _, l := range lives {
			if got, err := e.ov.Read(testFile, l.ref); err != nil || !bytes.Equal(got, l.value) {
				t.Fatalf("Read al final: %v", err)
			}
			for _, b := range chainBlocks(t, e.pool, l.ref) {
				if used[b] {
					t.Fatalf("el bloque %d está en dos cadenas", b)
				}
				used[b] = true
			}
		}
	}
	for _, b := range freeList(t, e.pool) {
		if used[b] {
			t.Fatalf("el bloque %d está en la lista de libres y en una cadena viva", b)
		}
	}
}

// sameSet dice si a y b tienen los mismos bloques.
func sameSet(a, b []page.BlockNumber) bool {
	return len(a) == len(b) && isSubset(a, b)
}

// isSubset dice si todos los bloques de a están en b.
func isSubset(a, b []page.BlockNumber) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}
